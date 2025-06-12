//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package test

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"
	"github.com/tianping526/eventbridge/app/job/internal/biz"
	"github.com/tianping526/eventbridge/app/job/internal/conf"
	"github.com/tianping526/eventbridge/app/job/internal/data"
	"github.com/tianping526/eventbridge/app/job/internal/server"
)

// wireApp init kratos application.
func wireApp(*conf.AppInfo) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, newApp))
}
