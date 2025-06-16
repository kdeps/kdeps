package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestAddPlaceholderImports(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	baseDir := t.TempDir()
	actionDir := filepath.Join(baseDir, "action")
	dataDir := filepath.Join(baseDir, "data")
	requestID := "req1"

	// create directories for placeholder files
	assert.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "exec"), 0o755))
	assert.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "data"), 0o755))

	// create minimal pkl file expected by AppendDataEntry
	dataPklPath := filepath.Join(actionDir, "data", requestID+"__data_output.pkl")
	minimalContent := []byte("files {}\n")
	assert.NoError(t, afero.WriteFile(fs, dataPklPath, minimalContent, 0o644))

	// create input file containing actionID
	targetPkl := filepath.Join(actionDir, "exec", "sample.pkl")
	fileContent := []byte("actionID = \"myAction\"\n")
	assert.NoError(t, afero.WriteFile(fs, targetPkl, fileContent, 0o644))

	dr := &DependencyResolver{
		Fs:        fs,
		ActionDir: actionDir,
		DataDir:   dataDir,
		RequestID: requestID,
		Context:   ctx,
		Logger:    logger,
	}

	// ensure DataDir has at least one file for PopulateDataFileRegistry
	assert.NoError(t, fs.MkdirAll(dataDir, 0o755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(dataDir, "dummy.txt"), []byte("abc"), 0o644))

	// run the function under test
	err := dr.AddPlaceholderImports(targetPkl)
	assert.Error(t, err)
}
