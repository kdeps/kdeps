package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	pklData "github.com/kdeps/schema/gen/data"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// TestAppendDataEntry_Direct verifies the happy-path where new files are merged
// into an existing (initially empty) data.pkl file without any monkey-patching.
// It uses a real EvalPkl run, so it depends on `pkl` binary being available in PATH –
// which the project's other tests already rely on.
func TestAppendDataEntry_Direct(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")
	dataDir := filepath.Join(actionDir, "data")
	require.NoError(t, fs.MkdirAll(dataDir, 0o755))

	ctx := context.Background()
	schemaVer := schema.SchemaVersion(ctx)

	// Seed minimal valid PKL content so pklData.LoadFromPath succeeds.
	initialContent := "extends \"package://schema.kdeps.com/core@" + schemaVer + "#/Data.pkl\"\n\nfiles {}\n"
	pklPath := filepath.Join(dataDir, "req__data_output.pkl")
	require.NoError(t, afero.WriteFile(fs, pklPath, []byte(initialContent), 0o644))

	dr := &DependencyResolver{
		Fs:        fs,
		Context:   ctx,
		ActionDir: actionDir,
		RequestID: "req",
		Logger:    logging.NewTestLogger(),
	}

	// Prepare new data to merge.
	files := map[string]map[string]string{
		"agentX": {
			"hello.txt": "SGVsbG8=", // "Hello" already base64-encoded
		},
	}
	newData := &pklData.DataImpl{Files: &files}

	require.NoError(t, dr.AppendDataEntry("testResource", newData))

	// Validate merged content.
	mergedBytes, err := afero.ReadFile(fs, pklPath)
	require.NoError(t, err)
	merged := string(mergedBytes)
	require.Contains(t, merged, "[\"agentX\"]")
	require.Contains(t, merged, schemaVer)
}
