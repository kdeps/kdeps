package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

func newHTTPTestResolver(t *testing.T) *DependencyResolver {
	tmp := t.TempDir()
	fs := afero.NewOsFs()
	// ensure tmp dir exists on host fs
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatalf("unable to create temp dir: %v", err)
	}
	return &DependencyResolver{
		Fs:        fs,
		FilesDir:  tmp,
		RequestID: "rid",
		Logger:    logging.NewTestLogger(),
	}
}

func TestWriteResponseBodyToFile(t *testing.T) {
	dr := newHTTPTestResolver(t)

	// happy path – encoded body should be decoded and written to file
	body := "hello world"
	enc := utils.EncodeValue(body)
	path, err := dr.WriteResponseBodyToFile("res1", &enc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Fatalf("expected non-empty path")
	}
	// Verify file exists and content matches (decoded value)
	data, err := afero.ReadFile(dr.Fs, path)
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}
	if string(data) != body {
		t.Errorf("file content mismatch: got %s want %s", string(data), body)
	}

	// nil body pointer should return empty path, nil error
	empty, err := dr.WriteResponseBodyToFile("res2", nil)
	if err != nil {
		t.Fatalf("unexpected error for nil input: %v", err)
	}
	if empty != "" {
		t.Errorf("expected empty path for nil input, got %s", empty)
	}

	// Ensure filename generation is as expected
	expectedFile := filepath.Join(dr.FilesDir, utils.GenerateResourceIDFilename("res1", dr.RequestID))
	if path != expectedFile {
		t.Errorf("unexpected file path: %s", path)
	}
}

func TestIsMethodWithBody_Cases(t *testing.T) {
	positive := []string{"POST", "put", "Patch", "DELETE"}
	for _, m := range positive {
		if !isMethodWithBody(m) {
			t.Errorf("expected %s to allow body", m)
		}
	}
	negative := []string{"GET", "HEAD", "OPTIONS"}
	for _, m := range negative {
		if isMethodWithBody(m) {
			t.Errorf("expected %s to not allow body", m)
		}
	}
}
