package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tianping526/eventbridge/app/internal/rule"
	"github.com/tianping526/eventbridge/app/internal/rule/target"
	"github.com/tianping526/eventbridge/app/internal/rule/transform"
)

func RuleTargetSyntaxCheck(ctx context.Context, t *rule.Target) error {
	logger := log.DefaultLogger
	_, err := target.NewDispatcher(ctx, logger, t)
	if err != nil {
		return err
	}
	_, err = transform.NewTransformer(ctx, logger, t)
	if err != nil {
		return err
	}
	return nil
}
