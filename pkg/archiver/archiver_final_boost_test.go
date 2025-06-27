package archiver_test

import (
	"testing"

	. "github.com/kdeps/kdeps/pkg/archiver"
	"github.com/stretchr/testify/assert"
)

func TestArchiverUtilityFunctions(t *testing.T) {
	// Test simple utility functions that might have 0% coverage

	t.Run("GetBackupPath", func(t *testing.T) {
		// Test GetBackupPath function
		tests := []struct {
			dst      string
			md5      string
			expected string
		}{
			{
				dst:      "/path/to/file.txt",
				md5:      "abcdef123456",
				expected: "/path/to/file_abcdef123456.txt",
			},
			{
				dst:      "/path/to/file",
				md5:      "123456",
				expected: "/path/to/file_123456",
			},
			{
				dst:      "file.conf",
				md5:      "xyz789",
				expected: "file_xyz789.conf",
			},
			{
				dst:      "/deep/nested/path/document.pdf",
				md5:      "hash123",
				expected: "/deep/nested/path/document_hash123.pdf",
			},
			{
				dst:      "noextension",
				md5:      "test",
				expected: "noextension_test",
			},
		}

		for _, test := range tests {
			t.Run(test.dst+"_"+test.md5, func(t *testing.T) {
				result := GetBackupPath(test.dst, test.md5)
				assert.Equal(t, test.expected, result)
			})
		}
	})

	t.Run("GetBackupPath_EdgeCases", func(t *testing.T) {
		// Test edge cases for GetBackupPath

		// Empty MD5
		result := GetBackupPath("/path/file.txt", "")
		expected := "/path/file_.txt"
		assert.Equal(t, expected, result)

		// Empty file path - filepath.Base("") returns "." and filepath.Dir("") returns "."
		result = GetBackupPath("", "hash")
		expected = "._hash"
		assert.Equal(t, expected, result)

		// Multiple dots in filename - filepath.Ext returns only the last extension
		result = GetBackupPath("/path/file.tar.gz", "hash123")
		expected = "/path/file.tar_hash123.gz"
		assert.Equal(t, expected, result)

		// Filename with just extension - filepath.Base(".bashrc") returns ".bashrc"
		result = GetBackupPath(".bashrc", "hash")
		expected = "_hash.bashrc"
		assert.Equal(t, expected, result)
	})
}
