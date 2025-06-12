package transform

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

var (
	_                     rule.NewTransformerFunc = NewTransformer
	newTransformFunctions                         = map[string]newTransformFunc{}
)

type (
	transformFunc    func(ctx context.Context, ext *rule.EventExt) (interface{}, error)
	newTransformFunc func(ctx context.Context, logger *log.Helper, value string, tmpl *string) (transformFunc, error)
)

// registerTransformFunc register transform function,
// note that this function cannot be called in more than one goroutine.
// recommended for use in func init() only
func registerTransformFunc(name string, newFunc newTransformFunc) {
	newTransformFunctions[name] = newFunc
}

func NewTransformer(ctx context.Context, logger log.Logger, target *rule.Target) (rule.Transformer, error) {
	lg := log.NewHelper(log.With(
		logger,
		"module", "transform/transformer",
		"caller", log.DefaultCaller,
	))
	fcs := make(map[string]transformFunc, len(target.Params))
	for _, tp := range target.Params {
		newFunc, ok := newTransformFunctions[tp.Form]
		if !ok {
			return nil, fmt.Errorf("unknown transformer(form=%s)", tp.Form)
		}
		fc, err := newFunc(ctx, lg, tp.Value, tp.Template)
		if err != nil {
			return nil, err
		}
		fcs[tp.Key] = fc
	}
	return &transformer{
		transformFunctions: fcs,
		log:                lg,
	}, nil
}

type transformer struct {
	log                *log.Helper
	transformFunctions map[string]transformFunc
}

// Transform if `target.Params` is empty, the entire original event is returned.
// The event is modified and returned because it is known that
// the upper layer assigns a separate event to each transformer,
// rather than generating a new event
func (t *transformer) Transform(ctx context.Context, event *rule.EventExt) (*rule.EventExt, error) {
	if len(t.transformFunctions) == 0 {
		data, err := protojson.Marshal(event.Event)
		if err != nil {
			return nil, err
		}
		event.Event.Data = string(data)
	} else {
		transformed := make(map[string]interface{}, len(t.transformFunctions))
		for key, fc := range t.transformFunctions {
			val, err := fc(ctx, event)
			if err != nil {
				return nil, err
			}
			transformed[key] = val
		}
		data, err := json.Marshal(transformed)
		if err != nil {
			return nil, err
		}
		event.Event.Data = string(data)
	}

	return event, nil
}
