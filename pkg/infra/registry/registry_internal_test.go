package registry

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestPublish_MarshalError(t *testing.T) {
	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("injected marshal error")
	}

	// Create a temp file as the archive
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.kdeps")
	require.NoError(t, os.WriteFile(archivePath, []byte("fake archive"), 0644))

	client := NewClient("http://localhost:9999")
	_, err := client.Publish(context.Background(), archivePath, &domain.KdepsPkg{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal manifest")
}
