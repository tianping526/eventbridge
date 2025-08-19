package target

import (
	"context"
	"encoding/json/jsontext"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/xeipuuv/gojsonschema"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

var (
	_                      rule.NewDispatcherFunc = NewDispatcher
	newDispatcherFunctions                        = map[string]newDispatcherFunc{}
	dispatcherSchemas                             = map[string]string{}
	validators                                    = map[string]*gojsonschema.Schema{}
)

type newDispatcherFunc func(context.Context, log.Logger, *rule.Target, *gojsonschema.Schema) (rule.Dispatcher, error)

// registerDispatcher register Dispatcher,
// note that this function cannot be called in more than one goroutine.
// recommended for use in func init() only
func registerDispatcher(name string, newDispatcherFunc newDispatcherFunc, schema string) {
	newDispatcherFunctions[name] = newDispatcherFunc
	schemaBytes := []byte(schema)
	err := (*jsontext.Value)(&schemaBytes).Compact()
	if err != nil {
		panic(fmt.Errorf("failed to compact schema for %s: %w", name, err))
	}
	dispatcherSchemas[name] = string(schemaBytes)
	v, err := gojsonschema.NewSchema(gojsonschema.NewStringLoader(schema))
	if err != nil {
		panic(err)
	}
	validators[name] = v
}

type dispatcher struct {
	log        *log.Helper
	dispatcher rule.Dispatcher
}

func NewDispatcher(ctx context.Context, logger log.Logger, target *rule.Target) (rule.Dispatcher, error) {
	newFunc, ok := newDispatcherFunctions[target.Type]
	if !ok {
		return nil, fmt.Errorf("unknown target type:%s", target.Type)
	}
	validator := validators[target.Type]
	d, err := newFunc(ctx, logger, target, validator)
	if err != nil {
		return nil, err
	}
	return &dispatcher{
		dispatcher: d,
		log: log.NewHelper(log.With(
			logger,
			"module", "target/dispatcher",
			"caller", log.DefaultCaller,
		)),
	}, nil
}

func (d *dispatcher) Dispatch(ctx context.Context, event *rule.EventExt) error {
	return d.dispatcher.Dispatch(ctx, event)
}

func (d *dispatcher) Close() error {
	return d.dispatcher.Close()
}

func ListAllDispatcherParamsSchema() map[string]string {
	schema := make(map[string]string, len(dispatcherSchemas))
	for key, val := range dispatcherSchemas {
		schema[key] = val
	}
	return schema
}
