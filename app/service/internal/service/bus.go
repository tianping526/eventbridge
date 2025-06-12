package service

import (
	"context"
	"fmt"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
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
			Name:                b.Name,
			Mode:                b.Mode,
			SourceTopic:         b.SourceTopic,
			SourceDelayTopic:    b.SourceDelayTopic,
			TargetExpDecayTopic: b.TargetExpDecayTopic,
			TargetBackoffTopic:  b.TargetBackoffTopic,
		}
		buses = append(buses, bus)
	}
	return &v1.ListBusResponse{
		Buses:     buses,
		NextToken: nt,
	}, nil
}

func (s *EventBridgeService) CreateBus(
	ctx context.Context, request *v1.CreateBusRequest,
) (*v1.CreateBusResponse, error) {
	sn := fmt.Sprintf("EBInterBus%s", request.Name)
	sdn := fmt.Sprintf("EBInterDelayBus%s", request.Name)
	ten := fmt.Sprintf("EBInterTargetExpDecayBus%s", request.Name)
	tbn := fmt.Sprintf("EBInterTargetBackoffBus%s", request.Name)
	if request.SourceTopic != nil {
		sn = *request.SourceTopic
	}
	if request.SourceDelayTopic != nil {
		sdn = *request.SourceDelayTopic
	}
	if request.TargetExpDecayTopic != nil {
		ten = *request.TargetExpDecayTopic
	}
	if request.TargetBackoffTopic != nil {
		tbn = *request.TargetBackoffTopic
	}
	if request.Mode == v1.BusWorkMode_BUS_WORK_MODE_UNSPECIFIED {
		request.Mode = v1.BusWorkMode_BUS_WORK_MODE_CONCURRENTLY
	}
	id, err := s.bc.CreateBus(
		ctx,
		request.Name,
		request.Mode,
		sn,
		sdn,
		ten,
		tbn,
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
