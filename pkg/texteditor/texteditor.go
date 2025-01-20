package texteditor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/x/editor"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// EditPkl opens the file at filePath with the 'kdeps' editor if the file exists and has a .pkl extension.
func EditPkl(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger) error {
	// Ensure the file has a .pkl extension
	if filepath.Ext(filePath) != ".pkl" {
		err := fmt.Sprintf("file '%s' does not have a .pkl extension", filePath)
		logger.Error(err)
		return errors.New(err)
	}

	// Check if the file exists
	if _, err := fs.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			errMsg := fmt.Sprintf("file '%s' does not exist", filePath)
			logger.Error(errMsg)
			return errors.New(errMsg)
		}
		errMsg := fmt.Sprintf("failed to stat file '%s': %v", filePath, err)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// Prepare the editor command
	cmd, err := editor.Cmd("kdeps", filePath)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create editor command: %v", err)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the editor command
	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("editor command failed: %v", err)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	return nil
}
