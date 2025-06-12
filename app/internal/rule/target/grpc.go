package target

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	t "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/dispatcher/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"
)

func init() {
	registerDispatcher(
		"gRPCDispatcher",
		newGRPCDispatcher,
		`
		{
		  "$schema": "https://json-schema.org/draft/2020-12/schema",
		  "title": "gRPC dispatcher",
		  "description": "The data format of the gRPC dispatcher",
		  "type": "object",
		  "properties": {
			"endpoint": {
			  "description": "Server endpoints to be pushed",
			  "type": "string"
			},
			"data": {
			  "description": "The json data to be pushed",
			  "type": "object"
			},
			"metadata": {
			  "description": "approximate header in http, use for auth etc...",
			  "type": "string"
			}
		  },
		  "required": [
			"endpoint",
			"data"
		  ]
		}`)
}

type gRPCDispatcher struct {
	log         *log.Helper
	clients     sync.Map
	connections sync.Map
	validator   *gojsonschema.Schema
	// Only one check is required per dispatcher
	validated int32
}

func newGRPCDispatcher(
	_ context.Context,
	logger log.Logger,
	_ *rule.Target,
	validator *gojsonschema.Schema,
) (rule.Dispatcher, error) {
	return &gRPCDispatcher{
		log: log.NewHelper(log.With(
			logger,
			"module",
			"target/gRPCDispatcher",
			"caller", log.DefaultCaller,
		)),
		validator: validator,
	}, nil
}

func (d *gRPCDispatcher) Dispatch(ctx context.Context, event *rule.EventExt) error {
	// validate
	val := atomic.LoadInt32(&d.validated)
	if val == 0 {
		result, err := d.validator.Validate(gojsonschema.NewStringLoader(event.Event.Data))
		if err != nil {
			return err
		}
		if !result.Valid() {
			return fmt.Errorf(
				"gRPC dispatcher target event data is not valid. see err: %s",
				result.Errors(),
			)
		}
	}
	atomic.AddInt32(&d.validated, 1)

	// fetch params
	jsonData := make(map[string]interface{})
	_ = json.Unmarshal([]byte(event.Event.Data), &jsonData)
	endpoint := jsonData["endpoint"].(string)
	data := jsonData["data"].(map[string]interface{})
	marshalData, _ := json.Marshal(data)

	// metadata
	if value, ok := jsonData["metadata"]; ok {
		md := value.(string)
		m := make(map[string]string)
		err := json.Unmarshal([]byte(md), &m)
		if err != nil {
			return err
		}
		for k, v := range m {
			ctx = metadata.AppendToOutgoingContext(ctx, k, v)
		}
	}

	// get client
	clientVal, ok := d.clients.Load(endpoint)
	var client v1.EventBridgeDispatcherServiceClient
	if ok {
		client = clientVal.(v1.EventBridgeDispatcherServiceClient)
	} else {
		conn, err := t.DialInsecure(
			context.Background(),
			t.WithEndpoint(endpoint),
			t.WithMiddleware(
				tracing.Client(),
				recovery.Recovery(),
			),
			t.WithTimeout(0),
			t.WithOptions(grpc.WithDisableHealthCheck()),
		)
		if err != nil {
			return err
		}
		connVal, exist := d.connections.LoadOrStore(endpoint, conn)
		if exist { // store by other goroutine
			err = conn.Close()
			if err != nil {
				return fmt.Errorf("close grpc connection(%s) failed: %w", endpoint, err)
			}
			conn = connVal.(*grpc.ClientConn)
		}
		client = v1.NewEventBridgeDispatcherServiceClient(conn)
		d.clients.Store(endpoint, client)
	}

	// call gRPC
	_, err := client.PostTargetEvent(ctx, &v1.PostTargetEventRequest{
		Id:              event.Event.Id,
		Source:          event.Event.Source,
		Datacontenttype: event.Event.Datacontenttype,
		Data:            string(marshalData),
	})
	if err != nil {
		return err
	}
	return nil
}

func (d *gRPCDispatcher) Close() error {
	errs := make([]error, 0)
	d.connections.Range(func(key, _ interface{}) bool {
		value, loaded := d.connections.LoadAndDelete(key)
		if loaded {
			err := value.(*grpc.ClientConn).Close()
			if err != nil {
				errs = append(errs, fmt.Errorf("close grpc connection(%s) failed: %w", key, err))
			}
		}
		return true
	})
	if len(errs) > 0 {
		errInfo := make([]string, 0, len(errs))
		for _, err := range errs {
			errInfo = append(errInfo, err.Error())
		}
		return fmt.Errorf("close gRPCDispatcher failed: %s", strings.Join(errInfo, ", "))
	}
	return nil
}
