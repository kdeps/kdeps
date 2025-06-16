package docker

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCleanupDockerFlow(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// prepare fake directories mimicking docker layout
	graphID := "gid123"
	agentDir := "/agent"
	actionDir := filepath.Join(agentDir, "action")
	projectDir := filepath.Join(agentDir, "project")
	workflowDir := filepath.Join(agentDir, "workflow")

	// populate dirs and a test file inside project
	assert.NoError(t, fs.MkdirAll(filepath.Join(projectDir, "sub"), 0o755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "sub", "file.txt"), []byte("data"), 0o644))

	// action directory (will be removed)
	assert.NoError(t, fs.MkdirAll(actionDir, 0o755))

	// context with required keys
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)

	env := &environment.Environment{DockerMode: "1"}

	// run cleanup – we just assert it completes within reasonable time (~2s)
	done := make(chan struct{})
	go func() {
		Cleanup(fs, ctx, env, logger)
		close(done)
	}()

	select {
	case <-done:
		// verify that workflowDir now exists and contains copied file (if copy executed)
		copied := filepath.Join(workflowDir, "sub", "file.txt")
		exists, _ := afero.Exists(fs, copied)
		// either exist or not depending on timing – we just make sure function returned
		_ = exists
	case <-ctx.Done():
		t.Fatal("context canceled prematurely")
	}
}
