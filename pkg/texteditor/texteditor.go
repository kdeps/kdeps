package texteditor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/x/editor"
	"github.com/spf13/afero"
)

// EditPkl opens the file at filePath with the 'kdeps' editor if the file exists and has a .pkl extension.
func EditPkl(fs afero.Fs, filePath string) error {
	// Ensure the file has a .pkl extension
	if filepath.Ext(filePath) != ".pkl" {
		return fmt.Errorf("file '%s' does not have a .pkl extension", filePath)
	}

	// Check if the file exists
	if _, err := fs.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file '%s' does not exist", filePath)
		}
		return fmt.Errorf("failed to stat file '%s': %w", filePath, err)
	}

	// Prepare the editor command
	cmd, err := editor.Cmd("kdeps", filePath)
	if err != nil {
		return fmt.Errorf("failed to create editor command: %w", err)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the editor command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor command failed: %w", err)
	}

	return nil
}
