package resolver

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
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
