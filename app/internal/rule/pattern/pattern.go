package pattern

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

var (
	_                 rule.NewMatcherFunc = NewMatcher
	newMatchFunctions                     = map[string]newMatchFunc{}
)

type (
	matchFunc    func(val interface{}) (ok bool, err error)
	newMatchFunc func(ctx context.Context, logger *log.Helper, spec interface{}) (fc matchFunc, err error)

	eventMatchFunc func(ctx context.Context, event *rule.EventExt) (ok bool, err error)
)

// registerMatchFunc register match function,
// note that this function cannot be called in more than one goroutine.
// recommended for use in func init() only
func registerMatchFunc(name string, newFunc newMatchFunc) {
	newMatchFunctions[name] = newFunc
}

// func newMatchFunction(ctx context.Context, name string, logger *log.Helper, spec interface{}) (matchFunc, error) {
//	newFunc, ok := newMatchFunctions[name]
//	if !ok {
//		return nil, fmt.Errorf("unknown matcher(name=%s)", name)
//	}
//	fc, err := newFunc(ctx, logger, spec)
//	if err != nil {
//		return nil, err
//	}
//	return fc, nil
// }

type matcher struct {
	log            *log.Helper
	eventMatchFunc eventMatchFunc
}

func NewMatcher(
	ctx context.Context,
	logger log.Logger,
	filterPattern map[string]interface{},
) (rule.Matcher, error) {
	lg := log.NewHelper(log.With(
		logger,
		"module", "pattern/matcher",
		"caller", log.DefaultCaller,
	))
	if len(filterPattern) == 0 {
		return &matcher{
			eventMatchFunc: func(context.Context, *rule.EventExt) (bool, error) {
				return false, nil
			},
			log: lg,
		}, nil
	}
	emf, err := parsePattern(ctx, lg, []string{}, filterPattern)
	if err != nil {
		return nil, err
	}
	return &matcher{
		eventMatchFunc: emf,
		log:            lg,
	}, nil
}

func parsePattern(
	ctx context.Context,
	logger *log.Helper,
	rootPath []string,
	relatedPattern interface{},
) (eventMatchFunc, error) {
	switch rp := relatedPattern.(type) {
	case string, float64: // match value
		return func(c context.Context, event *rule.EventExt) (bool, error) {
			val, err := event.GetFieldByPath(rootPath)
			if err != nil {
				if rule.IsDataUnmarshalError(err) {
					logger.WithContext(c).Error(err)
					return false, nil
				}
				return false, err
			}
			return rp == val, nil
		}, nil
	case []interface{}: // match an array
		vls := make(map[interface{}]bool)
		orFcs := make([][]matchFunc, 0, len(rp))
		for _, item := range rp {
			patternMap, ok := item.(map[string]interface{})
			if !ok { // value
				vls[item] = true
				continue
			}
			andFcs := make([]matchFunc, 0, len(patternMap))
			for name, spec := range patternMap { // pattern
				newFunc, ok := newMatchFunctions[name]
				if !ok {
					return nil, fmt.Errorf("unknown match func(name=%s)", name)
				}
				fc, err := newFunc(ctx, logger, spec)
				if err != nil {
					return nil, err
				}
				andFcs = append(andFcs, fc)
			}
			orFcs = append(orFcs, andFcs)
		}

		return func(c context.Context, event *rule.EventExt) (bool, error) {
			val, err := event.GetFieldByPath(rootPath)
			if err != nil {
				if rule.IsDataUnmarshalError(err) {
					logger.WithContext(c).Error(err)
					return false, nil
				}
				return false, err
			}

			// if any value matches, the event matches successfully
			mv, ok := val.([]interface{})
			if ok {
				for _, v := range mv { // any success
					if vls[v] {
						return true, nil
					}
				}
			} else {
				if vls[val] {
					return true, nil
				}
			}

			for _, andFcs := range orFcs { // any success
				if len(andFcs) == 0 {
					continue
				}

				res := true
				for _, fc := range andFcs { // all success
					mr, me := fc(val)
					if me != nil {
						return false, me
					}
					if !mr {
						res = false
						break
					}
				}
				if res {
					return true, nil
				}
			}

			return false, nil
		}, nil
	case map[string]interface{}:
		fcs := make([]eventMatchFunc, 0, len(rp))
		for key, val := range rp {
			emf, err := parsePattern(ctx, logger, append(rootPath, key), val)
			if err != nil {
				return nil, err
			}
			fcs = append(fcs, emf)
		}
		return func(c context.Context, event *rule.EventExt) (bool, error) {
			for _, fc := range fcs {
				mr, me := fc(c, event)
				if me != nil {
					return false, me
				}
				if !mr {
					return false, nil
				}
			}
			return true, nil
		}, nil
	default:
		return nil, fmt.Errorf("unexpect pattern(type=%T, val=%v)", relatedPattern, relatedPattern)
	}
}

// Pattern default support for specified value matching and array matching
func (m *matcher) Pattern(ctx context.Context, event *rule.EventExt) (bool, error) {
	return m.eventMatchFunc(ctx, event)
}
