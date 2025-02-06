package transform

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

func init() {
	registerTransformFunc("CONSTANT", newTransformFuncConstant)
}

func newTransformFuncConstant(_ context.Context, _ *log.Helper, value string, _ *string) (transformFunc, error) {
	return func(_ context.Context, _ *rule.EventExt) (interface{}, error) {
		return value, nil
	}, nil
}
