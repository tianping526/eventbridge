package pattern

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

func init() {
	registerMatchFunc("anything-but", newMatchFuncAnythingBut)
}

func newMatchFuncAnythingBut(ctx context.Context, logger *log.Helper, spec interface{}) (matchFunc, error) {
	switch abs := spec.(type) {
	case string, float64: // match value
		return func(val interface{}) (bool, error) {
			return abs != val, nil
		}, nil
	case []interface{}: // match an array
		mapFcs := make(map[string][]matchFunc)
		vls := make(map[interface{}]bool)
		for _, item := range abs {
			patternMap, ok := item.(map[string]interface{})
			if !ok { // value
				vls[item] = true
				continue
			}
			for name, s := range patternMap { // pattern
				fcs, ok := mapFcs[name]
				if !ok { // new match func
					newFunc, ok := newMatchFunctions[name]
					if !ok {
						return nil, fmt.Errorf("unknown match func(name=%s)", name)
					}
					fc, err := newFunc(ctx, logger, s)
					if err != nil {
						return nil, err
					}
					mapFcs[name] = []matchFunc{fc}
					continue
				}
				// append matcher
				newFunc := newMatchFunctions[name]
				fc, err := newFunc(ctx, logger, s)
				if err != nil {
					return nil, err
				}
				mapFcs[name] = append(fcs, fc)
			}
		}
		return func(val interface{}) (bool, error) {
			// if any value matches, the match fails
			mv, ok := val.([]interface{})
			if ok {
				for _, v := range mv { // any success
					if vls[v] {
						return false, nil
					}
				}
			} else {
				if vls[val] {
					return false, nil
				}
			}

			if len(mapFcs) == 0 {
				return true, nil
			}

			// returns failure if all patterns match successfully
			for _, fcs := range mapFcs { // all success
				res := false
				for _, fc := range fcs {
					mr, me := fc(val)
					if me != nil {
						return false, me
					}
					if mr {
						res = true
						break
					}
				}
				if !res {
					return true, nil
				}
			}

			return false, nil
		}, nil
	case map[string]interface{}:
		if len(abs) == 0 {
			return func(_ interface{}) (bool, error) {
				return true, nil
			}, nil
		}

		mapFcs := make(map[string][]matchFunc)
		for name, s := range abs { // pattern
			fcs, ok := mapFcs[name]
			if !ok { // new match func
				newFunc, ok := newMatchFunctions[name]
				if !ok {
					return nil, fmt.Errorf("unknown match func(name=%s)", name)
				}
				fc, err := newFunc(ctx, logger, s)
				if err != nil {
					return nil, err
				}
				mapFcs[name] = []matchFunc{fc}
				continue
			}
			// append matcher
			newFunc := newMatchFunctions[name]
			fc, err := newFunc(ctx, logger, s)
			if err != nil {
				return nil, err
			}
			mapFcs[name] = append(fcs, fc)
		}
		return func(val interface{}) (bool, error) {
			// returns failure if all patterns match successfully
			for _, fcs := range mapFcs { // all success
				res := false
				for _, fc := range fcs {
					mr, me := fc(val)
					if me != nil {
						return false, me
					}
					if mr {
						res = true
						break
					}
				}
				if !res {
					return true, nil
				}
			}

			return false, nil
		}, nil
	default:
		return nil, fmt.Errorf("anything-but unexpect pattern(type=%T, val=%+v)", spec, spec)
	}
}
