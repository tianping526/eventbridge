package data

import (
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/tianping526/eventbridge/app/job/internal/conf"
)

type Metric struct {
	ServerCodeTotal      metric.Int64Counter
	ServerDurationSec    metric.Float64Histogram
	DbDurationSec        metric.Float64Histogram
	PostEventCount       metric.Int64Counter
	PostEventDurationSec metric.Float64Histogram
	RunningWorkers       metric.Int64Gauge
	RuleExecTotal        metric.Int64Counter
	RuleExecSec          metric.Float64Histogram
}

func NewMetric(ai *conf.AppInfo) (*Metric, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	meter := provider.Meter(ai.Name)

	serverCodeTotal, err := metrics.DefaultRequestsCounter(meter, metrics.DefaultServerRequestsCounterName)
	if err != nil {
		return nil, err
	}
	serverDurationSec, err := metrics.DefaultSecondsHistogram(meter, metrics.DefaultServerSecondsHistogramName)
	if err != nil {
		return nil, err
	}
	dbDurationSec, err := meter.Float64Histogram(
		"db_client_requests_duration_sec",
		metric.WithUnit("s"),
		metric.WithDescription("DB requests duration(sec)."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1),
	)
	if err != nil {
		return nil, err
	}
	postEventCount, err := meter.Int64Counter(
		"job_event_post_count",
		metric.WithUnit("{call}"),
		metric.WithDescription("Number of events that have been posted."),
	)
	if err != nil {
		return nil, err
	}
	postEventDurationSec, err := meter.Float64Histogram(
		"job_event_post_duration_sec",
		metric.WithUnit("s"),
		metric.WithDescription("Post event duration(sec)."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1),
	)
	if err != nil {
		return nil, err
	}
	runningWorkers, err := meter.Int64Gauge(
		"job_event_running_workers",
		metric.WithUnit("{worker}"),
		metric.WithDescription("Number of workers per bus that are currently running."),
	)
	if err != nil {
		return nil, err
	}
	ruleExecTotal, err := meter.Int64Counter(
		"job_rule_execute_total",
		metric.WithUnit("{call}"),
		metric.WithDescription("The total number of rule executes."),
	)
	if err != nil {
		return nil, err
	}
	ruleExecSec, err := meter.Float64Histogram(
		"job_rule_execute_duration_sec",
		metric.WithUnit("s"),
		metric.WithDescription("Rule executes duration(sec)."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1),
	)
	if err != nil {
		return nil, err
	}

	return &Metric{
		ServerCodeTotal:      serverCodeTotal,
		ServerDurationSec:    serverDurationSec,
		DbDurationSec:        dbDurationSec,
		PostEventCount:       postEventCount,
		PostEventDurationSec: postEventDurationSec,
		RunningWorkers:       runningWorkers,
		RuleExecTotal:        ruleExecTotal,
		RuleExecSec:          ruleExecSec,
	}, nil
}
