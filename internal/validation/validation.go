package validation

import (
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
)

func ValidateFile(path string) (bool, error) {
	loader := openapi3.NewLoader()
	_, err := loader.LoadFromFile(path)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ValidateURL(rawURL string) (bool, error) {
	loader := openapi3.NewLoader()
	u, err := url.Parse(rawURL)
	if err != nil {
		return false, err
	}
	_, err = loader.LoadFromURI(u)
	if err != nil {
		return false, err
	}
	return true, nil
}
