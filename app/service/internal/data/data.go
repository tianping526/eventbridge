package data

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/service/internal/conf"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent"
	entBus "github.com/tianping526/eventbridge/app/service/internal/data/ent/bus"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent/hook"
	im "github.com/tianping526/eventbridge/app/service/internal/data/ent/migrate"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent/version"

	"ariga.io/atlas/sql/migrate"
	atlas "ariga.io/atlas/sql/schema"
	"entgo.io/ent/dialect"
	entSql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	kz "github.com/go-kratos/kratos/contrib/log/zap/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"
	"github.com/natefinch/lumberjack"
	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/signalfx/splunk-otel-go/instrumentation/database/sql/splunksql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSDK "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// init db driver
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4/stdlib"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewLogger,
	NewData,
	NewConfig,
	NewConfigBootstrap,
	NewRegistrar,
	NewMetric,
	NewTracerProvider,
	NewEntClient,
	NewRedisCmd,
	NewSchemaLocalCache,
	NewReflector,
	NewSender,
	NewBusRepo,
	NewEventRepo,
	NewRuleRepo,
)

// Data .
type Data struct {
	m *Metric

	db     *ent.Client
	rc     redis.Cmdable
	sender Sender
	sc     *cache.Cache
}

type Metric struct {
	CacheHits            metric.Int64Counter
	CacheMisses          metric.Int64Counter
	CacheDurationSec     metric.Float64Histogram
	CodeTotal            metric.Int64Counter
	DurationSec          metric.Float64Histogram
	DbDurationSec        metric.Float64Histogram
	PostEventCount       metric.Int64Counter
	PostEventDurationSec metric.Float64Histogram
}

// NewData .
func NewData(
	db *ent.Client,
	rcd redis.Cmdable,
	sender Sender,
	sc *cache.Cache,
	m *Metric,
) (*Data, error) {
	rc, _ := rcd.(*redis.Client)
	return &Data{
		db:     db,
		rc:     rc,
		sender: sender,
		sc:     sc,
		m:      m,
	}, nil
}

func NewMetric(ai *conf.AppInfo) (*Metric, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	meter := provider.Meter(ai.Name)

	cacheHits, err := meter.Int64Counter(
		"cache_redis_hits_total",
		metric.WithUnit("{call}"),
		metric.WithDescription("Redis hits total."),
	)
	if err != nil {
		return nil, err
	}
	cacheMisses, err := meter.Int64Counter(
		"cache_redis_misses_total",
		metric.WithUnit("{call}"),
		metric.WithDescription("Redis misses total."),
	)
	if err != nil {
		return nil, err
	}
	cacheDurationSec, err := meter.Float64Histogram(
		"cache_client_requests_duration_sec",
		metric.WithUnit("s"),
		metric.WithDescription("Cache requests duration(sec)."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1),
	)
	if err != nil {
		return nil, err
	}
	codeTotal, err := metrics.DefaultRequestsCounter(meter, metrics.DefaultServerRequestsCounterName)
	if err != nil {
		return nil, err
	}
	durationSec, err := metrics.DefaultSecondsHistogram(meter, metrics.DefaultServerSecondsHistogramName)
	if err != nil {
		return nil, err
	}
	dbDurationSec, err := meter.Float64Histogram(
		"db_client_requests_duration_sec",
		metric.WithUnit("s"),
		metric.WithDescription("DB requests duration(sec)."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1),
	)
	if err != nil {
		return nil, err
	}
	postEventCount, err := meter.Int64Counter(
		"job_event_post_count",
		metric.WithUnit("{call}"),
		metric.WithDescription("Number of events that have been posted."),
	)
	if err != nil {
		return nil, err
	}
	postEventDurationSec, err := meter.Float64Histogram(
		"job_event_post_duration_sec",
		metric.WithUnit("s"),
		metric.WithDescription("Post event duration(sec)."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1),
	)
	if err != nil {
		return nil, err
	}
	return &Metric{
		CacheHits:            cacheHits,
		CacheMisses:          cacheMisses,
		CacheDurationSec:     cacheDurationSec,
		CodeTotal:            codeTotal,
		DurationSec:          durationSec,
		DbDurationSec:        dbDurationSec,
		PostEventCount:       postEventCount,
		PostEventDurationSec: postEventDurationSec,
	}, nil
}

func NewLogger(ai *conf.AppInfo, cfg *conf.Bootstrap) (log.Logger, func(), error) {
	level := conf.Log_INFO
	encoding := conf.Log_JSON
	sampling := &zap.SamplingConfig{
		Initial:    100,
		Thereafter: 100,
	}
	outputPaths := []*conf.Log_Output{{Path: "stderr"}}

	if cfg.Log != nil {
		level = cfg.Log.Level
		encoding = cfg.Log.Encoding
		if cfg.Log.Sampling != nil {
			sampling = &zap.SamplingConfig{
				Initial:    int(cfg.Log.Sampling.Initial),
				Thereafter: int(cfg.Log.Sampling.Thereafter),
			}
		}
		if len(cfg.Log.OutputPaths) > 0 {
			outputPaths = cfg.Log.OutputPaths
		}
	}

	// encoder
	var encoder zapcore.Encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = zapcore.OmitKey
	encoderConfig.NameKey = zapcore.OmitKey
	encoderConfig.CallerKey = zapcore.OmitKey
	encoderConfig.StacktraceKey = zapcore.OmitKey
	if encoding == conf.Log_CONSOLE {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// sinks
	var sink zapcore.WriteSyncer
	closes := make([]func(), 0, len(outputPaths))
	paths := make([]string, 0, len(outputPaths))
	syncer := make([]zapcore.WriteSyncer, 0, len(outputPaths))
	for _, out := range outputPaths {
		if out.Rotate == nil {
			paths = append(paths, out.Path)
			continue
		}

		lg := &lumberjack.Logger{
			Filename:   out.Path,
			MaxSize:    int(out.Rotate.MaxSize),
			MaxAge:     int(out.Rotate.MaxAge),
			MaxBackups: int(out.Rotate.MaxBackups),
			Compress:   out.Rotate.Compress,
		}

		syncer = append(syncer, zapcore.AddSync(lg))
		closes = append(closes, func() {
			err := lg.Close()
			if err != nil {
				fmt.Printf("close lumberjack logger(%s) error(%s))", out.Path, err)
			}
		})
	}
	if len(paths) > 0 {
		writer, mc, err := zap.Open(paths...)
		if err != nil {
			for _, c := range closes {
				c()
			}
			return nil, nil, err
		}
		closes = append(closes, mc)
		syncer = append(syncer, writer)
	}
	sink = zap.CombineWriteSyncers(syncer...)

	zl := zap.New(
		zapcore.NewCore(encoder, sink, zap.NewAtomicLevelAt(zapcore.Level(level-1))),
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				core,
				time.Second,
				sampling.Initial,
				sampling.Thereafter,
			)
		}),
	)

	logger := log.With(
		kz.NewLogger(zl),
		"ts", log.DefaultTimestamp,
		"service.id", ai.Id,
		"service.name", ai.Name,
		"service.version", ai.Version,
		"trace_id", tracing.TraceID(),
		"span_id", tracing.SpanID(),
	)
	return logger, func() {
		err := zl.Sync()
		if err != nil {
			fmt.Printf("sync logger error(%s)", err)
		}
		for _, c := range closes {
			c()
		}
	}, nil
}

func NewConfig(ai *conf.AppInfo) (config.Config, func(), error) {
	fc := config.New(
		config.WithSource(
			file.NewSource(ai.FlagConf),
		),
	)
	if err := fc.Load(); err != nil {
		return nil, nil, err
	}
	return fc, func() {
		err := fc.Close()
		if err != nil {
			fmt.Printf("close file config(%s) error(%s))", ai.FlagConf, err)
		}
	}, nil
}

func NewConfigBootstrap(c config.Config) *conf.Bootstrap {
	var bc conf.Bootstrap
	if err := c.Value("bootstrap").Scan(&bc); err != nil {
		panic(err)
	}

	return &bc
}

func NewRegistrar(_ *conf.Bootstrap) (registry.Registrar, error) {
	return nil, nil
}

func NewTracerProvider(ai *conf.AppInfo, conf *conf.Bootstrap) (trace.TracerProvider, func(), error) {
	if conf.Trace == nil {
		return noop.NewTracerProvider(), func() {}, nil
	}
	exp, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpointURL(conf.Trace.EndpointUrl),
	)
	if err != nil {
		return nil, nil, err
	}
	tp := traceSDK.NewTracerProvider(
		traceSDK.WithBatcher(exp),
		traceSDK.WithResource(resource.NewSchemaless(
			semConv.ServiceNameKey.String(ai.Name),
		)),
		traceSDK.WithSampler(traceSDK.ParentBased(traceSDK.TraceIDRatioBased(1.0))),
	)
	otel.SetTracerProvider(tp)
	return tp, func() {
		err2 := tp.Shutdown(context.Background())
		if err2 != nil {
			fmt.Printf("close trace provider(%s) error(%s))", conf.Trace.EndpointUrl, err2)
		}
	}, nil
}

type queryMetricDriver struct {
	tableNameRe *regexp.Regexp
	durationSec metric.Float64Histogram
	addr        string
	*entSql.Driver
}

func (qmd queryMetricDriver) Query(ctx context.Context, query string, args, v interface{}) (err error) {
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

func SeedingHook(next schema.Applier) schema.Applier {
	return schema.ApplyFunc(func(ctx context.Context, conn dialect.ExecQuerier, plan *migrate.Plan) error {
		// Insert data seeding.
		for _, c := range plan.Changes {
			if m, ok := c.Source.(*atlas.AddTable); ok {
				if !strings.HasPrefix(c.Cmd, "CREATE TABLE") {
					continue
				}
				switch m.T.Name {
				case entBus.Table:
					// Insert bus data seeding.
					plan.Changes = append(plan.Changes, &migrate.Change{
						Cmd: fmt.Sprintf(
							"INSERT INTO %s (%s, %s, %s, %s, %s, %s, %s, %s, %s) VALUES "+
								"(100000000, 'Default', %d, 'EBInterBusDefault', 'EBInterDelayBusDefault', "+
								"'EBInterTargetExpDecayBusDefault', 'EBInterTargetBackoffBusDefault', NOW(), NOW())",
							entBus.Table, entBus.FieldID, entBus.FieldName, entBus.FieldMode,
							entBus.FieldSourceTopic, entBus.FieldSourceDelayTopic, entBus.FieldTargetExpDecayTopic,
							entBus.FieldTargetBackoffTopic, entBus.FieldCreateTime, entBus.FieldUpdateTime,
							v1.BusWorkMode_BUS_WORK_MODE_CONCURRENTLY,
						),
					})
				case version.Table:
					// Insert version data seeding.
					plan.Changes = append(plan.Changes, &migrate.Change{
						Cmd: fmt.Sprintf(
							"INSERT INTO %s (%s, %s) VALUES (%d, %d), (%d, %d)",
							version.Table, version.FieldID, version.FieldVersion,
							busesVersionID, 1,
							rulesVersionID, 0,
						),
					})
				}
			}
		}

		return next.Apply(ctx, conn, plan)
	})
}

func NewEntClient(conf *conf.Bootstrap, m *Metric) (*ent.Client, func(), error) {
	var (
		db  *sql.DB
		err error
	)
	switch conf.Data.Database.Driver {
	case dialect.MySQL:
		db, err = splunksql.Open(conf.Data.Database.Driver, conf.Data.Database.Source)
	case dialect.Postgres:
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
	drvWrap := &queryMetricDriver{
		tableNameRe: regexp.MustCompile(`FROM\s+"(\w+)"`),
		durationSec: m.DbDurationSec,
		addr:        metricAddr,
		Driver:      drv,
	}
	ec := ent.NewClient(ent.Driver(drvWrap))

	// Run the auto migration tool.
	if err = ec.Schema.Create(
		context.Background(),
		im.WithForeignKeys(false),
		schema.WithApplyHook(SeedingHook),
	); err != nil {
		return nil, nil, fmt.Errorf("failed creating schema resources: %v", err)
	}

	// Add a global hook that runs on all types and all operations.
	ec.Use(
		EntMetricsHook(
			WithEntEndpointAddr(metricAddr),
			WithEntRequestsDuration(m.DbDurationSec),
		),
	)
	ec.Use(
		hook.On(
			IDHook(),
			ent.OpCreate,
		), // Automatically set the ID field using sonyflake if no id is set when creating.
	)
	return ec, func() {
		err = ec.Close()
		if err != nil {
			fmt.Printf("failed closing ent client: %v", err)
		}
	}, nil
}

func NewRedisCmd(conf *conf.Bootstrap, m *Metric) (redis.Cmdable, func(), error) {
	client := redis.NewClient(&redis.Options{
		Addr:         conf.Data.Redis.Addr,
		Password:     conf.Data.Redis.Password,
		DB:           int(conf.Data.Redis.DbIndex),
		DialTimeout:  conf.Data.Redis.DialTimeout.AsDuration(),
		ReadTimeout:  conf.Data.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: conf.Data.Redis.WriteTimeout.AsDuration(),
	})
	// Enable tracing instrumentation.
	err := redisotel.InstrumentTracing(client)
	if err != nil {
		return nil, nil, err
	}
	client.AddHook(NewRedisMetricHook(
		WithRedisEndpointAddr(conf.Data.Redis.Addr),
		WithRedisRequestsDuration(m.CacheDurationSec),
	))
	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Second*2)
	defer cancelFunc()
	err = client.Ping(timeout).Err()
	if err != nil {
		return nil, nil, fmt.Errorf("redis connect error: %v", err)
	}
	return client, func() {
		err = client.Close()
		if err != nil {
			fmt.Printf("failed closing redis client: %v", err)
		}
	}, nil
}

func NewSchemaLocalCache() *cache.Cache {
	c := cache.New(2*time.Second, 10*time.Minute)
	return c
}
