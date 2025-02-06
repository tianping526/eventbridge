package pattern

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

func init() {
	registerMatchFunc("numeric", newMatchFuncNumeric)
}

type comparator func(val float64) bool

func newMatchFuncNumeric(_ context.Context, _ *log.Helper, spec interface{}) (matchFunc, error) {
	numeric, ok := spec.([]interface{})
	if !ok {
		return nil, fmt.Errorf("numeric spec(type=%T, val=%+v) should be []interface{}", spec, spec)
	}

	if len(numeric) < 2 { //nolint:mnd
		return func(_ interface{}) (bool, error) {
			return false, nil
		}, nil
	}

	comparators := make([]comparator, 0, len(numeric)/2)
	for i := 0; i < len(numeric)/2; i++ {
		num, ok := numeric[2*i+1].(float64)
		if !ok {
			return nil, fmt.Errorf(
				"numeric spec should be compared with number, not %T(value=%+v)",
				numeric[2*i+1], numeric[2*i+1],
			)
		}
		switch numeric[2*i] {
		case ">":
			comparators = append(comparators, func(val float64) bool {
				return val > num
			})
		case ">=":
			comparators = append(comparators, func(val float64) bool {
				return val >= num
			})
		case "=":
			comparators = append(comparators, func(val float64) bool {
				return val == num
			})
		case "<":
			comparators = append(comparators, func(val float64) bool {
				return val < num
			})
		case "<=":
			comparators = append(comparators, func(val float64) bool {
				return val <= num
			})
		default:
			return nil, fmt.Errorf(
				"unknown comparison operator %+v(index=%d)",
				numeric[2*i], 2*i,
			)
		}
	}

	return func(val interface{}) (bool, error) {
		floatVal, ok := val.(float64)
		if !ok {
			return false, nil
		}
		for _, cmp := range comparators {
			if !cmp(floatVal) {
				return false, nil
			}
		}
		return true, nil
	}, nil
}
