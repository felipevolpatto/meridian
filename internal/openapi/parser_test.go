package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFile(t *testing.T) {
	t.Run("ParseFile_ValidSpec", func(t *testing.T) {
		spec, err := ParseFile("../../docs/openapi.yaml")
		assert.NoError(t, err)
		assert.NotNil(t, spec)
		assert.Equal(t, "Simple API", spec.Info.Title)
		assert.Equal(t, "1.0.0", spec.Info.Version)
	})

	t.Run("ParseFile_InvalidPath", func(t *testing.T) {
		spec, err := ParseFile("nonexistent.yaml")
		assert.Error(t, err)
		assert.Nil(t, spec)
	})
}
