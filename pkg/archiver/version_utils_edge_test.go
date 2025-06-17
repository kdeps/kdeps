package archiver

import (
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestGetLatestVersionEdge(t *testing.T) {
	tmpDir := t.TempDir()

	// create version directories
	versions := []string{"1.0.0", "2.0.1", "0.9.9"}
	for _, v := range versions {
		if err := os.MkdirAll(tmpDir+"/"+v, 0o755); err != nil {
			t.Fatalf("failed mkdir: %v", err)
		}
	}

	logger := logging.NewTestLogger()
	latest, err := GetLatestVersion(tmpDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest != "2.0.1" {
		t.Fatalf("expected latest 2.0.1 got %s", latest)
	}
}

func TestGetLatestVersionNoVersions(t *testing.T) {
	dir := t.TempDir()
	logger := logging.NewTestLogger()
	if _, err := GetLatestVersion(dir, logger); err == nil {
		t.Fatalf("expected error when no versions present")
	}
}
