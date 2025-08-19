package data

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/informer"
	"github.com/tianping526/eventbridge/app/internal/rule"
	"github.com/tianping526/eventbridge/app/internal/rule/pattern"
	"github.com/tianping526/eventbridge/app/internal/rule/target"
	"github.com/tianping526/eventbridge/app/internal/rule/transform"
	"github.com/tianping526/eventbridge/app/job/internal/conf"
	"github.com/tianping526/eventbridge/app/job/internal/data/ent"
	entRule "github.com/tianping526/eventbridge/app/job/internal/data/ent/rule"
	"github.com/tianping526/eventbridge/app/job/internal/data/ent/version"
	"github.com/tianping526/eventbridge/app/job/internal/data/entext"
)

type ruleReflector struct {
	log *log.Helper

	db           *ent.Client
	rulesVersion uint64
	interval     time.Duration
	dbTimeout    time.Duration
	interRules   map[string]*rule.Rule
	closeCh      chan struct{}

	rules sync.Map // map[busName:ruleName]*rule.Rule
}

func NewRuleReflector(
	logger log.Logger,
	db *ent.Client,
) (informer.Reflector, error) {
	return &ruleReflector{
		log:       log.NewHelper(log.With(logger, "module", "rule/reflector")),
		db:        db,
		interval:  5 * time.Second,
		dbTimeout: 5 * time.Second,
		closeCh:   make(chan struct{}),
	}, nil
}

func (rr *ruleReflector) Watch() ([]string, error) {
	err := rr.fetchNextRulesVersion()
	if err != nil {
		return nil, err
	}

	rules, err := rr.fetchEnableRules()
	if err != nil {
		return nil, err
	}

	newRules := make(map[string]*rule.Rule, len(rules))
	for _, r := range rules {
		newRules[fmt.Sprintf("%s:%s", r.BusName, r.Name)] = r
	}

	updated := make([]string, 0)
	for key, r := range newRules {
		if old, ok := rr.interRules[key]; !ok || !reflect.DeepEqual(*old, *r) {
			rr.rules.Store(key, r)
			updated = append(updated, key)
		}
	}
	for key := range rr.interRules {
		if _, ok := newRules[key]; !ok {
			rr.rules.Delete(key)
			updated = append(updated, key)
		}
	}

	rr.interRules = newRules
	return updated, nil
}

func (rr *ruleReflector) Get(key string) (interface{}, bool) {
	return rr.rules.Load(key)
}

func (rr *ruleReflector) Close() error {
	close(rr.closeCh)
	return nil
}

func (rr *ruleReflector) fetchRulesVersion() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rr.dbTimeout)
	defer cancel()
	v, err := rr.db.Version.Query().Select(version.FieldVersion).Where(version.ID(entext.RulesVersionID)).Only(ctx)
	if err != nil {
		return 0, err
	}
	return v.Version, nil
}

func (rr *ruleReflector) fetchNextRulesVersion() error {
	v, err := rr.fetchRulesVersion()
	if err != nil {
		return err
	}
	if v > rr.rulesVersion {
		rr.rulesVersion = v
		return nil
	}

	t := time.NewTicker(rr.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			v, err = rr.fetchRulesVersion()
			if err != nil {
				return err
			}
			if v > rr.rulesVersion {
				rr.rulesVersion = v
				return nil
			}
		case <-rr.closeCh:
			return informer.NewReflectorClosedError()
		}
	}
}

func (rr *ruleReflector) fetchEnableRules() ([]*rule.Rule, error) {
	rules := make([]*rule.Rule, 0)
	next := uint64(0)
	limit := 100
	status := uint8(v1.RuleStatus_RULE_STATUS_ENABLE)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), rr.dbTimeout)
		rs, err := rr.db.Rule.Query().
			Where(
				entRule.Status(status),
				entRule.IDGTE(next),
			).
			Order(ent.Asc(entRule.FieldID)).
			Limit(limit + 1).
			All(ctx)
		cancel()
		if err != nil {
			return nil, err
		}
		if len(rs) > limit {
			next = rs[limit].ID
			rs = rs[:limit]
		} else {
			next = 0
		}
		for _, r := range rs {
			var targets []*rule.Target
			err = json.Unmarshal([]byte(r.Targets), &targets)
			if err != nil {
				return nil, err
			}
			rules = append(rules, &rule.Rule{
				Name:    r.Name,
				BusName: r.BusName,
				Status:  v1.RuleStatus(r.Status),
				Pattern: r.Pattern,
				Targets: targets,
			})
		}
		if next == 0 {
			break
		}
	}
	return rules, nil
}

func NewRules(logger log.Logger, conf *conf.Bootstrap, db *ent.Client, m *Metric) (rule.Rules, func(), error) {
	reflector, err := NewRuleReflector(logger, db)
	if err != nil {
		return nil, nil, err
	}
	return rule.NewRules(
		logger,
		reflector,
		rule.NewNewExecutorFunc(
			pattern.NewMatcher,
			transform.NewTransformer,
			target.NewDispatcher,
		),
		rule.WithExecuteDuration(m.RuleExecSec),
		rule.WithExecuteTotal(m.RuleExecTotal),
		rule.WithTransformParallelism(int(conf.Server.Event.TransformParallelism)),
	)
}
