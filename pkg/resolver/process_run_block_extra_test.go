package resolver

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklRes "github.com/kdeps/schema/gen/resource"
)

// TestProcessRunBlock_NoRunBlock verifies that when Run is nil the function returns without error
// but still increments the FileRunCounter.
func TestProcessRunBlock_NoRunBlock(t *testing.T) {
	dr := &DependencyResolver{
		Logger:         logging.NewTestLogger(),
		FileRunCounter: make(map[string]int),
		APIServerMode:  false,
	}

	resEntry := ResourceNodeEntry{ActionID: "act1", File: "foo.pkl"}
	rsc := &pklRes.Resource{} // Run is nil by default

	proceed, err := dr.processRunBlock(resEntry, rsc, "act1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proceed {
		t.Fatalf("expected proceed=false when Run is nil, got true")
	}
	if count := dr.FileRunCounter[resEntry.File]; count != 1 {
		t.Fatalf("expected FileRunCounter for %s to be 1, got %d", resEntry.File, count)
	}
}
