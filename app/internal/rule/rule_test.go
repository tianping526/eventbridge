package rule

import (
	"reflect"
	"testing"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
)

func TestRuleComparison(t *testing.T) {
	tpl1 := "value1"
	tpl2 := "value2"
	tpl3 := "value2"
	runs := []struct {
		name string
		r1   *Rule
		r2   *Rule
		eq   bool
	}{
		{
			name: "equal rules",
			r1: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1"},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			r2: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1"},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			eq: true,
		},
		{
			name: "different targets",
			r1: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: nil,
			},
			r2: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{},
			},
			eq: false,
		},
		{
			name: "different target params",
			r1: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1"},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			r2: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value2"},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			eq: false,
		},
		{
			name: "different target params1",
			r1: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1"},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			r2: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key2", Value: "value1"},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			eq: false,
		},
		{
			name: "different target params2",
			r1: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1"},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			r2: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1"},
							{Key: "key2", Value: "value2"},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			eq: false,
		},
		{
			name: "different template params",
			r1: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1", Form: "json", Template: &tpl1},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			r2: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1", Form: "json", Template: &tpl2},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			eq: false,
		},
		{
			name: "different template params1",
			r1: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1", Form: "json", Template: nil},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			r2: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1", Form: "json", Template: &tpl2},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			eq: false,
		},
		{
			name: "equal template params",
			r1: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1", Form: "json", Template: &tpl2},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			r2: &Rule{
				Name:    "rule1",
				BusName: "bus1",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: `{"source": ["app1"]}`,
				Targets: []*Target{
					{
						ID:   1,
						Type: "type1",
						Params: []*TargetParam{
							{Key: "key1", Value: "value1", Form: "json", Template: &tpl3},
						},
						RetryStrategy: v1.RetryStrategy_RETRY_STRATEGY_EXPONENTIAL_DECAY,
					},
				},
			},
			eq: true,
		},
	}

	for _, run := range runs {
		t.Run(run.name, func(t *testing.T) {
			eq := reflect.DeepEqual(*run.r1, *run.r2)
			if eq != run.eq {
				t.Errorf("expected equality: %v, got: %v", run.eq, eq)
			}
		})
	}
}
