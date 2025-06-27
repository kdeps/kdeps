package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsJSONNew(t *testing.T) {
	assert.True(t, IsJSON(`{"key": "value"}`))
	assert.False(t, IsJSON(`{invalid json`))
	assert.False(t, IsJSON(``))
	assert.False(t, IsJSON(`hello world`))
}

func TestFixJSONNew(t *testing.T) {
	assert.Equal(t, `{"key": "value"}`, FixJSON(`{"key": "value"}`))
	assert.Equal(t, ``, FixJSON(``))
}

func TestIsBase64EncodedNew(t *testing.T) {
	assert.True(t, IsBase64Encoded("SGVsbG8="))
	assert.False(t, IsBase64Encoded("Hello World!"))
	assert.False(t, IsBase64Encoded(""))
}

func TestEncodeBase64StringNew(t *testing.T) {
	assert.Equal(t, "SGVsbG8=", EncodeBase64String("Hello"))
	assert.Equal(t, "", EncodeBase64String(""))
}

func TestEncodeValueNew(t *testing.T) {
	assert.Equal(t, "aGVsbG8=", EncodeValue("hello"))
	assert.Equal(t, "", EncodeValue(""))
}
