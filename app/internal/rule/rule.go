package rule

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"

	"github.com/tianping526/eventbridge/app/internal/informer"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
)

const (
	// histogram: job_rule_execute_duration_seconds_bucket{"name", "event", "operation"}
	// counter:  job_rule_execute_total{"name", "event", "operation", "result"}
	metricLabelRuleName  = "name"
	metricLabelEvent     = "event"
	metricLabelOperation = "operation"
	metricLabelResult    = "result"
)

var (
	errNoMatcherAvailable     = errors.New("no matcher available")
	errNoTransformerAvailable = errors.New("no transformer available")
	errNoDispatcherAvailable  = errors.New("no dispatcher available")
)

type TargetParam struct {
	Key      string
	Form     string
	Value    string
	Template *string
}

type Target struct {
	ID            uint64
	Type          string
	Params        []*TargetParam
	RetryStrategy v1.RetryStrategy
}

type Rule struct {
	Name    string
	BusName string
	Status  v1.RuleStatus
	Pattern string
	Targets []*Target
}

type Matcher interface {
	Pattern(ctx context.Context, event *EventExt) (bool, error)
}

type Dispatcher interface {
	io.Closer
	Dispatch(ctx context.Context, event *EventExt) error
}

type Transformer interface {
	Transform(ctx context.Context, event *EventExt) (*EventExt, error)
}

type Executor interface {
	Matcher
	io.Closer
	Dispatch(context.Context, *EventExt) error
	Transform(ctx context.Context, event *EventExt) ([]*EventExt, error)
	Update(context.Context, *Rule) error
	IsFilterPatternEqual(filterPattern string) bool
	IsTargetsEqual(Targets []*Target) bool
}

type RulesManager interface {
	GetExecutors() (map[string]Executor, error)
	UpdateRules(ctx context.Context, rs []*Rule) error
	CleanupRules() error
}

type Rules interface {
	GetExecutors(busName string) (map[string]Executor, error)
}

type (
	NewMatcherFunc     func(ctx context.Context, logger log.Logger, pattern map[string]interface{}) (Matcher, error)
	NewTransformerFunc func(ctx context.Context, logger log.Logger, Target *Target) (Transformer, error)
	NewDispatcherFunc  func(ctx context.Context, logger log.Logger, Target *Target) (Dispatcher, error)
)

// Option is a functional option for configuring the executor.
type Option func(*options)

// WithExecuteDuration with executed duration histogram.
func WithExecuteDuration(c metric.Float64Histogram) Option {
	return func(o *options) {
		o.executeDuration = c
	}
}

// WithExecuteTotal with executed total counter.
func WithExecuteTotal(c metric.Int64Counter) Option {
	return func(o *options) {
		o.executeTotal = c
	}
}

// WithTransformParallelism sets the parallelism for transforming events.
func WithTransformParallelism(parallelism int) Option {
	return func(o *options) {
		if parallelism >= 2 { // nolint:mnd
			o.transformParallelism = parallelism
		}
	}
}

type options struct {
	// histogram: job_rule_execute_duration_seconds_bucket{"name", "event", "operation"}
	executeDuration metric.Float64Histogram
	// counter: job_rule_execute_total{"name", "event", "operation", "result"}
	executeTotal metric.Int64Counter
	// transformParallelism is the parallelism for transforming events.
	transformParallelism int
}

type executor struct {
	baseLog  log.Logger
	log      *log.Helper
	busName  string
	ruleName string
	opts     *options

	pattern string
	targets map[uint64]*Target

	matcher      Matcher
	transformers map[uint64]Transformer
	dispatchers  map[uint64]Dispatcher

	newMatcherFunc     NewMatcherFunc
	newTransformerFunc NewTransformerFunc
	newDispatcherFunc  NewDispatcherFunc

	sync.RWMutex
}

func NewNewExecutorFunc(
	nmf NewMatcherFunc,
	ntf NewTransformerFunc,
	ndf NewDispatcherFunc,
) NewExecutorFunc {
	return func(ctx context.Context, logger log.Logger, r *Rule, opts ...Option) (Executor, error) {
		ops := &options{
			transformParallelism: 20, // default parallelism
		}
		for _, o := range opts {
			o(ops)
		}
		exec := &executor{
			baseLog: logger,
			log: log.NewHelper(log.With(
				logger,
				"module", "rule/executor",
				"caller", log.DefaultCaller,
			)),
			busName:            r.BusName,
			ruleName:           r.Name,
			opts:               ops,
			newMatcherFunc:     nmf,
			newTransformerFunc: ntf,
			newDispatcherFunc:  ndf,
			targets:            map[uint64]*Target{},
			transformers:       map[uint64]Transformer{},
			dispatchers:        map[uint64]Dispatcher{},
		}
		err := exec.Update(ctx, r)
		if err != nil {
			return nil, err
		}
		return exec, nil
	}
}

func (d *executor) Pattern(ctx context.Context, event *EventExt) (ok bool, err error) {
	var matcher Matcher
	d.RLock()
	matcher = d.matcher
	d.RUnlock()
	if d.opts.executeTotal != nil {
		defer func() {
			res := "ok"
			if !ok {
				res = "pass"
			}
			if err != nil {
				res = fmt.Sprintf("%T", err)
			}
			d.opts.executeTotal.Add(
				ctx, 1,
				metric.WithAttributes(
					attribute.String(metricLabelRuleName, fmt.Sprintf("%s:%s", d.busName, d.ruleName)),
					attribute.String(metricLabelEvent, fmt.Sprintf("%s:%s", event.Event.Source, event.Event.Type)),
					attribute.String(metricLabelOperation, "Pattern"),
					attribute.String(metricLabelResult, res),
				),
			)
		}()
	}
	if matcher == nil {
		return false, errNoMatcherAvailable
	}
	return matcher.Pattern(ctx, event)
}

func (d *executor) Transform(ctx context.Context, event *EventExt) ([]*EventExt, error) {
	transformers := make(map[uint64]Transformer)
	d.RLock()
	for tID, t := range d.transformers {
		transformers[tID] = t
	}
	d.RUnlock()
	if len(transformers) == 0 {
		if d.opts.executeTotal != nil {
			defer d.opts.executeTotal.Add(
				ctx, 1,
				metric.WithAttributes(
					attribute.String(metricLabelRuleName, fmt.Sprintf("%s:%s", d.busName, d.ruleName)),
					attribute.String(metricLabelEvent, fmt.Sprintf("%s:%s", event.Event.Source, event.Event.Type)),
					attribute.String(metricLabelOperation, "Transform"),
					attribute.String(metricLabelResult, fmt.Sprintf("%T", errNoTransformerAvailable)),
				),
			)
		}
		return nil, errNoTransformerAvailable
	}

	if len(transformers) == 1 { // transform once, no parallel processing required
		for _, t := range transformers {
			targetEvent, err := t.Transform(ctx, event)
			if err != nil {
				return nil, err
			}
			return []*EventExt{targetEvent}, nil
		}
	}

	// parallel processing
	targetEvents := make([]*EventExt, 0, len(transformers))
	targetEventsLock := sync.Mutex{}
	eg := new(errgroup.Group)
	eg.SetLimit(d.opts.transformParallelism)

	for _, t := range transformers {
		eg.Go(func() error {
			targetEvent, err := t.Transform(ctx, event)
			if err != nil {
				return err
			}
			targetEventsLock.Lock()
			targetEvents = append(targetEvents, targetEvent)
			targetEventsLock.Unlock()
			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		return targetEvents, err
	}

	return targetEvents, nil
}

type wrapTransformer struct {
	transformer   Transformer
	executeTotal  metric.Int64Counter
	busName       string
	ruleName      string
	targetID      uint64
	retryStrategy v1.RetryStrategy
}

func (t *wrapTransformer) Transform(ctx context.Context, event *EventExt) (*EventExt, error) {
	newEvt := CloneEventExt(event)
	evt, err := t.transformer.Transform(ctx, newEvt)
	if err != nil {
		if t.executeTotal != nil {
			t.executeTotal.Add(
				ctx, 1,
				metric.WithAttributes(
					attribute.String(metricLabelRuleName, fmt.Sprintf("%s:%s:%d", t.busName, t.ruleName, t.targetID)),
					attribute.String(metricLabelEvent, fmt.Sprintf("%s:%s", event.Event.Source, event.Event.Type)),
					attribute.String(metricLabelOperation, "Transform"),
					attribute.String(metricLabelResult, fmt.Sprintf("%T", err)),
				),
			)
		}
		return nil, fmt.Errorf("transform target(id: %d) err: %w", t.targetID, err)
	}
	if t.executeTotal != nil {
		t.executeTotal.Add(
			ctx, 1,
			metric.WithAttributes(
				attribute.String(metricLabelRuleName, fmt.Sprintf("%s:%s:%d", t.busName, t.ruleName, t.targetID)),
				attribute.String(metricLabelEvent, fmt.Sprintf("%s:%s", event.Event.Source, event.Event.Type)),
				attribute.String(metricLabelOperation, "Transform"),
				attribute.String(metricLabelResult, "ok"),
			),
		)
	}
	evt.TargetId = t.targetID
	evt.RuleName = t.ruleName
	if t.retryStrategy != v1.RetryStrategy_RETRY_STRATEGY_UNSPECIFIED { // override event's retry strategy
		evt.RetryStrategy = t.retryStrategy
	}
	return evt, nil
}

func (d *executor) Dispatch(ctx context.Context, event *EventExt) (err error) {
	var dispatcher Dispatcher
	var dispatcherExist bool
	if d.opts.executeDuration != nil {
		startTime := time.Now()
		defer func() {
			d.opts.executeDuration.Record(
				ctx, time.Since(startTime).Seconds(),
				metric.WithAttributes(
					attribute.String(metricLabelRuleName, fmt.Sprintf("%s:%s:%d", d.busName, d.ruleName, event.TargetId)),
					attribute.String(metricLabelEvent, fmt.Sprintf("%s:%s", event.Event.Source, event.Event.Type)),
					attribute.String(metricLabelOperation, "Dispatch"),
				),
			)
		}()
	}
	d.RLock()
	dispatcher, dispatcherExist = d.dispatchers[event.TargetId]
	d.RUnlock()
	if d.opts.executeTotal != nil {
		defer func() {
			res := "ok"
			if err != nil {
				res = fmt.Sprintf("%T", err)
			}
			d.opts.executeTotal.Add(
				ctx, 1,
				metric.WithAttributes(
					attribute.String(metricLabelRuleName, fmt.Sprintf("%s:%s:%d", d.busName, d.ruleName, event.TargetId)),
					attribute.String(metricLabelEvent, fmt.Sprintf("%s:%s", event.Event.Source, event.Event.Type)),
					attribute.String(metricLabelOperation, "Dispatch"),
					attribute.String(metricLabelResult, res),
				),
			)
		}()
	}
	if !dispatcherExist {
		return errNoDispatcherAvailable
	}
	return dispatcher.Dispatch(ctx, event)
}

func IsMatcherNotFound(err error) bool {
	return errors.Is(err, errNoMatcherAvailable)
}

func IsTransformerNotFound(err error) bool {
	return errors.Is(err, errNoTransformerAvailable)
}

func IsDispatcherNotFound(err error) bool {
	return errors.Is(err, errNoDispatcherAvailable)
}

func (d *executor) Close() error {
	d.Lock()
	d.matcher = nil
	d.pattern = ""
	d.transformers = map[uint64]Transformer{}
	dispatchers := d.dispatchers
	d.dispatchers = map[uint64]Dispatcher{}
	d.targets = map[uint64]*Target{}
	d.Unlock()
	errs := make([]error, 0, len(dispatchers))
	for _, dispatcher := range dispatchers {
		if err := dispatcher.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		errInfo := make([]string, 0, len(errs))
		for _, err := range errs {
			errInfo = append(errInfo, err.Error())
		}
		return fmt.Errorf("close dispatchers err: %s", strings.Join(errInfo, ", "))
	}
	return nil
}

func (d *executor) Update(ctx context.Context, r *Rule) (err error) {
	if r == nil {
		return nil
	}
	newTargets := make(map[uint64]*Target, len(r.Targets))
	for _, t := range r.Targets {
		newTargets[t.ID] = t
	}

	delTargets := make(map[uint64]bool)
	oldTargets := make(map[uint64]*Target)
	d.RLock()
	for ti, t := range d.targets {
		oldTargets[ti] = t
	}
	d.RUnlock()

	for tID := range oldTargets {
		_, ok := newTargets[tID]
		if !ok {
			delTargets[tID] = true
		}
	}

	closeDispatchers := make([]Dispatcher, 0, len(delTargets)+len(newTargets))
	defer func() {
		errs := make([]error, 0, len(closeDispatchers))
		for _, dispatcher := range closeDispatchers {
			if err = dispatcher.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			errInfo := make([]string, 0, len(errs))
			for _, perErr := range errs {
				errInfo = append(errInfo, perErr.Error())
			}
			err = fmt.Errorf("close dispatchers err: %s", strings.Join(errInfo, ", "))
		}
	}()
	d.Lock()
	defer d.Unlock()
	if d.pattern != r.Pattern {
		parsedPattern := make(map[string]interface{})
		err = json.Unmarshal([]byte(r.Pattern), &parsedPattern)
		if err != nil {
			return err
		}
		var matcher Matcher
		matcher, err = d.newMatcherFunc(ctx, d.baseLog, parsedPattern)
		if err != nil {
			return err
		}
		d.matcher = matcher
		d.pattern = r.Pattern
	}
	for id := range delTargets {
		_, ok := d.targets[id]
		if ok {
			delete(d.transformers, id)
			closeDispatchers = append(closeDispatchers, d.dispatchers[id])
			delete(d.dispatchers, id)
			delete(d.targets, id)
		}
	}
	for id, t := range newTargets {
		if !reflect.DeepEqual(t, d.targets[id]) {
			d.targets[id] = t
			var transformer Transformer
			transformer, err = d.newTransformerFunc(ctx, d.baseLog, t)
			if err != nil {
				return err
			}
			d.transformers[id] = &wrapTransformer{
				transformer:   transformer,
				executeTotal:  d.opts.executeTotal,
				busName:       d.busName,
				ruleName:      d.ruleName,
				targetID:      id,
				retryStrategy: t.RetryStrategy,
			}
			var dispatcher Dispatcher
			dispatcher, err = d.newDispatcherFunc(ctx, d.baseLog, t)
			if err != nil {
				return err
			}
			oldDispatcher, ok := d.dispatchers[id]
			if ok {
				closeDispatchers = append(closeDispatchers, oldDispatcher)
			}
			d.dispatchers[id] = dispatcher
		}
	}
	return nil
}

func (d *executor) IsFilterPatternEqual(filterPattern string) bool {
	d.RLock()
	pattern := d.pattern
	d.RUnlock()
	return pattern == filterPattern
}

func (d *executor) IsTargetsEqual(targets []*Target) bool {
	newTargets := make(map[uint64]*Target, len(targets))
	for _, t := range targets {
		newTargets[t.ID] = t
	}
	oldTargets := make(map[uint64]*Target)
	d.RLock()
	for id, t := range d.targets {
		oldTargets[id] = t
	}
	d.RUnlock()
	return reflect.DeepEqual(newTargets, oldTargets)
}

type NewExecutorFunc func(context.Context, log.Logger, *Rule, ...Option) (Executor, error)

type rules struct {
	baseLog log.Logger
	log     *log.Helper

	executors       sync.Map // map[busName]map[ruleName]Executor
	newExecutorFunc NewExecutorFunc
	executorOpts    []Option
	ctx             context.Context
	updateTimeout   time.Duration
}

func NewRules(
	logger log.Logger,
	reflector informer.Reflector,
	nef NewExecutorFunc,
	ExecOpts ...Option,
) (Rules, func(), error) {
	rs := &rules{
		baseLog: logger,
		log: log.NewHelper(log.With(
			logger,
			"module", "rule/rules",
			"caller", log.DefaultCaller,
		)),
		newExecutorFunc: nef,
		executorOpts:    ExecOpts,
		ctx:             context.Background(),
		updateTimeout:   5 * time.Second,
	}
	h := newHandler(logger, reflector, rs)
	i := informer.NewInformer(logger, reflector, h)
	eg := new(errgroup.Group)
	eg.Go(func() error {
		i.WatchAndHandle()
		return nil
	})
	return rs, func() {
		i.Close()
		_ = eg.Wait()
		cleanup := make([]Executor, 0)
		rs.executors.Range(func(key, value interface{}) bool {
			rs.executors.Delete(key)
			rulesPerBus := value.(map[string]Executor)
			for _, e := range rulesPerBus {
				cleanup = append(cleanup, e)
			}
			return true
		})
		for _, e := range cleanup {
			err := e.Close()
			if err != nil {
				rs.log.Errorf("close executor failed: %v", err)
			}
		}
	}, nil
}

func (rs *rules) GetExecutors(busName string) (map[string]Executor, error) {
	if v, ok := rs.executors.Load(busName); ok {
		return v.(map[string]Executor), nil
	}
	return nil, nil
}

func (rs *rules) updateRule(r *Rule) error {
	v, ok := rs.executors.Load(r.BusName)
	if !ok { // add
		exec, err := rs.newExecutorFunc(rs.ctx, rs.baseLog, r, rs.executorOpts...)
		if err != nil {
			return err
		}
		rulesPerBus := map[string]Executor{r.Name: exec}
		rs.executors.Store(r.BusName, rulesPerBus)
		rs.log.Infof("bus %s rule %s added", r.BusName, r.Name)
		return nil
	}

	rulesPerBus := v.(map[string]Executor)
	if e, ok1 := rulesPerBus[r.Name]; ok1 { // update
		ctx, cancel := context.WithTimeout(rs.ctx, rs.updateTimeout)
		defer cancel()
		err := e.Update(ctx, r)
		if err != nil {
			return err
		}
		rs.log.Infof("bus %s rule %s updated", r.BusName, r.Name)
		return nil
	}

	// add
	exec, err := rs.newExecutorFunc(rs.ctx, rs.baseLog, r, rs.executorOpts...)
	if err != nil {
		return err
	}
	newRulesPerBus := make(map[string]Executor, len(rulesPerBus)+1)
	for name, e := range rulesPerBus {
		newRulesPerBus[name] = e
	}
	newRulesPerBus[r.Name] = exec
	rs.executors.Store(r.BusName, newRulesPerBus)
	rs.log.Infof("bus %s rule %s added", r.BusName, r.Name)
	return nil
}

func (rs *rules) deleteRule(key string) error {
	names := strings.Split(key, ":")
	if len(names) != 2 { //nolint:mnd
		return fmt.Errorf("invalid key: %s", key)
	}
	busName, ruleName := names[0], names[1]
	if v, ok := rs.executors.Load(busName); ok {
		rulesPerBus := v.(map[string]Executor)
		if e, ok1 := rulesPerBus[ruleName]; ok1 {
			newRulesPerBus := make(map[string]Executor, len(rulesPerBus)-1)
			for name, exec := range rulesPerBus {
				if name != ruleName {
					newRulesPerBus[name] = exec
				}
			}
			rs.executors.Store(busName, newRulesPerBus)
			rs.log.Infof("bus %s rule %s deleted", busName, ruleName)
			err := e.Close()
			if err != nil {
				rs.log.Errorf("close executor failed: %v", err)
			}
			return nil
		}
	}
	return nil
}

type handler struct {
	log *log.Helper

	reflector informer.Reflector
	rs        *rules
}

func newHandler(logger log.Logger, reflector informer.Reflector, rs *rules) informer.Handler {
	return &handler{
		log: log.NewHelper(log.With(
			logger,
			"module", "rule/handler",
			"caller", log.DefaultCaller,
		)),
		reflector: reflector,
		rs:        rs,
	}
}

func (h *handler) Handle(key string) error {
	if v, ok := h.reflector.Get(key); ok { // Add or update
		r := v.(*Rule)
		return h.rs.updateRule(r)
	}
	return h.rs.deleteRule(key)
}
