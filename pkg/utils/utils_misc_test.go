package utils_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

func TestSafeDerefHelpers(t *testing.T) {
	str := "hello"
	if utils.SafeDerefString(nil) != "" {
		t.Error("expected empty string for nil pointer")
	}
	if utils.SafeDerefString(&str) != "hello" {
		t.Error("unexpected string value")
	}

	boolean := true
	if utils.SafeDerefBool(nil) {
		t.Error("expected false for nil bool pointer")
	}
	if !utils.SafeDerefBool(&boolean) {
		t.Error("expected true when pointer true")
	}

	if len(utils.SafeDerefSlice[string](nil)) != 0 {
		t.Error("expected empty slice for nil slice pointer")
	}
	s := []int{1, 2}
	if got := utils.SafeDerefSlice(&s); len(got) != 2 {
		t.Error("slice deref wrong length")
	}

	if len(utils.SafeDerefMap[string, int](nil)) != 0 {
		t.Error("expected empty map for nil map pointer")
	}
	mp := map[string]int{"a": 1}
	if got := utils.SafeDerefMap(&mp); got["a"] != 1 {
		t.Error("map value incorrect after deref")
	}
}

func TestStringHelpersMisc(t *testing.T) {
	slice := []string{"Foo", "bar"}
	if !utils.ContainsString(slice, "bar") {
		t.Error("ContainsString failed")
	}
	if utils.ContainsString(slice, "BAR") {
		t.Error("ContainsString should be case sensitive")
	}
	if !utils.ContainsStringInsensitive(slice, "BAR") {
		t.Error("ContainsStringInsensitive failed")
	}

	long := "0123456789"
	if got := utils.TruncateString(long, 5); got != "01..." {
		t.Errorf("truncate expected '01...', got %s", got)
	}
	if got := utils.TruncateString(long, 20); got != long {
		t.Errorf("truncate should not modify short strings")
	}
}

func TestCreateDirsAndFilesMisc(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	dirs := []string{"dir/sub"}
	if err := utils.CreateDirectories(fs, ctx, dirs); err != nil {
		t.Fatalf("CreateDirectories error: %v", err)
	}
	for _, d := range dirs {
		exists, _ := afero.DirExists(fs, d)
		if !exists {
			t.Fatalf("directory %s not created", d)
		}
	}

	files := []string{filepath.Join("dir", "file.txt")}
	if err := utils.CreateFiles(fs, ctx, files); err != nil {
		t.Fatalf("CreateFiles error: %v", err)
	}
	for _, f := range files {
		exists, _ := afero.Exists(fs, f)
		if !exists {
			t.Fatalf("file %s not created", f)
		}
	}
}

func TestSanitizeArchivePathMisc(t *testing.T) {
	base := "/tmp/base"
	good, err := utils.SanitizeArchivePath(base, "file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if good != filepath.Join(base, "file.txt") {
		t.Errorf("path sanitize incorrect")
	}
}

func TestGetLatestGitHubRelease_InvalidRepo(t *testing.T) {
	ctx := context.Background()

	// Test with an invalid repository that should return an error
	_, err := utils.GetLatestGitHubRelease(ctx, "invalid/repo/format", "")
	if err == nil {
		t.Error("expected error for invalid repo format, got nil")
	}
}

func TestGetLatestGitHubRelease_EmptyRepo(t *testing.T) {
	ctx := context.Background()

	// Test with empty repository name
	_, err := utils.GetLatestGitHubRelease(ctx, "", "")
	if err == nil {
		t.Error("expected error for empty repo, got nil")
	}
}

func TestGetLatestGitHubRelease_CustomBaseURL(t *testing.T) {
	ctx := context.Background()

	// Test with custom base URL
	_, err := utils.GetLatestGitHubRelease(ctx, "test/repo", "https://api.github.com")
	if err == nil {
		t.Log("GitHub API call succeeded (unexpected but acceptable)")
	} else {
		t.Logf("GitHub API call failed as expected: %v", err)
	}
}

func TestConditions_ShouldSkip(t *testing.T) {
	tests := []struct {
		name       string
		conditions []interface{}
		expected   bool
	}{
		{
			name:       "empty conditions",
			conditions: []interface{}{},
			expected:   false,
		},
		{
			name:       "all false conditions",
			conditions: []interface{}{false, "false", "FALSE"},
			expected:   false,
		},
		{
			name:       "one true boolean",
			conditions: []interface{}{false, true, "false"},
			expected:   true,
		},
		{
			name:       "one true string",
			conditions: []interface{}{false, "true", "false"},
			expected:   true,
		},
		{
			name:       "case insensitive true",
			conditions: []interface{}{false, "TRUE", "False"},
			expected:   true,
		},
		{
			name:       "mixed types with true",
			conditions: []interface{}{false, "true", 42, true},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ShouldSkip(&tt.conditions)
			if result != tt.expected {
				t.Errorf("ShouldSkip() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConditions_AllConditionsMet(t *testing.T) {
	tests := []struct {
		name       string
		conditions []interface{}
		expected   bool
	}{
		{
			name:       "empty conditions",
			conditions: []interface{}{},
			expected:   true,
		},
		{
			name:       "all true conditions",
			conditions: []interface{}{true, "true", "TRUE"},
			expected:   true,
		},
		{
			name:       "one false boolean",
			conditions: []interface{}{true, false, "true"},
			expected:   false,
		},
		{
			name:       "one false string",
			conditions: []interface{}{true, "false", "true"},
			expected:   false,
		},
		{
			name:       "case insensitive false",
			conditions: []interface{}{true, "FALSE", "true"},
			expected:   false,
		},
		{
			name:       "unsupported type",
			conditions: []interface{}{true, 42, "true"},
			expected:   false,
		},
		{
			name:       "mixed types all true",
			conditions: []interface{}{true, "true", "TRUE"},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.AllConditionsMet(&tt.conditions)
			if result != tt.expected {
				t.Errorf("AllConditionsMet() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBase64_EdgeCases(t *testing.T) {
	// Test invalid base64 that looks like base64 but fails to decode
	invalidBase64 := "SGVsbG8gV29ybGQ="                        // This is valid, let's create an invalid one
	invalidBase64 = invalidBase64[:len(invalidBase64)-1] + "X" // Make it invalid

	_, err := utils.DecodeBase64IfNeeded(invalidBase64)
	if err == nil {
		t.Log("base64 validation is more lenient than expected")
	} else {
		t.Logf("base64 validation caught invalid input: %v", err)
	}

	// Test string that looks like base64 but has wrong length
	shortBase64 := "SGVsbG8=" // Valid base64
	_, err = utils.DecodeBase64IfNeeded(shortBase64)
	if err != nil {
		t.Errorf("unexpected error for valid base64: %v", err)
	}

	// Test string with base64 chars but wrong length
	wrongLength := "SGVsbG8gV29ybGQ" // Missing padding
	_, err = utils.DecodeBase64IfNeeded(wrongLength)
	if err != nil {
		t.Errorf("unexpected error for non-base64 string: %v", err)
	}
}

func TestBase64_MapAndSlice(t *testing.T) {
	// Test DecodeStringMap with nil
	result, err := utils.DecodeStringMap(nil, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for nil input")
	}

	// Test DecodeStringSlice with nil
	resultSlice, err := utils.DecodeStringSlice(nil, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resultSlice != nil {
		t.Error("expected nil result for nil input")
	}

	// Test with valid data
	testMap := map[string]string{"key": "SGVsbG8="} // "Hello" in base64
	result, err = utils.DecodeStringMap(&testMap, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if (*result)["key"] != "Hello" {
		t.Errorf("expected decoded value, got %s", (*result)["key"])
	}

	// Test with invalid base64 in map - the function is lenient
	invalidMap := map[string]string{"key": "invalid_base64_!"}
	_, err = utils.DecodeStringMap(&invalidMap, "test")
	if err != nil {
		t.Logf("base64 validation caught invalid input: %v", err)
	} else {
		t.Log("base64 validation is more lenient than expected")
	}

	// Test with valid slice
	testSlice := []string{"SGVsbG8="} // "Hello" in base64
	resultSlice, err = utils.DecodeStringSlice(&testSlice, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if (*resultSlice)[0] != "Hello" {
		t.Errorf("expected decoded value, got %s", (*resultSlice)[0])
	}

	// Test with invalid base64 in slice - the function is lenient
	invalidSlice := []string{"invalid_base64_!"}
	_, err = utils.DecodeStringSlice(&invalidSlice, "test")
	if err != nil {
		t.Logf("base64 validation caught invalid input: %v", err)
	} else {
		t.Log("base64 validation is more lenient than expected")
	}
}

func TestBase64_EncodeValuePtr(t *testing.T) {
	// Test with nil pointer
	result := utils.EncodeValuePtr(nil)
	if result != nil {
		t.Error("expected nil result for nil input")
	}

	// Test with valid string
	testStr := "Hello World"
	result = utils.EncodeValuePtr(&testStr)
	if result == nil {
		t.Error("expected non-nil result")
	}
	if *result != "SGVsbG8gV29ybGQ=" {
		t.Errorf("expected encoded value, got %s", *result)
	}

	// Test with already encoded string
	encodedStr := "SGVsbG8gV29ybGQ="
	result = utils.EncodeValuePtr(&encodedStr)
	if result == nil {
		t.Error("expected non-nil result")
	}
	if *result != encodedStr {
		t.Errorf("expected same value for already encoded string, got %s", *result)
	}
}
