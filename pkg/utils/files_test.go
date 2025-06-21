package utils_test

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// Note: All test functions have been removed from this file to avoid duplicate declarations. Tests are now in files_close_error_test.go.

func TestCreateDirectories(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	dirs := []string{}
	tmpDir, err := afero.TempDir(fs, "", "testdirs")
	assert.NoError(t, err)
	defer fs.RemoveAll(tmpDir)

	for i := 0; i < 3; i++ {
		dirs = append(dirs, filepath.Join(tmpDir, "dir", strconv.Itoa(i)))
	}
	// Should create all directories
	assert.NoError(t, utils.CreateDirectories(fs, ctx, dirs))
	for _, d := range dirs {
		exists, err := afero.DirExists(fs, d)
		assert.NoError(t, err)
		assert.True(t, exists)
	}
}

func TestGenerateResourceIDFilename_Unique(t *testing.T) {
	cases := []struct {
		input     string
		requestID string
		expected  string
	}{
		{"foo@bar/baz:qux", "req-", "req-foo_bar_baz_qux"},
		{"/leading/slash", "id-", "id-_leading_slash"},
		{"no_specials", "", "no_specials"},
	}
	for _, c := range cases {
		result := utils.GenerateResourceIDFilename(c.input, c.requestID)
		if result != c.expected {
			t.Errorf("input=%q, requestID=%q: got %q, want %q", c.input, c.requestID, result, c.expected)
		}
	}
}

func TestSanitizeArchivePath_Unique(t *testing.T) {
	tmpDir := t.TempDir()
	good, err := utils.SanitizeArchivePath(tmpDir, "file.txt")
	if err != nil {
		t.Fatalf("expected no error for good path, got %v", err)
	}
	if good == "" {
		t.Fatalf("expected non-empty path for good path")
	}

	// Should error if target is outside base dir
	_, err = utils.SanitizeArchivePath("/tmp", "../../etc/passwd")
	if err == nil {
		t.Fatalf("expected error for tainted path, got nil")
	}
}

func TestCreateFiles_Unique(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	tmpDir, err := afero.TempDir(fs, "", "testfiles")
	if err != nil {
		t.Fatalf("TempDir error: %v", err)
	}
	defer fs.RemoveAll(tmpDir)

	// Create parent directory for b/b.txt
	bDir := filepath.Join(tmpDir, "b")
	if err := fs.MkdirAll(bDir, 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	files := []string{
		filepath.Join(tmpDir, "a.txt"),
		filepath.Join(bDir, "b.txt"),
	}
	if err := utils.CreateFiles(fs, ctx, files); err != nil {
		t.Fatalf("CreateFiles error: %v", err)
	}
	for _, f := range files {
		exists, err := afero.Exists(fs, f)
		if err != nil || !exists {
			t.Errorf("file %s not created", f)
		}
	}
}
