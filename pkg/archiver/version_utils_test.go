package archiver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

// Test for compareVersions.
func TestCompareVersions(t *testing.T) {
	var ctx context.Context
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
				assert.Panics(t, func() { compareVersions(ctx, test.versions, logger) })
			} else {
				assert.Equal(t, test.expected, compareVersions(ctx, test.versions, logger))
			}
		})
	}
}

// Test for getLatestVersion.
func TestGetLatestVersion(t *testing.T) {
	var ctx context.Context
	logging.CreateLogger()
	logger := logging.GetLogger()

	// Set up a temporary directory with versioned subdirectories
	tempDir := t.TempDir()
	directories := []string{
		"1.0.0", "2.3.0", "1.2.1",
	}

	for _, dir := range directories {
		err := os.Mkdir(filepath.Join(tempDir, dir), os.ModePerm)
		assert.NoError(t, err, "Failed to create test directory")
	}

	t.Run("Valid directory with versions", func(t *testing.T) {
		latestVersion, err := getLatestVersion(ctx, tempDir, logger)
		assert.NoError(t, err, "Expected no error")
		assert.Equal(t, "2.3.0", latestVersion, "Expected latest version")
	})

	t.Run("Empty directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		latestVersion, err := getLatestVersion(ctx, emptyDir, logger)
		assert.Error(t, err, "Expected error for no versions found")
		assert.Equal(t, "", latestVersion, "Expected empty latest version")
	})

	t.Run("Invalid directory path", func(t *testing.T) {
		latestVersion, err := getLatestVersion(ctx, "/invalid/path", logger)
		assert.Error(t, err, "Expected error for invalid path")
		assert.Equal(t, "", latestVersion, "Expected empty latest version")
	})
}
