package transform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tianping526/eventbridge/app/internal/rule"
)

func init() {
	registerTransformFunc("TEMPLATE", newTransformFuncTemplate)
}

func newTransformFuncTemplate(
	ctx context.Context,
	logger *log.Helper,
	value string,
	tmpl *string,
) (transformFunc, error) {
	if tmpl == nil || *tmpl == "" {
		return func(_ context.Context, _ *rule.EventExt) (interface{}, error) {
			return nil, nil
		}, nil
	}

	fetcher := make(map[string]transformFunc)
	if value != "" {
		values := make(map[string]interface{})
		err := json.Unmarshal([]byte(value), &values)
		if err != nil {
			return nil, err
		}
		for key, val := range values {
			trimmedKey := strings.TrimSpace(key)
			jsonpath, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf(
					"transformer(TEMPLATE) value.%s(value=%+v, type=%T) should be string",
					trimmedKey, val, val,
				)
			}
			var fc transformFunc
			fc, err = newTransformFuncJsonpath(ctx, logger, jsonpath, nil)
			if err != nil {
				return nil, err
			}
			fetcher[trimmedKey] = fc
		}
	}

	tmplParsed := make([]interface{}, 0)
	// help the declaration of nested JSON pass the check.
	// the format of JSON should be {"a": "$.data.name"},
	// but the nested JSON declaration is {"a": $.data.name}
	tmplCanCheck := strings.Builder{}
	start := 0
	for end := 0; end < len(*tmpl); end++ {
		if (*tmpl)[end] == '$' && (*tmpl)[end+1] == '{' { // var
			if end > start {
				subStr := (*tmpl)[start:end]
				tmplParsed = append(tmplParsed, subStr)
				tmplCanCheck.WriteString(subStr)
			}
			varEndIdx := strings.IndexRune((*tmpl)[end:], '}')
			if varEndIdx == -1 {
				return nil, errors.New("template variables that start with ${ must have an } at the end")
			}
			varStr := (*tmpl)[end : end+varEndIdx]
			varName := strings.TrimSpace(
				strings.TrimPrefix(varStr, "${"),
			)
			fc, ok := fetcher[varName]
			if !ok {
				return nil, fmt.Errorf("template variable(key=%s) not found", varName)
			}
			tmplParsed = append(tmplParsed, fc)
			if (*tmpl)[end-1] != '"' {
				// use 1 to replace variable description that may not conform to the JSON format
				// to avoid JSON format check failures
				tmplCanCheck.WriteString("1")
			} else {
				tmplCanCheck.WriteString(varStr)
				tmplCanCheck.WriteString("}")
			}
			end += varEndIdx
			start = end + 1
		}
	}
	if len(*tmpl) > start {
		varStr := (*tmpl)[start:len(*tmpl)]
		tmplParsed = append(tmplParsed, varStr)
		tmplCanCheck.WriteString(varStr)
	}
	var checkSyntax interface{}
	err := json.Unmarshal([]byte(tmplCanCheck.String()), &checkSyntax)
	if err != nil {
		return nil, fmt.Errorf("template syntax err: %s", err)
	}
	return func(c context.Context, ext *rule.EventExt) (interface{}, error) {
		bld := strings.Builder{}
		for _, val := range tmplParsed {
			fc, ok := val.(transformFunc)
			if !ok {
				bld.WriteString(val.(string))
				continue
			}
			res, err1 := fc(c, ext)
			if err1 != nil {
				return nil, err1
			}
			strRes, ok := res.(string)
			if !ok {
				var byteRes []byte
				byteRes, err1 = json.Marshal(res)
				if err1 != nil {
					return nil, err1
				}
				bld.Write(byteRes)
			} else {
				bld.WriteString(strRes)
			}
		}
		var res interface{}
		err1 := json.Unmarshal([]byte(bld.String()), &res)
		if err1 != nil {
			return nil, err1
		}
		return res, nil
	}, nil
}
