package data

import (
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/tianping526/eventbridge/app/service/internal/conf"
)

type Metric struct {
	CacheHits            metric.Int64Counter
	CacheMisses          metric.Int64Counter
	CacheDurationSec     metric.Float64Histogram
	CodeTotal            metric.Int64Counter
	DurationSec          metric.Float64Histogram
	DbDurationSec        metric.Float64Histogram
	PostEventCount       metric.Int64Counter
	PostEventDurationSec metric.Float64Histogram
}

func NewMetric(ai *conf.AppInfo) (*Metric, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	meter := provider.Meter(ai.Name)

	cacheHits, err := meter.Int64Counter(
		"cache_redis_hits_total",
		metric.WithUnit("{call}"),
		metric.WithDescription("Redis hits total."),
	)
	if err != nil {
		return nil, err
	}
	cacheMisses, err := meter.Int64Counter(
		"cache_redis_misses_total",
		metric.WithUnit("{call}"),
		metric.WithDescription("Redis misses total."),
	)
	if err != nil {
		return nil, err
	}
	cacheDurationSec, err := meter.Float64Histogram(
		"cache_client_requests_duration",
		metric.WithUnit("s"),
		metric.WithDescription("Cache requests duration."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1),
	)
	if err != nil {
		return nil, err
	}
	codeTotal, err := metrics.DefaultRequestsCounter(meter, metrics.DefaultServerRequestsCounterName)
	if err != nil {
		return nil, err
	}
	durationSec, err := metrics.DefaultSecondsHistogram(meter, metrics.DefaultServerSecondsHistogramName)
	if err != nil {
		return nil, err
	}
	dbDurationSec, err := meter.Float64Histogram(
		"db_client_requests_duration",
		metric.WithUnit("s"),
		metric.WithDescription("DB requests duration."),
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
		"job_event_post_duration",
		metric.WithUnit("s"),
		metric.WithDescription("Post event duration."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1),
	)
	if err != nil {
		return nil, err
	}
	return &Metric{
		CacheHits:            cacheHits,
		CacheMisses:          cacheMisses,
		CacheDurationSec:     cacheDurationSec,
		CodeTotal:            codeTotal,
		DurationSec:          durationSec,
		DbDurationSec:        dbDurationSec,
		PostEventCount:       postEventCount,
		PostEventDurationSec: postEventDurationSec,
	}, nil
}
