package enforcer

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

func TestEnforcePklVersionComparisons(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()
	ver := schema.SchemaVersion(ctx)

	lineSame := "amends \"package://schema.kdeps.com/core@" + ver + "#/Workflow.pkl\""
	if err := EnforcePklVersion(ctx, lineSame, "file.pkl", ver, logger); err != nil {
		t.Fatalf("unexpected error for same version: %v", err)
	}

	lower := "0.0.1"
	lineLower := "amends \"package://schema.kdeps.com/core@" + lower + "#/Workflow.pkl\""
	if err := EnforcePklVersion(ctx, lineLower, "file.pkl", ver, logger); err != nil {
		t.Fatalf("unexpected error for lower version: %v", err)
	}

	higher := "999.999.999"
	lineHigher := "amends \"package://schema.kdeps.com/core@" + higher + "#/Workflow.pkl\""
	if err := EnforcePklVersion(ctx, lineHigher, "file.pkl", ver, logger); err != nil {
		t.Fatalf("unexpected error for higher version: %v", err)
	}

	bad := "amends \"package://schema.kdeps.com/core#/Workflow.pkl\"" // missing @version
	if err := EnforcePklVersion(ctx, bad, "file.pkl", ver, logger); err == nil {
		t.Fatalf("expected error for malformed line")
	}
}

func TestEnforceResourceRunBlock(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	fileOne := filepath.Join(dir, "single.pkl")
	contentSingle := "chat {\n}" // one run block
	_ = afero.WriteFile(fs, fileOne, []byte(contentSingle), 0o644)

	if err := EnforceResourceRunBlock(fs, context.Background(), fileOne, logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error for single run block: %v", err)
	}

	fileMulti := filepath.Join(dir, "multi.pkl")
	contentMulti := "chat {\n}\npython {\n}" // two run blocks
	_ = afero.WriteFile(fs, fileMulti, []byte(contentMulti), 0o644)

	if err := EnforceResourceRunBlock(fs, context.Background(), fileMulti, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for multiple run blocks, got nil")
	}
}
