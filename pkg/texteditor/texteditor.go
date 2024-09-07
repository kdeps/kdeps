package texteditor

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/x/editor"
	"github.com/spf13/afero"
)

func EditPkl(fs afero.Fs, filePath string) error {
	if _, err := fs.Stat(filePath); err == nil {
		c, err := editor.Cmd("kdeps", filePath)
		if err != nil {
			return errors.New(fmt.Sprintln("File does not exist!"))
		}

		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			return errors.New(fmt.Sprintf("Missing %s.", "$EDITOR"))
		}
	}

	return nil
}
