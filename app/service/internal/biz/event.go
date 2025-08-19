package biz

import (
	"context"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventRepo interface {
	PostEvent(ctx context.Context, eventExt *rule.EventExt, pubTime *timestamppb.Timestamp) (*EventInfo, error)
	ListSchema(
		ctx context.Context, source *string, sType *string, busName *string, time *timestamppb.Timestamp,
	) ([]*Schema, error)
	CreateSchema(ctx context.Context, source string, sType string, busName string, spec []byte) error
	UpdateSchema(ctx context.Context, source string, sType string, busName *string, spec []byte) error
	DeleteSchema(ctx context.Context, source string, sType *string) error
}

type EventInfo struct {
	ID         uint64
	MessageID  string
	MessageKey string
	TraceID    string
}

type Schema struct {
	Source  string
	Type    string
	BusName string
	Spec    string
	Time    *timestamppb.Timestamp

	validator *gojsonschema.Schema
}

func (s *Schema) ParseSpec() error {
	validator, err := gojsonschema.NewSchema(gojsonschema.NewStringLoader(s.Spec))
	if err != nil {
		return err
	}
	s.validator = validator
	return nil
}

func (s *Schema) GetValidator() *gojsonschema.Schema {
	return s.validator
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

func (uc *EventUseCase) PostEvent(
	ctx context.Context, eventExt *rule.EventExt, pubTime *timestamppb.Timestamp,
) (*EventInfo, error) {
	return uc.repo.PostEvent(ctx, eventExt, pubTime)
}

func (uc *EventUseCase) ListSchema(
	ctx context.Context, source *string, sType *string, busName *string, time *timestamppb.Timestamp,
) ([]*Schema, error) {
	return uc.repo.ListSchema(ctx, source, sType, busName, time)
}

func (uc *EventUseCase) CreateSchema(
	ctx context.Context, source string, sType string, busName string, spec []byte,
) error {
	err := EventSchemaSyntaxCheck(spec)
	if err != nil {
		return v1.ErrorSchemaSyntaxError(
			"syntax error: %s", err,
		)
	}
	return uc.repo.CreateSchema(ctx, source, sType, busName, spec)
}

func (uc *EventUseCase) UpdateSchema(
	ctx context.Context, source string, sType string, busName *string, spec []byte,
) error {
	err := EventSchemaSyntaxCheck(spec)
	if err != nil {
		return v1.ErrorSchemaSyntaxError(
			"syntax error: %s", err,
		)
	}
	return uc.repo.UpdateSchema(ctx, source, sType, busName, spec)
}

func (uc *EventUseCase) DeleteSchema(ctx context.Context, source string, sType *string) error {
	return uc.repo.DeleteSchema(ctx, source, sType)
}
