package service

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/service/internal/biz"
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	NewEventBridgeService,
)

type EventBridgeService struct {
	v1.UnimplementedEventBridgeServiceServer

	ec *biz.EventUseCase
	bc *biz.BusUseCase
	rc *biz.RuleUseCase

	log *log.Helper
}

func NewEventBridgeService(
	ec *biz.EventUseCase,
	bc *biz.BusUseCase,
	rc *biz.RuleUseCase,
	logger log.Logger,
) *EventBridgeService {
	return &EventBridgeService{
		log: log.NewHelper(log.With(
			logger,
			"module", "service",
			"caller", log.DefaultCaller,
		)),
		ec: ec,
		bc: bc,
		rc: rc,
	}
}
