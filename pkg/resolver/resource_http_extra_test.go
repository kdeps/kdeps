package resolver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklHTTP "github.com/kdeps/schema/gen/http"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestEncodeResponseHelpers(t *testing.T) {
	// Setup resolver
	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{
		Fs:       fs,
		Logger:   logging.NewTestLogger(),
		Context:  context.Background(),
		FilesDir: "/files",
	}

	t.Run("EncodeHeaders_Nil", func(t *testing.T) {
		got := encodeResponseHeaders(nil)
		require.Contains(t, got, "headers {[\"\"] = \"\"}\n")
	})

	t.Run("EncodeHeaders_WithValues", func(t *testing.T) {
		hdr := map[string]string{"Content-Type": "application/json"}
		resp := &pklHTTP.ResponseBlock{Headers: &hdr}
		got := encodeResponseHeaders(resp)
		require.Contains(t, got, "Content-Type")
		// value should be base64 encoded by EncodeValue
		require.Contains(t, got, utils.EncodeValue("application/json"))
	})

	t.Run("EncodeBody_Nil", func(t *testing.T) {
		got := encodeResponseBody(nil, dr, "id")
		require.Equal(t, "    body=\"\"\n", got)
	})

	t.Run("EncodeBody_WithValue", func(t *testing.T) {
		body := "hello"
		resp := &pklHTTP.ResponseBlock{Body: &body}
		got := encodeResponseBody(resp, dr, "res1")
		require.Contains(t, got, utils.EncodeValue(body))
		// file should have been written
		files, _ := afero.ReadDir(fs, dr.FilesDir)
		require.NotEmpty(t, files)
	})
}

func TestIsMethodWithBody(t *testing.T) {
	require.True(t, isMethodWithBody("POST"))
	require.True(t, isMethodWithBody("put"))
	require.False(t, isMethodWithBody("GET"))
}
