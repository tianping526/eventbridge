package data

import (
	"github.com/go-kratos/kratos/v2/registry"

	"github.com/tianping526/eventbridge/app/job/internal/conf"
)

func NewRegistrar(_ *conf.Bootstrap) (registry.Registrar, error) {
	return nil, nil
}
