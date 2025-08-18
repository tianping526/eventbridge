package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
)

// MQTopic should be able to compare values and cannot contain pointers
type MQTopic struct {
	Type      v1.MQType `json:"type"`
	Endpoints string    `json:"endpoints"` // "endpoint1;endpoint2"
	Topic     string    `json:"topic"`
}

// Bus should be able to compare values and cannot contain pointers
type Bus struct {
	Name           string
	Mode           v1.BusWorkMode
	Source         MQTopic
	SourceDelay    MQTopic
	TargetExpDecay MQTopic
	TargetBackoff  MQTopic
}

type BusRepo interface {
	ListBus(ctx context.Context, prefix *string, limit int32, nextToken uint64) ([]*Bus, uint64, error)
	CreateBus(
		ctx context.Context, bus string, mode v1.BusWorkMode, source MQTopic,
		sourceDelay MQTopic, targetExpDecay MQTopic, targetBackoff MQTopic,
	) (uint64, error)
	DeleteBus(ctx context.Context, bus string) error
}

type BusUseCase struct {
	repo BusRepo

	log *log.Helper
}

func NewBusUseCase(repo BusRepo, logger log.Logger) *BusUseCase {
	return &BusUseCase{
		repo: repo,
		log: log.NewHelper(log.With(
			logger,
			"module", "usecase/bus",
			"caller", log.DefaultCaller,
		)),
	}
}

func (uc *BusUseCase) ListBus(
	ctx context.Context, prefix *string, limit int32, nextToken uint64,
) ([]*Bus, uint64, error) {
	return uc.repo.ListBus(ctx, prefix, limit, nextToken)
}

func (uc *BusUseCase) CreateBus(
	ctx context.Context, bus string, mode v1.BusWorkMode, source MQTopic,
	sourceDelay MQTopic, targetExpDecay MQTopic, targetBackoff MQTopic,
) (uint64, error) {
	return uc.repo.CreateBus(ctx, bus, mode, source, sourceDelay, targetExpDecay, targetBackoff)
}

func (uc *BusUseCase) DeleteBus(ctx context.Context, bus string) error {
	return uc.repo.DeleteBus(ctx, bus)
}
