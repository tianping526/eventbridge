package data

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/redis/go-redis/extra/rediscmd/v9"
	"github.com/redis/go-redis/v9"
)

const (
	metricLabelCacheName   = "name"
	metricLabelCacheAddrs  = "addrs"
	metricLabelCacheResult = "res"
)

var _ redis.Hook = (*RedisMetricHook)(nil)

// RedisMetricsOption is a metrics option.
type RedisMetricsOption func(*redisMetricsOptions)

// WithRedisRequestsDuration with requests duration(s).
func WithRedisRequestsDuration(c metric.Float64Histogram) RedisMetricsOption {
	return func(o *redisMetricsOptions) {
		o.requestsDuration = c
	}
}

// WithRedisEndpointAddr with db Addr.
func WithRedisEndpointAddr(addrs ...string) RedisMetricsOption {
	return func(o *redisMetricsOptions) {
		o.Addrs = strings.Join(addrs, ",")
	}
}

func NewRedisMetricHook(opts ...RedisMetricsOption) *RedisMetricHook {
	op := redisMetricsOptions{}
	for _, o := range opts {
		o(&op)
	}

	return &RedisMetricHook{op: op}
}

type redisMetricsOptions struct {
	// histogram: cache_client_requests_duration_seconds_bucket{"name", "addrs", "res"}
	requestsDuration metric.Float64Histogram
	Addrs            string
}

type RedisMetricHook struct {
	op redisMetricsOptions
}

func (rmh *RedisMetricHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (rmh *RedisMetricHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		var err error
		if rmh.op.requestsDuration != nil {
			startTime := time.Now()
			defer func() {
				res := "ok"
				if err != nil && !errors.Is(redis.Nil, err) {
					res = fmt.Sprintf("%T", err)
				}
				rmh.op.requestsDuration.Record(
					ctx, time.Since(startTime).Seconds(),
					metric.WithAttributes(
						attribute.String(metricLabelCacheName, cmd.FullName()),
						attribute.String(metricLabelCacheAddrs, rmh.op.Addrs),
						attribute.String(metricLabelCacheResult, res),
					),
				)
			}()
		}
		err = next(ctx, cmd)
		return err
	}
}

func (rmh *RedisMetricHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cs []redis.Cmder) error {
		var err error
		if rmh.op.requestsDuration != nil {
			startTime := time.Now()
			defer func() {
				summary, _ := rediscmd.CmdsString(cs)
				res := "ok"
				if err != nil && !errors.Is(redis.Nil, err) {
					res = fmt.Sprintf("%T", err)
				}
				rmh.op.requestsDuration.Record(
					ctx, time.Since(startTime).Seconds(),
					metric.WithAttributes(
						attribute.String(metricLabelCacheName, fmt.Sprintf("pipeline%s", summary)),
						attribute.String(metricLabelCacheAddrs, rmh.op.Addrs),
						attribute.String(metricLabelCacheResult, res),
					),
				)
			}()
		}
		err = next(ctx, cs)
		return err
	}
}
