package data

import (
	"context"
	"fmt"
	"os"

	rmqClient "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"
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
			Endpoint: fmt.Sprintf("dns:///%s", endpoint),
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

func (r *rocketMQProducer) Send(
	ctx context.Context, topic string, mode v1.BusWorkMode, eventExt *rule.EventExt, pubTime *timestamppb.Timestamp,
) (string, error) {
	msg := &rmqClient.Message{
		Topic: topic,
		Body:  eventExt.Value(),
	}
	msg.SetKeys(eventExt.Key())
	if pubTime.IsValid() {
		msg.SetDelayTimestamp(pubTime.AsTime())
	} else if mode == v1.BusWorkMode_BUS_WORK_MODE_ORDERLY {
		// same source+type to the same message group
		msg.SetMessageGroup(fmt.Sprintf("%s:%s", eventExt.Event.Source, eventExt.Event.Type))
	}

	res, err := r.p.Send(ctx, msg)
	if err != nil {
		return "", err
	}

	if len(res) == 0 {
		return "", nil
	}
	return res[0].MessageID, nil
}

func (r *rocketMQProducer) Close() error {
	return r.p.GracefulStop()
}
