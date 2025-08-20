package data

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/service/internal/biz"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent"
	entBus "github.com/tianping526/eventbridge/app/service/internal/data/ent/bus"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent/eventschema"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent/rule"
	"github.com/tianping526/eventbridge/app/service/internal/data/entext"
)

type busRepo struct {
	log *log.Helper
	db  *ent.Client
	rc  redis.Cmdable
}

func NewBusRepo(logger log.Logger, db *ent.Client, rc redis.Cmdable) biz.BusRepo {
	return &busRepo{
		log: log.NewHelper(log.With(
			logger,
			"module", "repo/bus",
			"caller", log.DefaultCaller,
		)),
		db: db,
		rc: rc,
	}
}

func (repo *busRepo) ListBus(
	ctx context.Context, prefix *string, limit int32, nextToken uint64,
) ([]*biz.Bus, uint64, error) {
	stmt := repo.db.Bus.Query()
	if prefix != nil {
		stmt.Where(entBus.NameHasPrefix(*prefix))
	}
	if nextToken > 0 {
		stmt.Where(entBus.IDGTE(nextToken))
	}
	convertedLimit := int(limit)
	bs, err := stmt.Order(ent.Asc("id")).Limit(convertedLimit + 1).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	next := uint64(0)
	if len(bs) > convertedLimit {
		next = bs[convertedLimit].ID
		bs = bs[:convertedLimit]
	}
	buses := make([]*biz.Bus, 0, len(bs))
	for _, b := range bs {
		var source, sourceDelay, targetExpDecay, targetBackoff biz.MQTopic
		if err = json.Unmarshal([]byte(b.SourceTopic), &source); err != nil {
			return nil, 0, fmt.Errorf("unmarshal source topic: %w", err)
		}
		if err = json.Unmarshal([]byte(b.SourceDelayTopic), &sourceDelay); err != nil {
			return nil, 0, fmt.Errorf("unmarshal source delay topic: %w", err)
		}
		if err = json.Unmarshal([]byte(b.TargetExpDecayTopic), &targetExpDecay); err != nil {
			return nil, 0, fmt.Errorf("unmarshal target exp decay topic: %w", err)
		}
		if err = json.Unmarshal([]byte(b.TargetBackoffTopic), &targetBackoff); err != nil {
			return nil, 0, fmt.Errorf("unmarshal target backoff topic: %w", err)
		}
		buses = append(buses, &biz.Bus{
			Name:           b.Name,
			Source:         source,
			SourceDelay:    sourceDelay,
			TargetExpDecay: targetExpDecay,
			TargetBackoff:  targetBackoff,
			Mode:           v1.BusWorkMode(b.Mode),
		})
	}
	return buses, next, nil
}

func (repo *busRepo) CreateBus(
	ctx context.Context, bus string, mode v1.BusWorkMode, source biz.MQTopic,
	sourceDelay biz.MQTopic, targetExpDecay biz.MQTopic, targetBackoff biz.MQTopic,
) (uint64, error) {
	var id uint64
	sourceTopic, _ := json.Marshal(source)
	sourceDelayTopic, _ := json.Marshal(sourceDelay)
	targetExpDecayTopic, _ := json.Marshal(targetExpDecay)
	targetBackoffTopic, _ := json.Marshal(targetBackoff)
	err := entext.WithTx(ctx, repo.db, func(tx *ent.Tx) error {
		b, te := repo.db.Bus.Create().
			SetName(bus).
			SetMode(uint8(mode)).
			SetSourceTopic(string(sourceTopic)).
			SetSourceDelayTopic(string(sourceDelayTopic)).
			SetTargetExpDecayTopic(string(targetExpDecayTopic)).
			SetTargetBackoffTopic(string(targetBackoffTopic)).
			Save(ctx)
		if te != nil {
			if ent.IsConstraintError(te) {
				te = v1.ErrorBusNameRepeat(
					"bus name repeat. name: %s",
					bus,
				)
			}
			return te
		}

		// update version
		te = tx.Version.UpdateOneID(entext.BusesVersionID).AddVersion(1).Exec(ctx)
		if te != nil {
			return te
		}

		id = b.ID
		return nil
	})

	return id, err
}

func (repo *busRepo) DeleteBus(ctx context.Context, busName string) error {
	var schemaIDs []uint64
	err := entext.WithTx(ctx, repo.db, func(tx *ent.Tx) error {
		// query bus and lock
		bid, te := tx.Bus.Query().
			Where(
				entBus.Name(busName),
			).
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

		// query schema
		schemaIDs, te = tx.EventSchema.Query().
			Where(
				eventschema.BusName(busName),
			).
			IDs(ctx)
		if te != nil {
			return te
		}

		// Set schema bus name to ""
		te = tx.EventSchema.Update().
			Where(
				eventschema.IDIn(schemaIDs...),
				eventschema.BusName(busName),
			).
			SetBusName("").
			AddVersion(1).
			Exec(ctx)
		if te != nil {
			return te
		}

		// delete rule
		_, te = tx.Rule.Delete().Where(rule.BusName(busName)).Exec(ctx)
		if te != nil {
			return te
		}

		// delete data bus
		te = tx.Bus.DeleteOneID(bid).Exec(ctx)
		if te != nil {
			return te
		}

		// update version
		te = tx.Version.UpdateOneID(entext.BusesVersionID).AddVersion(1).Exec(ctx)
		if te != nil {
			return te
		}

		return nil
	})
	if err != nil {
		return err
	}

	// update cache
	schemas, err := repo.db.EventSchema.Query().
		Where(
			eventschema.IDIn(schemaIDs...),
		).
		All(ctx)
	if err != nil {
		repo.log.WithContext(ctx).Errorf("query schema: %v", err)
		return nil
	}
	for _, s := range schemas {
		err = SetCacheSchema(ctx, repo.rc, s.Source, s.Type, s)
		if err != nil {
			repo.log.WithContext(ctx).Errorf("SetCacheSchema: %v, schema: %v", err, s)
		}
	}

	return nil
}
