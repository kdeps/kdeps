package texteditor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/x/editor"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// EditPklFunc is the type for the EditPkl function
type EditPklFunc func(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger) error

// MockEditPkl is a mock version of EditPkl that doesn't actually open an editor
var MockEditPkl EditPklFunc = func(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger) error {
	// Ensure the file has a .pkl extension
	if filepath.Ext(filePath) != ".pkl" {
		err := errors.New("file '" + filePath + "' does not have a .pkl extension")
		logger.Error(err.Error())
		return err
	}

	// Check if the file exists
	if _, err := fs.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			errMsg := "file does not exist"
			logger.Error(errMsg)
			return errors.New(errMsg)
		}
		errMsg := "failed to stat file"
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// In the mock version, we just return success
	return nil
}

// EditorCmd abstracts the editor command for testability
//go:generate mockgen -destination=editorcmd_mock.go -package=texteditor . EditorCmd

type EditorCmd interface {
	Run() error
	SetIO(stdin, stdout, stderr *os.File)
}

type EditorCmdFunc func(editorName, filePath string) (EditorCmd, error)

// realEditorCmd wraps the real editor.Cmd
type realEditorCmd struct {
	cmd *exec.Cmd
}

func (r *realEditorCmd) Run() error {
	return r.cmd.Run()
}

func (r *realEditorCmd) SetIO(stdin, stdout, stderr *os.File) {
	r.cmd.Stdin = stdin
	r.cmd.Stdout = stdout
	r.cmd.Stderr = stderr
}

var editorCmd = editor.Cmd

func realEditorCmdFactory(editorName, filePath string) (EditorCmd, error) {
	cmd, err := editorCmd(editorName, filePath)
	if err != nil {
		return nil, err
	}
	return &realEditorCmd{cmd: cmd}, nil
}

// EditPklWithFactory opens the file at filePath with the 'kdeps' editor using the provided factory function.
func EditPklWithFactory(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger, factory EditorCmdFunc) error {
	if os.Getenv("NON_INTERACTIVE") == "1" {
		logger.Info("NON_INTERACTIVE=1, skipping editor")
		return nil
	}

	if filepath.Ext(filePath) != ".pkl" {
		err := fmt.Sprintf("file '%s' does not have a .pkl extension", filePath)
		logger.Error(err)
		return errors.New(err)
	}

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

	if factory == nil {
		factory = realEditorCmdFactory
	}

	edCmd, err := factory("kdeps", filePath)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create editor command: %v", err)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	edCmd.SetIO(os.Stdin, os.Stdout, os.Stderr)

	if err := edCmd.Run(); err != nil {
		errMsg := fmt.Sprintf("editor command failed: %v", err)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

// EditPkl provides backward compatibility for editing PKL files.
var EditPkl EditPklFunc = func(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger) error {
	return EditPklWithFactory(fs, ctx, filePath, logger, nil)
}
