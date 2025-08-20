package transform

import (
	"context"
	"encoding/json/v2"
	"reflect"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"
)

func TestTransform(t *testing.T) {
	logger := log.DefaultLogger
	type eventAndTransformRes struct {
		evt string
		res string
	}
	tmplTxt := "\"i am ${name}, my ip is ${  ip  }.\""
	tmplJSON := "{\"name\": \"${name}\", \"ip\": \"${  ip  }\"}"
	tmplVarConstJSON := "{\"name\": \"${name}\", \"ips\": [\"${  ip  }\", \"10.251.11.1\"], \"cc\":\"bb\"}"
	tmplNestedJSON := "{\"name\": \"${name}\", \"ips\": ${  ips  }}"
	tmplArray := "[{\"name\": \"${name}\", \"ips\": ${  ips  }}]"
	transformTests := []struct {
		target *rule.Target
		events []eventAndTransformRes
	}{
		// The whole event
		{
			target: &rule.Target{
				ID:     0,
				Type:   "",
				Params: []*rule.TargetParam{},
			},
			events: []eventAndTransformRes{
				{
					evt: `
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					res: `
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
				},
			},
		},
		// Partial event
		{
			target: &rule.Target{
				ID:   0,
				Type: "",
				Params: []*rule.TargetParam{
					{
						Key:   "resKey",
						Form:  "JSONPATH",
						Value: "$.data.name",
					},
					{
						Key:   "resKey1",
						Form:  "JSONPATH",
						Value: "$.data.source-ips",
					},
				},
			},
			events: []eventAndTransformRes{
				{
					evt: `
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ips\":[\"10.0.0.123\", \"10.0.0.124\"]}",
  "datacontenttype": "application/json"
}`,
					res: `
{
  "resKey": "test1",
  "resKey1": ["10.0.0.123", "10.0.0.124"]
}`,
				},
			},
		},
		// Constant
		{
			target: &rule.Target{
				ID:   0,
				Type: "",
				Params: []*rule.TargetParam{
					{
						Key:   "resKey",
						Form:  "CONSTANT",
						Value: "https://xxx",
					},
				},
			},
			events: []eventAndTransformRes{
				{
					evt: `
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					res: `
{
  "resKey": "https://xxx"
}`,
				},
			},
		},
		// Text template
		{
			target: &rule.Target{
				ID:   0,
				Type: "",
				Params: []*rule.TargetParam{
					{
						Key:      "resKey",
						Form:     "TEMPLATE",
						Value:    "{\"name  \":\"$.data.name\",\"ip\":\"$.data.source-ip\"}",
						Template: &tmplTxt,
					},
				},
			},
			events: []eventAndTransformRes{
				{
					evt: `
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					res: `
{
  "resKey": "i am test1, my ip is 10.0.0.123."
}`,
				},
			},
		},
		// JSON template
		{
			target: &rule.Target{
				ID:   0,
				Type: "",
				Params: []*rule.TargetParam{
					{
						Key:      "resKey",
						Form:     "TEMPLATE",
						Value:    "{\"name  \":\"$.data.name\",\"ip\":\"$.data.source-ip\"}",
						Template: &tmplJSON,
					},
				},
			},
			events: []eventAndTransformRes{
				{
					evt: `
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					res: `
{
  "resKey": {
    "name": "test1",
    "ip": "10.0.0.123"
  }
}`,
				},
			},
		},
		// JSON template with constant and variable
		{
			target: &rule.Target{
				ID:   0,
				Type: "",
				Params: []*rule.TargetParam{
					{
						Key:      "resKey",
						Form:     "TEMPLATE",
						Value:    "{\"name  \":\"$.data.name\",\"ip\":\"$.data.source-ip\"}",
						Template: &tmplVarConstJSON,
					},
				},
			},
			events: []eventAndTransformRes{
				{
					evt: `
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					res: `
{
  "resKey": {
    "name": "test1",
    "ips": [
      "10.0.0.123",
      "10.251.11.1"
    ],
    "cc": "bb"
  }
}`,
				},
			},
		},
		// Nested JSON template
		{
			target: &rule.Target{
				ID:   0,
				Type: "",
				Params: []*rule.TargetParam{
					{
						Key:      "resKey",
						Form:     "TEMPLATE",
						Value:    "{\"name  \":\"$.data.name\",\"ips\":\"$.data.ips\"}",
						Template: &tmplNestedJSON,
					},
				},
			},
			events: []eventAndTransformRes{
				{
					evt: `
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"ips\":[{\"host\":\"10.0.0.123\", \"port\":\"8080\"}]}",
  "datacontenttype": "application/json"
}`,
					res: `
{
  "resKey": {
    "name": "test1",
    "ips": [
      {
        "host": "10.0.0.123",
        "port": "8080"
      }
    ]
  }
}`,
				},
			},
		},
		// Array template
		{
			target: &rule.Target{
				ID:   0,
				Type: "",
				Params: []*rule.TargetParam{
					{
						Key:      "resKey",
						Form:     "TEMPLATE",
						Value:    "{\"name  \":\"$.data.0.name\",\"ips\":\"$.data.0.ips\"}",
						Template: &tmplArray,
					},
				},
			},
			events: []eventAndTransformRes{
				{
					evt: `
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "[{\"name\":\"test1\",\"ips\":[{\"host\":\"10.0.0.123\", \"port\":\"8080\"}]}]",
  "datacontenttype": "application/json"
}`,
					res: `
{
  "resKey": [{
    "name": "test1",
    "ips": [
      {
        "host": "10.0.0.123",
        "port": "8080"
      }
    ]
  }]
}`,
				},
			},
		},
	}
	for idx, tt := range transformTests {
		tfr, err := NewTransformer(context.Background(), logger, tt.target)
		if err != nil {
			t.Fatalf("case(index=%d) err: %v", idx, err)
		}
		for ei, evt := range tt.events {
			ee := &rule.EventExt{
				EventExt: &v1.EventExt{
					Event: &v1.Event{},
				},
			}
			err = protojson.Unmarshal([]byte(evt.evt), ee.Event)
			if err != nil {
				t.Fatal(err)
			}
			res, err := tfr.Transform(context.Background(), ee)
			if err != nil {
				t.Fatal(err)
			}
			var expectJSON interface{}
			var resJSON interface{}
			err = json.Unmarshal([]byte(evt.res), &expectJSON)
			if err != nil {
				t.Fatal(err)
			}
			err = json.Unmarshal([]byte(res.Event.Data), &resJSON)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expectJSON, resJSON) {
				t.Fatalf("case(index=%d, event_index=%d) test failure", idx, ei)
			}
		}
	}
}
