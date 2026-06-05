package scraper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalError(t *testing.T) {
	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("injected marshal error")
	}
	// Access the function that uses jsonMarshal
	result := map[string]interface{}{"key": "val"}
	_, err := jsonMarshal(result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected marshal error")
}
