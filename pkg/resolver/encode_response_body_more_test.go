package resolver

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklHTTP "github.com/kdeps/schema/gen/http"
	"github.com/spf13/afero"
)

func newMemResolver() *DependencyResolver {
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/files", 0o755) // nolint:errcheck
	return &DependencyResolver{
		Fs:        fs,
		FilesDir:  "/files",
		ActionDir: "/action",
		RequestID: "req1",
		Context:   context.Background(),
		Logger:    logging.NewTestLogger(),
	}
}

func TestEncodeResponseHeadersAndBody(t *testing.T) {
	dr := newMemResolver()

	body := "hello"
	hdrs := map[string]string{"X-Test": "val"}
	resp := &pklHTTP.ResponseBlock{
		Headers: &hdrs,
		Body:    &body,
	}

	// Test headers
	headersStr := encodeResponseHeaders(resp)
	if !strings.Contains(headersStr, "X-Test") {
		t.Fatalf("expected header name in output, got %s", headersStr)
	}

	// Test body encoding & file writing
	bodyStr := encodeResponseBody(resp, dr, "res1")
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	if !strings.Contains(bodyStr, encoded) {
		t.Fatalf("expected encoded body in output, got %s", bodyStr)
	}
	// The file should be created with decoded content
	files, _ := afero.ReadDir(dr.Fs, dr.FilesDir)
	if len(files) == 0 {
		t.Fatalf("expected file to be written in %s", dr.FilesDir)
	}
	content, _ := afero.ReadFile(dr.Fs, dr.FilesDir+"/"+files[0].Name())
	if string(content) != body {
		t.Fatalf("expected file content %q, got %q", body, string(content))
	}
}
