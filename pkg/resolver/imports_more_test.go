package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestPrepareImportFilesAndPrependDynamicImports(t *testing.T) {
	fs := afero.NewMemMapFs()

	actionDir := "/action"
	requestID := "abc123"
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// create resolver
	dr := &DependencyResolver{
		Fs:             fs,
		ActionDir:      actionDir,
		RequestID:      requestID,
		RequestPklFile: filepath.Join(actionDir, "api", requestID+"__request.pkl"),
		Logger:         logger,
		Context:        ctx,
	}

	// call PrepareImportFiles – should create multiple skeleton files
	assert.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "api"), 0o755))
	assert.NoError(t, dr.PrepareImportFiles())

	// check a couple of expected files exist
	expected := []string{
		filepath.Join(actionDir, "exec", requestID+"__exec_output.pkl"),
		filepath.Join(actionDir, "data", requestID+"__data_output.pkl"),
	}
	for _, f := range expected {
		exists, _ := afero.Exists(fs, f)
		assert.True(t, exists, f)
	}

	// create a dummy .pkl file with minimal content
	wfDir := "/workflow"
	assert.NoError(t, fs.MkdirAll(wfDir, 0o755))
	pklPath := filepath.Join(wfDir, "workflow.pkl")
	content := "amends \"base\"\n" // just an amends line
	assert.NoError(t, afero.WriteFile(fs, pklPath, []byte(content), 0o644))

	// run PrependDynamicImports
	assert.NoError(t, dr.PrependDynamicImports(pklPath))

	// the updated file should now contain some import lines (e.g., pkl:json)
	updated, err := afero.ReadFile(fs, pklPath)
	assert.NoError(t, err)
	assert.Contains(t, string(updated), "import \"pkl:json\"")
}
