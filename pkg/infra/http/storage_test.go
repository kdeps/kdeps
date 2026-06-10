// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package http_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"path/filepath"

	"github.com/kdeps/kdeps/v2/pkg/infra/http"
)

func TestNewTemporaryFileStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, store)
	// baseDir is not exported, so we test indirectly via operations
}

func TestTemporaryFileStore_Store(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	content := []byte("test file content")
	filename := "test.txt"
	contentType := "text/plain"

	file, err := store.Store(filename, content, contentType)
	require.NoError(t, err)
	assert.NotNil(t, file)
	assert.Equal(t, filename, file.Filename)
	assert.Equal(t, contentType, file.ContentType)
	assert.Equal(t, int64(len(content)), file.Size)
	assert.NotEmpty(t, file.ID)
	assert.NotEmpty(t, file.Path)

	// Verify file exists on disk
	_, err = os.Stat(file.Path)
	require.NoError(t, err)
}

func TestTemporaryFileStore_Get(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	file, err := store.Store("test.txt", []byte("content"), "text/plain")
	require.NoError(t, err)

	retrieved, err := store.Get(file.ID)
	require.NoError(t, err)
	assert.Equal(t, file.ID, retrieved.ID)
	assert.Equal(t, file.Filename, retrieved.Filename)

	// Test nonexistent file
	_, err = store.Get("nonexistent-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTemporaryFileStore_GetPath(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	file, err := store.Store("test.txt", []byte("content"), "text/plain")
	require.NoError(t, err)

	path, err := store.GetPath(file.ID)
	require.NoError(t, err)
	assert.Equal(t, file.Path, path)

	// Test nonexistent file
	_, err = store.GetPath("nonexistent-id")
	require.Error(t, err)
}

func TestTemporaryFileStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	file, err := store.Store("test.txt", []byte("content"), "text/plain")
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(file.Path)
	require.NoError(t, err)

	// Delete file
	err = store.Delete(file.ID)
	require.NoError(t, err)

	// Verify file is deleted from disk
	_, err = os.Stat(file.Path)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	// Verify file is removed from memory
	_, err = store.Get(file.ID)
	require.Error(t, err)

	// Test deleting nonexistent file
	err = store.Delete("nonexistent-id")
	require.Error(t, err)
}

func TestTemporaryFileStore_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	// Create old file - we'll test cleanup by waiting or using a short TTL
	// Note: We can't directly manipulate unexported fields, so we test
	// cleanup with a very short TTL after a delay, or test the cleanup
	// method directly by storing files and calling Cleanup with appropriate TTL
	oldFile, err := store.Store("old.txt", []byte("old"), "text/plain")
	require.NoError(t, err)

	// Cleanup immediately with very short TTL should not delete the just-created file
	err = store.Cleanup(1 * time.Hour)
	require.NoError(t, err)
	// File should still exist
	_, err = store.Get(oldFile.ID)
	require.NoError(t, err)

	// Create new file
	newFile, err := store.Store("new.txt", []byte("new"), "text/plain")
	require.NoError(t, err)

	// Cleanup files older than 1 hour
	// Both files are new, so neither should be cleaned up
	err = store.Cleanup(1 * time.Hour)
	require.NoError(t, err)

	// Both files should still exist (both are new)
	_, err = store.Get(oldFile.ID)
	require.NoError(t, err)

	retrieved, err := store.Get(newFile.ID)
	require.NoError(t, err)
	assert.Equal(t, newFile.ID, retrieved.ID)

	// Test cleanup with immediate TTL (should delete both)
	err = store.Cleanup(0)
	require.NoError(t, err)

	// Both should be deleted now
	_, err = store.Get(oldFile.ID)
	require.Error(t, err)
	_, err = store.Get(newFile.ID)
	require.Error(t, err)
}

func TestTemporaryFileStore_Close(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	// Store some files
	file1, err := store.Store("file1.txt", []byte("content1"), "text/plain")
	require.NoError(t, err)
	file2, err := store.Store("file2.txt", []byte("content2"), "text/plain")
	require.NoError(t, err)

	// Close store
	err = store.Close()
	require.NoError(t, err)

	// Files should be deleted from disk
	_, err = os.Stat(file1.Path)
	require.Error(t, err)
	_, err = os.Stat(file2.Path)
	require.Error(t, err)

	// Closing again should not error
	err = store.Close()
	require.NoError(t, err)
}

func TestTemporaryFileStore_SanitizeFilename(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	// Test with path in filename (should be sanitized)
	file, err := store.Store("../../../etc/passwd", []byte("content"), "text/plain")
	require.NoError(t, err)

	// Should only contain base filename
	assert.Equal(t, "passwd", file.Filename)
	assert.NotContains(t, file.Path, "../")
}

// TestNewTemporaryFileStore_MkdirAllError exercises the os.MkdirAll error branch
// at line 49-51 by providing a base directory whose parent is a regular file.
func TestNewTemporaryFileStore_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))
	baseDir := filepath.Join(blocker, "uploads")

	store, err := http.NewTemporaryFileStore(baseDir)
	require.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "failed to create upload directory")
}

// TestTemporaryFileStore_Store_WriteFileError exercises the os.WriteFile error
// branch at line 84-86 by writing to a read-only directory.
func TestTemporaryFileStore_Store_WriteFileError(t *testing.T) {
	tmpDir := t.TempDir()
	// Make directory read-only so os.WriteFile fails
	require.NoError(t, os.Chmod(tmpDir, 0444))
	defer func() {
		_ = os.Chmod(tmpDir, 0755) // restore for cleanup
	}()

	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	_, err = store.Store("test.txt", []byte("content"), "text/plain")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write file")
}

// TestTemporaryFileStore_Delete_DirError exercises the os.Remove error branch
// at line 143-145 when the file path is a non-empty directory (error is not
// IsNotExist).
func TestTemporaryFileStore_Delete_DirError(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := http.NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	// Store a file, record its path, then replace it with a non-empty directory
	file, err := store.Store("test.txt", []byte("content"), "text/plain")
	require.NoError(t, err)

	require.NoError(t, os.Remove(file.Path))
	require.NoError(t, os.MkdirAll(file.Path, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(file.Path, "child.txt"), []byte("x"), 0600))

	// os.Remove on a non-empty directory returns a non-IsNotExist error
	err = store.Delete(file.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete file")
}
