package data

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/informer"
	"github.com/tianping526/eventbridge/app/service/internal/biz"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent"
	entBus "github.com/tianping526/eventbridge/app/service/internal/data/ent/bus"
	"github.com/tianping526/eventbridge/app/service/internal/data/ent/version"
	"github.com/tianping526/eventbridge/app/service/internal/data/entext"
)

type reflector struct {
	log *log.Helper

	db           *ent.Client
	busesVersion uint64
	interval     time.Duration
	dbTimeout    time.Duration
	interBuses   map[string]*biz.Bus
	closeCh      chan struct{}

	buses sync.Map // map[busName]*biz.Bus
}

func NewReflector(
	logger log.Logger,
	db *ent.Client,
) (informer.Reflector, error) {
	return &reflector{
		log: log.NewHelper(log.With(
			logger,
			"module", "reflector",
			"caller", log.DefaultCaller,
		)),

		db:        db,
		interval:  5 * time.Second,
		dbTimeout: 5 * time.Second,
		closeCh:   make(chan struct{}),
	}, nil
}

func (r *reflector) Watch() ([]string, error) {
	err := r.fetchNextBusesVersion()
	if err != nil {
		return nil, err
	}

	buses, err := r.fetchBuses()
	if err != nil {
		return nil, err
	}

	newBuses := make(map[string]*biz.Bus, len(buses))
	for _, b := range buses {
		newBuses[b.Name] = b
	}

	updated := make([]string, 0)
	for name, b := range newBuses {
		if old, ok := r.interBuses[name]; !ok || *old != *b { // Add or update
			r.buses.Store(name, b)
			updated = append(updated, name)
		}
	}
	for name := range r.interBuses {
		if _, ok := newBuses[name]; !ok { // Delete
			r.buses.Delete(name)
			updated = append(updated, name)
		}
	}
	r.interBuses = newBuses
	return updated, nil
}

func (r *reflector) Get(key string) (interface{}, bool) {
	return r.buses.Load(key)
}

func (r *reflector) Close() error {
	close(r.closeCh)
	return nil
}

func (r *reflector) fetchBusesVersion() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.dbTimeout)
	defer cancel()
	v, err := r.db.Version.Query().Select(version.FieldVersion).Where(version.ID(entext.BusesVersionID)).Only(ctx)
	if err != nil {
		return 0, err
	}
	return v.Version, nil
}

func (r *reflector) fetchNextBusesVersion() error {
	v, err := r.fetchBusesVersion()
	if err != nil {
		return err
	}
	if v > r.busesVersion {
		r.busesVersion = v
		return nil
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-r.closeCh:
			return informer.NewReflectorClosedError()
		case <-ticker.C:
			v, err = r.fetchBusesVersion()
			if err != nil {
				return err
			}
			if v > r.busesVersion {
				r.busesVersion = v
				return nil
			}
		}
	}
}

func (r *reflector) fetchBuses() ([]*biz.Bus, error) {
	buses := make([]*biz.Bus, 0)
	next := uint64(0)
	limit := 100
	for {
		ctx, cancel := context.WithTimeout(context.Background(), r.dbTimeout)
		bs, err := r.db.Bus.Query().
			Where(
				entBus.IDGTE(next),
			).
			Select(
				entBus.FieldID,
				entBus.FieldName,
				entBus.FieldMode,
				entBus.FieldSourceTopic,
				entBus.FieldSourceDelayTopic,
			).
			Order(ent.Asc(entBus.FieldID)).
			Limit(limit + 1).
			All(ctx)
		cancel()
		if err != nil {
			return nil, err
		}
		if len(bs) > limit {
			next = bs[limit].ID
			bs = bs[:limit]
		} else {
			next = 0
		}
		for _, b := range bs {
			var source, sourceDelay biz.MQTopic
			if err = json.Unmarshal([]byte(b.SourceTopic), &source); err != nil {
				return nil, fmt.Errorf("unmarshal source topic: %w", err)
			}
			if err = json.Unmarshal([]byte(b.SourceDelayTopic), &sourceDelay); err != nil {
				return nil, fmt.Errorf("unmarshal source delay topic: %w", err)
			}
			buses = append(buses, &biz.Bus{
				Name:        b.Name,
				Mode:        v1.BusWorkMode(b.Mode),
				Source:      source,
				SourceDelay: sourceDelay,
			})
		}
		if next == 0 {
			break
		}
	}
	return buses, nil
}
