package data

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/tianping526/eventbridge/app/service/internal/conf"
)

func NewConfig(ai *conf.AppInfo) (config.Config, func(), error) {
	fc := config.New(
		config.WithSource(
			file.NewSource(ai.FlagConf),
			env.NewSource(),
		),
	)
	if err := fc.Load(); err != nil {
		return nil, nil, err
	}
	return fc, func() {
		err := fc.Close()
		if err != nil {
			fmt.Printf("close file config(%s) error(%s))\n", ai.FlagConf, err)
		}
	}, nil
}

func NewConfigBootstrap(c config.Config) (*conf.Bootstrap, error) {
	var bc conf.Bootstrap
	if err := c.Value("bootstrap").Scan(&bc); err != nil {
		return nil, err
	}

	// trace
	if bc.Trace != nil && len(bc.Trace.EndpointUrl) == 0 {
		return nil, fmt.Errorf("trace.endpoint_url config is required")
	}

	// server
	if bc.Server == nil {
		return nil, fmt.Errorf("server config is required")
	}

	// server.grpc
	if bc.Server.Grpc == nil {
		return nil, fmt.Errorf("server.grpc config is required")
	}
	if bc.Server.Grpc.Network == "" {
		bc.Server.Grpc.Network = "tcp"
	}
	if bc.Server.Grpc.Addr == "" {
		bc.Server.Grpc.Addr = ":9000"
	}
	if bc.Server.Grpc.Timeout == nil {
		bc.Server.Grpc.Timeout = durationpb.New(time.Second)
	}

	// server.http
	if bc.Server.Http == nil {
		return nil, fmt.Errorf("server.http config is required")
	}
	if bc.Server.Http.Network == "" {
		bc.Server.Http.Network = "tcp"
	}
	if bc.Server.Http.Addr == "" {
		bc.Server.Http.Addr = ":8000"
	}
	if bc.Server.Http.Timeout == nil {
		bc.Server.Http.Timeout = durationpb.New(time.Second)
	}

	// data
	if bc.Data == nil {
		return nil, fmt.Errorf("data config is required")
	}

	// data.database
	if bc.Data.Database == nil {
		return nil, fmt.Errorf("data.database config is required")
	}
	if bc.Data.Database.Driver == "" {
		bc.Data.Database.Driver = dialect.Postgres
	}
	if bc.Data.Database.Source == "" {
		return nil, fmt.Errorf("data.database.source config is required")
	}
	if bc.Data.Database.MaxOpen <= 0 {
		bc.Data.Database.MaxOpen = 100
	}
	if bc.Data.Database.MaxIdle <= 0 {
		bc.Data.Database.MaxIdle = 10
	}
	if bc.Data.Database.ConnMaxLifeTime == nil {
		bc.Data.Database.ConnMaxLifeTime = durationpb.New(0)
	}
	if bc.Data.Database.ConnMaxIdleTime == nil {
		bc.Data.Database.ConnMaxIdleTime = durationpb.New(300 * time.Second)
	}

	// data.redis
	if bc.Data.Redis == nil {
		return nil, fmt.Errorf("data.redis config is required")
	}
	if len(bc.Data.Redis.Addrs) == 0 {
		return nil, fmt.Errorf("data.redis.addrs config is required")
	}
	if bc.Data.Redis.DialTimeout == nil {
		bc.Data.Redis.DialTimeout = durationpb.New(1 * time.Second)
	}
	if bc.Data.Redis.ReadTimeout == nil {
		bc.Data.Redis.ReadTimeout = durationpb.New(200 * time.Millisecond)
	}
	if bc.Data.Redis.WriteTimeout == nil {
		bc.Data.Redis.WriteTimeout = durationpb.New(200 * time.Millisecond)
	}

	// data.default_mq
	if bc.Data.DefaultMq == "" {
		return nil, fmt.Errorf("data.default_mq config is required")
	}

	return &bc, nil
}
