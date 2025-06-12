package data

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	"github.com/tianping526/eventbridge/app/internal/rule"
	"github.com/tianping526/eventbridge/app/job/internal/biz"
)

const (
	maxFanout = 20

	metricPostEventBusName = "bus_name"
	metricPostEventType    = "event_type"
	metricPostEventResult  = "result"
)

var otr = otel.Tracer("/app/job/internal/data/event")

type eventRepo struct {
	data *Data
	log  *log.Helper
}

func NewEventRepo(data *Data, logger log.Logger) biz.EventRepo {
	return &eventRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/event")),
	}
}

func (repo *eventRepo) HandleEvent(ctx context.Context, evt *rule.EventExt) error {
	executors, err := repo.data.rs.GetExecutors(evt.BusName)
	if err != nil {
		return fmt.Errorf("get bus(%s) executors err: %s", evt.BusName, err)
	}
	if len(executors) == 0 {
		if evt.RuleName != "" {
			return fmt.Errorf("no executor for rule(%s) in bus(%s) to dispatch target event", evt.RuleName, evt.BusName)
		}
		return nil
	}

	if evt.RuleName != "" { // consume backoff queue and dispatch target event
		exec, ok := executors[evt.RuleName]
		if !ok {
			return fmt.Errorf("no executor for rule(%s) in bus(%s) to dispatch target event", evt.RuleName, evt.BusName)
		}
		return repo.retryDispatchTargetEvent(ctx, exec, evt)
	}

	// consume source event. match rule, transform event and dispatch target event.
	// send to backoff queue if dispatch failed.
	if len(executors) == 1 {
		for ruleName, exec := range executors {
			return repo.handleSourceEvent(ctx, ruleName, exec, evt)
		}
	} else {
		eg, c := errgroup.WithContext(ctx)
		eg.SetLimit(maxFanout)
		for ruleName, exec := range executors {
			eg.Go(func() error {
				return repo.handleSourceEvent(c, ruleName, exec, evt)
			})
		}
		return eg.Wait()
	}
	return nil
}

func (repo *eventRepo) retryDispatchTargetEvent(
	ctx context.Context, exec rule.Executor, evt *rule.EventExt,
) (err error) {
	// trace
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(semconv.MessagingConsumerIDKey.String(evt.Key()))
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}()

	// dispatch
	err = exec.Dispatch(ctx, evt)
	if err != nil {
		if rule.IsDispatcherNotFound(err) {
			err = fmt.Errorf(
				"no dispatcher for target(bus name: %s, rule name: %s, target id: %d) to dispatch",
				evt.BusName, evt.RuleName, evt.TargetId,
			)
			return
		}
		err = fmt.Errorf(
			"dispatch target(bus name: %s, rule name: %s, target id: %d) err: %s",
			evt.BusName, evt.RuleName, evt.TargetId, err,
		)
		return
	}
	return
}

func (repo *eventRepo) handleSourceEvent(
	ctx context.Context, ruleName string, exec rule.Executor, evt *rule.EventExt,
) (err error) {
	// trace
	var span trace.Span
	ctx, span = otr.Start(
		ctx, "HandleEvent", trace.WithSpanKind(trace.SpanKindConsumer),
	)
	span.SetAttributes(semconv.MessagingConsumerIDKey.String(evt.Key()))
	span.SetAttributes(attribute.String("ruleName", ruleName))
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}()

	// match rule
	var ok bool
	ok, err = exec.Pattern(ctx, evt)
	if err != nil {
		if rule.IsMatcherNotFound(err) {
			repo.log.Errorf("no matcher for rule(%s) in bus(%s) to match event", ruleName, evt.BusName)
			err = nil
			return
		}
		err = fmt.Errorf("match event err: %s, bus name: %s, rule name: %s", err, evt.BusName, ruleName)
		return
	}
	if !ok {
		return
	}

	if ctx.Err() != nil { // other goroutine failed
		err = context.Cause(ctx)
		return
	}

	// transform event
	var targetEvents []*rule.EventExt
	targetEvents, err = exec.Transform(ctx, evt)
	if err != nil {
		if rule.IsTransformerNotFound(err) {
			repo.log.Errorf("no transformer for rule(%s) in bus(%s) to transform event", ruleName, evt.BusName)
			err = nil
			return
		}
		err = fmt.Errorf("transform event err: %s, bus name: %s, rule name: %s", err, evt.BusName, ruleName)
		return
	}

	if ctx.Err() != nil { // other goroutine failed
		err = context.Cause(ctx)
		return
	}

	// dispatch target event
	if len(targetEvents) == 0 {
		repo.log.Errorf("no target event for rule(%s) in bus(%s) to dispatch", ruleName, evt.BusName)
		return
	}
	if len(targetEvents) == 1 {
		for _, targetEvt := range targetEvents {
			err = repo.dispatchTargetEvent(ctx, exec, targetEvt)
			return
		}
	}
	eg, c := errgroup.WithContext(ctx)
	eg.SetLimit(maxFanout)
	for _, targetEvt := range targetEvents {
		eg.Go(func() error {
			return repo.dispatchTargetEvent(c, exec, targetEvt)
		})
	}
	err = eg.Wait()
	return
}

func (repo *eventRepo) dispatchTargetEvent(ctx context.Context, exec rule.Executor, evt *rule.EventExt) (err error) {
	// trace
	var span trace.Span
	ctx, span = otr.Start(
		ctx, "DispatchTargetEvent", trace.WithSpanKind(trace.SpanKindProducer),
	)
	span.SetAttributes(semconv.MessagingConsumerIDKey.String(evt.Key()))
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}()

	// dispatch
	err = exec.Dispatch(ctx, evt)
	if err == nil {
		return
	}
	if rule.IsDispatcherNotFound(err) {
		repo.log.Errorf(
			"no dispatcher for target(bus name: %s, rule name: %s, target id: %d) to dispatch",
			evt.BusName, evt.RuleName, evt.TargetId,
		)
		err = nil
		return
	}

	// dispatch failed, send it to backoff queue
	startTime := time.Now()
	err = repo.data.sd.Send(ctx, evt)
	repo.data.m.PostEventDurationSec.Record(
		ctx, time.Since(startTime).Seconds(),
		metric.WithAttributes(
			attribute.String(metricPostEventBusName, evt.BusName),
			attribute.String(metricPostEventType, "target_event"),
		),
	)
	if err != nil {
		repo.data.m.PostEventCount.Add(
			ctx, 1,
			metric.WithAttributes(
				attribute.String(metricPostEventBusName, evt.BusName),
				attribute.String(metricPostEventType, "target_event"),
				attribute.String(metricPostEventResult, fmt.Sprintf("%T", err)),
			),
		)
		err = fmt.Errorf("send event(%s) to backoff queue err: %s", evt.Key(), err)
		return
	}
	repo.data.m.PostEventCount.Add(
		ctx, 1,
		metric.WithAttributes(
			attribute.String(metricPostEventBusName, evt.BusName),
			attribute.String(metricPostEventType, "target_event"),
			attribute.String(metricPostEventResult, "ok"),
		),
	)
	return
}
