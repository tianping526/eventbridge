package data

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"entgo.io/ent/dialect"
	entSql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/signalfx/splunk-otel-go/instrumentation/database/sql/splunksql"

	"github.com/tianping526/eventbridge/app/job/internal/conf"
	"github.com/tianping526/eventbridge/app/job/internal/data/ent"
	"github.com/tianping526/eventbridge/app/job/internal/data/ent/hook"
	"github.com/tianping526/eventbridge/app/job/internal/data/ent/migrate"
	"github.com/tianping526/eventbridge/app/job/internal/data/entext"

	// init db driver
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4/stdlib"
)

func NewEntClient(
	conf *conf.Bootstrap, m *Metric, l log.Logger,
) (*ent.Client, func(), error) {
	logger := log.NewHelper(log.With(l, "module", "data/ent", "caller", log.DefaultCaller))
	var (
		db  *sql.DB
		err error
	)
	switch conf.Data.Database.Driver {
	case dialect.MySQL:
		splunksql.Register(dialect.MySQL, splunksql.InstrumentationConfig{
			DBSystem: splunksql.DBSystemMySQL,
		})
		db, err = splunksql.Open(conf.Data.Database.Driver, conf.Data.Database.Source)
	case dialect.Postgres:
		splunksql.Register("pgx", splunksql.InstrumentationConfig{
			DBSystem: splunksql.DBSystemPostgreSQL,
		})
		db, err = splunksql.Open("pgx", conf.Data.Database.Source)
	default:
		return nil, nil, fmt.Errorf("unsupported db driver: %s", conf.Data.Database.Driver)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed opening connection to db: %v", err)
	}
	db.SetMaxOpenConns(int(conf.Data.Database.MaxOpen))
	db.SetMaxIdleConns(int(conf.Data.Database.MaxIdle))
	db.SetConnMaxLifetime(conf.Data.Database.ConnMaxLifeTime.AsDuration())
	db.SetConnMaxIdleTime(conf.Data.Database.ConnMaxIdleTime.AsDuration())

	sourceURL, err := url.Parse(conf.Data.Database.Source)
	if err != nil {
		return nil, nil, fmt.Errorf("failed parse source of db: %v", err)
	}
	metricAddr := fmt.Sprintf("%s%s", sourceURL.Host, sourceURL.Path)

	drv := entSql.OpenDB(conf.Data.Database.Driver, db)
	drvWrap := entext.NewQueryMetricDriver(drv, m.DbDurationSec, metricAddr)
	ec := ent.NewClient(ent.Driver(drvWrap))

	// Run the auto migration tool.
	if err = ec.Schema.Create(
		context.Background(),
		migrate.WithForeignKeys(false),
		schema.WithApplyHook(entext.SeedingHook),
	); err != nil {
		return nil, nil, fmt.Errorf("failed creating schema resources: %v", err)
	}

	// Add a global hook that runs on all types and all operations.
	ec.Use(
		entext.EntMetricsHook(
			entext.WithEntEndpointAddr(metricAddr),
			entext.WithEntRequestsDuration(m.DbDurationSec),
		),
	)
	ec.Use(
		hook.On(
			entext.IDHook(),
			ent.OpCreate,
		), // Automatically set the ID field using sonyflake if no id is set when creating.
	)
	return ec, func() {
		err = ec.Close()
		if err != nil {
			logger.Errorf("failed closing ent client: %v", err)
		}
	}, nil
}
