package pattern

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

func init() {
	registerMatchFunc("exists", newMatchFuncExists)
}

func newMatchFuncExists(_ context.Context, _ *log.Helper, spec interface{}) (matchFunc, error) {
	exists, ok := spec.(bool)
	if !ok {
		return nil, fmt.Errorf("exists spec(type=%T, val=%+v) should be bool", spec, spec)
	}
	if exists {
		return func(val interface{}) (bool, error) {
			if rule.IsNotExistsVal(val) {
				return false, nil
			}
			return true, nil
		}, nil
	}
	return func(val interface{}) (bool, error) {
		if rule.IsNotExistsVal(val) {
			return true, nil
		}
		return false, nil
	}, nil
}
