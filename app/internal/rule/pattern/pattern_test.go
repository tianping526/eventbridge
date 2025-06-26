package pattern

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/internal/rule"
)

func TestPattern(t *testing.T) {
	logger := log.DefaultLogger
	type eventAndMatchRes struct {
		evt string
		ok  bool
	}
	patternTests := []struct {
		pattern string
		events  []eventAndMatchRes
	}{
		// specific value
		{
			pattern: `
{
  "source": [
    "testSource1"
  ]
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"a\":\"i am test content ad\"}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource11",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"a\":\"i am test content ad\"}",
  "datacontenttype": "application/json"
}`,
					false,
				},
			},
		},
		{
			pattern: `
{
  "source": "testSource1"
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"a\":\"i am test content ad\"}",
  "datacontenttype": "application/json"
}`,
					true,
				},
			},
		},
		{
			pattern: `
{
  "data": {
    "name": [
      "test"
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
} `,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"tes\",\"scope\":100}",
  "datacontenttype": "application/json"
} `,
					false,
				},
			},
		},
		// prefix
		{
			pattern: `
{
  "source": [
    {
      "prefix": "testSource1"
    }
  ]
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource2",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"a\":\"i am test content ad\"}",
  "datacontenttype": "application/json"
}`,
					false,
				},
			},
		},
		// suffix
		{
			pattern: `
{
  "subject": [
    {
      "suffix": "est"
    },
    {
      "suffix": "xxx"
    }
  ]
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit xxx",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource2",
  "subject": "dolor mollit reprehenderit velit",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"a\":\"i am test content ad\"}",
  "datacontenttype": "application/json"
}`,
					false,
				},
			},
		},
		// anything-but
		{
			pattern: `
{
  "data": {
    "name": [
      {
        "anything-but": "test"
      }
    ],
    "scope": [
      {
        "anything-but": 100
      }
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					false,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit xxx",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"tes\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					false,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit xxx",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"tes\",\"scope\":10}",
  "datacontenttype": "application/json"
}`,
					true,
				},
			},
		},
		{
			pattern: `
{
  "data": {
    "name": [
      {
        "anything-but": [
          "test",
          "test1"
        ]
      }
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					false,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit xxx",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					false,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit xxx",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"tes\",\"scope\":10}",
  "datacontenttype": "application/json"
}`,
					true,
				},
			},
		},
		{
			pattern: `
{
  "data": {
    "name": [
      {
        "anything-but": {
          "prefix": "tes"
        }
      }
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					false,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit xxx",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"xxx\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					true,
				},
			},
		},
		// exists
		{
			pattern: `
{
  "data": {
    "name": [
      {
        "exists": true
      }
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit xxx",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name1\":\"xxx\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					false,
				},
			},
		},
		{
			pattern: `
{
  "data": {
    "name": [
      {
        "exists": false
      }
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					false,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit xxx",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name1\":\"xxx\",\"scope\":100}",
  "datacontenttype": "application/json"
}`,
					true,
				},
			},
		},
		// numeric
		{
			pattern: `
{
  "data": {
    "count1": [
      {
        "numeric": [
          ">",
          0,
          "<=",
          5
        ]
      }
    ],
    "count2": [
      {
        "numeric": [
          "<",
          10
        ]
      }
    ],
    "count3": [
      {
        "numeric": [
          "=",
          301.8
        ]
      }
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100,\"count1\":3,\"count2\":8,\"count3\":301.8}",
  "datacontenttype": "application/json"
}
`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100,\"count1\":6,\"count2\":8,\"count3\":301.8}",
  "datacontenttype": "application/json"
}
`,
					false,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100,\"count1\":3,\"count2\":8,\"count3\":301.9}",
  "datacontenttype": "application/json"
}
`,
					false,
				},
			},
		},
		// cidr
		{
			pattern: `
{
  "data": {
    "source-ip": [
      {
        "cidr": "10.0.0.0/24"
      }
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"source-ip\":\"10.0.1.123\"}",
  "datacontenttype": "application/json"
}`,
					false,
				},
			},
		},
		// multiple
		{
			pattern: `
{
  "source": [
    {
      "prefix": "testSource1"
    }
  ],
  "data": {
    "source-ip": [
      {
        "cidr": "10.0.0.0/24"
      }
    ],
    "name": [
      {
        "anything-but": "test"
      }
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					true,
				},
			},
		},
		{
			pattern: `
{
  "source": [
    {
      "prefix": "aa",
      "suffix": "bb"
    },
    {
      "prefix": "cc",
      "suffix": "dd"
    },
	{}
  ]
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "aa-bb",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "cc-dd",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "aa-dd",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					false,
				},
				{
					`
{
  "id": 123,
  "source": "cc-bb",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					false,
				},
			},
		},
		// array
		{
			pattern: `
{
  "source": [
    "testSource1",
    "testSource2",
    "testSource3"
  ]
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource2",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource3",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource4",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}`,
					false,
				},
			},
		},
		// empty string and null
		{
			pattern: `
{
  "data": {
    "value1": [
      ""
    ],
    "value2": [
      null
    ]
  }
}`,
			events: []eventAndMatchRes{
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\",\"value1\":\"\",\"value2\":null}",
  "datacontenttype": "application/json"
}`,
					true,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\",\"value1\":null,\"value2\":null}",
  "datacontenttype": "application/json"
}`,
					false,
				},
				{
					`
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\",\"value1\":\"\",\"value2\":\"\"}",
  "datacontenttype": "application/json"
}`,
					false,
				},
			},
		},
	}
	for idx, pt := range patternTests {
		filterPattern := make(map[string]interface{})
		err := json.Unmarshal([]byte(pt.pattern), &filterPattern)
		if err != nil {
			t.Fatal(err)
		}
		mhr, err := NewMatcher(context.Background(), logger, filterPattern)
		if err != nil {
			t.Fatal(err)
		}
		for ei, evt := range pt.events {
			ee := &rule.EventExt{
				EventExt: &v1.EventExt{
					Event: &v1.Event{},
				},
			}
			err = protojson.Unmarshal([]byte(evt.evt), ee.Event)
			if err != nil {
				t.Fatal(err)
			}
			res, err := mhr.Pattern(context.Background(), ee)
			if err != nil {
				t.Fatal(err)
			}
			if res != evt.ok {
				t.Fatalf("case(index=%d, event_index=%d) test failure", idx, ei)
			}
		}
	}
}
