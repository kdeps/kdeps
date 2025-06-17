package evaluator

import (
	"context"
	"errors"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCreateAndProcessPklFile_AmendsInPkg(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		// Simply return the header section to verify it flows through
		return headerSection + "\nprocessed", nil
	}

	final := "output_amends.pkl"
	sections := []string{"section1", "section2"}

	err := CreateAndProcessPklFile(fs, context.Background(), sections, final, "template.pkl", logger, processFunc, false)
	assert.NoError(t, err)

	// Verify final file exists and contains expected text
	content, err := afero.ReadFile(fs, final)
	assert.NoError(t, err)
	data := string(content)
	assert.True(t, strings.Contains(data, "amends \"package://schema.kdeps.com/core@"), "should contain amends relationship")
	assert.True(t, strings.Contains(data, "processed"))
}

func TestCreateAndProcessPklFile_ExtendsInPkg(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		return "result-" + headerSection, nil
	}

	final := "output_extends.pkl"
	err := CreateAndProcessPklFile(fs, context.Background(), nil, final, "template.pkl", logger, processFunc, true)
	assert.NoError(t, err)

	content, _ := afero.ReadFile(fs, final)
	str := string(content)
	assert.Contains(t, str, "extends \"package://schema.kdeps.com/core@")
	assert.Contains(t, str, "result-extends")
}

func TestCreateAndProcessPklFile_ProcessErrorInPkg(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		return "", assert.AnError
	}

	err := CreateAndProcessPklFile(fs, context.Background(), nil, "file.pkl", "template.pkl", logger, processFunc, false)
	assert.Error(t, err)
}

func TestEnsurePklBinaryExists(t *testing.T) {
	// Since mocking exec.LookPath directly is not possible, we can't easily test the binary lookup
	// Instead, we'll note that this test is limited and may need environment setup or alternative mocking
	// For now, we'll run the function as is, acknowledging it depends on the actual PATH
	ctx := context.Background()
	logger := logging.GetLogger()
	// This test will pass if 'pkl' is in PATH, fail with Fatal if not
	// We can't control the environment fully in this context
	err := EnsurePklBinaryExists(ctx, logger)
	if err != nil {
		t.Errorf("Expected no error if binary is in PATH, got: %v", err)
	}
	t.Log("EnsurePklBinaryExists test passed (dependent on PATH)")
}

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

func TestEvalPkl_InvalidExtensionAlt(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	if _, err := EvalPkl(fs, ctx, "/tmp/file.txt", "header", logger); err == nil {
		t.Fatalf("expected error for non-pkl extension")
	}
}

func TestCreateAndProcessPklFile_Minimal(t *testing.T) {
	memFs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	tmpDir := t.TempDir()
	finalFile := filepath.Join(tmpDir, "out.pkl")

	// Stub processFunc: just returns the header section.
	stub := func(fs afero.Fs, ctx context.Context, tmpFile string, header string, logger *logging.Logger) (string, error) {
		return header + "\ncontent", nil
	}

	err := CreateAndProcessPklFile(memFs, context.Background(), nil, finalFile, "Dummy.pkl", logger, stub, false)
	assert.NoError(t, err)

	// Verify file written with expected content.
	data, readErr := afero.ReadFile(memFs, finalFile)
	assert.NoError(t, readErr)
	assert.Contains(t, string(data), "content")
}

// stubProcessSuccess returns dummy content without error.
func stubProcessSuccess(fs afero.Fs, ctx context.Context, tmpFile string, header string, logger *logging.Logger) (string, error) {
	return header + "\ncontent", nil
}

// stubProcessFail returns an error to simulate processing failure.
func stubProcessFail(fs afero.Fs, ctx context.Context, tmpFile string, header string, logger *logging.Logger) (string, error) {
	return "", errors.New("process failed")
}

func TestCreateAndProcessPklFile_ProcessFuncError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	err := CreateAndProcessPklFile(fs, ctx, []string{"x = 1"}, "/ignored.pkl", "template.pkl", logger, stubProcessFail, false)
	if err == nil {
		t.Fatalf("expected error from processFunc, got nil")
	}
}

func TestCreateAndProcessPklFile_WritesFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	finalPath := "/out/final.pkl"

	if err := CreateAndProcessPklFile(fs, ctx, []string{"x = 1"}, finalPath, "template.pkl", logger, stubProcessSuccess, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert file now exists
	if ok, _ := afero.Exists(fs, finalPath); !ok {
		t.Fatalf("expected output file to be created")
	}
}

// TestCreateAndProcessPklFile verifies that CreateAndProcessPklFile creates the temporary
// file, invokes the supplied process function, and writes the final output file without
// returning an error. A no-op processFunc is provided so that the test remains hermetic.
func TestCreateAndProcessPklFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	finalFile := "/output.pkl"

	// Dummy process function that just returns fixed content
	processFunc := func(_ afero.Fs, _ context.Context, tmpFile string, _ string, _ *logging.Logger) (string, error) {
		// Ensure the temporary file actually exists
		if exists, err := afero.Exists(fs, tmpFile); err != nil || !exists {
			t.Fatalf("expected temporary file %s to exist", tmpFile)
		}
		return "processed-content", nil
	}

	sections := []string{"name = \"unit-test\""}

	// Execute the helper under test
	err := CreateAndProcessPklFile(fs, ctx, sections, finalFile, "Kdeps.pkl", logger, processFunc, false)
	assert.NoError(t, err)

	// Validate that the final file was written with the expected content
	content, err := afero.ReadFile(fs, finalFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "processed-content")
}

func TestCreateAndProcessPklFileNew(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := &logging.Logger{}
	sections := []string{
		`key = "value"`,
	}
	finalFileName := "test_output.pkl"
	pklTemplate := "template.pkl"
	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		// Simulate processing by reading the temp file
		content, err := afero.ReadFile(fs, tmpFile)
		if err != nil {
			return "", err
		}
		return string(content) + "\nprocessed", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, sections, finalFileName, pklTemplate, logger, processFunc, false)
	if err != nil {
		t.Errorf("CreateAndProcessPklFile failed: %v", err)
	}

	// Check if the final file was created and has content
	content, err := afero.ReadFile(fs, finalFileName)
	if err != nil {
		t.Errorf("Final file was not created or readable: %v", err)
	} else if len(content) == 0 {
		t.Errorf("Final file is empty")
	} else if !strings.Contains(string(content), "processed") {
		t.Errorf("Final file does not contain processed content: %s", string(content))
	}

	// Clean up
	fs.Remove(finalFileName)
}

func TestCreateAndProcessPklFileWithExtensionNew(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := &logging.Logger{}
	sections := []string{
		`key = "value"`,
	}
	finalFileName := "test_output_ext.pkl"
	pklTemplate := "template.pkl"
	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		// Simulate processing by reading the temp file
		content, err := afero.ReadFile(fs, tmpFile)
		if err != nil {
			return "", err
		}
		return string(content) + "\nprocessed with extension", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, sections, finalFileName, pklTemplate, logger, processFunc, true)
	if err != nil {
		t.Errorf("CreateAndProcessPklFile with extension failed: %v", err)
	}

	// Check if the final file was created and has content
	content, err := afero.ReadFile(fs, finalFileName)
	if err != nil {
		t.Errorf("Final file was not created or readable: %v", err)
	} else if len(content) == 0 {
		t.Errorf("Final file is empty")
	} else if !strings.Contains(string(content), "processed with extension") {
		t.Errorf("Final file does not contain processed content: %s", string(content))
	}

	// Clean up
	fs.Remove(finalFileName)
}

// TestEnsurePklBinaryExistsPositive adds a dummy `pkl` binary to PATH and
// asserts that EnsurePklBinaryExists succeeds.
func TestEnsurePklBinaryExistsPositive(t *testing.T) {
	logger := logging.NewTestLogger()

	tmpDir := t.TempDir()
	bin := "pkl"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	dummy := filepath.Join(tmpDir, bin)
	// create executable shell script file
	err := os.WriteFile(dummy, []byte("#!/bin/sh\nexit 0"), 0o755)
	assert.NoError(t, err)

	// prepend to PATH so lookPath finds it
	old := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+old)

	err = EnsurePklBinaryExists(context.Background(), logger)
	assert.NoError(t, err)
}
