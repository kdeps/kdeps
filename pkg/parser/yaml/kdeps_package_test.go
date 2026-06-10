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

package yaml

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

func TestExtractKdepsPackage_MkdirTempError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("chmod tests do not work as root")
	}

	dir := t.TempDir()

	// Create a valid package file that exists on disk.
	pkgPath := filepath.Join(dir, "test.kdeps")
	require.NoError(t, os.WriteFile(pkgPath, []byte("dummy"), 0o644))

	// Point TMPDIR at a non-writable directory so os.MkdirTemp("", ...) fails.
	noWrite := filepath.Join(dir, "nowrite")
	require.NoError(t, os.MkdirAll(noWrite, 0o755))
	require.NoError(t, os.Chmod(noWrite, 0o000))
	t.Cleanup(func() { _ = os.Chmod(noWrite, 0o755) })
	t.Setenv("TMPDIR", noWrite)

	_, _, err := extractKdepsPackage(pkgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temp directory")
}

func TestExtractKdepsPackage_OpenError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("chmod tests do not work as root")
	}

	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "test.kdeps")
	require.NoError(t, os.WriteFile(pkgPath, []byte("dummy"), 0o644))

	// Make the file non-readable so os.Open fails.
	require.NoError(t, os.Chmod(pkgPath, 0o000))
	t.Cleanup(func() { _ = os.Chmod(pkgPath, 0o644) })

	_, _, err := extractKdepsPackage(pkgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open package")
}

func TestExtractTarEntries_NextError(t *testing.T) {
	dir := t.TempDir()

	// A reader that returns a non-EOF error on every read forces Next() to
	// return an error that is NOT io.EOF, hitting the branch on line 96-98.
	tr := tar.NewReader(&errorReader{err: errors.New("simulated tar error")})

	err := extractTarEntries(tr, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read tar entry")
}

func TestExtractTarEntries_MkdirAllDirError(t *testing.T) {
	dir := t.TempDir()

	// Create a file that blocks the path a directory entry would need.
	blockPath := filepath.Join(dir, "block")
	require.NoError(t, os.WriteFile(blockPath, []byte("blocker"), 0o644))

	// Build a tar archive containing a directory entry whose path collides
	// with the file above.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "block/subdir/",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
	}))
	require.NoError(t, tw.Close())

	tr := tar.NewReader(&buf)
	err := extractTarEntries(tr, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}

func TestExtractTarFile_MkdirAllError(t *testing.T) {
	dir := t.TempDir()

	// A file sits where the parent directory of targetPath would be.
	parent := filepath.Join(dir, "parent")
	require.NoError(t, os.WriteFile(parent, []byte("not-a-dir"), 0o644))
	targetPath := filepath.Join(parent, "child", "file.txt")

	// Minimal tar.Reader — the error fires before any I/O.
	tr := tar.NewReader(bytes.NewReader(nil))
	hdr := &tar.Header{Name: "file.txt", Size: 5, Mode: 0o600}

	err := extractTarFile(tr, hdr, targetPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create parent directory")
}

func TestExtractTarFile_CreateError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("chmod tests do not work as root")
	}

	dir := t.TempDir()

	// A non-writable directory where the target file would be created.
	readOnly := filepath.Join(dir, "readonly")
	require.NoError(t, os.MkdirAll(readOnly, 0o755))
	require.NoError(t, os.Chmod(readOnly, 0o000))
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o755) })

	targetPath := filepath.Join(readOnly, "out.txt")

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "out.txt", Size: 5, Mode: 0o600}))
	_, wErr := tw.Write([]byte("hello"))
	require.NoError(t, wErr)
	require.NoError(t, tw.Close())

	tr := tar.NewReader(&buf)
	_, nErr := tr.Next()
	require.NoError(t, nErr)

	hdr := &tar.Header{Name: "out.txt", Size: 5, Mode: 0o600}
	err := extractTarFile(tr, hdr, targetPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create file")
}

func TestExtractTarFile_CopyNError(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "out.txt")

	// Build a valid tar header block (512 bytes) for an entry with Size: 5.
	var hdrBuf bytes.Buffer
	tw := tar.NewWriter(&hdrBuf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "file.txt", Size: 5, Mode: 0o600,
	}))
	_, wErr := tw.Write([]byte("hello"))
	require.NoError(t, wErr)
	require.NoError(t, tw.Close())

	sentinel := errors.New("injected read error")

	// Construct a stream: valid header block followed by an error sink.
	// tr.Next() reads the 512-byte header successfully and sets up a
	// regFileReader. When extractTarFile calls io.CopyN the data read
	// hits the errorReader.
	combined := io.MultiReader(
		bytes.NewReader(hdrBuf.Bytes()[:512]),
		&errorReader{err: sentinel},
	)

	tr := tar.NewReader(combined)
	hdr, nErr := tr.Next()
	require.NoError(t, nErr)

	err := extractTarFile(tr, hdr, targetPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write file")
}

func TestIsKdepsPackage_False(t *testing.T) {
	assert.False(t, isKdepsPackage("file.txt"))
	assert.False(t, isKdepsPackage("component.yaml"))
	assert.False(t, isKdepsPackage(""))
	assert.True(t, isKdepsPackage(".kdeps")) // bare .kdeps IS a suffix match
	assert.True(t, isKdepsPackage("agent.kdeps"))
}
