package archiver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCompareVersionsAndGetLatest(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("compareVersions", func(t *testing.T) {
		versions := []string{"1.0.0", "2.3.4", "2.10.0", "0.9.9"}
		latest := compareVersions(versions, logger)
		assert.Equal(t, "2.3.4", latest)
	})

	t.Run("GetLatestVersion", func(t *testing.T) {
		fs := afero.NewOsFs()
		tmpDir := t.TempDir()
		logger := logging.NewTestLogger()

		// create version dirs
		for _, v := range []string{"0.1.0", "1.2.3", "1.2.10"} {
			assert.NoError(t, fs.MkdirAll(filepath.Join(tmpDir, v), 0o755))
		}
		latest, err := GetLatestVersion(tmpDir, logger)
		assert.NoError(t, err)
		assert.Equal(t, "1.2.3", latest)

		emptyDir := filepath.Join(tmpDir, "empty")
		assert.NoError(t, fs.MkdirAll(emptyDir, 0o755))
		_, err = GetLatestVersion(emptyDir, logger)
		assert.Error(t, err)
	})
}

// Test for compareVersions.
func TestCompareVersions(t *testing.T) {
	logging.CreateLogger()
	logger := logging.GetLogger()

	tests := []struct {
		name        string
		versions    []string
		expected    string
		expectPanic bool
	}{
		{"Ascending order", []string{"1.0.0", "1.2.0", "1.1.1"}, "1.2.0", false},
		{"Descending order", []string{"2.0.0", "1.5.0", "1.0.0"}, "2.0.0", false},
		{"Same versions", []string{"1.0.0", "1.0.0", "1.0.0"}, "1.0.0", false},
		{"Empty slice", []string{}, "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectPanic {
				assert.Panics(t, func() { compareVersions(test.versions, logger) })
			} else {
				assert.Equal(t, test.expected, compareVersions(test.versions, logger))
			}
		})
	}
}

// Test for GetLatestVersion.
func TestGetLatestVersion(t *testing.T) {
	logging.CreateLogger()
	logger := logging.GetLogger()

	// Set up a temporary directory with versioned subdirectories
	tempDir := t.TempDir()
	directories := []string{
		"1.0.0", "2.3.0", "1.2.1",
	}

	for _, dir := range directories {
		err := os.Mkdir(filepath.Join(tempDir, dir), os.ModePerm)
		require.NoError(t, err, "Failed to create test directory")
	}

	t.Run("Valid directory with versions", func(t *testing.T) {
		latestVersion, err := GetLatestVersion(tempDir, logger)
		require.NoError(t, err, "Expected no error")
		assert.Equal(t, "2.3.0", latestVersion, "Expected latest version")
	})

	t.Run("Empty directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		latestVersion, err := GetLatestVersion(emptyDir, logger)
		require.Error(t, err, "Expected error for no versions found")
		assert.Equal(t, "", latestVersion, "Expected empty latest version")
	})

	t.Run("Invalid directory path", func(t *testing.T) {
		latestVersion, err := GetLatestVersion("/invalid/path", logger)
		require.Error(t, err, "Expected error for invalid path")
		assert.Equal(t, "", latestVersion, "Expected empty latest version")
	})
}
