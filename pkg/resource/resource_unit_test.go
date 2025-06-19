package resource_test

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

// TestLoadResource_FileNotFound verifies that LoadResource returns an error when
// provided with a non-existent file path. This exercises the error branch to
// ensure we log and wrap the underlying failure correctly.
func TestLoadResource_FileNotFound(t *testing.T) {
	_, err := LoadResource(context.Background(), "/path/to/nowhere/nonexistent.pkl", logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected error when reading missing resource file")
	}
}
