package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestGetFileMD5AndCopyFile(t *testing.T) {
	fsys := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	src := "/src.txt"
	content := []byte("hello world")
	assert.NoError(t, afero.WriteFile(fsys, src, content, 0o644))

	md5short, err := GetFileMD5(fsys, src, 8)
	assert.NoError(t, err)
	assert.Len(t, md5short, 8)

	dest := "/dest.txt"
	assert.NoError(t, CopyFile(fsys, ctx, src, dest, logger))

	// identical copy should not create backup
	assert.NoError(t, CopyFile(fsys, ctx, src, dest, logger))

	// modify src and copy again -> backup expected
	newContent := []byte("hello new world")
	assert.NoError(t, afero.WriteFile(fsys, src, newContent, 0o644))
	assert.NoError(t, CopyFile(fsys, ctx, src, dest, logger))

	backupName := "dest_" + md5short + ".txt"
	exists, _ := afero.Exists(fsys, "/"+backupName)
	assert.True(t, exists)
}

func TestMoveFolderAndCopyDir(t *testing.T) {
	fsys := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	srcDir := "/source"
	assert.NoError(t, fsys.MkdirAll(filepath.Join(srcDir, "nested"), 0o755))
	assert.NoError(t, afero.WriteFile(fsys, filepath.Join(srcDir, "file1.txt"), []byte("a"), 0o644))
	assert.NoError(t, afero.WriteFile(fsys, filepath.Join(srcDir, "nested", "file2.txt"), []byte("b"), 0o644))

	destDir := "/destination"
	assert.NoError(t, MoveFolder(fsys, srcDir, destDir))

	exists, _ := afero.DirExists(fsys, srcDir)
	assert.False(t, exists)

	for _, rel := range []string{"file1.txt", "nested/file2.txt"} {
		data, err := afero.ReadFile(fsys, filepath.Join(destDir, rel))
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	}

	compiledDir := "/compiled"
	assert.NoError(t, CopyDir(fsys, ctx, destDir, compiledDir, logger))
	d, err := afero.ReadFile(fsys, filepath.Join(compiledDir, "file1.txt"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("a"), d)
}
