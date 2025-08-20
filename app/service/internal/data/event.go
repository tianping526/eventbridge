package data

import (
	"context"
	_ "embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"
	"github.com/tianping526/eventbridge/app/service/internal/biz"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent"
	entBus "github.com/tianping526/eventbridge/app/service/internal/data/ent/bus"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent/eventschema"
	"github.com/tianping526/eventbridge/app/service/internal/data/entext"
)

const (
	metricPostEventBusName = "bus_name"
	metricPostEventType    = "event_type"
	metricPostEventResult  = "result"
)

var (
	maxDelSchemaKeys = 1000

	//go:embed cas.lua
	casLua string
	cas    = redis.NewScript(casLua)
)

type eventRepo struct {
	log    *log.Helper
	sf     *singleflight.Group
	db     *ent.Client
	sender Sender
	m      *Metric
	rc     redis.Cmdable
	sc     *cache.Cache
}

func NewEventRepo(
	logger log.Logger, db *ent.Client, sender Sender, m *Metric, rc redis.Cmdable, sc *cache.Cache,
) biz.EventRepo {
	return &eventRepo{
		log: log.NewHelper(log.With(
			logger,
			"module", "repo/event",
			"caller", log.DefaultCaller,
		)),
		sf:     new(singleflight.Group),
		db:     db,
		sender: sender,
		m:      m,
		rc:     rc,
		sc:     sc,
	}
}

func (repo *eventRepo) PostEvent(
	ctx context.Context, eventExt *rule.EventExt, pubTime *timestamppb.Timestamp,
) (*biz.EventInfo, error) {
	// validate eventExt
	schema, err := repo.GetLocalCacheSchema(ctx, eventExt.Event.Source, eventExt.Event.Type)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, v1.ErrorSourceTypeNotFound(
			"source(%s) + type(%s) not found.",
			eventExt.Event.Source, eventExt.Event.Type,
		)
	}
	if schema.BusName == "" {
		return nil, v1.ErrorDataBusRemoved(
			"data bus has been removed. source: %s, type: %s",
			eventExt.Event.Source, eventExt.Event.Type,
		)
	}
	err = eventExt.ValidateEventData(schema.GetValidator())
	if err != nil {
		return nil, err
	}

	// send a message in sync
	eventExt.BusName = schema.BusName
	var eventType, messageID string
	startTime := time.Now()
	if pubTime.IsValid() && time.Until(pubTime.AsTime()) >= time.Second { // delay
		eventType = "source_delay_event"
		messageID, err = repo.sender.Send(ctx, schema.BusName, eventExt, pubTime)
	} else {
		eventType = "source_event"
		messageID, err = repo.sender.Send(ctx, schema.BusName, eventExt, nil)
	}
	repo.m.PostEventDurationSec.Record(
		ctx, time.Since(startTime).Seconds(),
		metric.WithAttributes(
			attribute.String(metricPostEventBusName, schema.BusName),
			attribute.String(metricPostEventType, eventType),
		),
	)
	if err != nil {
		repo.m.PostEventCount.Add(
			ctx, 1,
			metric.WithAttributes(
				attribute.String(metricPostEventBusName, schema.BusName),
				attribute.String(metricPostEventType, eventType),
				attribute.String(metricPostEventResult, fmt.Sprintf("%T", err)),
			),
		)
		return nil, err
	}
	repo.m.PostEventCount.Add(
		ctx, 1,
		metric.WithAttributes(
			attribute.String(metricPostEventBusName, schema.BusName),
			attribute.String(metricPostEventType, eventType),
			attribute.String(metricPostEventResult, "ok"),
		),
	)

	return &biz.EventInfo{
		ID:         eventExt.Event.Id,
		MessageID:  messageID,
		MessageKey: eventExt.Key(),
	}, nil
}

func (repo *eventRepo) GetLocalCacheSchema(ctx context.Context, source string, sType string) (*biz.Schema, error) {
	lcKey := fmt.Sprintf("%s:%s", source, sType)
	val, ok := repo.sc.Get(lcKey)
	if ok {
		if val == nil {
			return nil, nil
		}
		return val.(*biz.Schema), nil
	}

	s, err := repo.FetchSchema(ctx, source, sType)
	if err != nil {
		return nil, err
	}

	var schema *biz.Schema
	if s != nil {
		schema = &biz.Schema{
			Source:  s.Source,
			Type:    s.Type,
			BusName: s.BusName,
			Spec:    s.Spec,
			Time:    timestamppb.New(s.CreateTime),
		}
		err = schema.ParseSpec()
		if err != nil {
			return nil, err
		}
	}

	repo.sc.Set(lcKey, schema, cache.DefaultExpiration)

	return schema, nil
}

func (repo *eventRepo) ListSchema(
	ctx context.Context, source *string, sType *string, busName *string, time *timestamppb.Timestamp,
) ([]*biz.Schema, error) {
	// fast path
	if source != nil && sType != nil {
		s, err := repo.FetchSchema(ctx, *source, *sType)
		if err != nil {
			return nil, err
		}
		if s == nil {
			return nil, nil
		}
		return []*biz.Schema{
			{
				Source:  s.Source,
				Type:    s.Type,
				BusName: s.BusName,
				Spec:    s.Spec,
				Time:    timestamppb.New(s.CreateTime),
			},
		}, nil
	}

	// slow path
	stmt := repo.db.EventSchema.Query()
	if source != nil {
		stmt.Where(eventschema.Source(*source))
	}
	if sType != nil {
		stmt.Where(eventschema.Type(*sType))
	}
	if busName != nil {
		stmt.Where(eventschema.BusName(*busName))
	}
	if time.IsValid() {
		stmt.Where(eventschema.CreateTimeGTE(time.AsTime()))
	}
	ss, err := stmt.All(ctx)
	if err != nil {
		return nil, err
	}

	schemas := make([]*biz.Schema, 0, len(ss))
	for _, s := range ss {
		schemas = append(schemas, &biz.Schema{
			Source:  s.Source,
			Type:    s.Type,
			BusName: s.BusName,
			Spec:    s.Spec,
			Time:    timestamppb.New(s.CreateTime),
		})
	}
	return schemas, nil
}

// FetchSchema from cache; if missing, calls the source method and then adds it to the cache.
func (repo *eventRepo) FetchSchema(ctx context.Context, source string, sType string) (*ent.EventSchema, error) {
	s, err := repo.FetchCacheSchema(ctx, source, sType)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			fetchRaw := false
			sfKey := fmt.Sprintf("sf:eb:event:schema:{%s}:%s", source, sType)
			var res interface{}
			res, err, _ = repo.sf.Do(sfKey, func() (interface{}, error) {
				fetchRaw = true
				repo.m.CacheMisses.Add(
					ctx, 1,
					metric.WithAttributes(
						attribute.String(metricLabelCacheName, "schema"),
					),
				)
				return repo.db.EventSchema.Query().
					Where(
						eventschema.Source(source),
						eventschema.Type(sType),
					).
					Only(ctx)
			})
			if err != nil {
				if !ent.IsNotFound(err) {
					return nil, err
				}
			} else {
				s = res.(*ent.EventSchema)
			}
			if !fetchRaw {
				return s, nil
			}
			err = SetCacheSchema(ctx, repo.rc, source, sType, s)
			if err != nil {
				repo.log.WithContext(ctx).Errorf("SetCacheSchema: %v, schema: %v", err, s)
			}
			return s, nil
		}
		return nil, err
	}
	repo.m.CacheHits.Add(
		ctx, 1,
		metric.WithAttributes(
			attribute.String(metricLabelCacheName, "schema"),
		),
	)
	return s, nil
}

// FetchCacheSchema from redis
func (repo *eventRepo) FetchCacheSchema(ctx context.Context, source string, sType string) (*ent.EventSchema, error) {
	key := fmt.Sprintf("eb:event:schema:{%s}:%s", source, sType)
	val, err := repo.rc.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	s := &ent.EventSchema{}
	err = json.Unmarshal(val, s)
	if err != nil {
		return nil, err
	}
	if s.ID == 0 {
		return nil, nil
	}
	return s, nil
}

// SetCacheSchema to redis cache
func SetCacheSchema(ctx context.Context, rc redis.Cmdable, source string, sType string, val *ent.EventSchema) error {
	key := fmt.Sprintf("eb:event:schema:{%s}:%s", source, sType)
	verKey := fmt.Sprintf("%s:version", key)
	bs, err := json.Marshal(val)
	if err != nil {
		return err
	}
	expire := 5 + rand.Int63n(5) //nolint:mnd
	ver := uint32(0)
	if val != nil && val.Version > 0 {
		ver = val.Version
	}
	return cas.Run(ctx, rc, []string{key, verKey}, bs, ver, expire).Err()
}

func (repo *eventRepo) CreateSchema(
	ctx context.Context, source string, sType string, busName string, spec []byte,
) error {
	var s *ent.EventSchema
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

		// save schema
		s, te = repo.db.EventSchema.Create().
			SetSource(source).
			SetType(sType).
			SetBusName(busName).
			SetSpec(string(spec)).
			SetVersion(1).
			Save(ctx)
		if te != nil {
			if ent.IsConstraintError(te) {
				te = v1.ErrorSourceTypeRepeat(
					"under each source, ensure that the type is unique.source: %s, type: %s",
					source, sType,
				)
			}
			return te
		}
		return nil
	})
	if err != nil {
		return err
	}

	// update cache
	err = SetCacheSchema(
		ctx,
		repo.rc,
		source,
		sType,
		s,
	)
	if err != nil {
		repo.log.WithContext(ctx).Errorf("SetCacheSchema: %v, schema: %v", err, s)
	}

	return nil
}

func (repo *eventRepo) UpdateSchema(
	ctx context.Context, source string, sType string, busName *string, spec []byte,
) error {
	stmt := repo.db.EventSchema.Update().
		Where(
			eventschema.Source(source),
			eventschema.Type(sType),
		)
	stmt.AddVersion(1)
	if spec != nil {
		stmt.SetSpec(string(spec))
	}
	var ar int
	var err error
	if busName != nil {
		stmt.SetBusName(*busName)
		err = entext.WithTx(ctx, repo.db, func(tx *ent.Tx) error {
			// query data bus and lock
			_, te := tx.Bus.Query().
				Where(entBus.Name(*busName)).
				ForUpdate().
				OnlyID(ctx)
			if te != nil {
				if ent.IsNotFound(te) {
					return v1.ErrorDataBusNotFound(
						"can't find the data bus. name: %s",
						*busName,
					)
				}
				return te
			}

			// update schema
			ar, te = stmt.Save(ctx)
			if te != nil {
				return te
			}
			return nil
		})
	} else {
		ar, err = stmt.Save(ctx)
	}
	if err != nil {
		return err
	}
	if ar != 1 {
		return v1.ErrorSchemaNotFound(
			"can't find the schema to update.source: %s, type: %s",
			source, sType,
		)
	}

	// update cache
	s, err := repo.db.EventSchema.Query().
		Where(
			eventschema.Source(source),
			eventschema.Type(sType),
		).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			repo.log.WithContext(ctx).Errorf("FetchDBSchema: %v", err)
			return nil
		}
	}
	err = SetCacheSchema(ctx, repo.rc, source, sType, s)
	if err != nil {
		repo.log.WithContext(ctx).Errorf("SetCacheSchema: %v, schema: %v", err, s)
	}

	return nil
}

func (repo *eventRepo) DeleteSchema(ctx context.Context, source string, sType *string) error {
	stmt := repo.db.EventSchema.Delete().Where(eventschema.Source(source))
	if sType != nil {
		stmt.Where(eventschema.Type(*sType))
	}
	_, err := stmt.Exec(ctx)
	if err != nil {
		return err
	}

	// update cache
	if sType != nil {
		key := fmt.Sprintf("eb:event:schema:{%s}:%s", source, *sType)
		verKey := fmt.Sprintf("%s:version", key)
		err = repo.rc.Del(ctx, verKey, key).Err()
		if err != nil {
			repo.log.WithContext(ctx).Errorf(
				"delete schema cache keys(%v): %v",
				[]string{verKey, key}, err,
			)
		}
		return nil
	}

	prefix := fmt.Sprintf("eb:event:schema:{%s}:*", source)
	keys, cursor, err := repo.rc.Scan(ctx, 0, prefix, 500).Result()
	if err != nil {
		repo.log.WithContext(ctx).Errorf("scan schema cache keys(%s): %v", prefix, err)
		return nil
	}
	for cursor > 0 {
		var ks []string
		ks, cursor, err = repo.rc.Scan(ctx, cursor, prefix, 500).Result()
		if err != nil {
			repo.log.WithContext(ctx).Errorf("scan schema cache keys(%s): %v", prefix, err)
			break
		}
		keys = append(keys, ks...)
		if len(keys) >= maxDelSchemaKeys {
			repo.log.WithContext(ctx).Error("delete schema: too many keys")
			break
		}
	}
	if len(keys) == 0 {
		return nil
	}
	err = repo.rc.Del(ctx, keys...).Err()
	if err != nil {
		repo.log.WithContext(ctx).Errorf(
			"delete schema cache keys(%v): %v",
			keys, err,
		)
	}

	return nil
}
