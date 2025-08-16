package test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	rmqClient "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"
	"github.com/tianping526/eventbridge/app/job/internal/conf"
	"github.com/tianping526/eventbridge/app/job/internal/data"
	"github.com/tianping526/eventbridge/app/job/internal/data/ent"
	"github.com/tianping526/eventbridge/app/job/internal/data/entext"

	// init db driver
	_ "github.com/jackc/pgx/v4/stdlib"
)

func TestPostEventOrderly(t *testing.T) { //nolint:gocyclo
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// setup docker-compose
	var err error
	if os.Getenv("TEST_ENV") != "CI" {
		cmdUp := exec.Command("docker-compose", "-f", "docker-compose.yaml", "up", "-d")
		err = cmdUp.Run()
		if err != nil {
			t.Errorf("docker-compose up error: %v", err)
		}
		defer func() {
			cmdDown := exec.Command("docker-compose", "-f", "docker-compose.yaml", "down")
			err = cmdDown.Run()
			if err != nil {
				t.Logf("docker-compose down error: %v", err)
			}
		}()
		time.Sleep(30 * time.Second) // wait for docker-compose up
	}

	// setup app
	appInfo := &conf.AppInfo{
		Name:     "eventbridge.test",
		Version:  "",
		FlagConf: "../configs",
		Id:       "test",
	}
	rocketmqEndpoint := "127.0.0.1:8081"
	if os.Getenv("TEST_ENV") == "CI" {
		appInfo.FlagConf = "./configs"
		rocketmqEndpoint = "proxy:8081"
	}
	app, cleanup, err := wireApp(appInfo)
	if err != nil {
		t.Errorf("app wire error: %v", err)
	}
	defer cleanup()

	eg := new(errgroup.Group)
	eg.Go(func() error {
		return app.Run()
	})

	// init config
	fc := config.New(
		config.WithSource(
			file.NewSource(appInfo.FlagConf),
		),
	)
	_ = fc.Load()
	defer func() {
		_ = fc.Close()
	}()
	var bc conf.Bootstrap
	_ = fc.Value("bootstrap").Scan(&bc)

	// init ent client
	db, err := sql.Open("pgx", bc.Data.Database.Source)
	if err != nil {
		t.Errorf("ent open error: %v", err)
	}
	drv := entsql.OpenDB(bc.Data.Database.Driver, db)
	client := ent.NewClient(ent.Driver(drv))
	defer func() {
		_ = client.Close()
	}()

	// init rocketmq producer
	producer, err := rmqClient.NewProducer(
		&rmqClient.Config{
			Endpoint: rocketmqEndpoint,
			Credentials: &credentials.SessionCredentials{
				AccessKey:    "",
				AccessSecret: "",
			},
		},
		rmqClient.WithMaxAttempts(3),
	)
	if err != nil {
		t.Errorf("rocketmq producer error: %v", err)
	}

	// start producer
	err = producer.Start()
	if err != nil {
		t.Errorf("rocketmq producer start error: %v", err)
	}
	defer func() {
		err = producer.GracefulStop()
		if err != nil {
			t.Logf("rocketmq producer close error: %v", err)
		}
	}()

	busName := "Orderly"

	// Insert data bus
	_, err = client.Bus.Create().
		SetName(busName).
		SetMode(uint8(v1.BusWorkMode_BUS_WORK_MODE_ORDERLY)).
		SetSourceTopic("EBInterBusOrderly").
		SetSourceDelayTopic("EBInterDelayBusOrderly").
		SetTargetExpDecayTopic("EBInterTargetExpDecayBusOrderly").
		SetTargetBackoffTopic("EBInterTargetBackoffBusOrderly").
		Save(context.Background())
	if err != nil {
		t.Errorf("bus create error: %v", err)
	}

	// Update busesVersion
	_, err = client.Version.UpdateOneID(data.BusesVersionID).AddVersion(1).Save(context.Background())
	if err != nil {
		t.Errorf("busesVersion update error: %v", err)
	}

	// Insert rule
	pattern := "{\"source\":[{\"prefix\":\"testSource1\"}]}"
	targets := `
		[
		  {
			"ID": 1,
			"Type": "HTTPDispatcher",
			"Params": [
			  {
				"Key": "url",
				"Form": "CONSTANT",
				"Value": "http://127.0.0.1:10188/target/event",
				"Template": null
			  },
			  {
				"Key": "method",
				"Form": "CONSTANT",
				"Value": "POST",
				"Template": null
			  },
			  {
				"Key": "body",
				"Form": "TEMPLATE",
				"Value": "{\"subject\":\"$.data.a\"}",
				"Template": "{\"code\":\"10188:${subject}\"}"
			  }
			],
			"RetryStrategy": 1
		  },
		  {
			"ID": 2,
			"Type": "HTTPDispatcher",
			"Params": [
			  {
				"Key": "url",
				"Form": "CONSTANT",
				"Value": "http://127.0.0.1:10189/target/event",
				"Template": null
			  },
			  {
				"Key": "method",
				"Form": "CONSTANT",
				"Value": "POST",
				"Template": null
			  },
			  {
				"Key": "body",
				"Form": "TEMPLATE",
				"Value": "{\"subject\":\"$.data.a\"}",
				"Template": "{\"code\":\"10189:${subject}\"}"
			  }
			],
			"RetryStrategy": 1
		  }
		]`
	_, err = client.Rule.Create().
		SetBusName(busName).
		SetName("testPostEventOrderly").
		SetStatus(uint8(v1.RuleStatus_RULE_STATUS_ENABLE)).
		SetPattern(pattern).
		SetTargets(targets).
		Save(context.Background())
	if err != nil {
		t.Errorf("rule create error: %v", err)
	}

	// Update rulesVersion
	_, err = client.Version.UpdateOneID(entext.RulesVersionID).AddVersion(1).Save(context.Background())
	if err != nil {
		t.Errorf("rulesVersion update error: %v", err)
	}

	// init http server
	var rcvRes [][2]string
	rcvResLock := sync.RWMutex{}
	mux := http.NewServeMux()
	mux.HandleFunc("/target/event", func(_ http.ResponseWriter, request *http.Request) {
		defer func(Body io.ReadCloser) {
			ce := Body.Close()
			if ce != nil {
				t.Log(ce)
			}
		}(request.Body)
		body, he := io.ReadAll(request.Body)
		if he != nil {
			t.Error(he)
		}
		parts := strings.Split(request.Host, ":")
		rcvResLock.Lock()
		rcvRes = append(rcvRes, [2]string{parts[1], string(body)})
		rcvResLock.Unlock()
	})

	server1 := &http.Server{Addr: ":10188", Handler: mux}
	eg.Go(func() error {
		_ = server1.ListenAndServe()
		return nil
	})
	server2 := &http.Server{Addr: ":10189", Handler: mux}
	eg.Go(func() error {
		_ = server2.ListenAndServe()
		return nil
	})

	// waiting job rule ready
	time.Sleep(10 * time.Second)

	// test if the event is sent successfully
	eventExt := &rule.EventExt{
		EventExt: &v1.EventExt{
			Event: &v1.Event{
				Id:              1,
				Source:          "testSource1",
				Type:            "testSourceType1",
				Time:            timestamppb.New(time.Now()),
				Datacontenttype: "application/json",
				Data:            `{"a": "i am test content"}`,
			},
			BusName:       busName,
			RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_BACKOFF,
		},
	}
	msg := &rmqClient.Message{
		Topic: "EBInterBusOrderly",
		Body:  eventExt.Value(),
	}
	msg.SetKeys(eventExt.Key())
	// same source+type to the same message group
	msg.SetMessageGroup(fmt.Sprintf("%s:%s", eventExt.Event.Source, eventExt.Event.Type))
	_, err = producer.Send(context.Background(), msg)
	if err != nil {
		t.Errorf("send event error: %v", err)
	}
	for i := 0; i < 10; i++ {
		rcvResLock.RLock()
		if len(rcvRes) >= 2 {
			rcvResLock.RUnlock()
			break
		}
		rcvResLock.RUnlock()
		if i == 9 {
			t.Errorf("waiting push timeout")
		}
		time.Sleep(time.Second)
	}
	rcvResLock.RLock()
	if len(rcvRes) != 2 {
		t.Errorf("unexpect target num: %d", len(rcvRes))
	}
	for _, rcv := range rcvRes {
		if !reflect.DeepEqual(rcv[1], fmt.Sprintf(`{"code":"%s:i am test content"}`, rcv[0])) {
			t.Errorf("unexpect body: %s", rcv[1])
		}
	}
	rcvResLock.RUnlock()
	rcvResLock.Lock()
	rcvRes = [][2]string{}
	rcvResLock.Unlock()

	// test delay event
	pubTime := time.Now().Add(time.Second * 5)
	eventExt = &rule.EventExt{
		EventExt: &v1.EventExt{
			Event: &v1.Event{
				Id:              1,
				Source:          "testSource1",
				Type:            "testSourceType1",
				Time:            timestamppb.New(pubTime),
				Datacontenttype: "application/json",
				Data:            fmt.Sprintf(`{"a": "%d"}`, pubTime.Unix()),
			},
			BusName:       busName,
			RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_BACKOFF,
		},
	}
	msg = &rmqClient.Message{
		Topic: "EBInterDelayBusOrderly",
		Body:  eventExt.Value(),
	}
	msg.SetKeys(eventExt.Key())
	msg.SetDelayTimestamp(pubTime)
	_, err = producer.Send(context.Background(), msg)
	if err != nil {
		t.Errorf("send event error: %v", err)
	}
	for i := 0; i < 10; i++ {
		rcvResLock.RLock()
		if len(rcvRes) >= 2 {
			rcvResLock.RUnlock()
			break
		}
		rcvResLock.RUnlock()
		if i == 9 {
			t.Errorf("waiting push timeout")
		}
		time.Sleep(time.Second)
	}
	rcvResLock.RLock()
	if len(rcvRes) != 2 {
		t.Errorf("unexpect target num: %d", len(rcvRes))
	}
	for _, rcv := range rcvRes {
		if !reflect.DeepEqual(rcv[1], fmt.Sprintf(`{"code":"%s:%d"}`, rcv[0], pubTime.Unix())) {
			t.Errorf("unexpect body: %s", rcv[1])
		}
	}
	now := time.Now()
	if !pubTime.Before(now) {
		t.Errorf("The current time(%s) should be after the delayed publish time(%s)",
			now, pubTime)
	}
	rcvResLock.RUnlock()
	rcvResLock.Lock()
	rcvRes = [][2]string{}
	rcvResLock.Unlock()

	// test post events orderly
	events := make([]*rule.EventExt, 0, 100)
	for i := 0; i < 100; i++ {
		events = append(events, &rule.EventExt{
			EventExt: &v1.EventExt{
				Event: &v1.Event{
					Id:              uint64(i),
					Source:          "testSource1",
					Type:            "testSourceType1",
					Time:            timestamppb.New(time.Now()),
					Datacontenttype: "application/json",
					Data:            fmt.Sprintf(`{"a": "%d"}`, i),
				},
				BusName:       busName,
				RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_BACKOFF,
			},
		})
	}
	for _, evt := range events {
		msg = &rmqClient.Message{
			Topic: "EBInterBusOrderly",
			Body:  evt.Value(),
		}
		msg.SetKeys(evt.Key())
		// same source+type to the same message group
		msg.SetMessageGroup(fmt.Sprintf("%s:%s", evt.Event.Source, evt.Event.Type))
		_, err = producer.Send(context.Background(), msg)
		if err != nil {
			t.Errorf("send event error: %v", err)
		}
	}
	for i := 0; i < 10; i++ {
		rcvResLock.RLock()
		if len(rcvRes) >= 200 {
			rcvResLock.RUnlock()
			break
		}
		rcvResLock.RUnlock()
		if i == 9 {
			t.Errorf("waiting push timeout")
		}
		time.Sleep(time.Second)
	}
	rcvResLock.RLock()
	if len(rcvRes) != 200 {
		t.Errorf("unexpect target num: %d", len(rcvRes))
	}
	lastRes1 := int64(0)
	lastRes2 := int64(0)
	for _, rcv := range rcvRes {
		parts1 := strings.Split(rcv[1], ":")
		content := parts1[2]
		parts2 := strings.Split(content, `"`)
		curNum, pe := strconv.ParseInt(parts2[0], 10, 64)
		if pe != nil {
			t.Error(pe)
		}
		if rcv[0] == "10188" { //nolint:staticcheck
			if curNum <= lastRes1 {
				if lastRes1 != 0 {
					t.Errorf("the order of results is disrupted: %d, %d", curNum, lastRes1)
				}
			}
			lastRes1 = curNum
		} else if rcv[0] == "10189" { //nolint:staticcheck
			if curNum <= lastRes2 {
				if lastRes2 != 0 {
					t.Errorf("the order of results is disrupted: %d, %d", curNum, lastRes2)
				}
			}
			lastRes2 = curNum
		} else {
			t.Errorf("unexpect port: %s", rcv[0])
		}
	}
	rcvResLock.RUnlock()
	rcvResLock.Lock()
	rcvRes = [][2]string{}
	rcvResLock.Unlock()

	err = server2.Close()
	if err != nil {
		t.Errorf("http server close error: %v", err)
	}
	err = server1.Close()
	if err != nil {
		t.Errorf("http server close error: %v", err)
	}
	err = app.Stop()
	if err != nil {
		t.Logf("app stop error: %v", err)
	}
	err = eg.Wait()
	if err != nil {
		t.Log(err)
	}
}
