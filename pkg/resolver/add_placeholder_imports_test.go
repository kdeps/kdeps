package resolver_test

import (
	"context"
	"testing"

	. "github.com/kdeps/kdeps/pkg/resolver"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestAddPlaceholderImports_NoActionID(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temporary PKL file without actionID
	tmpFile, err := afero.TempFile(fs, "", "*.pkl")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_, _ = tmpFile.WriteString("# sample pkl file without id\n")
	tmpFile.Close()

	dr := &DependencyResolver{
		Fs:      fs,
		Context: ctx,
		Logger:  logger,
	}

	if err := dr.AddPlaceholderImports(tmpFile.Name()); err == nil {
		t.Fatalf("expected error for missing action id, got nil")
	}
}
