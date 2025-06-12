//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"
	"github.com/tianping526/eventbridge/app/service/internal/biz"
	"github.com/tianping526/eventbridge/app/service/internal/conf"
	"github.com/tianping526/eventbridge/app/service/internal/data"
	"github.com/tianping526/eventbridge/app/service/internal/server"
	"github.com/tianping526/eventbridge/app/service/internal/service"
)

// wireApp init kratos application.
func wireApp(*conf.AppInfo) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
