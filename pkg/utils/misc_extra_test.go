package utils

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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
			assert.True(t, ShouldSkip(&cond))
		}
	})

	t.Run("ShouldSkip_FalseCase", func(t *testing.T) {
		cond := []interface{}{false, "no"}
		assert.False(t, ShouldSkip(&cond))
	})

	t.Run("AllConditionsMet", func(t *testing.T) {
		trueSet := []interface{}{true, "TRUE"}
		falseSet := []interface{}{true, "false"}
		assert.True(t, AllConditionsMet(&trueSet))
		assert.False(t, AllConditionsMet(&falseSet))
	})
}

func TestPKLHTTPFormatters(t *testing.T) {
	headers := map[string][]string{
		"X-Test": {"val1", "val2"},
	}
	formattedHeaders := FormatRequestHeaders(headers)
	// Expect outer block and encoded inner values
	assert.True(t, strings.HasPrefix(formattedHeaders, "headers {"))
	assert.Contains(t, formattedHeaders, EncodeBase64String("val1"))
	assert.Contains(t, formattedHeaders, EncodeBase64String("val2"))

	params := map[string][]string{"q": {" go ", "lang"}}
	formattedParams := FormatRequestParams(params)
	assert.True(t, strings.HasPrefix(formattedParams, "params {"))
	assert.Contains(t, formattedParams, EncodeBase64String("go")) // Trimmed

	respHeaders := map[string]string{"Content-Type": "application/json"}
	formattedRespHeaders := FormatResponseHeaders(respHeaders)
	assert.True(t, strings.HasPrefix(formattedRespHeaders, "headers {"))
	assert.Contains(t, formattedRespHeaders, "application/json")

	props := map[string]string{"duration": "120"}
	formattedProps := FormatResponseProperties(props)
	assert.True(t, strings.HasPrefix(formattedProps, "properties {"))
	assert.Contains(t, formattedProps, "120")
}

func TestFileHelpers(t *testing.T) {
	// GenerateResourceIDFilename sanitization
	got := GenerateResourceIDFilename("@/path:val", "req-")
	assert.Equal(t, "req-__path_val", got)

	// SanitizeArchivePath should allow inside paths and reject escape attempts
	base := "/tmp/base"
	good, err := SanitizeArchivePath(base, "inner/file.txt")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(good, base))

	_, err = SanitizeArchivePath(base, "../../etc/passwd")
	assert.Error(t, err)

	// CreateDirectories & CreateFiles integration test (in-mem FS)
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	dirs := []string{"/a/b/c", "/a/b/d"}
	files := []string{"/a/b/c/file1.txt", "/a/b/d/file2.txt"}

	assert.NoError(t, CreateDirectories(fs, ctx, dirs))
	for _, d := range dirs {
		exists, _ := afero.DirExists(fs, d)
		assert.True(t, exists)
	}

	assert.NoError(t, CreateFiles(fs, ctx, files))
	for _, f := range files {
		exists, _ := afero.Exists(fs, f)
		assert.True(t, exists)
	}

	// Ensure CreateFiles writes to correct paths relative to previously created dirs
	stat, err := fs.Stat("/a/b/c/file1.txt")
	assert.NoError(t, err)
	assert.False(t, stat.IsDir())
}
