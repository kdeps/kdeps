// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package docker

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTarEntryToFile_Errors(t *testing.T) {
	t.Run("mkdir error", func(t *testing.T) {
		tmpDir := t.TempDir()
		blocker := filepath.Join(tmpDir, "blocker")
		require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))
		dst := filepath.Join(blocker, "nested", "file.txt")

		err := extractTarEntryToFile(tar.NewReader(bytes.NewReader(nil)), dst, 1024)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create output directory")
	})

	t.Run("create error", func(t *testing.T) {
		tmpDir := t.TempDir()
		dst := filepath.Join(tmpDir, "existing")
		require.NoError(t, os.Mkdir(dst, 0750))

		err := extractTarEntryToFile(tar.NewReader(bytes.NewReader([]byte("data"))), dst, 1024)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create output file")
	})

	t.Run("copy error", func(t *testing.T) {
		tmpDir := t.TempDir()
		dst := filepath.Join(tmpDir, "out.txt")

		var tarBuf bytes.Buffer
		tw := tar.NewWriter(&tarBuf)
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: "payload.txt",
			Mode: 0600,
			Size: 100,
		}))
		_, err := tw.Write(bytes.Repeat([]byte("x"), 10))
		require.NoError(t, err)

		tr := tar.NewReader(io.MultiReader(
			bytes.NewReader(tarBuf.Bytes()),
			errReader{err: errors.New("read failed")},
		))
		_, err = tr.Next()
		require.NoError(t, err)

		err = extractTarEntryToFile(tr, dst, 1024)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write file")
	})
}

type errReader struct {
	err error
}

func (r errReader) Read(_ []byte) (int, error) {
	return 0, r.err
}
