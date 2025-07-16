package entext

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/tianping526/eventbridge/app/service/internal/data/ent"
)

const (
	metricLabelDBTableName = "name"
	metricLabelDBAddr      = "addr"
	metricLabelDBCommand   = "command"
	metricLabelDBResult    = "res"
)

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

type queryMetricDriver struct {
	tableNameRe *regexp.Regexp
	durationSec metric.Float64Histogram
	addr        string
	*sql.Driver
}

func NewQueryMetricDriver(
	driver *sql.Driver,
	durationSec metric.Float64Histogram,
	addr string,
) dialect.Driver {
	return &queryMetricDriver{
		tableNameRe: regexp.MustCompile(`FROM\s+"(\w+)"`),
		durationSec: durationSec,
		addr:        addr,
		Driver:      driver,
	}
}

func (qmd *queryMetricDriver) Query(ctx context.Context, query string, args, v interface{}) (err error) {
	res := qmd.tableNameRe.FindStringSubmatch(query)
	tableName := ""
	if len(res) > 1 {
		tableName = res[1]
	}
	if qmd.durationSec != nil {
		startTime := time.Now()
		defer func() {
			result := "ok"
			if err != nil {
				result = fmt.Sprintf("%T", err)
			}
			qmd.durationSec.Record(
				ctx, time.Since(startTime).Seconds(),
				metric.WithAttributes(
					attribute.String(metricLabelDBTableName, tableName),
					attribute.String(metricLabelDBAddr, qmd.addr),
					attribute.String(metricLabelDBCommand, "query"),
					attribute.String(metricLabelDBResult, result),
				),
			)
		}()
	}

	err = qmd.Driver.Query(ctx, query, args, v)
	return err
}
