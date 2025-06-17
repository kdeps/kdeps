package archiver

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestCompareVersionsOrdering(t *testing.T) {
	versions := []string{"1.2.3", "2.0.0", "1.10.1"}
	latest := compareVersions(versions, logging.NewTestLogger())
	if latest != "2.0.0" {
		t.Fatalf("expected latest 2.0.0 got %s", latest)
	}

	// already sorted descending should keep first element
	versions2 := []string{"3.1.0", "2.9.9", "0.0.1"}
	if got := compareVersions(versions2, logging.NewTestLogger()); got != "3.1.0" {
		t.Fatalf("unexpected latest %s", got)
	}
} 