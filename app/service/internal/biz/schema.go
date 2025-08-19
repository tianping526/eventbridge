package biz

import (
	"github.com/xeipuuv/gojsonschema"
)

func EventSchemaSyntaxCheck(spec []byte) error {
	if spec == nil {
		return nil
	}
	jsl := gojsonschema.NewBytesLoader(spec)
	_, err := gojsonschema.NewSchema(jsl)
	if err != nil {
		return err
	}
	return nil
}
