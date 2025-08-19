package service

import (
	"context"
	"encoding/json/jsontext"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"
)

func (s *EventBridgeService) ListRule(ctx context.Context, request *v1.ListRuleRequest) (*v1.ListRuleResponse, error) {
	limit := request.Limit
	if limit == 0 {
		limit = 100
	}
	rs, nt, err := s.rc.ListRule(ctx, request.BusName, request.Prefix, request.Status, limit, request.NextToken)
	if err != nil {
		return nil, err
	}
	rules := make([]*v1.ListRuleResponse_Rule, 0, len(rs))
	for _, r := range rs {
		targets := make([]*v1.Target, 0, len(r.Targets))
		for _, t := range r.Targets {
			params := make([]*v1.TargetParam, 0, len(t.Params))
			for _, p := range t.Params {
				param := &v1.TargetParam{
					Key:      p.Key,
					Form:     p.Form,
					Value:    p.Value,
					Template: p.Template,
				}
				params = append(params, param)
			}
			target := &v1.Target{
				Id:            t.ID,
				Type:          t.Type,
				Params:        params,
				RetryStrategy: t.RetryStrategy,
			}
			targets = append(targets, target)
		}
		rr := &v1.ListRuleResponse_Rule{
			Name:    r.Name,
			BusName: r.BusName,
			Status:  r.Status,
			Pattern: r.Pattern,
			Targets: targets,
		}
		rules = append(rules, rr)
	}
	return &v1.ListRuleResponse{
		Rules:     rules,
		NextToken: nt,
	}, nil
}

func (s *EventBridgeService) CreateRule(
	ctx context.Context, request *v1.CreateRuleRequest,
) (*v1.CreateRuleResponse, error) {
	status := request.Status
	if request.Status == v1.RuleStatus_RULE_STATUS_UNSPECIFIED {
		status = v1.RuleStatus_RULE_STATUS_ENABLE
	}
	targetMapping := make(map[uint64]*rule.Target, len(request.Targets))
	for _, t := range request.Targets {
		params := make([]*rule.TargetParam, 0, len(t.Params))
		for _, p := range t.Params {
			param := &rule.TargetParam{
				Key:      p.Key,
				Form:     p.Form,
				Value:    p.Value,
				Template: p.Template,
			}
			params = append(params, param)
		}
		target := &rule.Target{
			ID:            t.Id,
			Type:          t.Type,
			Params:        params,
			RetryStrategy: t.RetryStrategy,
		}
		targetMapping[t.Id] = target
	}
	targets := make([]*rule.Target, 0, len(targetMapping))
	for _, t := range targetMapping {
		targets = append(targets, t)
	}
	pattern := []byte(request.Pattern)
	err := (*jsontext.Value)(&pattern).Compact()
	if err != nil {
		return nil, v1.ErrorPatternSyntaxError(
			"syntax error: %s", err,
		)
	}
	id, err := s.rc.CreateRule(ctx, request.BusName, request.Name, status, pattern, targets)
	if err != nil {
		return nil, err
	}
	return &v1.CreateRuleResponse{
		Id: id,
	}, nil
}

func (s *EventBridgeService) UpdateRule(
	ctx context.Context, request *v1.UpdateRuleRequest,
) (*v1.UpdateRuleResponse, error) {
	var pattern []byte
	if request.Pattern != nil {
		pattern = []byte(*request.Pattern)
		err := (*jsontext.Value)(&pattern).Compact()
		if err != nil {
			return nil, v1.ErrorPatternSyntaxError(
				"syntax error: %s", err,
			)
		}
	}
	err := s.rc.UpdateRule(ctx, request.BusName, request.Name, request.Status, pattern)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateRuleResponse{}, nil
}

func (s *EventBridgeService) DeleteRule(
	ctx context.Context, request *v1.DeleteRuleRequest,
) (*v1.DeleteRuleResponse, error) {
	err := s.rc.DeleteRule(ctx, request.BusName, request.Name)
	if err != nil {
		return nil, err
	}
	return &v1.DeleteRuleResponse{}, nil
}

func (s *EventBridgeService) CreateTargets(
	ctx context.Context, request *v1.CreateTargetsRequest,
) (*v1.CreateTargetsResponse, error) {
	targetMapping := make(map[uint64]*rule.Target, len(request.Targets))
	for _, t := range request.Targets {
		params := make([]*rule.TargetParam, 0, len(t.Params))
		for _, p := range t.Params {
			param := &rule.TargetParam{
				Key:      p.Key,
				Form:     p.Form,
				Value:    p.Value,
				Template: p.Template,
			}
			params = append(params, param)
		}
		target := &rule.Target{
			ID:            t.Id,
			Type:          t.Type,
			Params:        params,
			RetryStrategy: t.RetryStrategy,
		}
		targetMapping[t.Id] = target
	}
	targets := make([]*rule.Target, 0, len(targetMapping))
	for _, t := range targetMapping {
		targets = append(targets, t)
	}
	err := s.rc.CreateTargets(ctx, request.BusName, request.RuleName, targets)
	if err != nil {
		return nil, err
	}
	return &v1.CreateTargetsResponse{}, nil
}

func (s *EventBridgeService) DeleteTargets(
	ctx context.Context, request *v1.DeleteTargetsRequest,
) (*v1.DeleteTargetsResponse, error) {
	err := s.rc.DeleteTargets(ctx, request.BusName, request.RuleName, request.Targets)
	if err != nil {
		return nil, err
	}
	return &v1.DeleteTargetsResponse{}, nil
}

func (s *EventBridgeService) ListDispatcherSchema(
	ctx context.Context, request *v1.ListDispatcherSchemaRequest,
) (*v1.ListDispatcherSchemaResponse, error) {
	dss, err := s.rc.ListDispatcherSchema(ctx, request.Types)
	if err != nil {
		return nil, err
	}
	dispatcherSchemas := make([]*v1.ListDispatcherSchemaResponse_DispatcherSchema, 0, len(dss))
	for _, ds := range dss {
		dispatcherSchema := &v1.ListDispatcherSchemaResponse_DispatcherSchema{
			Type:         ds.Type,
			ParamsSchema: ds.ParamsSchema,
		}
		dispatcherSchemas = append(dispatcherSchemas, dispatcherSchema)
	}
	return &v1.ListDispatcherSchemaResponse{
		DispatcherSchemas: dispatcherSchemas,
	}, nil
}
