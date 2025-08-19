package biz

import (
	"context"
	"encoding/json/v2"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tianping526/eventbridge/app/internal/rule/pattern"
)

func RulePatternSyntaxCheck(ctx context.Context, spec *string) error {
	if spec == nil {
		return nil
	}
	logger := log.DefaultLogger
	filterPattern := make(map[string]interface{})
	err := json.Unmarshal([]byte(*spec), &filterPattern)
	if err != nil {
		return err
	}
	_, err = pattern.NewMatcher(ctx, logger, filterPattern)
	if err != nil {
		return err
	}
	return nil
}
