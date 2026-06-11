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

package targz_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/archive/targz"
)

type errorReader struct{ err error }

func (e *errorReader) Read([]byte) (int, error) { return 0, e.err }

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func TestExtractGzipTar_Success(t *testing.T) {
	dir := t.TempDir()
	data := buildTarGz(t, map[string]string{"workflow.yaml": "apiVersion: kdeps.io/v1\n"})
	err := targz.ExtractGzipTar(bytes.NewReader(data), dir, targz.DefaultOptions())
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "workflow.yaml"))
}

func TestExtractGzipTar_InvalidGzip(t *testing.T) {
	err := targz.ExtractGzipTar(
		bytes.NewReader([]byte("not gzip")),
		t.TempDir(),
		targz.DefaultOptions(),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gzip header")
}

func TestExtractTar_NextError(t *testing.T) {
	tr := tar.NewReader(&errorReader{err: errors.New("simulated tar error")})
	err := targz.ExtractTar(tr, t.TempDir(), targz.DefaultOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read tar entry")
}

func TestExtractTar_TraversalRejected(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "../outside.txt", Size: 1, Mode: 0o644}))
	_, err := tw.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	err = targz.ExtractTar(tar.NewReader(&buf), t.TempDir(), targz.DefaultOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid archive path")
}

func TestExtractTar_FileSizeLimit(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	payload := bytes.Repeat([]byte("x"), 64)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "big.bin", Size: int64(len(payload)), Typeflag: tar.TypeReg, Mode: 0o644,
	}))
	_, err := tw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	opts := targz.DefaultOptions()
	opts.MaxFileSize = 32
	err = targz.ExtractTar(tar.NewReader(&buf), t.TempDir(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

func TestExtractTar_MaxEntries(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for i := range 3 {
		name := filepath.Join("f", string(rune('a'+i))+".txt")
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Size: 1, Mode: 0o644}))
		_, err := tw.Write([]byte("x"))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())

	opts := targz.DefaultOptions()
	opts.MaxEntries = 2
	err := targz.ExtractTar(tar.NewReader(&buf), t.TempDir(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry count exceeds")
}

func TestExtractToTemp_MissingArchive(t *testing.T) {
	_, _, err := targz.ExtractToTemp("/nonexistent/archive.kdeps", "kdeps-*", targz.DefaultOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archive not found")
}

func TestExtractFile_FromDisk(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "pkg.kdeps")
	require.NoError(t, os.WriteFile(archive, buildTarGz(t, map[string]string{"a.txt": "hi"}), 0o644))

	dest := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(dest, 0o755))
	require.NoError(t, targz.ExtractFile(archive, dest, targz.DefaultOptions()))
	assert.FileExists(t, filepath.Join(dest, "a.txt"))
}

func TestExtractTar_SkipsDotEntry(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: ".", Typeflag: tar.TypeReg, Size: 0, Mode: 0o644,
	}))
	require.NoError(t, tw.Close())

	dest := t.TempDir()
	require.NoError(t, targz.ExtractTar(tar.NewReader(&buf), dest, targz.RegistryOptions()))
}

func TestExtractTar_RegularOnlyExactSizeLimit(t *testing.T) {
	payload := bytes.Repeat([]byte("a"), 4)
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "f.bin", Size: int64(len(payload)), Typeflag: tar.TypeReg, Mode: 0o644,
	}))
	_, err := tw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	opts := targz.RegistryOptions()
	opts.MaxFileSize = 4
	err = targz.ExtractTar(tar.NewReader(&buf), t.TempDir(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

func TestExtractTar_RegularOnly(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "dir/", Typeflag: tar.TypeDir, Mode: 0o755}))
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "dir/file.txt", Size: 2, Typeflag: tar.TypeReg, Mode: 0o644,
	}))
	_, err := tw.Write([]byte("ok"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	dest := t.TempDir()
	require.NoError(t, targz.ExtractTar(tar.NewReader(&buf), dest, targz.RegistryOptions()))
	assert.DirExists(t, filepath.Join(dest, "dir"))
	assert.FileExists(t, filepath.Join(dest, "dir", "file.txt"))
}

func TestExtractTar_CopyError(t *testing.T) {
	data := buildTarGz(t, map[string]string{"file.txt": "hello"})
	opts := targz.DefaultOptions()
	opts.Hooks.IOCopyN = func(_ io.Writer, _ io.Reader, _ int64) (int64, error) {
		return 0, errors.New("copy fail")
	}

	err := targz.ExtractGzipTar(bytes.NewReader(data), t.TempDir(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract file")
}

func TestExtractToTemp_Success(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "pkg.kdeps")
	require.NoError(t, os.WriteFile(archive, buildTarGz(t, map[string]string{"a.txt": "ok"}), 0o644))

	tempDir, cleanup, err := targz.ExtractToTemp(archive, "kdeps-*", targz.DefaultOptions())
	require.NoError(t, err)
	require.NotEmpty(t, tempDir)
	t.Cleanup(cleanup)
	assert.FileExists(t, filepath.Join(tempDir, "a.txt"))
}

func TestExtractToTemp_MkdirTempError(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "pkg.kdeps")
	require.NoError(t, os.WriteFile(archive, buildTarGz(t, map[string]string{"a.txt": "ok"}), 0o644))

	opts := targz.DefaultOptions()
	opts.Hooks.MkdirTemp = func(string, string) (string, error) {
		return "", errors.New("temp fail")
	}
	_, _, err := targz.ExtractToTemp(archive, "kdeps-*", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temp directory")
}

func TestExtractToTemp_ExtractFailureCleansUp(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "bad.kdeps")
	require.NoError(t, os.WriteFile(archive, []byte("not gzip"), 0o644))

	_, _, err := targz.ExtractToTemp(archive, "kdeps-*", targz.DefaultOptions())
	require.Error(t, err)
}

func TestExtractFile_OpenError(t *testing.T) {
	err := targz.ExtractFile(
		filepath.Join(t.TempDir(), "missing.kdeps"),
		t.TempDir(),
		targz.DefaultOptions(),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open archive")
}

func TestExtractTar_Hook(t *testing.T) {
	targz.ExtractTarHook = func(_ *tar.Reader, _ string, _ targz.Options) error {
		return errors.New("hooked")
	}
	t.Cleanup(func() { targz.ExtractTarHook = nil })

	err := targz.ExtractTar(tar.NewReader(bytes.NewReader(nil)), t.TempDir(), targz.DefaultOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hooked")
}

func TestWriteEntry_Success(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := "payload"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "out.txt", Size: int64(len(content)), Mode: 0o644,
	}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	dest := t.TempDir()
	target := filepath.Join(dest, "out.txt")
	opts := targz.DefaultOptions()
	opts.FilePerm = 0o600
	require.NoError(t, targz.WriteEntry(
		tar.NewReader(&buf),
		&tar.Header{Name: "out.txt", Size: int64(len(content))},
		target,
		opts,
	))
	assert.FileExists(t, target)
}

func TestExtractTar_MaxTotalSize(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for i := range 2 {
		name := fmt.Sprintf("f%d.txt", i)
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Size: 8, Mode: 0o644}))
		_, err := tw.Write(bytes.Repeat([]byte("x"), 8))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())

	opts := targz.DefaultOptions()
	opts.MaxTotalSize = 10
	err := targz.ExtractTar(tar.NewReader(&buf), t.TempDir(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "total uncompressed size exceeds")
}

func TestExtractTar_SkipsNonRegularEntry(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "link", Typeflag: tar.TypeSymlink, Linkname: "else", Mode: 0o644,
	}))
	require.NoError(t, tw.Close())

	dest := t.TempDir()
	require.NoError(t, targz.ExtractTar(tar.NewReader(&buf), dest, targz.RegistryOptions()))
	entries, err := os.ReadDir(dest)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestExtractTar_MkdirDirError(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "dir/", Typeflag: tar.TypeDir, Mode: 0o755}))
	require.NoError(t, tw.Close())

	opts := targz.DefaultOptions()
	opts.Hooks.MkdirAll = func(string, os.FileMode) error {
		return errors.New("mkdir fail")
	}
	err := targz.ExtractTar(tar.NewReader(&buf), t.TempDir(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}

func TestExtractTar_CloseError(t *testing.T) {
	data := buildTarGz(t, map[string]string{"file.txt": "hello"})
	opts := targz.DefaultOptions()
	opts.Hooks.FileClose = func(*os.File) error {
		return errors.New("close fail")
	}
	err := targz.ExtractGzipTar(bytes.NewReader(data), t.TempDir(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to close file")
}

func TestExtractTar_ParentMkdirError(t *testing.T) {
	data := buildTarGz(t, map[string]string{"nested/file.txt": "x"})
	opts := targz.DefaultOptions()
	opts.Hooks.MkdirAll = func(path string, _ os.FileMode) error {
		if filepath.Base(path) == "nested" {
			return errors.New("parent mkdir fail")
		}
		return os.MkdirAll(path, opts.DirPerm)
	}
	err := targz.ExtractGzipTar(bytes.NewReader(data), t.TempDir(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create parent directory")
}

func TestWriteEntry_CreateFileError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("chmod tests do not work as root")
	}

	dest := t.TempDir()
	readOnly := filepath.Join(dest, "readonly")
	require.NoError(t, os.MkdirAll(readOnly, 0o755))
	require.NoError(t, os.Chmod(readOnly, 0o000))
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o755) })

	target := filepath.Join(readOnly, "out.txt")
	content := "hello"
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "out.txt", Size: int64(len(content)), Mode: 0o644,
	}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	err = targz.WriteEntry(
		tar.NewReader(&buf),
		&tar.Header{Name: "out.txt", Size: int64(len(content))},
		target,
		targz.DefaultOptions(),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create file")
}

func TestDefaultOptions_ZeroMaxFileSizeUsesDefault(t *testing.T) {
	opts := targz.Options{MaxFileSize: 0}
	data := buildTarGz(t, map[string]string{"file.txt": "hello"})
	require.NoError(t, targz.ExtractGzipTar(bytes.NewReader(data), t.TempDir(), opts))
}
