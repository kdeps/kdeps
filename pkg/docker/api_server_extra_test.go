package docker

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestValidateMethodExtra(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "http://example.com", nil)
	allowed := []string{http.MethodGet, http.MethodPost}
	out, err := validateMethod(r, allowed)
	require.NoError(t, err)
	require.Equal(t, `method = "POST"`, out)

	// Disallowed method
	r.Method = http.MethodDelete
	_, err = validateMethod(r, allowed)
	require.Error(t, err)
}

func TestFormatResponseJSONExtra(t *testing.T) {
	// Prepare response with data that is itself JSON string
	inner := map[string]any{"foo": "bar"}
	innerBytes, _ := json.Marshal(inner)
	resp := map[string]any{
		"response": map[string]any{
			"data": []string{string(innerBytes)},
		},
	}
	raw, _ := json.Marshal(resp)
	pretty := formatResponseJSON(raw)

	// It should now be pretty-printed and contain nested object without quotes
	require.Contains(t, string(pretty), "\"foo\": \"bar\"")
}

func TestCleanOldFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &resolver.DependencyResolver{Fs: fs, Logger: logging.NewTestLogger(), ResponseTargetFile: "old.json"}

	// Case where file exists
	require.NoError(t, afero.WriteFile(fs, dr.ResponseTargetFile, []byte("x"), 0644))
	require.NoError(t, cleanOldFiles(dr))
	exists, _ := afero.Exists(fs, dr.ResponseTargetFile)
	require.False(t, exists)

	// Case where file does not exist should be no-op
	require.NoError(t, cleanOldFiles(dr))
}

func TestDecodeResponseContentExtra(t *testing.T) {
	// Prepare APIResponse JSON with Base64 encoded data
	dataJSON := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(dataJSON))
	respStruct := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{encoded}},
	}
	raw, _ := json.Marshal(respStruct)

	logger := logging.NewTestLogger()
	out, err := decodeResponseContent(raw, logger)
	require.NoError(t, err)
	require.Len(t, out.Response.Data, 1)
	require.JSONEq(t, dataJSON, out.Response.Data[0])
}
