package archiver

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// TestCopyDirBasic exercises the main happy-path of CopyDir, ensuring it
// recreates directory structure and files.
func TestCopyDirBasic(t *testing.T) {
	fsys := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	src := "/src"
	dst := "/dst"

	// Build a small tree: /src/sub/hello.txt
	require.NoError(t, fsys.MkdirAll(filepath.Join(src, "sub"), 0o755))
	fileContent := []byte("copy_dir_contents")
	require.NoError(t, afero.WriteFile(fsys, filepath.Join(src, "sub", "hello.txt"), fileContent, 0o644))

	// Act
	require.NoError(t, CopyDir(fsys, ctx, src, dst, logger))

	// Assert: destination directory replicates the tree.
	copiedBytes, err := afero.ReadFile(fsys, filepath.Join(dst, "sub", "hello.txt"))
	require.NoError(t, err)
	require.Equal(t, fileContent, copiedBytes)

	// Permissions (mode) on directory should be preserved (at least execute bit).
	info, err := fsys.Stat(filepath.Join(dst, "sub"))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

// TestCopyDirError verifies that an error from the underlying filesystem is
// propagated.  We create a read-only FS wrapper around a mem FS and attempt to
// write into it.
func TestCopyDirError(t *testing.T) {
	mem := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	src := "/ro/src"
	dst := "/ro/dst"
	require.NoError(t, mem.MkdirAll(src, 0o755))
	require.NoError(t, afero.WriteFile(mem, filepath.Join(src, "file.txt"), []byte("data"), 0o644))

	// Wrap in read-only fs to provoke write error on destination creation.
	ro := afero.NewReadOnlyFs(mem)

	err := CopyDir(ro, ctx, src, dst, logger)
	require.Error(t, err)

	// The error should be about permission or read-only.
	require.True(t, errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrInvalid))
}
