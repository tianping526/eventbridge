package service

import (
	"context"
	"sort"
	"strings"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/service/internal/biz"
)

func (s *EventBridgeService) ListBus(ctx context.Context, request *v1.ListBusRequest) (*v1.ListBusResponse, error) {
	limit := request.Limit
	if limit == 0 {
		limit = 100
	}
	bs, nt, err := s.bc.ListBus(ctx, request.Prefix, limit, request.NextToken)
	if err != nil {
		return nil, err
	}
	buses := make([]*v1.ListBusResponse_Bus, 0, len(bs))
	for _, b := range bs {
		bus := &v1.ListBusResponse_Bus{
			Name: b.Name,
			Mode: b.Mode,
			Source: &v1.MQTopic{
				MqType:    b.Source.Type,
				Endpoints: strings.Split(b.Source.Endpoints, ";"),
				Topic:     b.Source.Topic,
			},
			SourceDelay: &v1.MQTopic{
				MqType:    b.SourceDelay.Type,
				Endpoints: strings.Split(b.SourceDelay.Endpoints, ";"),
				Topic:     b.SourceDelay.Topic,
			},
			TargetExpDecay: &v1.MQTopic{
				MqType:    b.TargetExpDecay.Type,
				Endpoints: strings.Split(b.TargetExpDecay.Endpoints, ";"),
				Topic:     b.TargetExpDecay.Topic,
			},
			TargetBackoff: &v1.MQTopic{
				MqType:    b.TargetBackoff.Type,
				Endpoints: strings.Split(b.TargetBackoff.Endpoints, ";"),
				Topic:     b.TargetBackoff.Topic,
			},
		}
		buses = append(buses, bus)
	}
	return &v1.ListBusResponse{
		Buses:     buses,
		NextToken: nt,
	}, nil
}

func deduplicateStrings(input []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(input))
	for _, v := range input {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

func (s *EventBridgeService) CreateBus(
	ctx context.Context, request *v1.CreateBusRequest,
) (*v1.CreateBusResponse, error) {
	if request.Source.MqType == v1.MQType_MQ_TYPE_UNSPECIFIED {
		request.Source.MqType = v1.MQType_MQ_TYPE_ROCKETMQ
	}
	if request.SourceDelay.MqType == v1.MQType_MQ_TYPE_UNSPECIFIED {
		request.SourceDelay.MqType = v1.MQType_MQ_TYPE_ROCKETMQ
	}
	if request.TargetExpDecay.MqType == v1.MQType_MQ_TYPE_UNSPECIFIED {
		request.TargetExpDecay.MqType = v1.MQType_MQ_TYPE_ROCKETMQ
	}
	if request.TargetBackoff.MqType == v1.MQType_MQ_TYPE_UNSPECIFIED {
		request.TargetBackoff.MqType = v1.MQType_MQ_TYPE_ROCKETMQ
	}
	if request.Mode == v1.BusWorkMode_BUS_WORK_MODE_UNSPECIFIED {
		request.Mode = v1.BusWorkMode_BUS_WORK_MODE_CONCURRENTLY
	}
	request.Source.Endpoints = deduplicateStrings(request.Source.Endpoints)
	sort.Strings(request.Source.Endpoints)
	request.SourceDelay.Endpoints = deduplicateStrings(request.SourceDelay.Endpoints)
	sort.Strings(request.SourceDelay.Endpoints)
	request.TargetExpDecay.Endpoints = deduplicateStrings(request.TargetExpDecay.Endpoints)
	sort.Strings(request.TargetExpDecay.Endpoints)
	request.TargetBackoff.Endpoints = deduplicateStrings(request.TargetBackoff.Endpoints)
	sort.Strings(request.TargetBackoff.Endpoints)
	id, err := s.bc.CreateBus(
		ctx,
		request.Name,
		request.Mode,
		biz.MQTopic{
			Type:      request.Source.MqType,
			Endpoints: strings.Join(request.Source.Endpoints, ";"),
			Topic:     request.Source.Topic,
		},
		biz.MQTopic{
			Type:      request.SourceDelay.MqType,
			Endpoints: strings.Join(request.SourceDelay.Endpoints, ";"),
			Topic:     request.SourceDelay.Topic,
		},
		biz.MQTopic{
			Type:      request.TargetExpDecay.MqType,
			Endpoints: strings.Join(request.TargetExpDecay.Endpoints, ";"),
			Topic:     request.TargetExpDecay.Topic,
		},
		biz.MQTopic{
			Type:      request.TargetBackoff.MqType,
			Endpoints: strings.Join(request.TargetBackoff.Endpoints, ";"),
			Topic:     request.TargetBackoff.Topic,
		},
	)
	if err != nil {
		return nil, err
	}
	return &v1.CreateBusResponse{
		Id: id,
	}, nil
}

func (s *EventBridgeService) DeleteBus(
	ctx context.Context, request *v1.DeleteBusRequest,
) (*v1.DeleteBusResponse, error) {
	err := s.bc.DeleteBus(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	return &v1.DeleteBusResponse{}, nil
}
