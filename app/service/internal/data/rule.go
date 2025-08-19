package data

import (
	"context"
	"encoding/json/v2"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	ir "github.com/tianping526/eventbridge/app/internal/rule"
	"github.com/tianping526/eventbridge/app/internal/rule/target"
	"github.com/tianping526/eventbridge/app/service/internal/biz"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent"
	entBus "github.com/tianping526/eventbridge/app/service/internal/data/ent/bus"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent/rule"
	"github.com/tianping526/eventbridge/app/service/internal/data/entext"
)

type ruleRepo struct {
	logger *log.Helper
	db     *ent.Client
}

func NewRuleRepo(logger log.Logger, db *ent.Client) biz.RuleRepo {
	return &ruleRepo{
		logger: log.NewHelper(log.With(
			logger,
			"module", "repo/rule",
			"caller", log.DefaultCaller,
		)),
		db: db,
	}
}

func (repo *ruleRepo) ListRule(
	ctx context.Context, bus string, prefix *string, status v1.RuleStatus, limit int32, nextToken uint64,
) ([]*ir.Rule, uint64, error) {
	convertedLimit := int(limit)
	stmt := repo.db.Rule.Query().Where(rule.BusName(bus))
	if prefix != nil {
		stmt.Where(rule.NameHasPrefix(*prefix))
	}
	if status != v1.RuleStatus_RULE_STATUS_UNSPECIFIED {
		stmt.Where(rule.Status(uint8(status)))
	}
	if nextToken > 0 {
		stmt.Where(rule.IDGTE(nextToken))
	}
	rs, err := stmt.Order(ent.Asc("id")).Limit(convertedLimit + 1).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	next := uint64(0)
	if len(rs) > convertedLimit {
		next = rs[convertedLimit].ID
		rs = rs[:convertedLimit]
	}
	rules := make([]*ir.Rule, 0, len(rs))
	for _, r := range rs {
		var targets []*ir.Target
		err = json.Unmarshal([]byte(r.Targets), &targets)
		if err != nil {
			return nil, 0, err
		}
		rules = append(rules, &ir.Rule{
			Name:    r.Name,
			BusName: r.BusName,
			Status:  v1.RuleStatus(r.Status),
			Pattern: r.Pattern,
			Targets: targets,
		})
	}
	return rules, next, nil
}

func (repo *ruleRepo) CreateRule(
	ctx context.Context, busName string, name string, status v1.RuleStatus, pattern *string, targets []*ir.Target,
) (uint64, error) {
	uniqueIDTargets := make([]*ir.Target, 0, len(targets))
	idTargetMap := make(map[uint64]*ir.Target, len(targets))
	for _, t := range targets {
		idTargetMap[t.ID] = t
	}
	for _, t := range idTargetMap {
		uniqueIDTargets = append(uniqueIDTargets, t)
	}
	targets = uniqueIDTargets

	var r *ent.Rule
	err := entext.WithTx(ctx, repo.db, func(tx *ent.Tx) error {
		// query data bus and lock
		_, te := tx.Bus.Query().
			Where(entBus.Name(busName)).
			ForUpdate().
			OnlyID(ctx)
		if te != nil {
			if ent.IsNotFound(te) {
				return v1.ErrorDataBusNotFound(
					"can't find the data bus. name: %s",
					busName,
				)
			}
			return te
		}

		// save rule
		bts, te := json.Marshal(targets)
		if te != nil {
			return te
		}
		r, te = tx.Rule.Create().
			SetBusName(busName).
			SetName(name).
			SetStatus(uint8(status)).
			SetPattern(*pattern).
			SetTargets(string(bts)).
			Save(ctx)
		if te != nil {
			if ent.IsConstraintError(te) {
				return v1.ErrorRuleNameRepeat(
					"rule name repeat. bus name: %s, rule name: %s",
					busName, name,
				)
			}
			return te
		}

		// update version
		te = tx.Version.UpdateOneID(entext.RulesVersionID).AddVersion(1).Exec(ctx)
		if te != nil {
			return te
		}

		return nil
	})
	if err != nil {
		return 0, err
	}
	return r.ID, nil
}

func (repo *ruleRepo) UpdateRule(
	ctx context.Context, bus string, name string, status v1.RuleStatus, pattern *string,
) error {
	stmt := repo.db.Rule.Update().
		Where(
			rule.BusName(bus),
			rule.Name(name),
		)
	if status != v1.RuleStatus_RULE_STATUS_UNSPECIFIED {
		stmt.SetStatus(uint8(status))
	}
	if pattern != nil {
		stmt.SetPattern(*pattern)
	}

	return entext.WithTx(ctx, repo.db, func(tx *ent.Tx) error {
		ar, err := stmt.Save(ctx)
		if err != nil {
			return err
		}
		if ar != 1 {
			return v1.ErrorRuleNotFound(
				"rule not found. bus name: %s, rule name: %s",
				bus, name,
			)
		}

		// update version
		err = tx.Version.UpdateOneID(entext.RulesVersionID).AddVersion(1).Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (repo *ruleRepo) DeleteRule(ctx context.Context, bus string, name string) error {
	return entext.WithTx(ctx, repo.db, func(tx *ent.Tx) error {
		dr, err := repo.db.Rule.Delete().
			Where(
				rule.BusName(bus),
				rule.Name(name),
			).
			Exec(ctx)
		if err != nil {
			return err
		}
		if dr != 1 {
			return v1.ErrorRuleNotFound(
				"rule not found. bus name: %s, rule name: %s",
				bus, name,
			)
		}

		// update version
		err = tx.Version.UpdateOneID(entext.RulesVersionID).AddVersion(1).Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (repo *ruleRepo) CreateTargets(ctx context.Context, bus string, ruleName string, targets []*ir.Target) error {
	err := entext.WithTx(ctx, repo.db, func(tx *ent.Tx) error {
		r, te := tx.Rule.Query().
			Where(
				rule.BusName(bus),
				rule.Name(ruleName),
			).
			Select(
				rule.FieldID,
				rule.FieldTargets,
			).
			ForUpdate().
			Only(ctx)
		if te != nil {
			if ent.IsNotFound(te) {
				return v1.ErrorRuleNotFound(
					"rule not found. bus name: %s, rule name: %s",
					bus, ruleName,
				)
			}
			return te
		}
		var ts []*ir.Target
		te = json.Unmarshal([]byte(r.Targets), &ts)
		if te != nil {
			return te
		}
		ts = append(ts, targets...)
		tm := make(map[uint64]*ir.Target, len(ts))
		for _, t := range ts {
			tm[t.ID] = t
		}
		nts := make([]*ir.Target, 0, len(tm))
		for _, t := range tm {
			nts = append(nts, t)
		}
		bts, te := json.Marshal(nts)
		if te != nil {
			return te
		}
		_, te = tx.Rule.UpdateOneID(r.ID).SetTargets(string(bts)).Save(ctx)
		if te != nil {
			return te
		}

		// update version
		te = tx.Version.UpdateOneID(entext.RulesVersionID).AddVersion(1).Exec(ctx)
		if te != nil {
			return te
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (repo *ruleRepo) DeleteTargets(ctx context.Context, bus string, ruleName string, targetIDs []uint64) error {
	tdm := make(map[uint64]bool)
	for _, tid := range targetIDs {
		tdm[tid] = true
	}
	err := entext.WithTx(ctx, repo.db, func(tx *ent.Tx) error {
		r, te := tx.Rule.Query().
			Where(
				rule.BusName(bus),
				rule.Name(ruleName),
			).
			Select(
				rule.FieldID,
				rule.FieldTargets,
			).
			ForUpdate().
			Only(ctx)
		if te != nil {
			if ent.IsNotFound(te) {
				return v1.ErrorRuleNotFound(
					"rule not found. bus name: %s, rule name: %s",
					bus, ruleName,
				)
			}
			return te
		}
		var ts []*ir.Target
		te = json.Unmarshal([]byte(r.Targets), &ts)
		if te != nil {
			return te
		}
		nts := make([]*ir.Target, 0, len(ts))
		for _, t := range ts {
			if !tdm[t.ID] {
				nts = append(nts, t)
			}
		}
		bts, te := json.Marshal(nts)
		if te != nil {
			return te
		}
		_, te = tx.Rule.UpdateOneID(r.ID).SetTargets(string(bts)).Save(ctx)
		if te != nil {
			return te
		}

		// update version
		te = tx.Version.UpdateOneID(entext.RulesVersionID).AddVersion(1).Exec(ctx)
		if te != nil {
			return te
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (repo *ruleRepo) ListDispatcherSchema(_ context.Context, types []string) ([]*biz.DispatcherSchema, error) {
	schemaMap := target.ListAllDispatcherParamsSchema()

	schemas := make([]*biz.DispatcherSchema, 0, len(schemaMap))
	if len(types) != 0 {
		for _, typ := range types {
			schema, ok := schemaMap[typ]
			if ok {
				schemas = append(schemas, &biz.DispatcherSchema{
					Type:         typ,
					ParamsSchema: schema,
				})
			}
		}
	} else {
		for typ, schema := range schemaMap {
			schemas = append(schemas, &biz.DispatcherSchema{
				Type:         typ,
				ParamsSchema: schema,
			})
		}
	}

	return schemas, nil
}
