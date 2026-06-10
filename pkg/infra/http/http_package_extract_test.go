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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPackageEntry_PrefixGuard(t *testing.T) {
	var total int64
	err := extractPackageEntry(
		&tar.Header{Name: "file.txt"},
		"/tmp/base",
		"/other/file.txt",
		tar.NewReader(bytes.NewReader(nil)),
		&total,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path in package")
}

func TestExtractKdepsPackage_InvalidGzip(t *testing.T) {
	tmpDir := t.TempDir()
	err := extractKdepsPackage([]byte("not-a-gzip-archive"), tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid gzip archive")
}

func TestWriteExtractedFile_PrefixGuard(t *testing.T) {
	var total int64
	err := writeExtractedFile("/tmp/base", "/other/file.txt", bytes.NewReader([]byte("x")), &total)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target path")
}
