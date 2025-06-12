//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package test

import (
	"github.com/google/wire"
	"github.com/tianping526/eventbridge/app/service/internal/biz"
	"github.com/tianping526/eventbridge/app/service/internal/conf"
	"github.com/tianping526/eventbridge/app/service/internal/data"
	"github.com/tianping526/eventbridge/app/service/internal/service"
)

func wireService(*conf.AppInfo) (*service.EventBridgeService, func(), error) {
	panic(wire.Build(data.ProviderSet, biz.ProviderSet, service.ProviderSet))
}
