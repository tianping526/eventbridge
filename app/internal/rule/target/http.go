package target

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/xeipuuv/gojsonschema"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

func init() {
	registerDispatcher(
		"HTTPDispatcher",
		newHTTPDispatcher,
		`
		{
		  "$schema": "https://json-schema.org/draft/2020-12/schema",
		  "title": "HTTP dispatcher",
		  "description": "The data format of the HTTP dispatcher",
		  "type": "object",
		  "properties": {
			"method": {
			  "description": "method specifies the HTTP method (GET, POST, PUT, etc.)",
			  "type": "string"
			},
			"url": {
			  "description": "url specifies the URL to access",
			  "type": "string"
			},
			"header": {
			  "description": "header contains the request header fields",
			  "type": "object",
			  "patternProperties": {
				".*": {
				  "type": "string"
				}
			  }
			},
			"body": {
			  "description": "body is the request's body",
			  "type": "object"
			}
		  },
		  "required": [
			"method",
			"url"
		  ]
		}`)
}

type httpDispatcher struct {
	log       *log.Helper
	client    atomic.Value
	validator *gojsonschema.Schema
	// Only one check is required per dispatcher
	validated int32
}

func newHTTPDispatcher(
	_ context.Context,
	logger log.Logger,
	_ *rule.Target,
	validator *gojsonschema.Schema,
) (rule.Dispatcher, error) {
	return &httpDispatcher{
		log: log.NewHelper(log.With(
			logger,
			"module", "target/httpDispatcher",
			"caller", log.DefaultCaller,
		)),
		validator: validator,
	}, nil
}

func (d *httpDispatcher) Dispatch(ctx context.Context, event *rule.EventExt) (err error) {
	// validate
	val := atomic.LoadInt32(&d.validated)
	if val == 0 {
		var result *gojsonschema.Result
		result, err = d.validator.Validate(gojsonschema.NewStringLoader(event.Event.Data))
		if err != nil {
			return err
		}
		if !result.Valid() {
			return fmt.Errorf(
				"http dispatcher target event data is not valid. see err: %s",
				result.Errors(),
			)
		}
	}
	atomic.AddInt32(&d.validated, 1)

	// fetch params
	jsonData := make(map[string]interface{})
	_ = json.Unmarshal([]byte(event.Event.Data), &jsonData)
	method := jsonData["method"].(string)
	url := jsonData["url"].(string)
	var body io.Reader
	bodyData, ok := jsonData["body"]
	if ok {
		marshalBody, _ := json.Marshal(bodyData)
		body = bytes.NewReader(marshalBody)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	headerData, ok := jsonData["header"]
	if ok {
		header := headerData.(map[string]string)
		for key, val := range header {
			req.Header.Set(key, val)
		}
	}

	// get client
	clientVal := d.client.Load()
	var client *http.Client
	if clientVal == nil {
		client = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 30 * time.Second,
			},
			Timeout: 0,
		}
		d.client.Store(client)
	} else {
		client = clientVal.(*http.Client)
	}

	// call http request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		err = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		var rb []byte
		rb, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf(
				"response status code: %d, read body err: %s",
				resp.StatusCode, err,
			)
		}
		return fmt.Errorf(
			"response status code: %d, body: %s",
			resp.StatusCode, rb,
		)
	}
	return nil
}

func (d *httpDispatcher) Close() error {
	return nil
}
