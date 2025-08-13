package data

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/event"
	"github.com/tianping526/eventbridge/app/internal/rule"

	rmqClient "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	v2 "github.com/apache/rocketmq-clients/golang/v5/protocol/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/metadata"
)

const (
	rmqReqTimeout = 3 * time.Second

	metricLabelBusName  = "bus_name"
	metricLabelBusTopic = "bus_topic"
)

func init() {
	_ = os.Setenv(rmqClient.ENABLE_CONSOLE_APPENDER, "true")
	_ = os.Setenv(rmqClient.CLIENT_LOG_LEVEL, "warn")
	rmqClient.ResetLogger()
}

type rocketMQProducer struct {
	p rmqClient.Producer
}

func NewRocketMQProducer(endpoint string) (MQProducer, error) {
	producer, err := rmqClient.NewProducer(
		&rmqClient.Config{
			Endpoint: endpoint,
			Credentials: &credentials.SessionCredentials{
				AccessKey:    "",
				AccessSecret: "",
			},
		},
		rmqClient.WithMaxAttempts(3),
	)
	if err != nil {
		return nil, err
	}

	// start producer
	err = producer.Start()
	if err != nil {
		return nil, err
	}
	return &rocketMQProducer{p: producer}, nil
}

func (r *rocketMQProducer) Send(ctx context.Context, topic string, mode v1.BusWorkMode, eventExt *rule.EventExt) error {
	msg := &rmqClient.Message{
		Topic: topic,
		Body:  eventExt.Value(),
	}
	msg.SetKeys(eventExt.Key())
	if mode == v1.BusWorkMode_BUS_WORK_MODE_ORDERLY {
		// same source+type to the same message group
		msg.SetMessageGroup(fmt.Sprintf("%s:%s", eventExt.Event.Source, eventExt.Event.Type))
	}

	_, err := r.p.Send(ctx, msg)
	return err
}

func (r *rocketMQProducer) Close() error {
	return r.p.GracefulStop()
}

type rocketMQConsumer struct {
	log *log.Helper

	busName  string
	busTopic string
	c        rmqClient.SimpleConsumer
	closeC   chan struct{}
}

func NewRocketMQConsumer(logger log.Logger, endpoint string, topic string) (MQConsumer, error) {
	// new simpleConsumer instance
	simpleConsumer, err := rmqClient.NewSimpleConsumer(&rmqClient.Config{
		Endpoint: endpoint,
		ConsumerGroup: fmt.Sprintf(
			"%s%s",
			strings.Replace(strings.ReplaceAll(endpoint, ".", ""), ":", "", 1),
			topic,
		),
		Credentials: &credentials.SessionCredentials{
			AccessKey:    "",
			AccessSecret: "",
		},
	},
		rmqClient.WithAwaitDuration(time.Second*5),
		rmqClient.WithSubscriptionExpressions(map[string]*rmqClient.FilterExpression{
			topic: rmqClient.SUB_ALL,
		}),
	)
	if err != nil {
		return nil, err
	}
	// start simpleConsumer
	err = simpleConsumer.Start()
	if err != nil {
		return nil, err
	}
	return &rocketMQConsumer{
		log: log.NewHelper(log.With(
			logger,
			"module", "rocketmq/consumer",
			"caller", log.DefaultCaller,
		)),

		c:      simpleConsumer,
		closeC: make(chan struct{}),
	}, nil
}

func (r *rocketMQConsumer) Receive(
	ctx context.Context,
	handler event.Handler,
	mode v1.BusWorkMode,
	timeout time.Duration,
	workers int32,
	runningWorkers metric.Int64Gauge,
) error {
	if mode == v1.BusWorkMode_BUS_WORK_MODE_ORDERLY {
		invisibleDuration := 10*time.Second + rmqReqTimeout + timeout
		for {
			mvs, err := r.c.Receive(
				ctx, 1, invisibleDuration,
			)
			if err != nil {
				select {
				case <-r.closeC:
					return nil
				default:
					var errRPC *rmqClient.ErrRpcStatus
					ok := errors.As(err, &errRPC)
					if ok && errRPC.Code == int32(v2.Code_MESSAGE_NOT_FOUND) {
						continue
					}
					r.log.Error(err)
					time.Sleep(time.Second)
					continue
				}
			}
			// handle a message
			for _, mv := range mvs {
				r.messageHandle(ctx, mv, handler, timeout)
			}
		}
	} else { // CONCURRENTLY
		eg := new(errgroup.Group)
		defer func() {
			_ = eg.Wait() // wait for all goroutines to finish
		}()
		sem := make(chan struct{}, workers)
		if runningWorkers != nil {
			eg.Go(func() error {
				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-r.closeC:
						return nil
					case <-ticker.C:
						runningWorkers.Record(
							ctx, int64(len(sem)),
							metric.WithAttributes(
								attribute.String(metricLabelBusName, r.busName),
								attribute.String(metricLabelBusTopic, r.busTopic),
							),
						)
					}
				}
			})
		}
		for {
			select {
			case <-r.closeC:
				return nil
			case sem <- struct{}{}: // acquire a semaphore
				<-sem // release a semaphore
			}
			maxMessageNum := workers - int32(len(sem))
			invisibleDuration := time.Duration(maxMessageNum+10)*time.Second + rmqReqTimeout + timeout
			mvs, err := r.c.Receive(
				ctx, maxMessageNum, invisibleDuration,
			)
			if err != nil {
				select {
				case <-r.closeC:
					return nil
				default:
					var errRPC *rmqClient.ErrRpcStatus
					ok := errors.As(err, &errRPC)
					if ok && errRPC.Code == int32(v2.Code_MESSAGE_NOT_FOUND) {
						continue
					}
					r.log.Error(err)
					time.Sleep(time.Second)
					continue
				}
			}
			// handle a message
			for _, mv := range mvs {
				select {
				case <-r.closeC:
					return nil
				case sem <- struct{}{}: // acquire a semaphore
					eg.Go(func() error {
						defer func() {
							<-sem // release a semaphore
						}()
						r.messageHandle(ctx, mv, handler, timeout)
						return nil
					})
				}
			}
		}
	}
}

func (r *rocketMQConsumer) messageHandle(
	ctx context.Context,
	mv *rmqClient.MessageView,
	f event.Handler,
	timeout time.Duration,
) {
	// set timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// unmarshal event
	evt, err := rule.NewEventExtFromBytes(mv.GetBody())
	if err != nil {
		r.log.Errorf("unmarshal event err: %s, event: %s", err, mv.GetBody())
		return
	}

	if evt.Metadata != nil {
		md := make(map[string][]string, len(evt.Metadata))
		for k, v := range evt.Metadata {
			md[k] = []string{v}
		}
		ctx = metadata.NewServerContext(ctx, md)
	}

	// handle
	if evt.RetryStrategy == v1.RetryStrategy_RETRY_STRATEGY_BACKOFF {
		// retries 3 times, with each retry interval being a random value between 10 and 20 seconds
		// total of 4 executions.
		_, err = f(ctx, evt) // handle
		if err == nil {      // success
			err = r.c.Ack(ctx, mv)
			if err != nil {
				r.log.Errorf("ack event(%s) err: %s", evt.Key(), err)
			}
		}

		if err != nil { // failed
			if mv.GetDeliveryAttempt() >= 4 { // nolint:mnd
				r.log.Errorf(
					"failed %d times, event key: %s, will into DLQ",
					mv.GetDeliveryAttempt(), evt.Key(),
				)
				// err = r.c.ForwardMessageToDeadLetterQueue(ctx, mv)
				// if err != nil {
				// 	r.log.Errorf("forward event(%s) to DLQ err: %s", evt.Key(), err)
				// }
			} else {
				delayS := 10 + rand.Intn(10)
				r.log.Errorf(
					"failed %d times, event key: %s, will retry after %ds",
					mv.GetDeliveryAttempt(), evt.Key(), delayS,
				)
				err = r.c.ChangeInvisibleDuration(mv, time.Duration(delayS)*time.Second)
				if err != nil {
					r.log.Errorf("change event(%s) invisible duration err: %s", evt.Key(), err)
				}
			}
		}
	} else {
		// runAndExponentialDecayRetry retries 176 times, each retry interval exponential increment to 512 seconds,
		// total retry time of 1 day;
		// each retry specific interval: 1, 2, 4, 8, 16, 32, 64, 128, 256, 512,
		// 512 ... . 512 seconds (a total of 167 512)
		// total of 177 executions.
		_, err = f(ctx, evt) // handle
		if err == nil {      // success
			err = r.c.Ack(ctx, mv)
			if err != nil {
				r.log.Errorf("ack event(%s) err: %s", evt.Key(), err)
			}
		}

		if err != nil { // failed
			if mv.GetDeliveryAttempt() >= 177 { // nolint:mnd
				r.log.Errorf(
					"failed %d times, event key: %s, will into DLQ",
					mv.GetDeliveryAttempt(), evt.Key(),
				)
				// err = r.c.ForwardMessageToDeadLetterQueue(ctx, mv)
				// if err != nil {
				// 	r.log.Errorf("forward event(%s) to DLQ err: %s", evt.Key(), err)
				// }
			} else {
				delayS := 512
				if mv.GetDeliveryAttempt() < 10 { // nolint:mnd
					delayS = int(math.Exp2(float64(mv.GetDeliveryAttempt() - 1)))
				}
				r.log.Errorf(
					"failed %d times, event key: %s, will retry after %ds",
					mv.GetDeliveryAttempt(), evt.Key(), delayS,
				)
				err = r.c.ChangeInvisibleDuration(mv, time.Duration(delayS)*time.Second)
				if err != nil {
					r.log.Errorf("change event(%s) invisible duration err: %s", evt.Key(), err)
				}
			}
		}
	}
}

func (r *rocketMQConsumer) Close() error {
	select {
	case <-r.closeC:
		return nil
	default:
		close(r.closeC)
		return r.c.GracefulStop()
	}
}
