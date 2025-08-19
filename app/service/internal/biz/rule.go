package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"
)

type DispatcherSchema struct {
	Type         string
	ParamsSchema string
}

type RuleRepo interface {
	ListRule(
		ctx context.Context, bus string, prefix *string, status v1.RuleStatus, limit int32, nextToken uint64,
	) ([]*rule.Rule, uint64, error)
	CreateRule(
		ctx context.Context, bus string, name string, status v1.RuleStatus, pattern []byte, targets []*rule.Target,
	) (uint64, error)
	UpdateRule(ctx context.Context, bus string, name string, status v1.RuleStatus, pattern []byte) error
	DeleteRule(ctx context.Context, bus string, name string) error
	CreateTargets(ctx context.Context, bus string, ruleName string, targets []*rule.Target) error
	DeleteTargets(ctx context.Context, bus string, ruleName string, targetIDs []uint64) error
	ListDispatcherSchema(ctx context.Context, types []string) ([]*DispatcherSchema, error)
}

type RuleUseCase struct {
	repo RuleRepo

	log *log.Helper
}

func NewRuleUseCase(repo RuleRepo, logger log.Logger) *RuleUseCase {
	return &RuleUseCase{
		repo: repo,
		log: log.NewHelper(log.With(
			logger,
			"module", "usecase/rule",
			"caller", log.DefaultCaller,
		)),
	}
}

func (uc *RuleUseCase) ListRule(
	ctx context.Context, bus string, prefix *string, status v1.RuleStatus, limit int32, nextToken uint64,
) ([]*rule.Rule, uint64, error) {
	return uc.repo.ListRule(ctx, bus, prefix, status, limit, nextToken)
}

func (uc *RuleUseCase) CreateRule(
	ctx context.Context, bus string, name string, status v1.RuleStatus, pattern []byte, targets []*rule.Target,
) (uint64, error) {
	err := RulePatternSyntaxCheck(ctx, pattern)
	if err != nil {
		return 0, v1.ErrorPatternSyntaxError(
			"syntax error: %s", err,
		)
	}
	for _, t := range targets {
		errCheck := RuleTargetSyntaxCheck(ctx, t)
		if errCheck != nil {
			return 0, v1.ErrorTargetParamSyntaxError(
				"parameter syntax error: %s", errCheck,
			)
		}
	}
	return uc.repo.CreateRule(ctx, bus, name, status, pattern, targets)
}

func (uc *RuleUseCase) UpdateRule(
	ctx context.Context, bus string, name string, status v1.RuleStatus, pattern []byte,
) error {
	if pattern != nil {
		err := RulePatternSyntaxCheck(ctx, pattern)
		if err != nil {
			return v1.ErrorPatternSyntaxError(
				"syntax error: %s", err,
			)
		}
	}
	return uc.repo.UpdateRule(ctx, bus, name, status, pattern)
}

func (uc *RuleUseCase) DeleteRule(ctx context.Context, bus string, name string) error {
	return uc.repo.DeleteRule(ctx, bus, name)
}

func (uc *RuleUseCase) CreateTargets(ctx context.Context, bus string, ruleName string, targets []*rule.Target) error {
	for _, t := range targets {
		errCheck := RuleTargetSyntaxCheck(ctx, t)
		if errCheck != nil {
			return v1.ErrorTargetParamSyntaxError(
				"parameter syntax error: %s", errCheck,
			)
		}
	}
	return uc.repo.CreateTargets(ctx, bus, ruleName, targets)
}

func (uc *RuleUseCase) DeleteTargets(ctx context.Context, bus string, ruleName string, targetIDs []uint64) error {
	return uc.repo.DeleteTargets(ctx, bus, ruleName, targetIDs)
}

func (uc *RuleUseCase) ListDispatcherSchema(ctx context.Context, types []string) ([]*DispatcherSchema, error) {
	return uc.repo.ListDispatcherSchema(ctx, types)
}
