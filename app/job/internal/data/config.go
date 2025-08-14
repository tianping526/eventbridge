package data

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/tianping526/eventbridge/app/job/internal/conf"
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

	// server.event
	defaultEventConf := &conf.Server_Event{
		SourceTimeout:         durationpb.New(1 * time.Second),
		DelayTimeout:          durationpb.New(1 * time.Second),
		TargetExpDecayTimeout: durationpb.New(1 * time.Second),
		TargetBackoffTimeout:  durationpb.New(1 * time.Second),
		WorkersPerMqTopic:     256,
		RuleParallelism:       20,
		TransformParallelism:  20,
		DispatchParallelism:   20,
	}
	if bc.Server.Event == nil {
		bc.Server.Event = defaultEventConf
	} else {
		if bc.Server.Event.SourceTimeout == nil {
			bc.Server.Event.SourceTimeout = defaultEventConf.SourceTimeout
		}
		if bc.Server.Event.DelayTimeout == nil {
			bc.Server.Event.DelayTimeout = defaultEventConf.DelayTimeout
		}
		if bc.Server.Event.TargetExpDecayTimeout == nil {
			bc.Server.Event.TargetExpDecayTimeout = defaultEventConf.TargetExpDecayTimeout
		}
		if bc.Server.Event.TargetBackoffTimeout == nil {
			bc.Server.Event.TargetBackoffTimeout = defaultEventConf.TargetBackoffTimeout
		}
		if bc.Server.Event.WorkersPerMqTopic <= 1 {
			bc.Server.Event.WorkersPerMqTopic = defaultEventConf.WorkersPerMqTopic
		}
		if bc.Server.Event.RuleParallelism <= 1 {
			bc.Server.Event.RuleParallelism = defaultEventConf.RuleParallelism
		}
		if bc.Server.Event.TransformParallelism <= 1 {
			bc.Server.Event.TransformParallelism = defaultEventConf.TransformParallelism
		}
		if bc.Server.Event.DispatchParallelism <= 1 {
			bc.Server.Event.DispatchParallelism = defaultEventConf.DispatchParallelism
		}
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

	// data.default_mq
	if bc.Data.DefaultMq == "" {
		return nil, fmt.Errorf("data.default_mq config is required")
	}

	return &bc, nil
}
