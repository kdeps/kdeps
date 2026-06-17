package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestStoredUploadPath(t *testing.T) {
	t.Parallel()
	p := storedUploadPath("/uploads", "abc123", "file.txt")
	assert.Equal(t, "/uploads/abc123_file.txt", p)
}

func TestNewUploadedFileRecord(t *testing.T) {
	t.Parallel()
	rec := newUploadedFileRecord("id1", "photo.jpg", "image/jpeg", "/tmp/photo.jpg", 4096)
	assert.Equal(t, "id1", rec.ID)
	assert.Equal(t, "photo.jpg", rec.Filename)
	assert.Equal(t, "image/jpeg", rec.ContentType)
	assert.Equal(t, "/tmp/photo.jpg", rec.Path)
	assert.Equal(t, int64(4096), rec.Size)
	assert.NotNil(t, rec.Metadata)
}

func TestLookupStoredFile_Hit(t *testing.T) {
	t.Parallel()
	files := map[string]*domain.UploadedFile{
		"f1": {ID: "f1", Filename: "x.txt"},
	}
	f, err := lookupStoredFile(files, "f1")
	require.NoError(t, err)
	assert.Equal(t, "f1", f.ID)
}

func TestLookupStoredFile_Miss(t *testing.T) {
	t.Parallel()
	files := map[string]*domain.UploadedFile{}
	_, err := lookupStoredFile(files, "missing")
	assert.Error(t, err)
}

func TestGenerateUploadID(t *testing.T) {
	t.Parallel()
	id := generateUploadID([]byte("content"))
	assert.Len(t, id, 16)
	id2 := generateUploadID([]byte("content"))
	assert.Len(t, id2, 16)
}

func TestExpiredFileIDs(t *testing.T) {
	t.Parallel()
	old := time.Now().Add(-2 * time.Hour)
	recent := time.Now()
	files := map[string]*domain.UploadedFile{
		"old":    {UploadedAt: old},
		"recent": {UploadedAt: recent},
	}
	cutoff := time.Now().Add(-1 * time.Hour)
	ids := expiredFileIDs(files, cutoff)
	assert.Equal(t, []string{"old"}, ids)
}

func TestRemoveStoredFileEntry_DeletesFromMap(t *testing.T) {
	t.Parallel()
	files := map[string]*domain.UploadedFile{
		"f1": {ID: "f1", Path: "/nonexistent/file.txt"},
	}
	removeStoredFileEntry(files, "f1")
	_, exists := files["f1"]
	assert.False(t, exists)
}
