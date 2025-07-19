package utils_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionsHelpers(t *testing.T) {
	t.Run("ShouldSkip_TrueCases", func(t *testing.T) {
		cases := [][]interface{}{
			{true, false},
			{"TRUE", false},
			{false, "TrUe"},
		}
		for _, c := range cases {
			cond := c
			assert.True(t, utils.ShouldSkip(&cond))
		}
	})

	t.Run("ShouldSkip_FalseCase", func(t *testing.T) {
		cond := []interface{}{false, "no"}
		assert.False(t, utils.ShouldSkip(&cond))
	})

	t.Run("AllConditionsMet", func(t *testing.T) {
		trueSet := []interface{}{true, "TRUE"}
		falseSet := []interface{}{true, "false"}
		assert.True(t, utils.AllConditionsMet(&trueSet))
		assert.False(t, utils.AllConditionsMet(&falseSet))
	})
}

func TestPKLHTTPFormattersMisc(t *testing.T) {
	headers := map[string][]string{
		"X-Test": {"val1", "val2"},
	}
	formattedHeaders := utils.FormatRequestHeaders(headers)
	// Expect outer block and encoded inner values
	assert.True(t, strings.HasPrefix(formattedHeaders, "Headers {"))
	assert.Contains(t, formattedHeaders, utils.EncodeBase64String("val1"))
	assert.Contains(t, formattedHeaders, utils.EncodeBase64String("val2"))

	params := map[string][]string{"q": {" go ", "lang"}}
	formattedParams := utils.FormatRequestParams(params)
	assert.True(t, strings.HasPrefix(formattedParams, "Params {"))
	assert.Contains(t, formattedParams, utils.EncodeBase64String("go")) // Trimmed and base64 encoded

	respHeaders := map[string]string{"Content-Type": "application/json"}
	formattedRespHeaders := utils.FormatResponseHeaders(respHeaders)
	assert.True(t, strings.HasPrefix(formattedRespHeaders, "Headers {"))
	assert.Contains(t, formattedRespHeaders, "application/json")

	props := map[string]string{"duration": "120"}
	formattedProps := utils.FormatResponseProperties(props)
	assert.True(t, strings.HasPrefix(formattedProps, "Properties {"))
	assert.Contains(t, formattedProps, "120")
}

func TestFileHelpers(t *testing.T) {
	// GenerateResourceIDFilename sanitization
	got := utils.GenerateResourceIDFilename("@/path:val", "req-")
	assert.Equal(t, "req-__path_val", got)

	// SanitizeArchivePath should allow inside paths and reject escape attempts
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	base := filepath.Join(tmpDir, "base")
	good, err := utils.SanitizeArchivePath(base, "inner/file.txt")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(good, base))

	_, err = utils.SanitizeArchivePath(base, "../../etc/passwd")
	require.Error(t, err)

	// CreateDirectories & CreateFiles integration test (in-mem FS)
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	dirs := []string{filepath.Join(tmpDir, "a", "b", "c"), filepath.Join(tmpDir, "a", "b", "d")}
	files := []string{filepath.Join(dirs[0], "file1.txt"), filepath.Join(dirs[1], "file2.txt")}

	err = utils.CreateDirectories(ctx, fs, dirs)
	require.NoError(t, err)
	for _, d := range dirs {
		exists, _ := afero.DirExists(fs, d)
		assert.True(t, exists)
	}

	err = utils.CreateFiles(ctx, fs, files)
	require.NoError(t, err)
	for _, f := range files {
		exists, _ := afero.Exists(fs, f)
		assert.True(t, exists)
	}

	// Ensure CreateFiles writes to correct paths relative to previously created dirs
	stat, err := fs.Stat(files[0])
	require.NoError(t, err)
	assert.False(t, stat.IsDir())
}
