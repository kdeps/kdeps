package docker

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestDecodeResponseContent_Success(t *testing.T) {
	logger := logging.NewTestLogger()

	// Prepare an APIResponse JSON with base64-encoded JSON payload in data.
	inner := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(inner))

	raw := APIResponse{
		Success: true,
		Response: ResponseData{
			Data: []string{encoded},
		},
		Meta: ResponseMeta{
			RequestID: "abc",
		},
	}

	rawBytes, err := json.Marshal(raw)
	assert.NoError(t, err)

	decoded, err := decodeResponseContent(rawBytes, logger)
	assert.NoError(t, err)
	assert.Equal(t, "abc", decoded.Meta.RequestID)
	assert.Contains(t, decoded.Response.Data[0], "\"hello\": \"world\"")
}

func TestDecodeResponseContent_InvalidJSON(t *testing.T) {
	logger := logging.NewTestLogger()
	_, err := decodeResponseContent([]byte(`not-json`), logger)
	assert.Error(t, err)
}

func TestFormatResponseJSONPretty(t *testing.T) {
	// Create a response that will be decodable by formatResponseJSON
	inner := map[string]string{"foo": "bar"}
	innerBytes, _ := json.Marshal(inner)

	resp := map[string]interface{}{
		"response": map[string]interface{}{
			"data": []interface{}{string(innerBytes)},
		},
	}
	bytesIn, _ := json.Marshal(resp)

	pretty := formatResponseJSON(bytesIn)

	// The formatted JSON should contain nested object without quotes around keys
	assert.Contains(t, string(pretty), "\"foo\": \"bar\"")
}
