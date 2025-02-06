package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

type EventRepo interface {
	HandleEvent(context.Context, *rule.EventExt) error
}

type EventUseCase struct {
	repo EventRepo

	log *log.Helper
}

func NewEventUseCase(repo EventRepo, logger log.Logger) *EventUseCase {
	return &EventUseCase{
		repo: repo,
		log: log.NewHelper(log.With(
			logger,
			"module", "usecase/event",
			"caller", log.DefaultCaller,
		)),
	}
}

func (uc *EventUseCase) HandleEvent(ctx context.Context, evt *rule.EventExt) (interface{}, error) {
	err := uc.repo.HandleEvent(ctx, evt)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
