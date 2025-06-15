package evaluator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// createDummyPklBinary writes an executable fake "pkl" binary to dir and returns its path.
func createDummyPklBinary(t *testing.T, dir string) string {
	t.Helper()
	file := filepath.Join(dir, "pkl")
	content := "#!/bin/sh\necho '{}'; exit 0\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0o755))
	// Windows executables need .exe suffix
	if runtime.GOOS == "windows" {
		exePath := file + ".exe"
		require.NoError(t, os.Rename(file, exePath))
		file = exePath
	}
	return file
}

func TestEnsurePklBinaryExists_WithDummyBinary(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	tmpDir := t.TempDir()
	_ = createDummyPklBinary(t, tmpDir)

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })

	require.NoError(t, EnsurePklBinaryExists(ctx, logger))
}

func TestEvalPkl_WithDummyBinary(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	tmpDir := t.TempDir()
	dummy := createDummyPklBinary(t, tmpDir)
	_ = dummy
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })

	// Create a fake .pkl file on the OS filesystem because the external command
	// receives the path directly.
	pklPath := filepath.Join(tmpDir, "sample.pkl")
	require.NoError(t, os.WriteFile(pklPath, []byte("{}"), 0o644))

	fs := afero.NewOsFs()
	header := "amends \"pkg://dummy\""
	output, err := EvalPkl(fs, ctx, pklPath, header, logger)
	require.NoError(t, err)
	require.Contains(t, output, header)
}

func TestEvalPkl_InvalidExtension(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	// Should error when file does not have .pkl extension
	_, err := EvalPkl(fs, context.Background(), "file.txt", "header", logger)
	require.Error(t, err)
	require.Contains(t, err.Error(), ".pkl extension")
}

func TestCreateAndProcessPklFile_Basic(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()
	fs := afero.NewMemMapFs()

	sections := []string{"section1", "section2"}
	finalFile := filepath.Join(t.TempDir(), "out.pkl")

	// simple process func echoes header + sections concatenated
	process := func(fs afero.Fs, ctx context.Context, tmpFile string, header string, logger *logging.Logger) (string, error) {
		data, err := afero.ReadFile(fs, tmpFile)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	err := CreateAndProcessPklFile(fs, ctx, sections, finalFile, "Workflow.pkl", logger, process, false)
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, finalFile)
	require.NoError(t, err)
	// ensure both sections exist
	require.Contains(t, string(content), "section1")
	require.Contains(t, string(content), "section2")
}

func TestCreateAndProcessPklFile_Simple(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "/out/result.pkl"
	sections := []string{"sec1", "sec2"}
	// processFunc writes content combining headerSection and sections
	var receivedHeader string
	processFunc := func(f afero.Fs, c context.Context, tmpFile string, headerSection string, l *logging.Logger) (string, error) {
		receivedHeader = headerSection
		return headerSection + "-processed", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, sections, finalPath, "Template.pkl", logger, processFunc, false)
	require.NoError(t, err)
	// Verify output file exists with expected content
	data, err := afero.ReadFile(fs, finalPath)
	require.NoError(t, err)
	require.Equal(t, receivedHeader+"-processed", string(data))
}

func TestCreateAndProcessPklFile_Extends(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "result_ext.pkl"
	sections := []string{"alpha"}
	// processFunc checks that headerSection starts with 'extends'
	processFunc := func(f afero.Fs, c context.Context, tmpFile string, headerSection string, l *logging.Logger) (string, error) {
		if !strings.HasPrefix(headerSection, "extends") {
			return "", errors.New("unexpected header")
		}
		return "ok", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, sections, finalPath, "Template.pkl", logger, processFunc, true)
	require.NoError(t, err)
	data, err := afero.ReadFile(fs, finalPath)
	require.NoError(t, err)
	require.Equal(t, "ok", string(data))
}
