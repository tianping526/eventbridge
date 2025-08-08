package data

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"github.com/tianping526/eventbridge/app/service/internal/conf"
)

func NewRedisCmd(
	conf *conf.Bootstrap, m *Metric, l log.Logger,
) (redis.Cmdable, func(), error) {
	logger := log.NewHelper(log.With(l, "module", "data/redis", "caller", log.DefaultCaller))
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:          conf.Data.Redis.Addrs,
		MasterName:     conf.Data.Redis.MasterName,
		Password:       conf.Data.Redis.Password,
		DB:             int(conf.Data.Redis.DbIndex),
		DialTimeout:    conf.Data.Redis.DialTimeout.AsDuration(),
		ReadTimeout:    conf.Data.Redis.ReadTimeout.AsDuration(),
		WriteTimeout:   conf.Data.Redis.WriteTimeout.AsDuration(),
		RouteByLatency: true,
	})
	// Enable tracing instrumentation.
	err := redisotel.InstrumentTracing(client)
	if err != nil {
		return nil, nil, err
	}
	client.AddHook(NewRedisMetricHook(
		WithRedisEndpointAddr(conf.Data.Redis.Addrs...),
		WithRedisRequestsDuration(m.CacheDurationSec),
	))
	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Second*2)
	defer cancelFunc()
	err = client.Ping(timeout).Err()
	if err != nil {
		return nil, nil, fmt.Errorf("redis connect error: %v", err)
	}
	return client, func() {
		err = client.Close()
		if err != nil {
			logger.Errorf("failed closing redis client: %v", err)
		}
	}, nil
}
