package target

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/xeipuuv/gojsonschema"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

func init() {
	registerDispatcher(
		"noopDispatcher",
		newNoopDispatcher,
		`{}`)
}

type noopDispatcher struct{}

func newNoopDispatcher(
	_ context.Context,
	_ log.Logger,
	_ *rule.Target,
	_ *gojsonschema.Schema,
) (rule.Dispatcher, error) {
	return &noopDispatcher{}, nil
}

func (d *noopDispatcher) Dispatch(_ context.Context, _ *rule.EventExt) error {
	return nil
}

func (d *noopDispatcher) Close() error {
	return nil
}
