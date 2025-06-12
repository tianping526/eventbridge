package pattern

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

func init() {
	registerMatchFunc("suffix", newMatchFuncSuffix)
}

func newMatchFuncSuffix(_ context.Context, _ *log.Helper, spec interface{}) (matchFunc, error) {
	suffix, ok := spec.(string)
	if !ok {
		return nil, fmt.Errorf("suffix spec(type=%T, val=%+v) should be string", spec, spec)
	}
	return func(val interface{}) (bool, error) {
		strVal, ok := val.(string)
		if !ok {
			return false, nil
		}
		return strings.HasSuffix(strVal, suffix), nil
	}, nil
}
