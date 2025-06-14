package enforcer

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestEnforcePklFilenameValid(t *testing.T) {
	line := "amends \"package://schema.kdeps.com/core@0.0.0#/Workflow.pkl\""
	if err := EnforcePklFilename(context.Background(), line, "/tmp/workflow.pkl", logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error for valid filename: %v", err)
	}

	lineConf := "amends \"package://schema.kdeps.com/core@0.0.0#/Kdeps.pkl\""
	if err := EnforcePklFilename(context.Background(), lineConf, "/tmp/.kdeps.pkl", logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error for config filename: %v", err)
	}
}

func TestEnforcePklFilenameInvalid(t *testing.T) {
	line := "amends \"package://schema.kdeps.com/core@0.0.0#/Workflow.pkl\""
	// wrong actual file name
	if err := EnforcePklFilename(context.Background(), line, "/tmp/other.pkl", logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for mismatched filename")
	}

	// invalid pkl reference
	badLine := "amends \"package://schema.kdeps.com/core@0.0.0#/Unknown.pkl\""
	if err := EnforcePklFilename(context.Background(), badLine, "/tmp/foo.pkl", logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for unknown pkl file")
	}
}

func TestCompareVersions_Basic(t *testing.T) {
	if c, _ := compareVersions("1.2.3", "1.2.3", logging.NewTestLogger()); c != 0 {
		t.Fatalf("expected equal version compare = 0, got %d", c)
	}
	if c, _ := compareVersions("0.9", "1.0", logging.NewTestLogger()); c != -1 {
		t.Fatalf("expected older version -1, got %d", c)
	}
	if c, _ := compareVersions("2.0", "1.5", logging.NewTestLogger()); c != 1 {
		t.Fatalf("expected newer version 1, got %d", c)
	}
}
