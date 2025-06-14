package evaluator

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestEvalPkl_InvalidExtensionAlt(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	if _, err := EvalPkl(fs, ctx, "/tmp/file.txt", "header", logger); err == nil {
		t.Fatalf("expected error for non-pkl extension")
	}
}
