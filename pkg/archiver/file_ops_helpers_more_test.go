package archiver

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBackupPathAdditional(t *testing.T) {
	dst := filepath.Join("/tmp", "file.txt")
	md5 := "abcdef12"
	expected := filepath.Join("/tmp", "file_"+md5+".txt")
	assert.Equal(t, expected, getBackupPath(dst, md5))
}
