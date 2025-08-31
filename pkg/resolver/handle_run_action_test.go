package resolver

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	pklResource "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// TestHandleRunAction_BasicFlow simulates a minimal happy-path execution where
// all heavy dependencies are stubbed via the injectable helpers. It asserts
// that the injected helpers are invoked and that no error is returned.
func TestHandleRunAction_BasicFlow(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

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
		SessionDBPath:  "/tmp/session.db",
		ItemDBPath:     "/tmp/item.db",
		MemoryReader:   &memory.PklResourceReader{DB: openDB()},
		SessionReader:  &session.PklResourceReader{DB: openDB()},
		ToolReader:     &tool.PklResourceReader{DB: openDB()},
		ItemReader:     &item.PklResourceReader{DB: openDB()},
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
		return &pklResource.Resource{ActionID: "act1"}, nil // Run is nil
	}

	var prbCalled bool
	dr.ProcessRunBlockFn = func(res ResourceNodeEntry, rsc *pklResource.Resource, actionID string, hasItems bool) (bool, error) {
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
