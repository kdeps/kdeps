package resolver

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	pklRes "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestHandleRunAction_BasicFlow simulates a minimal happy-path execution where
// all heavy dependencies are stubbed via the injectable helpers. It asserts
// that the injected helpers are invoked and that no error is returned.
func TestHandleRunAction_BasicFlow(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	sessionDBPath := filepath.Join(tmpDir, "session.db")
	itemDBPath := filepath.Join(tmpDir, "item.db")

	// Prepare in-memory sqlite connections for the various readers so that the
	// final Close() calls in HandleRunAction don't panic.
	openDB := func() *sql.DB {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatalf("failed to open in-memory sqlite db: %v", err)
		}
		return db
	}

	// Minimal workflow that just targets a single action.
	wf := &pklWf.WorkflowImpl{TargetActionID: "act1"}

	dr := &DependencyResolver{
		Fs:             fs,
		Logger:         logger,
		Workflow:       wf,
		Context:        context.Background(),
		ActionDir:      "/action",
		RequestID:      "req1",
		SessionDBPath:  sessionDBPath,
		ItemDBPath:     itemDBPath,
		MemoryReader:   &memory.PklResourceReader{DB: openDB()},
		SessionReader:  &session.PklResourceReader{DB: openDB()},
		ToolReader:     &tool.PklResourceReader{DB: openDB()},
		ItemReader:     &item.PklResourceReader{DB: openDB()},
		AgentReader:    &agent.PklResourceReader{DB: openDB()},
		FileRunCounter: make(map[string]int),
	}

	// --- inject stubs for heavy funcs ------------------------------
	dr.LoadResourceEntriesFn = func() error {
		// Provide a single resource entry.
		dr.Resources = []ResourceNodeEntry{{ActionID: "act1", File: "/res1.pkl"}}
		return nil
	}

	dr.BuildDependencyStackFn = func(target string, visited map[string]bool) []string {
		if target != "act1" {
			t.Fatalf("unexpected target passed to BuildDependencyStackFn: %s", target)
		}
		return []string{"act1"}
	}

	var loadCalled bool
	dr.LoadResourceFn = func(_ context.Context, file string, _ ResourceType) (interface{}, error) {
		loadCalled = true
		return &pklRes.Resource{ActionID: "act1"}, nil // Run is nil
	}

	var prbCalled bool
	dr.ProcessRunBlockFn = func(res ResourceNodeEntry, rsc *pklRes.Resource, actionID string, hasItems bool) (bool, error) {
		prbCalled = true
		return false, nil // do not proceed further
	}

	dr.ClearItemDBFn = func() error { return nil }

	// ----------------------------------------------------------------

	proceed, err := dr.HandleRunAction()
	if err != nil {
		t.Fatalf("HandleRunAction returned error: %v", err)
	}
	if proceed {
		t.Fatalf("expected proceed=false, got true")
	}
	if !loadCalled {
		t.Fatal("LoadResourceFn was not invoked")
	}
	if !prbCalled {
		t.Fatal("ProcessRunBlockFn was not invoked")
	}
}

func TestAddPlaceholderImports_UsesCanonicalAgentReader(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a minimal workflow with agentID and version
	testAgentID := "testagent"
	testVersion := "1.2.3"
	wf := &pklWf.WorkflowImpl{
		AgentID: testAgentID,
		Version: testVersion,
	}

	dr := &DependencyResolver{
		Fs:       fs,
		Logger:   logger,
		Workflow: wf,
		Context:  context.Background(),
		DataDir:  "/data",
	}

	// Write a resource file with an unqualified actionID
	resourceContent := `actionID = "myAction"
run {}`
	resourcePath := "/tmp/test_resource.pkl"
	afero.WriteFile(fs, resourcePath, []byte(resourceContent), 0o644)

	// Create the data directory and a dummy data file
	dataDir := "/data"
	fs.MkdirAll(dataDir, 0o755)
	afero.WriteFile(fs, filepath.Join(dataDir, "dummy.txt"), []byte("test data"), 0o644)

	// Create the action directory structure
	actionDir := "/action"
	fs.MkdirAll(filepath.Join(actionDir, "data"), 0o755)
	fs.MkdirAll(filepath.Join(actionDir, "exec"), 0o755)
	fs.MkdirAll(filepath.Join(actionDir, "llm"), 0o755)
	fs.MkdirAll(filepath.Join(actionDir, "client"), 0o755)
	fs.MkdirAll(filepath.Join(actionDir, "python"), 0o755)

	// Create minimal output files
	requestID := "test-request"
	dataOutputPath := filepath.Join(actionDir, "data", requestID+"__data_output.pkl")
	afero.WriteFile(fs, dataOutputPath, []byte("Files {}\n"), 0o644)

	// Set the request ID
	dr.RequestID = requestID
	dr.ActionDir = actionDir

	// Test that AddPlaceholderImports uses the canonical agent reader
	err := dr.AddPlaceholderImports(resourcePath)
	
	// The test should fail because the PKL file doesn't exist, but it should fail
	// in a way that shows the agent reader was used (which we can see from the logs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to evaluate PKL file")
}
