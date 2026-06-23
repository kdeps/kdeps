//go:build !js

package llm

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHFDownloadWithToken_Success(t *testing.T) {
	content := []byte("fake-gguf-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer tok123", r.Header.Get("Authorization"))
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := dir + "/model.gguf"
	origFS := AppFS
	AppFS = afero.NewOsFs()
	defer func() { AppFS = origFS }()

	err := hfDownloadWithToken(context.Background(), srv.URL, dest, "tok123")
	require.NoError(t, err)
	data, readErr := os.ReadFile(dest)
	require.NoError(t, readErr)
	assert.Equal(t, content, data)
}

func TestHFDownloadWithToken_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	origFS := AppFS
	AppFS = afero.NewOsFs()
	defer func() { AppFS = origFS }()

	err := hfDownloadWithToken(context.Background(), srv.URL, t.TempDir()+"/m.gguf", "tok")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestHFDownloadAria2c_Success(t *testing.T) {
	binDir := t.TempDir()
	fakeAria2c := binDir + "/aria2c"
	script := "#!/bin/sh\nfor a in \"$@\"; do\n  case \"$a\" in\n    --dir=*) d=\"${a#--dir=}\" ;;\n    --out=*) o=\"${a#--out=}\" ;;\n  esac\ndone\nprintf 'data' > \"$d/$o\"\n"
	require.NoError(t, os.WriteFile(fakeAria2c, []byte(script), 0o755))

	dir := t.TempDir()
	dest := dir + "/model.gguf"
	err := hfDownloadAria2c(
		context.Background(),
		fakeAria2c,
		"http://example.com/m.gguf",
		dest,
		"",
		slog.Default(),
	)
	require.NoError(t, err)
	_, statErr := os.Stat(dest)
	assert.NoError(t, statErr)
}

func TestHFDownloadAria2c_WithToken(t *testing.T) {
	binDir := t.TempDir()
	fakeAria2c := binDir + "/aria2c"
	script := "#!/bin/sh\nfor a in \"$@\"; do case \"$a\" in --dir=*) d=\"${a#--dir=}\" ;; --out=*) o=\"${a#--out=}\" ;; esac; done\nprintf 'ok' > \"$d/$o\"\n"
	require.NoError(t, os.WriteFile(fakeAria2c, []byte(script), 0o755))

	dir := t.TempDir()
	dest := dir + "/model.gguf"
	err := hfDownloadAria2c(
		context.Background(),
		fakeAria2c,
		"http://example.com/m.gguf",
		dest,
		"mytoken",
		slog.Default(),
	)
	require.NoError(t, err)
}

func TestHFDownloadAria2c_Failure(t *testing.T) {
	binDir := t.TempDir()
	fakeAria2c := binDir + "/aria2c"
	require.NoError(t, os.WriteFile(fakeAria2c, []byte("#!/bin/sh\nexit 1\n"), 0o755))

	dir := t.TempDir()
	dest := dir + "/model.gguf"
	err := hfDownloadAria2c(
		context.Background(),
		fakeAria2c,
		"http://x.com/m.gguf",
		dest,
		"",
		slog.Default(),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aria2c")
}

func TestHFDownloadFile_WithToken(t *testing.T) {
	content := []byte("model-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	t.Setenv("HF_TOKEN", "mytoken")
	dir := t.TempDir()
	dest := dir + "/m.gguf"
	origFS := AppFS
	AppFS = afero.NewOsFs()
	defer func() { AppFS = origFS }()

	err := hfDownloadFile(context.Background(), srv.URL, "m.gguf", dest, dir, slog.Default())
	require.NoError(t, err)
	data, readErr := os.ReadFile(dest)
	require.NoError(t, readErr)
	assert.Equal(t, content, data)
}

func TestHFDownloadFile_NoToken_NoAria2c(t *testing.T) {
	// No HF_TOKEN, no aria2c on PATH -> falls through to downloadModelFile (which fails).
	// This test just exercises the fallback path.
	t.Setenv("HF_TOKEN", "")
	t.Setenv("PATH", t.TempDir()) // empty PATH so aria2c is not found

	origFS := AppFS
	AppFS = afero.NewOsFs()
	defer func() { AppFS = origFS }()

	dir := t.TempDir()
	dest := dir + "/m.gguf"
	// Non-routable address ensures fast failure.
	_ = hfDownloadFile(
		context.Background(),
		"http://192.0.2.1/model.gguf",
		"m.gguf",
		dest,
		dir,
		slog.Default(),
	)
}

func TestHFSearchWithoutFilter_FiltersNonGGUF(t *testing.T) {
	all := []HFModelResult{
		{ID: "org/has-gguf", Siblings: []HFFileEntry{{Filename: "model.gguf", Size: 100}}},
		{ID: "org/no-gguf", Siblings: []HFFileEntry{{Filename: "model.bin", Size: 100}}},
		{ID: "org/empty", Siblings: nil},
	}
	// Server: returns empty for tagged request, full list for untagged.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("filter") == "gguf" {
			_ = json.NewEncoder(w).Encode([]HFModelResult{})
			return
		}
		_ = json.NewEncoder(w).Encode(all)
	}))
	defer srv.Close()

	got, err := HFSearchGGUFWithBase(context.Background(), srv.URL+"/api/models", "org", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "org/has-gguf", got[0].ID)
}
