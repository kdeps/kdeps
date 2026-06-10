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
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePackageEntryPath_AbsError(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(string) (string, error) {
		return "", errors.New("abs failed")
	}

	_, err := resolvePackageEntryPath("/tmp/dest", "file.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve target path")
}

func TestExtractKdepsPackage_AbsDestError(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(path string) (string, error) {
		if path == "/bad/dest" {
			return "", errors.New("abs failed")
		}
		return path, nil
	}

	err := extractKdepsPackage([]byte{}, "/bad/dest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve destination directory")
}

func TestExtractKdepsPackage_MaxEntryCount(t *testing.T) {
	orig := maxPackageEntryCountLimit
	maxPackageEntryCountLimit = 2
	t.Cleanup(func() { maxPackageEntryCountLimit = orig })

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	const entryContent = "x"
	for i := range 3 {
		name := fmt.Sprintf("file%d.txt", i)
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(entryContent))}))
		_, err := tw.Write([]byte(entryContent))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	tmpDir := t.TempDir()
	err := extractKdepsPackage(buf.Bytes(), tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum entry count")
}

func TestWriteExtractedFile_FileSizeLimit(t *testing.T) {
	orig := maxPackageFileSizeLimit
	maxPackageFileSizeLimit = 4
	t.Cleanup(func() { maxPackageFileSizeLimit = orig })

	tmpDir := t.TempDir()
	baseAbs, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	target := filepath.Join(tmpDir, "large.bin")

	var total int64
	err = writeExtractedFile(baseAbs, target, bytes.NewReader(bytes.Repeat([]byte("x"), 8)), &total)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

func TestWriteExtractedFile_TotalSizeLimit(t *testing.T) {
	origFile := maxPackageFileSizeLimit
	origTotal := maxPackageTotalUncompressedLimit
	maxPackageFileSizeLimit = 16
	maxPackageTotalUncompressedLimit = 6
	t.Cleanup(func() {
		maxPackageFileSizeLimit = origFile
		maxPackageTotalUncompressedLimit = origTotal
	})

	tmpDir := t.TempDir()
	baseAbs, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	target := filepath.Join(tmpDir, "part.bin")

	var total int64
	err = writeExtractedFile(baseAbs, target, bytes.NewReader([]byte("1234567")), &total)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum total uncompressed size")
}

func TestWriteExtractedFile_CloseError(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "file.txt")

	orig := closeExtractedFile
	t.Cleanup(func() { closeExtractedFile = orig })
	closeExtractedFile = func(_ *os.File) error {
		return errors.New("close failed")
	}

	baseAbs, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	var totalBytes int64
	err = writeExtractedFile(baseAbs, target, bytes.NewReader([]byte("payload")), &totalBytes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close failed")
}

func TestGetManagementWorkflowPath_AppDirectoryWithWorkflowFile(t *testing.T) {
	origStat := osStat
	origFind := findWorkflowFileHook
	t.Cleanup(func() {
		osStat = origStat
		findWorkflowFileHook = origFind
	})
	osStat = func(name string) (os.FileInfo, error) {
		if name == "/app" {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}
	findWorkflowFileHook = func(string) string {
		return "/app/workflow.yaml.j2"
	}

	server, err := NewServer(nil, nil, slog.Default())
	require.NoError(t, err)

	assert.Equal(t, "/app/workflow.yaml.j2", server.getManagementWorkflowPath())
}

func TestGetManagementWorkflowPath_AppDirectory(t *testing.T) {
	orig := osStat
	t.Cleanup(func() { osStat = orig })
	osStat = func(name string) (os.FileInfo, error) {
		if name == "/app" {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}

	server, err := NewServer(nil, nil, slog.Default())
	require.NoError(t, err)

	assert.Equal(t, "/app/workflow.yaml", server.getManagementWorkflowPath())
}
