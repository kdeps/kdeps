package texteditor

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestEditPkl(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("WrongExtension", func(t *testing.T) {
		err := EditPkl(fs, ctx, "/tmp/file.txt", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ".pkl extension")
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		err := EditPkl(fs, ctx, "/tmp/missing.pkl", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	// We cannot easily simulate editor.Cmd or cmd.Run errors without refactoring,
	// so we focus on the above error cases for now.
}
