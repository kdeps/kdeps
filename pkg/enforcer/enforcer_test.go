package enforcer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

var (
	testFs              = afero.NewOsFs()
	homeDirPath         string
	currentDirPath      string
	ctx                 = context.Background()
	systemConfiguration *kdeps.Kdeps
	fileThatExist       string
	logger              *logging.Logger
	agentPath           string
	doc                 string
	workflowAmendsLine  = fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"`, schema.SchemaVersion(ctx))
	configAmendsLine    = fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Kdeps.pkl"`, schema.SchemaVersion(ctx))
	resourceAmendsLine  = fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"`, schema.SchemaVersion(ctx))
	resourceValues      = `
ActionID = "helloWorld"
Name = "name"
Description = "description"
Category = "category"
`
	configValues = `
RunMode = "docker"
DockerGPU = "cpu"
`
	workflowValues = `
Settings {
  APIServerMode = false
  APIServer {
    PortNum = 3000
    Routes {
      new {
	Path = "/api"
	Methods {
	  "POST"
	}
      }
    }
  }
}
AgentID = "myAgent"
Description = "My awesome AI Agent"
Version = "1.0.0"
TargetActionID = "helloWorld"
`
	testingT *testing.T
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Configuration steps
			ctx.Step(`^we have a blank config file$`, weHaveABlankFile)
			ctx.Step(`^a file "([^"]*)" exists in the current directory$`, aFileExistsInTheCurrentDirectory)
			ctx.Step(`^a system configuration is defined$`, aSystemConfigurationIsDefined)
			ctx.Step(`^it does not have a config amends line on top of the file$`, itDoesNotHaveAConfigAmendsLineOnTopOfTheFile)
			ctx.Step(`^it have a "([^"]*)" amends url line on top of the file$`, itHaveAAmendsURLLineOnTopOfTheFile)
			ctx.Step(`^it have a config amends line on top of the file$`, itHaveAConfigAmendsLineOnTopOfTheFile)
			ctx.Step(`^the current directory is "([^"]*)"$`, theCurrentDirectoryIs)
			ctx.Step(`^the home directory is "([^"]*)"$`, theHomeDirectoryIs)
			// Workflow steps
			ctx.Step(`^a file "([^"]*)" exists in the "([^"]*)"$`, aFileExistsInThe)
			ctx.Step(`^an agent folder "([^"]*)" exists in the current directory$`, anAgentFolderExistsInTheCurrentDirectory)
			ctx.Step(`^it is a valid agent$`, itIsAValidAgent)
			ctx.Step(`^it is an invalid agent$`, itIsAnInvalidAgent)
			ctx.Step(`^it is a valid pkl file$`, itIsAValidPklFile)
			ctx.Step(`^it is an invalid pkl file$`, itIsAnInvalidPklFile)
			ctx.Step(`^we have a blank workflow file$`, weHaveABlankFile)
			ctx.Step(`^it does not have a workflow amends line on top of the file$`, itDoesNotHaveAWorkflowAmendsLineOnTopOfTheFile)
			ctx.Step(`^it have a workflow amends line on top of the file$`, itHaveAWorkflowAmendsLineOnTopOfTheFile)
			ctx.Step(`^a folder named "([^"]*)" exists in the "([^"]*)"$`, aFolderNamedExistsInThe)
			// Resource steps
			ctx.Step(`^it have a resource amends line on top of the file$`, itHaveAResourceAmendsLineOnTopOfTheFile)
			ctx.Step(`^it does not have a resource amends line on top of the file$`, itDoesNotHaveAResourceAmendsLineOnTopOfTheFile)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/enforcer"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

// Config tests

func weHaveABlankFile() error {
	logger = logging.GetLogger()
	doc = ""
	return nil
}

func theHomeDirectoryIs(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", "")
	if err != nil {
		return err
	}

	homeDirPath = tempDir

	return nil
}

func theCurrentDirectoryIs(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", "")
	if err != nil {
		return err
	}

	currentDirPath = tempDir

	return nil
}

func aFileExistsInTheCurrentDirectory(arg1 string) error {
	file := filepath.Join(currentDirPath, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}

	f.Close()

	fileThatExist = file

	return nil
}

func aSystemConfigurationIsDefined() error {
	env := &environment.Environment{
		Home:           homeDirPath,
		Pwd:            "",
		NonInteractive: "1",
	}

	environ, err := environment.NewEnvironment(testFs, env)
	if err != nil {
		return err
	}

	cfgFile, err := cfg.GenerateConfiguration(ctx, testFs, environ, logger)
	if err != nil {
		return err
	}

	scfg, err := cfg.LoadConfiguration(ctx, testFs, cfgFile, logger)
	if err != nil {
		return err
	}

	systemConfiguration = scfg

	return nil
}

func itDoesNotHaveAConfigAmendsLineOnTopOfTheFile() error {
	doc = configValues

	return nil
}

func itHaveAAmendsURLLineOnTopOfTheFile(arg1 string) error {
	doc = strings.ReplaceAll(doc, "kdeps.com", arg1)

	return nil
}

func itHaveAConfigAmendsLineOnTopOfTheFile() error {
	doc = fmt.Sprintf("%s\n%s", configAmendsLine, configValues)

	return nil
}

func itIsAnInvalidAgent() error {
	if err := EnforceFolderStructure(testFs, ctx, agentPath, logger); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func itIsAValidAgent() error {
	if err := EnforceFolderStructure(testFs, ctx, agentPath, logger); err != nil {
		return err
	}

	return nil
}

func itIsAnInvalidPklFile() error {
	if err := EnforcePklTemplateAmendsRules(testFs, fileThatExist, context.Background(), logger); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func itIsAValidPklFile() error {
	if err := EnforcePklTemplateAmendsRules(testFs, fileThatExist, context.Background(), logger); err != nil {
		return err
	}

	if _, err := evaluator.EvalPkl(testFs, ctx, fileThatExist, "", nil, logger); err != nil {
		return err
	}

	return nil
}

// Workflow tests

func aFileExistsInThe(arg1, arg2 string) error {
	p := agentPath

	if arg2 != "my-agent" {
		p = agentPath + "/" + arg2
	}

	file := filepath.Join(p, arg1)
	fmt.Printf("Creating %s file!", file) //nolint:forbidigo // Test debug output

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	fileThatExist = file

	return nil
}

func anAgentFolderExistsInTheCurrentDirectory(arg1 string) error {
	agentPath = currentDirPath + "/my-agent"
	if err := testFs.MkdirAll(agentPath, 0o755); err != nil {
		return err
	}
	fmt.Printf("Agent path %s created!", agentPath) //nolint:forbidigo // Test debug output

	return nil
}

func itDoesNotHaveAWorkflowAmendsLineOnTopOfTheFile() error {
	doc = workflowValues

	return nil
}

func itHaveAWorkflowAmendsLineOnTopOfTheFile() error {
	doc = fmt.Sprintf("%s\n%s", workflowAmendsLine, workflowValues)

	return nil
}

func aFolderNamedExistsInThe(arg1, arg2 string) error {
	agentPath = currentDirPath + "/my-agent"
	subfolderPath := agentPath + "/" + arg1
	if err := testFs.MkdirAll(subfolderPath, 0o755); err != nil {
		return err
	}
	fmt.Printf("Agent path %s created!", subfolderPath) //nolint:forbidigo // Test debug output

	return nil
}

// Resource steps

func itHaveAResourceAmendsLineOnTopOfTheFile() error {
	doc = fmt.Sprintf("%s\n%s", resourceAmendsLine, resourceValues)

	return nil
}

func itDoesNotHaveAResourceAmendsLineOnTopOfTheFile() error {
	doc = resourceValues

	return nil
}

func TestEnforcePklVersion(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := t.Context()
	schemaVersion := "1.2.3"

	goodLine := "amends \"package://schema.kdeps.com/core@1.2.3#/Kdeps.pkl\""
	require.NoError(t, EnforcePklVersion(ctx, goodLine, "file.pkl", schemaVersion, logger))

	// lower version should warn but not error
	lowLine := "amends \"package://schema.kdeps.com/core@1.0.0#/Kdeps.pkl\""
	require.NoError(t, EnforcePklVersion(ctx, lowLine, "file.pkl", schemaVersion, logger))

	// higher version also no error
	highLine := "amends \"package://schema.kdeps.com/core@2.0.0#/Kdeps.pkl\""
	require.NoError(t, EnforcePklVersion(ctx, highLine, "file.pkl", schemaVersion, logger))

	// invalid version format should error
	badLine := "amends \"package://schema.kdeps.com/core@1.x#/Kdeps.pkl\""
	require.Error(t, EnforcePklVersion(ctx, badLine, "file.pkl", schemaVersion, logger))
}

func TestEnforcePklFilename(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := t.Context()

	// Good configuration .kdeps.pkl
	lineCfg := "amends \"package://schema.kdeps.com/core@1.0.0#/Kdeps.pkl\""
	require.NoError(t, EnforcePklFilename(ctx, lineCfg, "/path/to/.kdeps.pkl", logger))

	// Good workflow.pkl
	lineWf := "amends \"package://schema.kdeps.com/core@1.0.0#/Workflow.pkl\""
	require.NoError(t, EnforcePklFilename(ctx, lineWf, "/some/workflow.pkl", logger))

	// Resource.pkl must not have those filenames
	lineResource := "amends \"package://schema.kdeps.com/core@1.0.0#/Resource.pkl\""
	require.NoError(t, EnforcePklFilename(ctx, lineResource, "/path/to/resources/custom.pkl", logger))

	// Invalid file extension for config
	err := EnforcePklFilename(ctx, lineCfg, "/path/to/wrongname.txt", logger)
	require.Error(t, err)

	// Resource.pkl with forbidden filename
	err = EnforcePklFilename(ctx, lineResource, "/path/to/.kdeps.pkl", logger)
	require.Error(t, err)

	// Unknown pkl filename in amends line -> expect error
	unknownLine := "amends \"package://schema.kdeps.com/core@1.0.0#/Unknown.pkl\""
	err = EnforcePklFilename(ctx, unknownLine, "/path/to/unknown.pkl", logger)
	require.Error(t, err)
}

func TestEnforcePklFilenameValid(t *testing.T) {
	line := "amends \"package://schema.kdeps.com/core@0.0.0#/Workflow.pkl\""
	if err := EnforcePklFilename(t.Context(), line, "/tmp/workflow.pkl", logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error for valid filename: %v", err)
	}

	lineConf := "amends \"package://schema.kdeps.com/core@0.0.0#/Kdeps.pkl\""
	if err := EnforcePklFilename(t.Context(), lineConf, "/tmp/.kdeps.pkl", logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error for config filename: %v", err)
	}
}

func TestEnforcePklFilenameInvalid(t *testing.T) {
	line := "amends \"package://schema.kdeps.com/core@0.0.0#/Workflow.pkl\""
	// wrong actual file name
	if err := EnforcePklFilename(t.Context(), line, "/tmp/other.pkl", logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for mismatched filename")
	}

	// invalid pkl reference
	badLine := "amends \"package://schema.kdeps.com/core@0.0.0#/Unknown.pkl\""
	if err := EnforcePklFilename(t.Context(), badLine, "/tmp/foo.pkl", logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for unknown pkl file")
	}
}

func TestCompareVersions_Basic(t *testing.T) {
	if c, _ := compareVersions("1.2.3", "1.2.3", logging.NewTestLogger()); c != 0 {
		t.Fatalf("expected equal version compare = 0, got %d", c)
	}
	if c, _ := compareVersions("0.9", "1.0", logging.NewTestLogger()); c != -1 {
		t.Fatalf("expected older version -1, got %d", c)
	}
	if c, _ := compareVersions("2.0", "1.5", logging.NewTestLogger()); c != 1 {
		t.Fatalf("expected newer version 1, got %d", c)
	}
}

// createFiles helper creates nested files and dirs on provided fs.
func createFiles(t *testing.T, fsys afero.Fs, paths []string) {
	for _, p := range paths {
		dir := filepath.Dir(p)
		if err := fsys.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := afero.WriteFile(fsys, p, []byte("data"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

func TestEnforceFolderStructure_Happy(t *testing.T) {
	fsys := afero.NewOsFs()
	tmpDir := t.TempDir()

	// required layout
	createFiles(t, fsys, []string{
		filepath.Join(tmpDir, "workflow.pkl"),
		filepath.Join(tmpDir, "resources", "foo.pkl"),
		filepath.Join(tmpDir, "data", "agent", "1.0", "file.txt"),
	})

	if err := EnforceFolderStructure(fsys, t.Context(), tmpDir, logging.NewTestLogger()); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	_ = schema.SchemaVersion(t.Context())
}

func TestEnforceFolderStructure_BadExtraDir(t *testing.T) {
	fsys := afero.NewOsFs()
	tmpDir := t.TempDir()

	createFiles(t, fsys, []string{
		filepath.Join(tmpDir, "workflow.pkl"),
		filepath.Join(tmpDir, "resources", "foo.pkl"),
		filepath.Join(tmpDir, "extras", "bad.txt"),
	})

	if err := EnforceFolderStructure(fsys, context.Background(), tmpDir, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for unexpected folder")
	}

	_ = schema.SchemaVersion(t.Context())
}

func TestEnforcePklTemplateAmendsRules(t *testing.T) {
	fsys := afero.NewOsFs()
	tmp := t.TempDir()
	validFile := filepath.Join(tmp, "workflow.pkl")
	content := "amends \"package://schema.kdeps.com/core@" + schema.SchemaVersion(t.Context()) + "#/Workflow.pkl\"\n"
	if err := afero.WriteFile(fsys, validFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := EnforcePklTemplateAmendsRules(fsys, validFile, t.Context(), logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	invalidFile := filepath.Join(tmp, "bad.pkl")
	if err := afero.WriteFile(fsys, invalidFile, []byte("invalid line\n"), 0o644); err != nil {
		t.Fatalf("write2: %v", err)
	}
	if err := EnforcePklTemplateAmendsRules(fsys, invalidFile, t.Context(), logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for bad amends line")
	}
}

func TestEnforcePklVersionComparisons(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := t.Context()
	ver := schema.SchemaVersion(ctx)

	lineSame := "amends \"package://schema.kdeps.com/core@" + ver + "#/Workflow.pkl\""
	if err := EnforcePklVersion(ctx, lineSame, "file.pkl", ver, logger); err != nil {
		t.Fatalf("unexpected error for same version: %v", err)
	}

	lower := "0.0.1"
	lineLower := "amends \"package://schema.kdeps.com/core@" + lower + "#/Workflow.pkl\""
	if err := EnforcePklVersion(ctx, lineLower, "file.pkl", ver, logger); err != nil {
		t.Fatalf("unexpected error for lower version: %v", err)
	}

	higher := "999.999.999"
	lineHigher := "amends \"package://schema.kdeps.com/core@" + higher + "#/Workflow.pkl\""
	if err := EnforcePklVersion(ctx, lineHigher, "file.pkl", ver, logger); err != nil {
		t.Fatalf("unexpected error for higher version: %v", err)
	}

	bad := "amends \"package://schema.kdeps.com/core#/Workflow.pkl\"" // missing @version
	if err := EnforcePklVersion(ctx, bad, "file.pkl", ver, logger); err == nil {
		t.Fatalf("expected error for malformed line")
	}
}

func TestEnforceResourceRunBlock(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	fileOne := filepath.Join(dir, "single.pkl")
	contentSingle := "Chat {\n}" // one run block
	_ = afero.WriteFile(fs, fileOne, []byte(contentSingle), 0o644)

	if err := EnforceResourceRunBlock(t.Context(), fs, fileOne, logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error for single run block: %v", err)
	}

	fileMulti := filepath.Join(dir, "multi.pkl")
	contentMulti := "Chat {\n}\nPython {\n}" // two run blocks
	_ = afero.WriteFile(fs, fileMulti, []byte(contentMulti), 0o644)

	if err := EnforceResourceRunBlock(t.Context(), fs, fileMulti, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for multiple run blocks, got nil")
	}
}

func TestCompareVersions(t *testing.T) {
	logger := logging.NewTestLogger()

	tests := []struct {
		name     string
		v1, v2   string
		expected int
		wantErr  bool
	}{
		{"equal versions", "1.2.3", "1.2.3", 0, false},
		{"v1 greater patch", "1.2.4", "1.2.3", 1, false},
		{"v1 greater minor", "1.3.0", "1.2.9", 1, false},
		{"v1 less major", "1.2.3", "2.0.0", -1, false},
		{"different length v1 longer", "1.2.3.1", "1.2.3", 1, false},
		{"different length v2 longer", "1.2", "1.2.0.1", -1, false},
		{"invalid v1 format", "1.2.x", "1.2.0", 0, true},
		{"invalid v2 format", "1.2.0", "1.2.x", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := compareVersions(tc.v1, tc.v2, logger)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestCompareVersionsAdditional(t *testing.T) {
	logger := logging.NewTestLogger()
	tests := []struct {
		name   string
		v1, v2 string
		want   int
	}{
		{"equal", "1.2.3", "1.2.3", 0},
		{"v1< v2", "0.9", "1.0", -1},
		{"v1>v2", "2.0", "1.5", 1},
		{"different lengths", "1.2.3", "1.2", 1},
	}
	for _, tc := range tests {
		got, err := compareVersions(tc.v1, tc.v2, logger)
		assert.NoError(t, err)
		assert.Equal(t, tc.want, got, tc.name)
	}
}

func TestEnforcePklTemplateAmendsRules_MultipleAmends(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a test file with multiple amends statements
	content := `amends "package://schema.kdeps.com/core@0.3.1-dev#/Resource.pkl"
amends "package://schema.kdeps.com/core@0.3.1-dev#/Utils.pkl"

import "pkl:json"
import "pkl:math"

ActionID = "testResource"
Name = "Test Resource"
Description = "A test resource with multiple amends"
Category = ""

Run {
  Exec {
    Command = "echo test"
  }
}
`
	filePath := "/test.pkl"
	err := afero.WriteFile(fs, filePath, []byte(content), 0o644)
	require.NoError(t, err)

	// Test that validation passes with multiple amends statements
	err = EnforcePklTemplateAmendsRules(fs, filePath, t.Context(), logger)
	assert.NoError(t, err, "Validation should pass with multiple amends statements")
}

func TestEnforcePklTemplateAmendsRules_InvalidAmends(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a test file with one valid and one invalid amends statement
	content := `amends "package://schema.kdeps.com/core@0.3.1-dev#/Resource.pkl"
amends "package://invalid.com/core@0.3.1-dev#/Invalid.pkl"

ActionID = "testResource"
Name = "Test Resource"
Description = "A test resource with invalid amends"
Category = ""

Run {
  Exec {
    Command = "echo test"
  }
}
`
	filePath := "/test.pkl"
	err := afero.WriteFile(fs, filePath, []byte(content), 0o644)
	require.NoError(t, err)

	// Test that validation fails with invalid amends statement
	err = EnforcePklTemplateAmendsRules(fs, filePath, t.Context(), logger)
	assert.Error(t, err, "Validation should fail with invalid amends statement")
	assert.Contains(t, err.Error(), "schema URL validation failed")
}

func TestEnforcePklTemplateAmendsRules_NoAmends(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a test file with no amends statements
	content := `import "pkl:json"

ActionID = "testResource"
Name = "Test Resource"
Description = "A test resource with no amends"
Category = ""

Run {
  Exec {
    Command = "echo test"
  }
}
`
	filePath := "/test.pkl"
	err := afero.WriteFile(fs, filePath, []byte(content), 0o644)
	require.NoError(t, err)

	// Test that validation fails with no amends statements
	err = EnforcePklTemplateAmendsRules(fs, filePath, t.Context(), logger)
	assert.Error(t, err, "Validation should fail with no amends statements")
	assert.Contains(t, err.Error(), "no valid 'amends' line found")
}
