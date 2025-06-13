//go:build skip
// +build skip

package resolver_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bouk/monkey"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	pklData "github.com/kdeps/schema/gen/data"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestAppendDataEntry_SuccessMerge(t *testing.T) {
	// Monkey-patch pklData.LoadFromPath to bypass real parsing.
	patchLoad := monkey.Patch(pklData.LoadFromPath, func(ctx context.Context, path string) (pklData.Data, error) {
		m := make(map[string]map[string]string)
		return &pklData.DataImpl{Files: &m}, nil
	})
	defer patchLoad.Unpatch()

	// Monkey-patch evaluator.EvalPkl to a no-op that just echoes back its input.
	patchEval := monkey.Patch(evaluator.EvalPkl, func(fs afero.Fs, ctx context.Context, pklPath, _ string, _ *logging.Logger) (string, error) {
		bytes, _ := afero.ReadFile(fs, pklPath)
		return string(bytes), nil
	})
	defer patchEval.Unpatch()

	fs := afero.NewOsFs()
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")
	_ = fs.MkdirAll(filepath.Join(actionDir, "data"), 0o755)

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Context:   context.Background(),
		ActionDir: actionDir,
		RequestID: "req123",
		Logger:    logging.NewTestLogger(),
	}

	// new data to merge
	files := map[string]map[string]string{
		"agentX": {
			"hello.txt": "SGVsbG8=", // already base64
		},
	}
	newData := &pklData.DataImpl{Files: &files}

	err := dr.AppendDataEntry("res1", newData)
	assert.NoError(t, err)

	// Verify file written and contains the merged agent key.
	pklPath := filepath.Join(actionDir, "data", dr.RequestID+"__data_output.pkl")
	contentBytes, readErr := afero.ReadFile(fs, pklPath)
	assert.NoError(t, readErr)
	content := string(contentBytes)
	assert.Contains(t, content, "[\"agentX\"]")
	assert.Contains(t, content, schema.SchemaVersion(dr.Context))
}
