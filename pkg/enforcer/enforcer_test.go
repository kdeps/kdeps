package enforcer_test

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
	"github.com/kdeps/schema/assets"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/enforcer"
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
	globalWorkspace     *assets.PKLWorkspace // Global workspace for all tests
	resourceValues      = `
ActionID = "helloWorld"
Name = "agentname"
Description = "description"
Category = "category"
`
	configValues = `
Mode = "docker"
DockerGPU = "cpu"
`
	workflowValues = `
AgentID = "myAgent"
Description = "My awesome AI Agent"
Version = "1.0.0"
TargetActionID = "helloWorld"
`
	testingT *testing.T
)

func init() {
	// Setup global PKL workspace once for all tests
	var err error
	globalWorkspace, err = assets.SetupPKLWorkspaceInTmpDir()
	if err != nil {
		panic(fmt.Sprintf("Failed to setup global PKL workspace: %v", err))
	}
}

func TestFeatures(t *testing.T) {
	defer globalWorkspace.Cleanup() // Clean up at the end of all tests

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

func theHomeDirectoryIs(_ string) error {
	tempDir, err := afero.TempDir(testFs, "", "")
	if err != nil {
		return err
	}

	homeDirPath = tempDir

	return nil
}

func theCurrentDirectoryIs(_ string) error {
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
	// Initialize evaluator for this test
	evaluator.TestSetup(nil)
	defer evaluator.TestTeardown(nil)

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
	// For assets-based paths, we need to handle domain validation differently
	// If the domain is not "kdeps.com", we should create an invalid amends line
	if arg1 != "kdeps.com" {
		// Create an invalid amends line for testing domain validation
		// Determine which type of file we're working with based on the current doc content
		if strings.Contains(doc, "Workflow.pkl") {
			doc = fmt.Sprintf(`amends "package://%s/core@0.4.4#/Workflow.pkl"
%s`, arg1, workflowValues)
		} else if strings.Contains(doc, "Resource.pkl") {
			doc = fmt.Sprintf(`amends "package://%s/core@0.4.4#/Resource.pkl"
%s`, arg1, resourceValues)
		} else {
			// Default to config file
			doc = fmt.Sprintf(`amends "package://%s/core@0.4.4#/Kdeps.pkl"
%s`, arg1, configValues)
		}
	} else {
		// For valid domain, replace the assets path with legacy schema URL
		// This simulates the old behavior for testing
		if strings.Contains(doc, "Workflow.pkl") {
			doc = fmt.Sprintf(`amends "package://schema.kdeps.com/core@0.4.4#/Workflow.pkl"
%s`, workflowValues)
		} else if strings.Contains(doc, "Resource.pkl") {
			doc = fmt.Sprintf(`amends "package://schema.kdeps.com/core@0.4.4#/Resource.pkl"
%s`, resourceValues)
		} else {
			// Default to config file
			doc = fmt.Sprintf(`amends "package://schema.kdeps.com/core@0.4.4#/Kdeps.pkl"
%s`, configValues)
		}
	}

	return nil
}

func itHaveAConfigAmendsLineOnTopOfTheFile() error {
	// Use assets to get the correct import path for the latest version
	assetsImportPath := fmt.Sprintf(`amends "%s"`, globalWorkspace.GetImportPath("Kdeps.pkl"))
	doc = fmt.Sprintf("%s\n%s", assetsImportPath, configValues)

	return nil
}

func itIsAnInvalidAgent() error {
	if err := enforcer.EnforceFolderStructure(ctx, testFs, agentPath, logger); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func itIsAValidAgent() error {
	if err := enforcer.EnforceFolderStructure(ctx, testFs, agentPath, logger); err != nil {
		return err
	}

	return nil
}

func itIsAnInvalidPklFile() error {
	if err := enforcer.EnforcePklTemplateAmendsRules(ctx, testFs, fileThatExist, logger); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func itIsAValidPklFile() error {
	// Initialize evaluator for this test
	evaluator.TestSetup(nil)
	defer evaluator.TestTeardown(nil)

	if err := enforcer.EnforcePklTemplateAmendsRules(ctx, testFs, fileThatExist, logger); err != nil {
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
	// Creating file (log output removed)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	fileThatExist = file

	return nil
}

func anAgentFolderExistsInTheCurrentDirectory(_ string) error {
	agentPath = currentDirPath + "/my-agent"
	if err := testFs.MkdirAll(agentPath, 0o755); err != nil {
		return err
	}
	// Agent path created (log output removed)

	return nil
}

func itDoesNotHaveAWorkflowAmendsLineOnTopOfTheFile() error {
	doc = workflowValues

	return nil
}

func itHaveAWorkflowAmendsLineOnTopOfTheFile() error {
	// Use assets to get the correct import path for the latest version
	assetsImportPath := fmt.Sprintf(`amends "%s"`, globalWorkspace.GetImportPath("Workflow.pkl"))
	doc = fmt.Sprintf("%s%s", assetsImportPath, workflowValues)

	return nil
}

func aFolderNamedExistsInThe(arg1, _ string) error {
	agentPath = currentDirPath + "/my-agent"
	subfolderPath := agentPath + "/" + arg1
	if err := testFs.MkdirAll(subfolderPath, 0o755); err != nil {
		return err
	}
	// Agent path created (log output removed)

	return nil
}

// Resource steps

func itHaveAResourceAmendsLineOnTopOfTheFile() error {
	// Use assets to get the correct import path for the latest version
	assetsImportPath := fmt.Sprintf(`amends "%s"`, globalWorkspace.GetImportPath("Resource.pkl"))
	doc = fmt.Sprintf("%s%s", assetsImportPath, resourceValues)

	return nil
}

func itDoesNotHaveAResourceAmendsLineOnTopOfTheFile() error {
	doc = resourceValues

	return nil
}

func TestEnforcePklVersion(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()
	schemaVersion := "1.2.3"

	goodLine := "amends \"package://schema.kdeps.com/core@1.2.3#/Kdeps.pkl\""
	require.NoError(t, enforcer.EnforcePklVersion(ctx, goodLine, "file.pkl", schemaVersion, logger))

	// lower version should warn but not error
	lowLine := "amends \"package://schema.kdeps.com/core@1.0.0#/Kdeps.pkl\""
	require.NoError(t, enforcer.EnforcePklVersion(ctx, lowLine, "file.pkl", schemaVersion, logger))

	// higher version also no error
	highLine := "amends \"package://schema.kdeps.com/core@2.0.0#/Kdeps.pkl\""
	require.NoError(t, enforcer.EnforcePklVersion(ctx, highLine, "file.pkl", schemaVersion, logger))

	// invalid version format should error
	badLine := "amends \"package://schema.kdeps.com/core@1.x#/Kdeps.pkl\""
	require.Error(t, enforcer.EnforcePklVersion(ctx, badLine, "file.pkl", schemaVersion, logger))
}

func TestEnforcePklFilename(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()
	ver := schema.Version(ctx)

	// Good configuration .kdeps.pkl
	lineCfg := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Kdeps.pkl\"", ver)
	require.NoError(t, enforcer.EnforcePklFilename(ctx, lineCfg, "/path/to/.kdeps.pkl", logger))

	// Good workflow.pkl
	lineWf := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Workflow.pkl\"", ver)
	require.NoError(t, enforcer.EnforcePklFilename(ctx, lineWf, "/some/workflow.pkl", logger))

	// Resource.pkl must not have those filenames
	lineResource := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Resource.pkl\"", ver)
	require.NoError(t, enforcer.EnforcePklFilename(ctx, lineResource, "/path/to/resources/custom.pkl", logger))

	// Invalid file extension for config
	err := enforcer.EnforcePklFilename(ctx, lineCfg, "/path/to/wrongname.txt", logger)
	require.Error(t, err)

	// Resource.pkl with forbidden filename
	err = enforcer.EnforcePklFilename(ctx, lineResource, "/path/to/.kdeps.pkl", logger)
	require.Error(t, err)

	// Unknown pkl filename in amends line -> expect error
	unknownLine := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Unknown.pkl\"", ver)
	err = enforcer.EnforcePklFilename(ctx, unknownLine, "/path/to/unknown.pkl", logger)
	require.Error(t, err)
}

func TestEnforcePklFilenameValid(t *testing.T) {
	ctx := context.Background()
	ver := schema.Version(ctx)

	line := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Workflow.pkl\"", ver)
	workflowPath := filepath.Join(t.TempDir(), "workflow.pkl")
	if err := enforcer.EnforcePklFilename(ctx, line, workflowPath, logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error for valid filename: %v", err)
	}

	lineConf := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Kdeps.pkl\"", ver)
	configPath := filepath.Join(t.TempDir(), ".kdeps.pkl")
	if err := enforcer.EnforcePklFilename(ctx, lineConf, configPath, logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error for config filename: %v", err)
	}
}

func TestEnforcePklFilenameInvalid(t *testing.T) {
	ctx := context.Background()
	ver := schema.Version(ctx)

	line := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Workflow.pkl\"", ver)
	// wrong actual file name
	otherPath := filepath.Join(t.TempDir(), "other.pkl")
	if err := enforcer.EnforcePklFilename(ctx, line, otherPath, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for mismatched filename")
	}

	// invalid pkl reference
	badLine := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Unknown.pkl\"", ver)
	fooPath := filepath.Join(t.TempDir(), "foo.pkl")
	if err := enforcer.EnforcePklFilename(ctx, badLine, fooPath, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for unknown pkl file")
	}
}

func TestCompareVersions_Basic(t *testing.T) {
	if c, _ := enforcer.CompareVersions("1.2.3", "1.2.3", logging.NewTestLogger()); c != 0 {
		t.Fatalf("expected equal version compare = 0, got %d", c)
	}
	if c, _ := enforcer.CompareVersions("0.9", "1.0", logging.NewTestLogger()); c != -1 {
		t.Fatalf("expected older version -1, got %d", c)
	}
	if c, _ := enforcer.CompareVersions("2.0", "1.5", logging.NewTestLogger()); c != 1 {
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
	ctx := context.Background()
	fsys := afero.NewMemMapFs()
	tmpDir := t.TempDir()

	// required layout
	createFiles(t, fsys, []string{
		filepath.Join(tmpDir, "workflow.pkl"),
		filepath.Join(tmpDir, "resources", "foo.pkl"),
		filepath.Join(tmpDir, "data", "agent", "1.0", "file.txt"),
	})

	if err := enforcer.EnforceFolderStructure(ctx, fsys, tmpDir, logging.NewTestLogger()); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	_ = schema.Version(ctx)
}

func TestEnforceFolderStructure_BadExtraDir(t *testing.T) {
	ctx := context.Background()
	fsys := afero.NewMemMapFs()
	tmpDir := t.TempDir()

	createFiles(t, fsys, []string{
		filepath.Join(tmpDir, "workflow.pkl"),
		filepath.Join(tmpDir, "resources", "foo.pkl"),
		filepath.Join(tmpDir, "extras", "bad.txt"),
	})

	if err := enforcer.EnforceFolderStructure(ctx, fsys, tmpDir, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for unexpected folder")
	}

	_ = schema.Version(ctx)
}

func TestEnforcePklTemplateAmendsRules(t *testing.T) {
	fsys := afero.NewOsFs()
	tmp := t.TempDir()
	validFile := filepath.Join(tmp, "workflow.pkl")
	content := "amends \"package://schema.kdeps.com/core@" + schema.Version(ctx) + "#/Workflow.pkl\"\n"
	if err := afero.WriteFile(fsys, validFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := enforcer.EnforcePklTemplateAmendsRules(ctx, fsys, validFile, logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	invalidFile := filepath.Join(tmp, "bad.pkl")
	if err := afero.WriteFile(fsys, invalidFile, []byte("invalid line\n"), 0o644); err != nil {
		t.Fatalf("write2: %v", err)
	}
	if err := enforcer.EnforcePklTemplateAmendsRules(ctx, fsys, invalidFile, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for bad amends line")
	}
}

func TestEnforcePklVersionComparisons(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()
	ver := schema.Version(ctx)

	lineSame := "amends \"package://schema.kdeps.com/core@" + ver + "#/Workflow.pkl\""
	if err := enforcer.EnforcePklVersion(ctx, lineSame, "file.pkl", ver, logger); err != nil {
		t.Fatalf("unexpected error for same version: %v", err)
	}

	lower := "0.0.1"
	lineLower := "amends \"package://schema.kdeps.com/core@" + lower + "#/Workflow.pkl\""
	if err := enforcer.EnforcePklVersion(ctx, lineLower, "file.pkl", ver, logger); err != nil {
		t.Fatalf("unexpected error for lower version: %v", err)
	}

	higher := "999.999.999"
	lineHigher := "amends \"package://schema.kdeps.com/core@" + higher + "#/Workflow.pkl\""
	if err := enforcer.EnforcePklVersion(ctx, lineHigher, "file.pkl", ver, logger); err != nil {
		t.Fatalf("unexpected error for higher version: %v", err)
	}

	bad := "amends \"package://schema.kdeps.com/core#/Workflow.pkl\"" // missing @version
	if err := enforcer.EnforcePklVersion(ctx, bad, "file.pkl", ver, logger); err == nil {
		t.Fatalf("expected error for malformed line")
	}
}

func TestEnforceResourceRunBlock(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	fileOne := filepath.Join(dir, "single.pkl")
	contentSingle := "chat {\n}" // one run block
	_ = afero.WriteFile(fs, fileOne, []byte(contentSingle), 0o644)

	if err := enforcer.EnforceResourceRunBlock(ctx, fs, fileOne, logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error for single run block: %v", err)
	}

	fileMulti := filepath.Join(dir, "multi.pkl")
	contentMulti := "chat {\n}\npython {\n}" // two run blocks
	_ = afero.WriteFile(fs, fileMulti, []byte(contentMulti), 0o644)

	if err := enforcer.EnforceResourceRunBlock(ctx, fs, fileMulti, logging.NewTestLogger()); err == nil {
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
			result, err := enforcer.CompareVersions(tc.v1, tc.v2, logger)
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
		got, err := enforcer.CompareVersions(tc.v1, tc.v2, logger)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, tc.name)
	}
}
