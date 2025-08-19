package service

import (
	"context"
	"encoding/json/jsontext"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"
)

func (s *EventBridgeService) PostEvent(
	ctx context.Context, request *v1.PostEventRequest,
) (*v1.PostEventResponse, error) {
	if request.RetryStrategy == v1.RetryStrategy_RETRY_STRATEGY_UNSPECIFIED {
		request.RetryStrategy = v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY
	}
	eventExt, err := rule.NewEventExt(request.Event, request.RetryStrategy)
	if err != nil {
		return nil, err
	}
	event, err := s.ec.PostEvent(ctx, eventExt, request.PubTime)
	if err != nil {
		return nil, err
	}
	return &v1.PostEventResponse{
		Id:         event.ID,
		MessageId:  event.MessageID,
		MessageKey: event.MessageKey,
		TraceId:    event.TraceID,
	}, nil
}

func (s *EventBridgeService) ListSchema(
	ctx context.Context, request *v1.ListSchemaRequest,
) (*v1.ListSchemaResponse, error) {
	ss, err := s.ec.ListSchema(ctx, request.Source, request.Type, request.BusName, request.Time)
	if err != nil {
		return nil, err
	}
	schemas := make([]*v1.Schema, 0, len(ss))
	for _, scm := range ss {
		schema := &v1.Schema{
			Source:  scm.Source,
			Type:    scm.Type,
			BusName: scm.BusName,
			Spec:    scm.Spec,
			Time:    scm.Time,
		}
		schemas = append(schemas, schema)
	}
	return &v1.ListSchemaResponse{
		Schemas: schemas,
	}, nil
}

func (s *EventBridgeService) CreateSchema(
	ctx context.Context, request *v1.CreateSchemaRequest,
) (*v1.CreateSchemaResponse, error) {
	specBytes := []byte(request.Spec)
	err := (*jsontext.Value)(&specBytes).Compact()
	if err != nil {
		return nil, v1.ErrorSchemaSyntaxError("syntax error: %s", err)
	}
	err = s.ec.CreateSchema(ctx, request.Source, request.Type, request.BusName, specBytes)
	if err != nil {
		return nil, err
	}
	return &v1.CreateSchemaResponse{}, nil
}

func (s *EventBridgeService) UpdateSchema(
	ctx context.Context, request *v1.UpdateSchemaRequest,
) (*v1.UpdateSchemaResponse, error) {
	var specBytes []byte
	if request.Spec != nil {
		specBytes = []byte(*request.Spec)
		err := (*jsontext.Value)(&specBytes).Compact()
		if err != nil {
			return nil, v1.ErrorSchemaSyntaxError("syntax error: %s", err)
		}
	}
	err := s.ec.UpdateSchema(ctx, request.Source, request.Type, request.BusName, specBytes)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateSchemaResponse{}, nil
}

func (s *EventBridgeService) DeleteSchema(
	ctx context.Context, request *v1.DeleteSchemaRequest,
) (*v1.DeleteSchemaResponse, error) {
	err := s.ec.DeleteSchema(ctx, request.Source, request.Type)
	if err != nil {
		return nil, err
	}
	return &v1.DeleteSchemaResponse{}, nil
}
