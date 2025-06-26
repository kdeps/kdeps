package utils_test

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestCreateKdepsTempDir tests the CreateKdepsTempDir function
func TestCreateKdepsTempDir(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		suffix    string
		expectErr bool
	}{
		{
			name:      "Valid request ID with suffix",
			requestID: "test-request-123",
			suffix:    "pkl-eval",
			expectErr: false,
		},
		{
			name:      "Valid request ID without suffix",
			requestID: "test-request-456",
			suffix:    "",
			expectErr: false,
		},
		{
			name:      "Empty request ID should fail",
			requestID: "",
			suffix:    "pkl-eval",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			dir, err := utils.CreateKdepsTempDir(fs, tt.requestID, tt.suffix)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Empty(t, dir)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, dir)

			// Verify directory was created
			exists, err := afero.DirExists(fs, dir)
			assert.NoError(t, err)
			assert.True(t, exists)

			// Verify directory structure contains request ID
			assert.Contains(t, dir, tt.requestID)

			// If suffix provided, verify it's in the path
			if tt.suffix != "" {
				assert.Contains(t, dir, tt.suffix)
			}
		})
	}
}

// TestCreateKdepsTempFile tests the CreateKdepsTempFile function
func TestCreateKdepsTempFile(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		pattern   string
		expectErr bool
	}{
		{
			name:      "Valid request ID with pattern",
			requestID: "test-request-789",
			pattern:   "test-*.pkl",
			expectErr: false,
		},
		{
			name:      "Valid request ID with empty pattern",
			requestID: "test-request-abc",
			pattern:   "",
			expectErr: false,
		},
		{
			name:      "Empty request ID should fail",
			requestID: "",
			pattern:   "test-*.pkl",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			file, err := utils.CreateKdepsTempFile(fs, tt.requestID, tt.pattern)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, file)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, file)

			// Get file name and verify it exists
			fileName := file.Name()
			assert.NotEmpty(t, fileName)

			// Verify file was created
			exists, err := afero.Exists(fs, fileName)
			assert.NoError(t, err)
			assert.True(t, exists)

			// Verify file path contains request ID
			assert.Contains(t, fileName, tt.requestID)

			// If pattern provided with extension, verify it's preserved
			if strings.Contains(tt.pattern, ".") {
				ext := filepath.Ext(tt.pattern)
				if ext != "" {
					assert.True(t, strings.HasSuffix(fileName, ext))
				}
			}

			// Close the file
			file.Close()
		})
	}
}

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
