package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
)

func ParseFile(filename string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	return loader.LoadFromFile(filename)
}
