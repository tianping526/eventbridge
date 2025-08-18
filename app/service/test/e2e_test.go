package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/service/internal/conf"
	"github.com/tianping526/eventbridge/app/service/internal/service"
)

var (
	sv               *service.EventBridgeService
	rocketmqEndpoint = "127.0.0.1:8082"
)

// TestMain setup docker-compose
func TestMain(m *testing.M) {
	code := 0
	defer func() {
		if code != 0 {
			os.Exit(code)
		}
	}()

	// setup docker-compose
	var err error
	if os.Getenv("TEST_ENV") != "CI" {
		cmdUp := exec.Command("docker-compose", "-f", "docker-compose.yaml", "-p", "eb-svc-t", "up", "-d")
		err = cmdUp.Run()
		if err != nil {
			panic(fmt.Sprintf("docker-compose up error: %v", err))
		}
		defer func() {
			cmdDown := exec.Command("docker-compose", "-f", "docker-compose.yaml", "-p", "eb-svc-t", "down")
			err = cmdDown.Run()
			if err != nil {
				panic(fmt.Sprintf("docker-compose down error: %v", err))
			}
		}()
		time.Sleep(10 * time.Second) // wait for docker-compose up
	}

	// setup app
	appInfo := &conf.AppInfo{
		Name:     "eventbridge.test",
		Version:  "",
		FlagConf: "../configs",
		Id:       "test",
	}
	if os.Getenv("TEST_ENV") == "CI" {
		appInfo.FlagConf = "./configs"
		rocketmqEndpoint = "proxy:8081"
	}
	var cleanup func()
	sv, cleanup, err = wireService(appInfo)
	if err != nil {
		panic(fmt.Sprintf("new service error: %v", err))
	}
	defer cleanup()

	// Create Default bus
	_, err = sv.CreateBus(context.Background(), &v1.CreateBusRequest{
		Name: "Default",
		Mode: v1.BusWorkMode_BUS_WORK_MODE_CONCURRENTLY,
		Source: &v1.MQTopic{
			MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
			Endpoints: []string{
				rocketmqEndpoint,
			},
			Topic: "EBInterBusDefault",
		},
		SourceDelay: &v1.MQTopic{
			MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
			Endpoints: []string{
				rocketmqEndpoint,
			},
			Topic: "EBInterDelayBusDefault",
		},
		TargetExpDecay: &v1.MQTopic{
			MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
			Endpoints: []string{
				rocketmqEndpoint,
			},
			Topic: "EBInterTargetExpDecayBusDefault",
		},
		TargetBackoff: &v1.MQTopic{
			MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
			Endpoints: []string{
				rocketmqEndpoint,
			},
			Topic: "EBInterTargetBackoffBusDefault",
		},
	})
	if err != nil {
		if !v1.IsBusNameRepeat(err) {
			panic(fmt.Sprintf("create default bus error: %v", err))
		}
	}
	time.Sleep(5 * time.Second) // wait for bus creation

	code = m.Run()
}

func TestPostEvent(t *testing.T) {
	convey.Convey("Given everything positive", t, func() {
		spec := "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\"," +
			"\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}"
		_, err := sv.CreateSchema(context.Background(), &v1.CreateSchemaRequest{
			Source:  "PostEventSource",
			Type:    "PostEventType",
			BusName: "Default",
			Spec:    spec,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("When PostEvent", func() {
			_, err = sv.PostEvent(context.Background(), &v1.PostEventRequest{
				Event: &v1.Event{
					Source:          "PostEventSource",
					Type:            "PostEventType",
					Data:            `{"a":"b"}`,
					Datacontenttype: "application/json",
				},
			})
			convey.Convey("Then err should be nil.", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})
	})
}

func TestListSchema(t *testing.T) {
	convey.Convey("Given a schema to a source", t, func() {
		var (
			ctx    = context.Background()
			source = "source5"
			req    = &v1.ListSchemaRequest{
				Source: &source,
			}
		)
		spec := "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\"," +
			"\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}"
		_, err := sv.CreateSchema(ctx, &v1.CreateSchemaRequest{
			Source:  source,
			Type:    "type5",
			BusName: "Default",
			Spec:    spec,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("When ListSchema", func() {
			p1, err := sv.ListSchema(ctx, req)
			convey.Convey("Then err should be nil.len(p1.Schemas) should not be 1.", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(p1.Schemas), convey.ShouldEqual, 1)
			})
		})
	})
}

func TestCreateSchema(t *testing.T) {
	convey.Convey("Given duplicate source+type", t, func() {
		spec := "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\"," +
			"\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}"
		var (
			ctx = context.Background()
			req = &v1.CreateSchemaRequest{
				Source:  "sourceRepeat",
				Type:    "typeRepeat",
				BusName: "Default",
				Spec:    spec,
			}
		)
		convey.Convey("When CreateSchema", func() {
			_, err := sv.CreateSchema(ctx, req)
			_, err2 := sv.CreateSchema(ctx, req)
			convey.Convey("Then err1 should be nil.err2 should not be SOURCE_TYPE_REPEAT.", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(v1.IsSourceTypeRepeat(err2), convey.ShouldBeTrue)
			})
		})
	})
	convey.Convey("Given a schema whose bus name does not exist", t, func() {
		spec := "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\"," +
			"\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}"
		var (
			ctx = context.Background()
			req = &v1.CreateSchemaRequest{
				Source:  "source6",
				Type:    "type6",
				BusName: "dada",
				Spec:    spec,
			}
		)
		convey.Convey("When CreateSchema", func() {
			_, err := sv.CreateSchema(ctx, req)
			convey.Convey("Then err should be DATA_BUS_NOT_FOUND.", func() {
				convey.So(v1.IsDataBusNotFound(err), convey.ShouldBeTrue)
			})
		})
	})
	convey.Convey("Given a schema whose spec syntax err", t, func() {
		spec := "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\"," +
			"\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"fakerType\"}}}"
		var (
			ctx = context.Background()
			req = &v1.CreateSchemaRequest{
				Source:  "source6",
				Type:    "type6",
				BusName: "dada",
				Spec:    spec,
			}
		)
		convey.Convey("When CreateSchema", func() {
			_, err := sv.CreateSchema(ctx, req)
			convey.Convey("Then err should be SCHEMA_SYNTAX_ERROR.", func() {
				convey.So(v1.IsSchemaSyntaxError(err), convey.ShouldBeTrue)
			})
		})
	})
}

func TestUpdateSchema(t *testing.T) {
	convey.Convey("Given a schema whose source+type does not exist", t, func() {
		spec := "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\"," +
			"\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}"
		var (
			ctx = context.Background()
			req = &v1.UpdateSchemaRequest{
				Source: "source_not_found",
				Type:   "type_not_found",
				Spec:   &spec,
			}
		)
		convey.Convey("When UpdateSchema", func() {
			p1, err := sv.UpdateSchema(ctx, req)
			convey.Convey("Then err should be SCHEMA_NOT_FOUND.p1 should not be nil.", func() {
				convey.So(v1.IsSchemaNotFound(err), convey.ShouldBeTrue)
				convey.So(p1, convey.ShouldBeNil)
			})
		})
	})
	convey.Convey("Given a schema whose bus_name does not exist", t, func() {
		var (
			ctx     = context.Background()
			busName = "not_found_bus_name"
			req     = &v1.UpdateSchemaRequest{
				Source:  "source8",
				Type:    "source8",
				BusName: &busName,
			}
		)
		spec := "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\"," +
			"\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}"
		_, err := sv.CreateSchema(ctx, &v1.CreateSchemaRequest{
			Source:  "source8",
			Type:    "type8",
			BusName: "Default",
			Spec:    spec,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("When UpdateSchema", func() {
			p1, err := sv.UpdateSchema(ctx, req)
			convey.Convey("Then err should be DATA_BUS_NOT_FOUND.p1 should be nil.", func() {
				convey.So(v1.IsDataBusNotFound(err), convey.ShouldBeTrue)
				convey.So(p1, convey.ShouldBeNil)
			})
		})
	})
	convey.Convey("Given a schema whose spec syntax err", t, func() {
		spec := "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\"," +
			"\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"fakerType\"}}}"
		var (
			ctx     = context.Background()
			busName = "not_found_bus_name"
			req     = &v1.UpdateSchemaRequest{
				Source:  "source8",
				Type:    "source8",
				BusName: &busName,
				Spec:    &spec,
			}
		)
		convey.Convey("When UpdateSchema", func() {
			p1, err := sv.UpdateSchema(ctx, req)
			convey.Convey("Then err should be SCHEMA_SYNTAX_ERROR.p1 should be nil.", func() {
				convey.So(v1.IsSchemaSyntaxError(err), convey.ShouldBeTrue)
				convey.So(p1, convey.ShouldBeNil)
			})
		})
	})
}

func TestDeleteSchema(t *testing.T) {
	convey.Convey("Given everything positive", t, func() {
		var (
			ctx   = context.Background()
			sType = "type9"
			req   = &v1.DeleteSchemaRequest{
				Source: "source9",
				Type:   &sType,
			}
		)
		spec := "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\"," +
			"\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}"
		_, err := sv.CreateSchema(ctx, &v1.CreateSchemaRequest{
			Source:  "source9",
			Type:    sType,
			BusName: "Default",
			Spec:    spec,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("When DeleteSchema", func() {
			_, err := sv.DeleteSchema(ctx, req)
			convey.Convey("Then err should be nil.p1 should not be nil.", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})
	})
}

func TestListBus(t *testing.T) {
	convey.Convey("Given a bus other than Default", t, func() {
		_, err := sv.CreateBus(context.Background(), &v1.CreateBusRequest{
			Name: "Default1",
			Mode: v1.BusWorkMode_BUS_WORK_MODE_CONCURRENTLY,
			Source: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterBusDefault1",
			},
			SourceDelay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterDelayBusDefault1",
			},
			TargetExpDecay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetExpDecayBusDefault1",
			},
			TargetBackoff: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetBackoffBusDefault1",
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("When ListBus with limit 1", func() {
			prefix := "Default"
			reply, err := sv.ListBus(context.Background(), &v1.ListBusRequest{
				Prefix: &prefix,
				Limit:  1,
			})
			convey.So(err, convey.ShouldBeNil)
			reply1, err := sv.ListBus(context.Background(), &v1.ListBusRequest{
				Prefix:    &prefix,
				Limit:     1,
				NextToken: reply.NextToken,
			})
			convey.So(err, convey.ShouldBeNil)
			convey.Convey("Then first len(reply.Buses) should be 1 and nextToken should not be 0, "+
				"second len(reply1.Buses) should be 1 and nextToken should be 0", func() {
				convey.So(len(reply.Buses), convey.ShouldEqual, 1)
				convey.So(reply.NextToken, convey.ShouldNotEqual, 0)
				convey.So(len(reply1.Buses), convey.ShouldEqual, 1)
				convey.So(reply1.NextToken, convey.ShouldEqual, 0)
			})
		})
	})
}

func TestDeleteBus(t *testing.T) {
	convey.Convey("Given everything positive", t, func() {
		var (
			ctx = context.Background()
			req = &v1.DeleteBusRequest{
				Name: "CreatedBus",
			}
		)
		_, err := sv.CreateBus(ctx, &v1.CreateBusRequest{
			Name: "CreatedBus",
			Mode: v1.BusWorkMode_BUS_WORK_MODE_CONCURRENTLY,
			Source: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterBusCreatedBus",
			},
			SourceDelay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterDelayBusCreatedBus",
			},
			TargetExpDecay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetExpDecayBusCreatedBus",
			},
			TargetBackoff: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetBackoffBusCreatedBus",
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("When DeleteBus and ListBus", func() {
			_, err = sv.DeleteBus(ctx, req)
			prefix := "CreatedBus"
			p1, _ := sv.ListBus(ctx, &v1.ListBusRequest{
				Prefix: &prefix,
			})
			convey.Convey("Then err should be nil.len(p1.Buses) should not be 0.", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(p1.Buses), convey.ShouldEqual, 0)
			})
		})
	})
}

func TestCreateRule(t *testing.T) {
	convey.Convey("Given a Rule whose bus_name does not exist", t, func() {
		pattern := "{\"subject\":[{\"prefix\":\"acs:oss:cn-hangzhou:1234567:xls-papk/\"}," +
			"{\"suffix\":\".txt\"},{\"suffix\":\".jpg\"}]}"
		var (
			ctx = context.Background()
			req = &v1.CreateRuleRequest{
				Name:    "TestRule",
				BusName: "not_found_bus_name",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: pattern,
				Targets: []*v1.Target{
					{
						Id:   1,
						Type: "gRPCDispatcher",
						Params: []*v1.TargetParam{
							{
								Value: "https://oapi.dingtalk.com/robot/send?" +
									"access_token=1560abe367f48877c69bb6a9916244979927abbbbf82f4fe8801692cd6ea****",
								Form: "CONSTANT",
								Key:  "URL",
							},
						},
					},
				},
			}
		)
		convey.Convey("When CreateRule", func() {
			p1, err := sv.CreateRule(ctx, req)
			convey.Convey("Then err should be DATA_BUS_NOT_FOUND.p1 should be nil.", func() {
				convey.So(v1.IsDataBusNotFound(err), convey.ShouldBeTrue)
				convey.So(p1, convey.ShouldBeNil)
			})
		})
	})
	convey.Convey("Given a Rule whose target.type does not exist", t, func() {
		pattern := "{\"subject\":[{\"prefix\":\"acs:oss:cn-hangzhou:1234567:xls-papk/\"}," +
			"{\"suffix\":\".txt\"},{\"suffix\":\".jpg\"}]}"
		var (
			ctx = context.Background()
			req = &v1.CreateRuleRequest{
				Name:    "TestRule",
				BusName: "not_found_bus_name",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: pattern,
				Targets: []*v1.Target{
					{
						Id:   1,
						Type: "fakerType",
						Params: []*v1.TargetParam{
							{
								Value: "https://oapi.dingtalk.com/robot/send?" +
									"access_token=1560abe367f48877c69bb6a9916244979927abbbbf82f4fe8801692cd6ea****",
								Form: "CONSTANT",
								Key:  "URL",
							},
						},
					},
				},
			}
		)
		convey.Convey("When CreateRule", func() {
			p1, err := sv.CreateRule(ctx, req)
			convey.Convey("Then err should be TARGET_PARAM_SYNTAX_ERROR.p1 should be nil.", func() {
				convey.So(v1.IsTargetParamSyntaxError(err), convey.ShouldBeTrue)
				convey.So(p1, convey.ShouldBeNil)
			})
		})
	})
	convey.Convey("Given a Rule whose target.type does not exist", t, func() {
		pattern := "{\"subject\":[{\"prefix\":\"acs:oss:cn-hangzhou:1234567:xls-papk/\"}," +
			"{\"suffix\":\".txt\"},{\"suffix\":\".jpg\"}]}"
		var (
			ctx  = context.Background()
			tmpl = "{\"code\":\"10188:${subject}\"a}"
			req  = &v1.CreateRuleRequest{
				Name:    "TestRule",
				BusName: "not_found_bus_name",
				Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
				Pattern: pattern,
				Targets: []*v1.Target{
					{
						Id:   1,
						Type: "gRPCDispatcher",
						Params: []*v1.TargetParam{
							{
								Value:    `{"subject":"$.data.a"}`,
								Form:     "TEMPLATE",
								Key:      "body",
								Template: &tmpl,
							},
						},
					},
				},
			}
		)
		convey.Convey("When CreateRule", func() {
			p1, err := sv.CreateRule(ctx, req)
			convey.Convey("Then err should be TARGET_PARAM_SYNTAX_ERROR.p1 should be nil.", func() {
				convey.So(v1.IsTargetParamSyntaxError(err), convey.ShouldBeTrue)
				convey.So(p1, convey.ShouldBeNil)
			})
		})
	})
}

func TestUpdateRule(t *testing.T) {
	convey.Convey("Given everything positive", t, func() {
		var (
			ctx = context.Background()
			req = &v1.UpdateRuleRequest{
				Name:    "TestRule",
				BusName: "CreatedBus1",
				Status:  v1.RuleStatus_RULE_STATUS_DISABLE,
			}
		)
		_, err := sv.CreateBus(ctx, &v1.CreateBusRequest{
			Name: "CreatedBus1",
			Mode: v1.BusWorkMode_BUS_WORK_MODE_CONCURRENTLY,
			Source: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterBusCreatedBus1",
			},
			SourceDelay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterDelayBusCreatedBus1",
			},
			TargetExpDecay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetExpDecayBusCreatedBus1",
			},
			TargetBackoff: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetBackoffBusCreatedBus1",
			},
		})
		convey.So(err, convey.ShouldBeNil)
		pattern := "{\"subject\":[{\"prefix\":\"acs:oss:cn-hangzhou:1234567:xls-papk/\"}," +
			"{\"suffix\":\".txt\"},{\"suffix\":\".jpg\"}]}"
		_, err = sv.CreateRule(ctx, &v1.CreateRuleRequest{
			Name:    "TestRule",
			BusName: "CreatedBus1",
			Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
			Pattern: pattern,
			Targets: []*v1.Target{
				{
					Id:   1,
					Type: "gRPCDispatcher",
					Params: []*v1.TargetParam{
						{
							Value: "https://oapi.dingtalk.com/robot/send?" +
								"access_token=1560abe367f48877c69bb6a9916244979927abbbbf82f4fe8801692cd6ea****",
							Form: "CONSTANT",
							Key:  "URL",
						},
					},
				},
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("When UpdateRule", func() {
			_, err := sv.UpdateRule(ctx, req)
			convey.Convey("Then err should be nil.", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})
	})
	convey.Convey("Given rule does not exist", t, func() {
		var (
			ctx = context.Background()
			req = &v1.UpdateRuleRequest{
				Name:    "TestRule1",
				BusName: "CreatedBus1",
				Status:  v1.RuleStatus_RULE_STATUS_DISABLE,
			}
		)
		convey.Convey("When UpdateRule", func() {
			_, err := sv.UpdateRule(ctx, req)
			convey.Convey("Then err should be RULE_NOT_FOUND.", func() {
				convey.So(v1.IsRuleNotFound(err), convey.ShouldBeTrue)
			})
		})
	})
}

func TestDeleteRule(t *testing.T) {
	convey.Convey("Given everything positive", t, func() {
		var (
			ctx = context.Background()
			req = &v1.DeleteRuleRequest{
				Name:    "TestRule2",
				BusName: "CreatedBus2",
			}
		)
		_, err := sv.CreateBus(ctx, &v1.CreateBusRequest{
			Name: "CreatedBus2",
			Mode: v1.BusWorkMode_BUS_WORK_MODE_CONCURRENTLY,
			Source: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterBusCreatedBus2",
			},
			SourceDelay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterDelayBusCreatedBus2",
			},
			TargetExpDecay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetExpDecayBusCreatedBus2",
			},
			TargetBackoff: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetBackoffBusCreatedBus2",
			},
		})
		convey.So(err, convey.ShouldBeNil)
		pattern := "{\"subject\":[{\"prefix\":\"acs:oss:cn-hangzhou:1234567:xls-papk/\"}," +
			"{\"suffix\":\".txt\"},{\"suffix\":\".jpg\"}]}"
		_, err = sv.CreateRule(ctx, &v1.CreateRuleRequest{
			Name:    "TestRule2",
			BusName: "CreatedBus2",
			Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
			Pattern: pattern,
			Targets: []*v1.Target{
				{
					Id:   1,
					Type: "gRPCDispatcher",
					Params: []*v1.TargetParam{
						{
							Value: "https://oapi.dingtalk.com/robot/send?" +
								"access_token=1560abe367f48877c69bb6a9916244979927abbbbf82f4fe8801692cd6ea****",
							Form: "CONSTANT",
							Key:  "URL",
						},
					},
				},
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("When DeleteRule", func() {
			_, err := sv.DeleteRule(ctx, req)
			prefix := "TestRule2"
			reply, _ := sv.ListRule(ctx, &v1.ListRuleRequest{
				Prefix:  &prefix,
				BusName: "CreatedBus2",
			})
			convey.Convey("Then err should be nil.list rule len(reply.Rules) should be 0.", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(reply.Rules), convey.ShouldEqual, 0)
			})
		})
	})
	convey.Convey("Given rule does not exist", t, func() {
		var (
			ctx = context.Background()
			req = &v1.DeleteRuleRequest{
				Name:    "TestRuleNotFound",
				BusName: "CreatedBus1",
			}
		)
		convey.Convey("When DeleteRule", func() {
			_, err := sv.DeleteRule(ctx, req)
			convey.Convey("Then err should be RULE_NOT_FOUND.", func() {
				convey.So(v1.IsRuleNotFound(err), convey.ShouldBeTrue)
			})
		})
	})
}

func TestCreateTargets(t *testing.T) {
	convey.Convey("Given rule does not exist", t, func() {
		var (
			ctx = context.Background()
			req = &v1.CreateTargetsRequest{
				RuleName: "not_found_rule_name",
				BusName:  "not_found_bus_name",
				Targets: []*v1.Target{
					{
						Id:   2,
						Type: "gRPCDispatcher",
						Params: []*v1.TargetParam{
							{
								Value: "https://oapi.dingtalk.com/robot/send?" +
									"access_token=1560abe367f48877c69bb6a9916244979927abbbbf82f4fe8801692cd6ea****",
								Form: "CONSTANT",
								Key:  "URL",
							},
						},
					},
				},
			}
		)
		convey.Convey("When CreateTargets", func() {
			_, err := sv.CreateTargets(ctx, req)
			convey.Convey("Then err should be RULE_NOT_FOUND.", func() {
				convey.So(v1.IsRuleNotFound(err), convey.ShouldBeTrue)
			})
		})
	})
	convey.Convey("Given rule.target.type does not exist", t, func() {
		var (
			ctx = context.Background()
			req = &v1.CreateTargetsRequest{
				RuleName: "not_found_rule_name",
				BusName:  "not_found_bus_name",
				Targets: []*v1.Target{
					{
						Id:   2,
						Type: "fakeType",
						Params: []*v1.TargetParam{
							{
								Value: "https://oapi.dingtalk.com/robot/send?" +
									"access_token=1560abe367f48877c69bb6a9916244979927abbbbf82f4fe8801692cd6ea****",
								Form: "CONSTANT",
								Key:  "URL",
							},
						},
					},
				},
			}
		)
		convey.Convey("When CreateTargets", func() {
			_, err := sv.CreateTargets(ctx, req)
			convey.Convey("Then err should be RULE_NOT_FOUND.", func() {
				convey.So(v1.IsTargetParamSyntaxError(err), convey.ShouldBeTrue)
			})
		})
	})
}

func TestDeleteTargets(t *testing.T) {
	convey.Convey("When everything goes positive", t, func() {
		var (
			ctx = context.Background()
			req = &v1.DeleteTargetsRequest{
				RuleName: "TestRule3",
				BusName:  "CreatedBus3",
				Targets:  []uint64{3},
			}
		)
		_, err := sv.CreateBus(ctx, &v1.CreateBusRequest{
			Name: "CreatedBus3",
			Mode: v1.BusWorkMode_BUS_WORK_MODE_CONCURRENTLY,
			Source: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterBusCreatedBus3",
			},
			SourceDelay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterDelayBusCreatedBus3",
			},
			TargetExpDecay: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetExpDecayBusCreatedBus3",
			},
			TargetBackoff: &v1.MQTopic{
				MqType: v1.MQType_MQ_TYPE_ROCKETMQ,
				Endpoints: []string{
					rocketmqEndpoint,
				},
				Topic: "EBInterTargetBackoffBusCreatedBus3",
			},
		})
		convey.So(err, convey.ShouldBeNil)
		pattern := "{\"subject\":[{\"prefix\":\"acs:oss:cn-hangzhou:1234567:xls-papk/\"}," +
			"{\"suffix\":\".txt\"},{\"suffix\":\".jpg\"}]}"
		_, err = sv.CreateRule(ctx, &v1.CreateRuleRequest{
			Name:    "TestRule3",
			BusName: "CreatedBus3",
			Status:  v1.RuleStatus_RULE_STATUS_ENABLE,
			Pattern: pattern,
			Targets: []*v1.Target{
				{
					Id:   3,
					Type: "gRPCDispatcher",
					Params: []*v1.TargetParam{
						{
							Value: "https://oapi.dingtalk.com/robot/send?" +
								"access_token=1560abe367f48877c69bb6a9916244979927abbbbf82f4fe8801692cd6ea****",
							Form: "CONSTANT",
							Key:  "URL",
						},
					},
				},
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("When DeleteTargets", func() {
			_, err := sv.DeleteTargets(ctx, req)
			prefix := "TestRule3"
			reply, _ := sv.ListRule(ctx, &v1.ListRuleRequest{
				Prefix:  &prefix,
				BusName: "CreatedBus3",
			})
			convey.Convey("Then err should be nil.target len should be 0", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(reply.Rules[0].Targets), convey.ShouldEqual, 0)
			})
		})
	})
}

func TestListDispatcherSchema(t *testing.T) {
	convey.Convey("When types is empty", t, func() {
		var (
			ctx = context.Background()
			req = &v1.ListDispatcherSchemaRequest{}
		)
		convey.Convey("When ListDispatcherSchema", func() {
			reply, err := sv.ListDispatcherSchema(ctx, req)
			convey.So(err, convey.ShouldBeNil)
			convey.Convey("Then err should be nil. DispatcherSchemas len should be greater than 0", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(reply.DispatcherSchemas), convey.ShouldBeGreaterThan, 0)
			})
		})
	})
	convey.Convey("When types is noopDispatcher", t, func() {
		var (
			ctx = context.Background()
			req = &v1.ListDispatcherSchemaRequest{
				Types: []string{"noopDispatcher"},
			}
		)
		convey.Convey("When ListDispatcherSchema", func() {
			reply, err := sv.ListDispatcherSchema(ctx, req)
			convey.So(err, convey.ShouldBeNil)
			convey.Convey("Then err should be nil. DispatcherSchemas len should be 1", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(reply.DispatcherSchemas), convey.ShouldEqual, 1)
			})
		})
	})
	convey.Convey("When types is faker", t, func() {
		var (
			ctx = context.Background()
			req = &v1.ListDispatcherSchemaRequest{
				Types: []string{"faker"},
			}
		)
		convey.Convey("When ListDispatcherSchema", func() {
			reply, err := sv.ListDispatcherSchema(ctx, req)
			convey.So(err, convey.ShouldBeNil)
			convey.Convey("Then err should be nil. DispatcherSchemas len should be 0", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(reply.DispatcherSchemas), convey.ShouldEqual, 0)
			})
		})
	})
}
