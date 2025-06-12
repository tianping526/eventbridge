package transform

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

func init() {
	registerTransformFunc("JSONPATH", newTransformFuncJsonpath)
}

func newTransformFuncJsonpath(_ context.Context, _ *log.Helper, value string, _ *string) (transformFunc, error) {
	path := strings.Split(value, ".")
	if len(path) != 0 && path[0] == "$" {
		path = path[1:]
	}
	return func(_ context.Context, ext *rule.EventExt) (interface{}, error) {
		val, err := ext.GetFieldByPath(path)
		if err != nil {
			return nil, err
		}
		if rule.IsNotExistsVal(val) {
			return nil, nil
		}
		return val, nil
	}, nil
}
