package data

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"github.com/tianping526/eventbridge/app/service/internal/conf"
)

func NewRedisCmd(conf *conf.Bootstrap, m *Metric) (redis.Cmdable, func(), error) {
	client := redis.NewClient(&redis.Options{
		Addr:         conf.Data.Redis.Addr,
		Password:     conf.Data.Redis.Password,
		DB:           int(conf.Data.Redis.DbIndex),
		DialTimeout:  conf.Data.Redis.DialTimeout.AsDuration(),
		ReadTimeout:  conf.Data.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: conf.Data.Redis.WriteTimeout.AsDuration(),
	})
	// Enable tracing instrumentation.
	err := redisotel.InstrumentTracing(client)
	if err != nil {
		return nil, nil, err
	}
	client.AddHook(NewRedisMetricHook(
		WithRedisEndpointAddr(conf.Data.Redis.Addr),
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
			fmt.Printf("failed closing redis client: %v", err)
		}
	}, nil
}
