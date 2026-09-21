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

package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTempFileOfSize creates a temp file of exactly n bytes and registers
// its cleanup, mirroring pkg/input/file's own test helper of the same
// shape.
func writeTempFileOfSize(t *testing.T, n int) string {
	t.Helper()
	tmp, err := os.CreateTemp(t.TempDir(), "kdeps-loader-size-*.bin")
	require.NoError(t, err)
	defer tmp.Close()
	require.NoError(t, tmp.Truncate(int64(n)))
	return tmp.Name()
}

func TestMaxLoaderInputBytes_DefaultWhenUnset(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "")
	assert.Equal(t, int64(defaultMaxLoaderInputBytes), maxLoaderInputBytes())
}

func TestMaxLoaderInputBytes_EnvOverride(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "1024")
	assert.Equal(t, int64(1024), maxLoaderInputBytes())
}

func TestMaxLoaderInputBytes_ZeroDisables(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "0")
	assert.Equal(t, int64(0), maxLoaderInputBytes())
}

func TestMaxLoaderInputBytes_InvalidEnvFallsBackToDefault(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "not-a-number")
	assert.Equal(t, int64(defaultMaxLoaderInputBytes), maxLoaderInputBytes())
}

func TestMaxLoaderInputBytes_NegativeEnvFallsBackToDefault(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "-1")
	assert.Equal(t, int64(defaultMaxLoaderInputBytes), maxLoaderInputBytes())
}

func TestCheckFileSizeLimit_OverLimitRejected(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "1024")
	path := writeTempFileOfSize(t, 2048)

	err := checkFileSizeLimit(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "over the")
	assert.Contains(t, err.Error(), "KDEPS_LOADER_MAX_BYTES")
}

func TestCheckFileSizeLimit_AtLimitAllowed(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "1024")
	path := writeTempFileOfSize(t, 1024)

	assert.NoError(t, checkFileSizeLimit(path))
}

func TestCheckFileSizeLimit_DisabledByZero(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "0")
	path := writeTempFileOfSize(t, 2048)

	assert.NoError(t, checkFileSizeLimit(path))
}

func TestCheckFileSizeLimit_MissingFilePropagatesStatError(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "1024")
	err := checkFileSizeLimit(filepath.Join(t.TempDir(), "does-not-exist.txt"))
	require.Error(t, err)
}

func TestFormatByteSize(t *testing.T) {
	cases := map[int64]string{
		500:       "500 B",
		1024:      "1.0 KiB",
		1536:      "1.5 KiB",
		1 << 20:   "1.0 MiB",
		256 << 20: "256.0 MiB",
		1 << 30:   "1.0 GiB",
	}
	for n, want := range cases {
		assert.Equal(t, want, formatByteSize(n), "formatByteSize(%d)", n)
	}
}

// --- End-to-end: the size limit actually gates the real loaders, not just
// the helper function in isolation. ---

func TestLoadText_OverLimitRejected(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "16")
	path := writeTempFileOfSize(t, 1024)

	_, err := loadText(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loader text")
	assert.Contains(t, err.Error(), "KDEPS_LOADER_MAX_BYTES")
}

func TestLoadText_UnderLimitAllowed(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "1024")
	dir := t.TempDir()
	path := filepath.Join(dir, "small.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600))

	docs, err := loadText(path)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "hello", docs[0].Content)
}

func TestLoadDirectory_OversizedFileSkippedNotFatal(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "16")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "small.txt"), []byte("ok"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.txt"), make([]byte, 1024), 0o600))

	docs, err := loadDirectory(dir)
	require.NoError(t, err, "an oversized file must be skipped, not fail the whole directory load")
	require.Len(t, docs, 1, "only the small file should have been loaded")
	assert.Equal(t, "ok", docs[0].Content)
}

func TestLoadNotionDirectory_OversizedFileSkippedNotFatal(t *testing.T) {
	t.Setenv("KDEPS_LOADER_MAX_BYTES", "16")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "small.md"), []byte("ok"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.md"), make([]byte, 1024), 0o600))

	docs, err := loadNotionDirectory(dir)
	require.NoError(t, err)
	require.Len(t, docs, 1, "only the small file should have been loaded")
	assert.Equal(t, "ok", docs[0].Content)
}
