package entext

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"ariga.io/atlas/sql/migrate"
	atlas "ariga.io/atlas/sql/schema"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"github.com/sony/sonyflake"

	"github.com/tianping526/eventbridge/app/job/internal/data/ent/version"
)

const (
	BusesVersionID = 1
	RulesVersionID = 2
)

// IDHook Using sonyflake to generate IDs with hook.
func IDHook() ent.Hook {
	var sonyflakeMap sync.Map
	type IDSetter interface {
		SetID(uint64)
	}
	type IDGetter interface {
		ID() (id uint64, exists bool)
	}
	type TypeGetter interface {
		Type() string
	}

	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			ig, ok := m.(IDGetter)
			if !ok {
				return nil, fmt.Errorf("mutation %T did not implement IDGetter", m)
			}
			_, exists := ig.ID()
			if !exists {
				var is IDSetter
				is, ok = m.(IDSetter)
				if !ok {
					return nil, fmt.Errorf("mutation %T did not implement IDSetter", m)
				}
				var tg TypeGetter
				tg, ok = m.(TypeGetter)
				if !ok {
					return nil, fmt.Errorf("mutation %T did not implement TypeGetter", m)
				}
				typ := tg.Type()
				var val interface{}
				val, ok = sonyflakeMap.Load(typ)
				var idGen *sonyflake.Sonyflake
				if ok {
					idGen = val.(*sonyflake.Sonyflake)
				} else {
					st, _ := time.Parse("2006-01-02", "2024-12-10")
					idGen = sonyflake.NewSonyflake(
						sonyflake.Settings{
							StartTime: st,
						},
					)
					sonyflakeMap.Store(typ, idGen)
				}
				id, err := idGen.NextID()
				if err != nil {
					return nil, err
				}
				is.SetID(id)
			}
			return next.Mutate(ctx, m)
		})
	}
}

func SeedingHook(next schema.Applier) schema.Applier {
	return schema.ApplyFunc(func(ctx context.Context, conn dialect.ExecQuerier, plan *migrate.Plan) error {
		// Insert data seeding.
		for _, c := range plan.Changes {
			if m, ok := c.Source.(*atlas.AddTable); ok {
				if !strings.HasPrefix(c.Cmd, "CREATE TABLE") {
					continue
				}
				switch m.T.Name {
				case version.Table:
					// Insert version data seeding.
					plan.Changes = append(plan.Changes, &migrate.Change{
						Cmd: fmt.Sprintf(
							"INSERT INTO %s (%s, %s) VALUES (%d, %d), (%d, %d)",
							version.Table, version.FieldID, version.FieldVersion,
							BusesVersionID, 1,
							RulesVersionID, 0,
						),
					})
				}
			}
		}

		return next.Apply(ctx, conn, plan)
	})
}
