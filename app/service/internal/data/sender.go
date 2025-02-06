package data

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/informer"
	"github.com/tianping526/eventbridge/app/internal/rule"
	"github.com/tianping526/eventbridge/app/service/internal/biz"
	"github.com/tianping526/eventbridge/app/service/internal/conf"
)

const (
	SchemaRocketMQ = "rocketmq"
)

var (
	_ informer.Handler = (*handler)(nil)

	ppg = propagation.NewCompositeTextMapPropagator(
		tracing.Metadata{},
		propagation.Baggage{},
		propagation.TraceContext{},
	)
)

type Sender interface {
	Send(ctx context.Context, busName string, eventExt *rule.EventExt, pubTime *timestamppb.Timestamp) (string, error)
}

type MQProducer interface {
	Send(
		ctx context.Context, topic string, mode v1.BusWorkMode, eventExt *rule.EventExt, pubTime *timestamppb.Timestamp,
	) (string, error)
	io.Closer
}

type bus struct {
	sourceTopic      string
	sourceDelayTopic string

	sourceMQ      string // schema://host:port/topic
	sourceDelayMQ string // schema://host:port/topic

	mode v1.BusWorkMode

	sourceMQProducer      MQProducer
	sourceDelayMQProducer MQProducer
}

type sender struct {
	log *log.Helper

	defaultMQ *url.URL

	buses sync.Map // map[busName]*bus
}

func NewSender(
	logger log.Logger,
	reflector informer.Reflector,
	bc *conf.Bootstrap,
) (Sender, func(), error) {
	mq, err := url.Parse(bc.Data.DefaultMq)
	if err != nil {
		return nil, nil, err
	}
	if mq.Scheme == "" {
		mq.Scheme = SchemaRocketMQ
	}
	s := &sender{
		log: log.NewHelper(log.With(
			logger,
			"module", "sender",
			"caller", log.DefaultCaller,
		)),

		defaultMQ: mq,
	}
	h := newHandler(logger, reflector, s)
	i := informer.NewInformer(logger, reflector, h)
	eg := new(errgroup.Group)
	eg.Go(func() error {
		i.WatchAndHandle()
		return nil
	})
	return s, func() {
		i.Close()
		_ = eg.Wait()
		cleanup := make([]*bus, 0)
		s.buses.Range(func(key, value interface{}) bool {
			s.buses.Delete(key)
			b := value.(*bus)
			cleanup = append(cleanup, b)
			return true
		})
		for _, b := range cleanup {
			err = b.sourceMQProducer.Close()
			if err != nil {
				s.log.Errorf("close sourceMQProducer failed: %v", err)
			}
			err = b.sourceDelayMQProducer.Close()
			if err != nil {
				s.log.Errorf("close sourceDelayMQProducer failed: %v", err)
			}
		}
	}, nil
}

func (s *sender) Send(
	ctx context.Context,
	busName string,
	eventExt *rule.EventExt,
	pubTime *timestamppb.Timestamp,
) (string, error) {
	// inject propagation
	carrier := propagation.MapCarrier{}
	ppg.Inject(ctx, carrier)
	eventExt.Metadata = carrier

	v, ok := s.buses.Load(busName)
	if !ok {
		return "", fmt.Errorf("bus %s not found", busName)
	}

	b := v.(*bus)
	if pubTime.IsValid() {
		return b.sourceDelayMQProducer.Send(ctx, b.sourceDelayTopic, b.mode, eventExt, pubTime)
	}
	return b.sourceMQProducer.Send(ctx, b.sourceTopic, b.mode, eventExt, pubTime)
}

func (s *sender) updateBus(b *biz.Bus) error {
	source, err := parseTopic(s.defaultMQ, b.SourceTopic)
	if err != nil {
		return err
	}
	sourceDelay, err := parseTopic(s.defaultMQ, b.SourceDelayTopic)
	if err != nil {
		return err
	}

	v, ok := s.buses.Load(b.Name)
	if !ok { // Add
		var sourceMQProducer MQProducer
		sourceMQProducer, err = s.newMQProducer(source)
		if err != nil {
			return err
		}
		var sourceDelayMQProducer MQProducer
		sourceDelayMQProducer, err = s.newMQProducer(sourceDelay)
		if err != nil {
			return err
		}
		s.buses.Store(b.Name, &bus{
			sourceTopic:           source.Path,
			sourceDelayTopic:      sourceDelay.Path,
			sourceMQ:              source.String(),
			sourceDelayMQ:         sourceDelay.String(),
			mode:                  b.Mode,
			sourceMQProducer:      sourceMQProducer,
			sourceDelayMQProducer: sourceDelayMQProducer,
		})
		return nil
	}

	// Update
	old := v.(*bus)
	nb := &bus{
		sourceTopic:      source.Path,
		sourceDelayTopic: sourceDelay.Path,
		sourceMQ:         source.String(),
		sourceDelayMQ:    sourceDelay.String(),
		mode:             b.Mode,
	}
	var cleanup []io.Closer
	if old.sourceMQ == nb.sourceMQ {
		nb.sourceMQProducer = old.sourceMQProducer
	} else {
		var sourceMQProducer MQProducer
		sourceMQProducer, err = s.newMQProducer(source)
		if err != nil {
			return err
		}
		nb.sourceMQProducer = sourceMQProducer
		cleanup = append(cleanup, old.sourceMQProducer)
	}
	if old.sourceDelayMQ == nb.sourceDelayMQ {
		nb.sourceDelayMQProducer = old.sourceDelayMQProducer
	} else {
		var sourceDelayMQProducer MQProducer
		sourceDelayMQProducer, err = s.newMQProducer(sourceDelay)
		if err != nil {
			return err
		}
		nb.sourceDelayMQProducer = sourceDelayMQProducer
		cleanup = append(cleanup, old.sourceDelayMQProducer)
	}
	s.buses.Store(b.Name, nb)
	for _, c := range cleanup {
		err = c.Close()
		if err != nil {
			s.log.Errorf("close mq producer failed: %v", err)
		}
	}
	return nil
}

func (s *sender) newMQProducer(mq *url.URL) (MQProducer, error) {
	switch mq.Scheme {
	case SchemaRocketMQ:
		return NewRocketMQProducer(mq.Host)
	default:
		return nil, fmt.Errorf("unsupported mq scheme: %s", mq.Scheme)
	}
}

func parseTopic(defaultMQ *url.URL, topic string) (*url.URL, error) {
	parsedTopic, err := url.Parse(topic)
	if err != nil {
		return nil, err
	}

	if parsedTopic.Scheme == "" {
		parsedTopic.Scheme = defaultMQ.Scheme
	}
	if parsedTopic.Host == "" {
		parsedTopic.Host = defaultMQ.Host
	}
	if parsedTopic.Path == "" {
		return nil, fmt.Errorf("bus topic is empty")
	}

	return parsedTopic, nil
}

func (s *sender) deleteBus(busName string) error {
	if v, loaded := s.buses.LoadAndDelete(busName); loaded {
		b := v.(*bus)
		err := b.sourceMQProducer.Close()
		if err != nil {
			s.log.Errorf("close source mq producer failed: %v", err)
		}
		err = b.sourceDelayMQProducer.Close()
		if err != nil {
			s.log.Errorf("close source delay mq producer failed: %v", err)
		}
	}
	return nil
}

type handler struct {
	log *log.Helper

	reflector informer.Reflector
	sender    *sender
}

func newHandler(
	logger log.Logger,
	reflector informer.Reflector,
	sender *sender,
) informer.Handler {
	return &handler{
		log: log.NewHelper(log.With(
			logger,
			"module", "sender/handler",
			"caller", log.DefaultCaller,
		)),

		reflector: reflector,
		sender:    sender,
	}
}

func (h *handler) Handle(key string) error {
	if v, ok := h.reflector.Get(key); ok { // Add or update
		b := v.(*biz.Bus)
		return h.sender.updateBus(b)
	}
	return h.sender.deleteBus(key)
}
