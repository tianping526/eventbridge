package data

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/sync/errgroup"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/event"
	"github.com/tianping526/eventbridge/app/internal/informer"
	"github.com/tianping526/eventbridge/app/internal/rule"
	"github.com/tianping526/eventbridge/app/job/internal/conf"
	"github.com/tianping526/eventbridge/app/job/internal/data/ent"
	entBus "github.com/tianping526/eventbridge/app/job/internal/data/ent/bus"
	"github.com/tianping526/eventbridge/app/job/internal/data/ent/version"
)

const (
	BusesVersionID         = 1
	SchemaRocketMQ         = "rocketmq"
	consumerDefaultTimeout = 1 * time.Second
)

var ppg = propagation.NewCompositeTextMapPropagator(
	tracing.Metadata{},
	propagation.Baggage{},
	propagation.TraceContext{},
)

type Sender interface {
	Send(ctx context.Context, eventExt *rule.EventExt) error
}

type Bus interface {
	Sender
	event.Receiver
}

type MQProducer interface {
	Send(ctx context.Context, topic string, mode v1.BusWorkMode, eventExt *rule.EventExt) error
	io.Closer
}

type MQConsumer interface {
	Receive(
		ctx context.Context, handler event.Handler, mode v1.BusWorkMode,
		timeout time.Duration, workers int32, runningWorkers metric.Int64Gauge,
	) error
	io.Closer
}

// busInfo should be able to compare values and cannot contain pointers
type busInfo struct {
	name                string
	mode                v1.BusWorkMode
	sourceTopic         string
	sourceDelayTopic    string
	targetExpDecayTopic string
	targetBackoffTopic  string
}

type busReflector struct {
	log *log.Helper

	db           *ent.Client
	busesVersion uint64
	interval     time.Duration
	dbTimeout    time.Duration
	interBuses   map[string]*busInfo
	closeCh      chan struct{}

	buses sync.Map // map[busName]*busInfo
}

func NewBusReflector(
	logger log.Logger,
	db *ent.Client,
) (informer.Reflector, error) {
	return &busReflector{
		log: log.NewHelper(log.With(
			logger,
			"module", "bus/reflector",
			"caller", log.DefaultCaller,
		)),

		db:        db,
		interval:  5 * time.Second,
		dbTimeout: 5 * time.Second,
		closeCh:   make(chan struct{}),
	}, nil
}

func (br *busReflector) Watch() ([]string, error) {
	err := br.fetchNextBusesVersion()
	if err != nil {
		return nil, err
	}

	busInfos, err := br.fetchBuses()
	if err != nil {
		return nil, err
	}

	newBuses := make(map[string]*busInfo, len(busInfos))
	for _, b := range busInfos {
		newBuses[b.name] = b
	}

	updated := make([]string, 0)
	for name, b := range newBuses {
		if old, ok := br.interBuses[name]; !ok || *old != *b { // Add or update
			br.buses.Store(name, b)
			updated = append(updated, name)
		}
	}
	for name := range br.interBuses {
		if _, ok := newBuses[name]; !ok { // Delete
			br.buses.Delete(name)
			updated = append(updated, name)
		}
	}
	br.interBuses = newBuses
	return updated, nil
}

func (br *busReflector) Get(key string) (interface{}, bool) {
	return br.buses.Load(key)
}

func (br *busReflector) Close() error {
	close(br.closeCh)
	return nil
}

func (br *busReflector) fetchBusesVersion() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), br.dbTimeout)
	defer cancel()
	v, err := br.db.Version.Query().Select(version.FieldVersion).Where(version.ID(BusesVersionID)).Only(ctx)
	if err != nil {
		return 0, err
	}
	return v.Version, nil
}

func (br *busReflector) fetchNextBusesVersion() error {
	v, err := br.fetchBusesVersion()
	if err != nil {
		return err
	}
	if v > br.busesVersion {
		br.busesVersion = v
		return nil
	}

	timer := time.NewTicker(br.interval)
	defer timer.Stop()
	for {
		select {
		case <-br.closeCh:
			return informer.NewReflectorClosedError()
		case <-timer.C:
			v, err = br.fetchBusesVersion()
			if err != nil {
				return err
			}
			if v > br.busesVersion {
				br.busesVersion = v
				return nil
			}
		}
	}
}

func (br *busReflector) fetchBuses() ([]*busInfo, error) {
	busInfos := make([]*busInfo, 0)
	next := uint64(0)
	limit := 100
	for {
		ctx, cancel := context.WithTimeout(context.Background(), br.dbTimeout)
		bs, err := br.db.Bus.Query().
			Where(
				entBus.IDGTE(next),
			).
			Select(
				entBus.FieldID,
				entBus.FieldName,
				entBus.FieldMode,
				entBus.FieldSourceTopic,
				entBus.FieldSourceDelayTopic,
				entBus.FieldTargetExpDecayTopic,
				entBus.FieldTargetBackoffTopic,
			).
			Order(ent.Asc(entBus.FieldID)).
			Limit(limit + 1).
			All(ctx)
		cancel()
		if err != nil {
			return nil, err
		}
		if len(bs) > limit {
			next = bs[limit].ID
			bs = bs[:limit]
		} else {
			next = 0
		}
		for _, b := range bs {
			busInfos = append(busInfos, &busInfo{
				name:                b.Name,
				mode:                v1.BusWorkMode(b.Mode),
				sourceTopic:         b.SourceTopic,
				sourceDelayTopic:    b.SourceDelayTopic,
				targetExpDecayTopic: b.TargetExpDecayTopic,
				targetBackoffTopic:  b.TargetBackoffTopic,
			})
		}
		if next == 0 {
			break
		}
	}
	return busInfos, nil
}

type bus struct {
	sourceTopic         string
	sourceDelayTopic    string
	targetExpDecayTopic string
	targetBackoffTopic  string

	sourceMQ         string // schema://host:port/topic
	sourceDelayMQ    string // schema://host:port/topic
	targetExpDecayMQ string // schema://host:port/topic
	targetBackoffMQ  string // schema://host:port/topic

	mode v1.BusWorkMode

	sourceMQConsumer         MQConsumer
	sourceDelayMQConsumer    MQConsumer
	targetExpDecayMQConsumer MQConsumer
	targetBackoffMQConsumer  MQConsumer
	targetExpDecayMQProducer MQProducer
	targetBackoffMQProducer  MQProducer
}

type buses struct {
	baseLog        log.Logger
	log            *log.Helper
	runningWorkers metric.Int64Gauge

	defaultMQ             *url.URL
	workersPerMqTopic     int32
	sourceTimeout         time.Duration
	sourceDelayTimeout    time.Duration
	targetExpDecayTimeout time.Duration
	targetBackoffTimeout  time.Duration
	informer              *informer.Informer
	ctx                   context.Context
	eg                    *errgroup.Group
	eventHandler          event.Handler
	buses                 sync.Map // map[busName]*bus
	closed                chan struct{}
}

func NewBuses(
	logger log.Logger,
	reflector informer.Reflector,
	bc *conf.Bootstrap,
	m *Metric,
) (Bus, error) {
	mq, err := url.Parse(bc.Data.DefaultMq)
	if err != nil {
		return nil, err
	}
	if mq.Scheme == "" {
		mq.Scheme = SchemaRocketMQ
	}
	workersPerMqTopic := bc.Data.WorkersPerMqTopic
	if workersPerMqTopic < 1 {
		workersPerMqTopic = 256
	}
	sourceTimeout := consumerDefaultTimeout
	if bc.Server.Event.SourceTimeout.IsValid() {
		sourceTimeout = bc.Server.Event.SourceTimeout.AsDuration()
	}
	sourceDelayTimeout := consumerDefaultTimeout
	if bc.Server.Event.DelayTimeout.IsValid() {
		sourceDelayTimeout = bc.Server.Event.DelayTimeout.AsDuration()
	}
	targetExpDecayTimeout := consumerDefaultTimeout
	if bc.Server.Event.TargetExpDecayTimeout.IsValid() {
		targetExpDecayTimeout = bc.Server.Event.TargetExpDecayTimeout.AsDuration()
	}
	targetBackoffTimeout := consumerDefaultTimeout
	if bc.Server.Event.TargetBackoffTimeout.IsValid() {
		targetBackoffTimeout = bc.Server.Event.TargetBackoffTimeout.AsDuration()
	}
	bs := &buses{
		baseLog: logger,
		log: log.NewHelper(log.With(
			logger,
			"module", "buses",
			"caller", log.DefaultCaller,
		)),
		runningWorkers: m.RunningWorkers,

		defaultMQ:             mq,
		workersPerMqTopic:     workersPerMqTopic,
		sourceTimeout:         sourceTimeout,
		sourceDelayTimeout:    sourceDelayTimeout,
		targetExpDecayTimeout: targetExpDecayTimeout,
		targetBackoffTimeout:  targetBackoffTimeout,
		eg:                    new(errgroup.Group),
		closed:                make(chan struct{}),
	}
	h := newBusHandler(logger, reflector, bs)
	i := informer.NewInformer(logger, reflector, h)
	bs.informer = i // NOTE: circular reference

	return bs, nil
}

func (bs *buses) Send(ctx context.Context, eventExt *rule.EventExt) error {
	// inject propagation
	carrier := propagation.MapCarrier{}
	ppg.Inject(ctx, carrier)
	eventExt.Metadata = carrier

	v, ok := bs.buses.Load(eventExt.BusName)
	if !ok {
		return fmt.Errorf("bus %s not found", eventExt.BusName)
	}

	b := v.(*bus)
	if eventExt.RetryStrategy == v1.RetryStrategy_RETRY_STRATEGY_BACKOFF {
		return b.targetBackoffMQProducer.Send(ctx, b.targetBackoffTopic, b.mode, eventExt)
	}
	return b.targetExpDecayMQProducer.Send(ctx, b.targetExpDecayTopic, b.mode, eventExt)
}

func (bs *buses) Receive(ctx context.Context, handler event.Handler) error {
	bs.eventHandler = handler
	bs.ctx = ctx
	bs.informer.WatchAndHandle()
	close(bs.closed)
	return nil
}

func (bs *buses) Close(ctx context.Context) error {
	bs.informer.Close()
	bs.informer = nil // NOTE: avoid circular reference
	<-bs.closed
	cleanup := make([]*bus, 0)
	bs.buses.Range(func(key, value interface{}) bool {
		bs.buses.Delete(key)
		b := value.(*bus)
		cleanup = append(cleanup, b)
		return true
	})
	for _, b := range cleanup {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := b.sourceMQConsumer.Close()
			if err != nil {
				bs.log.Errorf("close sourceMQConsumer err: %s", err)
			}
			err = b.sourceDelayMQConsumer.Close()
			if err != nil {
				bs.log.Errorf("close sourceDelayMQConsumer err: %s", err)
			}
			err = b.targetExpDecayMQConsumer.Close()
			if err != nil {
				bs.log.Errorf("close targetExpDecayMQConsumer err: %s", err)
			}
			err = b.targetBackoffMQConsumer.Close()
			if err != nil {
				bs.log.Errorf("close targetBackoffMQConsumer err: %s", err)
			}
			err = b.targetExpDecayMQProducer.Close()
			if err != nil {
				bs.log.Errorf("close targetExpDecayMQProducer err: %s", err)
			}
			err = b.targetBackoffMQProducer.Close()
			if err != nil {
				bs.log.Errorf("close targetBackoffMQProducer err: %s", err)
			}
		}
	}
	_ = bs.eg.Wait()
	return nil
}

func (bs *buses) updateBus(b *busInfo) error {
	source, err := parseTopic(bs.defaultMQ, b.sourceTopic)
	if err != nil {
		return err
	}
	sourceDelay, err := parseTopic(bs.defaultMQ, b.sourceDelayTopic)
	if err != nil {
		return err
	}
	targetExpDecay, err := parseTopic(bs.defaultMQ, b.targetExpDecayTopic)
	if err != nil {
		return err
	}
	targetBackoff, err := parseTopic(bs.defaultMQ, b.targetBackoffTopic)
	if err != nil {
		return err
	}

	v, ok := bs.buses.Load(b.name)
	if !ok { // Add
		var targetExpDecayMQProducer, targetBackoffMQProducer MQProducer
		targetExpDecayMQProducer, err = bs.newMQProducer(targetExpDecay)
		if err != nil {
			return err
		}
		targetBackoffMQProducer, err = bs.newMQProducer(targetBackoff)
		if err != nil {
			return err
		}
		var targetExpDecayMQConsumer, targetBackoffMQConsumer, sourceMQConsumer, sourceDelayMQConsumer MQConsumer
		targetExpDecayMQConsumer, err = bs.newMQConsumer(b.name, targetExpDecay, b.mode, bs.targetExpDecayTimeout)
		if err != nil {
			return err
		}
		targetBackoffMQConsumer, err = bs.newMQConsumer(b.name, targetBackoff, b.mode, bs.targetBackoffTimeout)
		if err != nil {
			return err
		}
		sourceMQConsumer, err = bs.newMQConsumer(b.name, source, b.mode, bs.sourceTimeout)
		if err != nil {
			return err
		}
		sourceDelayMQConsumer, err = bs.newMQConsumer(b.name, sourceDelay, b.mode, bs.sourceDelayTimeout)
		if err != nil {
			return err
		}
		bs.buses.Store(b.name, &bus{
			sourceTopic:         source.Path,
			sourceDelayTopic:    sourceDelay.Path,
			targetExpDecayTopic: targetExpDecay.Path,
			targetBackoffTopic:  targetBackoff.Path,

			sourceMQ:         source.String(),
			sourceDelayMQ:    sourceDelay.String(),
			targetExpDecayMQ: targetExpDecay.String(),
			targetBackoffMQ:  targetBackoff.String(),

			mode: b.mode,

			sourceMQConsumer:         sourceMQConsumer,
			sourceDelayMQConsumer:    sourceDelayMQConsumer,
			targetExpDecayMQConsumer: targetExpDecayMQConsumer,
			targetBackoffMQConsumer:  targetBackoffMQConsumer,
			targetExpDecayMQProducer: targetExpDecayMQProducer,
			targetBackoffMQProducer:  targetBackoffMQProducer,
		})
		return nil
	}

	// Update
	old := v.(*bus)
	nb := &bus{
		sourceTopic:         source.Path,
		sourceDelayTopic:    sourceDelay.Path,
		targetExpDecayTopic: targetExpDecay.Path,
		targetBackoffTopic:  targetBackoff.Path,

		sourceMQ:         source.String(),
		sourceDelayMQ:    sourceDelay.String(),
		targetExpDecayMQ: targetExpDecay.String(),
		targetBackoffMQ:  targetBackoff.String(),

		mode: b.mode,
	}
	var cleanup []io.Closer
	if old.targetExpDecayMQ == nb.targetExpDecayMQ {
		nb.targetExpDecayMQProducer = old.targetExpDecayMQProducer
	} else {
		var targetExpDecayMQProducer MQProducer
		targetExpDecayMQProducer, err = bs.newMQProducer(targetExpDecay)
		if err != nil {
			return err
		}
		nb.targetExpDecayMQProducer = targetExpDecayMQProducer
		cleanup = append(cleanup, old.targetExpDecayMQProducer)
	}
	if old.targetBackoffMQ == nb.targetBackoffMQ {
		nb.targetBackoffMQProducer = old.targetBackoffMQProducer
	} else {
		var targetBackoffMQProducer MQProducer
		targetBackoffMQProducer, err = bs.newMQProducer(targetBackoff)
		if err != nil {
			return err
		}
		nb.targetBackoffMQProducer = targetBackoffMQProducer
		cleanup = append(cleanup, old.targetBackoffMQProducer)
	}
	if old.mode == nb.mode {
		if old.targetExpDecayMQ == nb.targetExpDecayMQ {
			nb.targetExpDecayMQConsumer = old.targetExpDecayMQConsumer
		} else {
			var targetExpDecayMQConsumer MQConsumer
			targetExpDecayMQConsumer, err = bs.newMQConsumer(b.name, targetExpDecay, b.mode, bs.targetExpDecayTimeout)
			if err != nil {
				return err
			}
			nb.targetExpDecayMQConsumer = targetExpDecayMQConsumer
			cleanup = append(cleanup, old.targetExpDecayMQConsumer)
		}
		if old.targetBackoffMQ == nb.targetBackoffMQ {
			nb.targetBackoffMQConsumer = old.targetBackoffMQConsumer
		} else {
			var targetBackoffMQConsumer MQConsumer
			targetBackoffMQConsumer, err = bs.newMQConsumer(b.name, targetBackoff, b.mode, bs.targetBackoffTimeout)
			if err != nil {
				return err
			}
			nb.targetBackoffMQConsumer = targetBackoffMQConsumer
			cleanup = append(cleanup, old.targetBackoffMQConsumer)
		}
		if old.sourceMQ == nb.sourceMQ {
			nb.sourceMQConsumer = old.sourceMQConsumer
		} else {
			var sourceMQConsumer MQConsumer
			sourceMQConsumer, err = bs.newMQConsumer(b.name, source, b.mode, bs.sourceTimeout)
			if err != nil {
				return err
			}
			nb.sourceMQConsumer = sourceMQConsumer
			cleanup = append(cleanup, old.sourceMQConsumer)
		}
		if old.sourceDelayMQ == nb.sourceDelayMQ {
			nb.sourceDelayMQConsumer = old.sourceDelayMQConsumer
		} else {
			var sourceDelayMQConsumer MQConsumer
			sourceDelayMQConsumer, err = bs.newMQConsumer(b.name, sourceDelay, b.mode, bs.sourceDelayTimeout)
			if err != nil {
				return err
			}
			nb.sourceDelayMQConsumer = sourceDelayMQConsumer
			cleanup = append(cleanup, old.sourceDelayMQConsumer)
		}
	} else {
		var targetExpDecayMQConsumer, targetBackoffMQConsumer, sourceMQConsumer, sourceDelayMQConsumer MQConsumer
		targetExpDecayMQConsumer, err = bs.newMQConsumer(b.name, targetExpDecay, b.mode, bs.targetExpDecayTimeout)
		if err != nil {
			return err
		}
		targetBackoffMQConsumer, err = bs.newMQConsumer(b.name, targetBackoff, b.mode, bs.targetBackoffTimeout)
		if err != nil {
			return err
		}
		sourceMQConsumer, err = bs.newMQConsumer(b.name, source, b.mode, bs.sourceTimeout)
		if err != nil {
			return err
		}
		sourceDelayMQConsumer, err = bs.newMQConsumer(b.name, sourceDelay, b.mode, bs.sourceDelayTimeout)
		if err != nil {
			return err
		}
		nb.targetExpDecayMQConsumer = targetExpDecayMQConsumer
		nb.targetBackoffMQConsumer = targetBackoffMQConsumer
		nb.sourceMQConsumer = sourceMQConsumer
		nb.sourceDelayMQConsumer = sourceDelayMQConsumer
		cleanup = append(
			cleanup,
			old.targetExpDecayMQConsumer, old.targetBackoffMQConsumer,
			old.sourceMQConsumer, old.sourceDelayMQConsumer,
		)
	}
	bs.buses.Store(b.name, nb)
	for i := len(cleanup) - 1; i >= 0; i-- {
		err = cleanup[i].Close()
		if err != nil {
			bs.log.Errorf("close consumer or producer err: %s", err)
		}
	}
	return nil
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

func (bs *buses) newMQProducer(mq *url.URL) (MQProducer, error) {
	switch mq.Scheme {
	case SchemaRocketMQ:
		return NewRocketMQProducer(mq.Host)
	default:
		return nil, fmt.Errorf("unsupported mq scheme: %s", mq.Scheme)
	}
}

func (bs *buses) newMQConsumer(
	busName string, mq *url.URL, mode v1.BusWorkMode, timeout time.Duration,
) (MQConsumer, error) {
	var consumer MQConsumer
	var err error
	switch mq.Scheme {
	case SchemaRocketMQ:
		consumer, err = NewRocketMQConsumer(bs.baseLog, busName, mq.Host, mq.Path)
	default:
		return nil, fmt.Errorf("unsupported mq scheme: %s", mq.Scheme)
	}
	if err != nil {
		return nil, err
	}
	bs.eg.Go(func() error {
		err = consumer.Receive(bs.ctx, bs.eventHandler, mode, timeout, bs.workersPerMqTopic, bs.runningWorkers)
		if err != nil {
			bs.log.Errorf("consumer(%s) receive err: %s", mq.String(), err)
		}
		return nil
	})
	return consumer, nil
}

func (bs *buses) deleteBus(busName string) error {
	if v, ok := bs.buses.LoadAndDelete(busName); ok {
		b := v.(*bus)
		err := b.sourceMQConsumer.Close()
		if err != nil {
			bs.log.Errorf("close sourceMQConsumer err: %s", err)
		}
		err = b.sourceDelayMQConsumer.Close()
		if err != nil {
			bs.log.Errorf("close sourceDelayMQConsumer err: %s", err)
		}
		err = b.targetExpDecayMQProducer.Close()
		if err != nil {
			bs.log.Errorf("close targetExpDecayMQProducer err: %s", err)
		}
		err = b.targetBackoffMQProducer.Close()
		if err != nil {
			bs.log.Errorf("close targetBackoffMQProducer err: %s", err)
		}
	}
	return nil
}

type busHandler struct {
	log *log.Helper

	reflector informer.Reflector
	buses     *buses
}

func newBusHandler(
	logger log.Logger,
	reflector informer.Reflector,
	buses *buses,
) *busHandler {
	return &busHandler{
		log: log.NewHelper(log.With(
			logger,
			"module", "bus/handler",
			"caller", log.DefaultCaller,
		)),

		reflector: reflector,
		buses:     buses,
	}
}

func (h *busHandler) Handle(key string) error {
	if v, ok := h.reflector.Get(key); ok { // Add or update
		b := v.(*busInfo)
		return h.buses.updateBus(b)
	}
	return h.buses.deleteBus(key)
}
