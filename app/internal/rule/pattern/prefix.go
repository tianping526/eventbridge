package pattern

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

func init() {
	registerMatchFunc("prefix", newMatchFuncPrefix)
}

func newMatchFuncPrefix(_ context.Context, _ *log.Helper, spec interface{}) (matchFunc, error) {
	prefix, ok := spec.(string)
	if !ok {
		return nil, fmt.Errorf("prefix spec(type=%T, val=%v) should be string", spec, spec)
	}
	return func(val interface{}) (bool, error) {
		strVal, ok := val.(string)
		if !ok {
			return false, nil
		}
		return strings.HasPrefix(strVal, prefix), nil
	}, nil
}
