package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestAddPlaceholderImports_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	dr := &DependencyResolver{Fs: fs, Logger: logger}
	if err := dr.AddPlaceholderImports("/no/such/file.pkl"); err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}

func TestNewGraphResolver_Minimal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	env, err := environment.NewEnvironment(fs, nil)
	if err != nil {
		t.Fatalf("env err: %v", err)
	}

	dr, err := NewGraphResolver(fs, context.Background(), env, nil, logger)
	if err == nil {
		// If resolver succeeded, sanity-check key fields
		if dr.Graph == nil || dr.FileRunCounter == nil {
			t.Errorf("expected Graph and FileRunCounter initialized")
		}
	}
}

func TestPrepareImportFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}

	dr := &DependencyResolver{
		Fs:          fs,
		Context:     ctx,
		ActionDir:   "/tmp/action",
		RequestID:   "req1",
		Logger:      logging.NewTestLogger(),
		Environment: env,
	}

	// call function
	require.NoError(t, dr.PrepareImportFiles())

	// verify that expected stub files were created with minimal header lines
	expected := []struct{ folder, key string }{
		{"llm", "LLM.pkl"},
		{"client", "HTTP.pkl"},
		{"exec", "Exec.pkl"},
		{"python", "Python.pkl"},
		{"data", "Data.pkl"},
	}

	for _, e := range expected {
		p := filepath.Join(dr.ActionDir, e.folder, dr.RequestID+"__"+e.folder+"_output.pkl")
		exists, _ := afero.Exists(fs, p)
		require.True(t, exists, "file %s should exist", p)
		// simple read check
		b, err := afero.ReadFile(fs, p)
		require.NoError(t, err)
		require.Contains(t, string(b), e.key)
	}
}
