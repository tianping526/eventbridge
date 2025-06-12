package data

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/tianping526/eventbridge/app/job/internal/data/ent"

	"github.com/sony/sonyflake"
)

const (
	metricLabelDBTableName = "name"
	metricLabelDBAddr      = "addr"
	metricLabelDBCommand   = "command"
	metricLabelDBResult    = "res"
)

// WithTx runs callbacks in a transaction.
func WithTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) (err error) {
	tx, err := client.Tx(ctx)
	if err != nil {
		return
	}
	defer func() {
		if v := recover(); v != nil {
			err = tx.Rollback()
			if err != nil {
				return
			}
			panic(v)
		}
	}()
	if err = fn(tx); err != nil {
		if rer := tx.Rollback(); rer != nil {
			return rer
		}
		return
	}
	err = tx.Commit()
	return
}

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

// EntMetricsHookOption is a metrics option.
type EntMetricsHookOption func(*entMetricsHookOptions)

// WithEntRequestsDuration with requests duration(s).
func WithEntRequestsDuration(c metric.Float64Histogram) EntMetricsHookOption {
	return func(o *entMetricsHookOptions) {
		o.requestsDuration = c
	}
}

// WithEntEndpointAddr with db Addr.
func WithEntEndpointAddr(a string) EntMetricsHookOption {
	return func(o *entMetricsHookOptions) {
		o.Addr = a
	}
}

type entMetricsHookOptions struct {
	// histogram: db_client_requests_duration_sec_bucket{"name", "Addr", "command", "res"}
	requestsDuration metric.Float64Histogram
	Addr             string
}

// EntMetricsHook Using prometheus to monitor db with hook.
func EntMetricsHook(opts ...EntMetricsHookOption) ent.Hook {
	op := entMetricsHookOptions{}
	for _, o := range opts {
		o(&op)
	}
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(
			func(ctx context.Context, m ent.Mutation) (val ent.Value, err error) {
				dbOp := ""
				switch m.Op() {
				case ent.OpCreate: // node creation.
					dbOp = "create"
				case ent.OpUpdate: // update nodes by predicate (if any).
					dbOp = "update"
				case ent.OpUpdateOne: // update one node.
					dbOp = "update_one"
				case ent.OpDelete: // delete nodes by predicate (if any).
					dbOp = "delete"
				case ent.OpDeleteOne: // delete one node.
					dbOp = "delete_one"
				}
				if op.requestsDuration != nil {
					startTime := time.Now()
					defer func() {
						res := "ok"
						if err != nil {
							res = fmt.Sprintf("%T", err)
						}
						op.requestsDuration.Record(
							ctx, time.Since(startTime).Seconds(),
							metric.WithAttributes(
								attribute.String(metricLabelDBTableName, m.Type()),
								attribute.String(metricLabelDBAddr, op.Addr),
								attribute.String(metricLabelDBCommand, dbOp),
								attribute.String(metricLabelDBResult, res),
							),
						)
					}()
				}
				val, err = next.Mutate(ctx, m)
				return
			},
		)
	}
}
