package biz

import (
	"github.com/xeipuuv/gojsonschema"
)

func EventSchemaSyntaxCheck(spec *string) error {
	if spec == nil {
		return nil
	}
	jsl := gojsonschema.NewStringLoader(*spec)
	_, err := gojsonschema.NewSchema(jsl)
	if err != nil {
		return err
	}
	return nil
}
