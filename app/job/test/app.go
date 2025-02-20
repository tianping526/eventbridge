package test

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/tianping526/eventbridge/app/internal/event"
	"github.com/tianping526/eventbridge/app/job/internal/conf"
)

func newApp(
	logger log.Logger, rr registry.Registrar, hs *http.Server, es *event.Server, cfg *conf.AppInfo,
) *kratos.App {
	return kratos.New(
		kratos.ID(cfg.Id),
		kratos.Name(cfg.Name),
		kratos.Version(cfg.Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			hs,
			es,
		),
		kratos.Registrar(rr),
	)
}
