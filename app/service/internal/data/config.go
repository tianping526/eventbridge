package data

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"

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

	return &bc, nil
}
