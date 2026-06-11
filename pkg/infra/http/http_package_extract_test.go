// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestPackage(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0600, Size: int64(len(content)),
		}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	return buf.Bytes()
}

func TestExtractKdepsPackage_InvalidGzip(t *testing.T) {
	tmpDir := t.TempDir()
	err := extractKdepsPackage([]byte("not-a-gzip-archive"), tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid gzip archive")
}

func TestExtractKdepsPackage_Success(t *testing.T) {
	tmpDir := t.TempDir()
	data := buildTestPackage(t, map[string]string{"workflow.yaml": "apiVersion: kdeps.io/v1\n"})
	require.NoError(t, extractKdepsPackage(data, tmpDir))
}

func TestExtractKdepsPackage_InvalidArchivePath(t *testing.T) {
	tmpDir := t.TempDir()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "../escape.txt", Size: 1, Mode: 0600}))
	_, err := tw.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	err = extractKdepsPackage(buf.Bytes(), tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path in package")
}

func TestExtractKdepsPackage_FileSizeLimit(t *testing.T) {
	orig := maxPackageFileSizeLimit
	maxPackageFileSizeLimit = 4
	t.Cleanup(func() { maxPackageFileSizeLimit = orig })

	tmpDir := t.TempDir()
	data := buildTestPackage(t, map[string]string{"big.bin": "12345678"})
	err := extractKdepsPackage(data, tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

func TestMapHTTPExtractError_DirectoryCreate(t *testing.T) {
	err := mapHTTPExtractError(errors.New(`failed to create directory "dir": mkdir failed`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}

func TestMapHTTPExtractError_ParentDirectoryCreate(t *testing.T) {
	err := mapHTTPExtractError(errors.New(`failed to create parent directory for "file": mkdir failed`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create parent directory")
}

func TestMapHTTPExtractError_TotalSizeExceeded(t *testing.T) {
	err := mapHTTPExtractError(errors.New("archive total uncompressed size exceeds limit of 10 bytes"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum total uncompressed size")
}

func TestMapHTTPExtractError_ResolveDestDir(t *testing.T) {
	err := mapHTTPExtractError(errors.New("resolve dest dir: abs failed"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve destination directory")
}

func TestMapHTTPExtractError_DefaultPassthrough(t *testing.T) {
	err := errors.New("custom extraction failure")
	assert.Equal(t, err, mapHTTPExtractError(err))
}

func TestExtractHTTPErrorPath(t *testing.T) {
	assert.Equal(t, "entry", extractHTTPErrorPath(`failed: "entry"`))
	assert.Equal(t, "plain", extractHTTPErrorPath("plain"))
}

func TestExtractKdepsPackage_MaxEntryCount(t *testing.T) {
	orig := maxPackageEntryCountLimit
	maxPackageEntryCountLimit = 2
	t.Cleanup(func() { maxPackageEntryCountLimit = orig })

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	for i := range 3 {
		name := fmt.Sprintf("file%d.txt", i)
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: 1}))
		_, err := tw.Write([]byte("x"))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	tmpDir := t.TempDir()
	err := extractKdepsPackage(buf.Bytes(), tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum entry count")
}
