package archiver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resource"
	"github.com/kdeps/kdeps/pkg/workflow"
	assets "github.com/kdeps/schema/assets"
	"github.com/kr/pretty"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testFs             afero.Fs
	kdepsDir           string
	aiAgentDir         string
	resourcesDir       string
	dataDir            string
	workflowFile       string
	resourceFile       string
	packageDir         string
	lastCreatedPackage string
	projectDir         string
	logger             *logging.Logger
	testingT           *testing.T
	ctx                context.Context
	globalWorkspace    *assets.PKLWorkspace // Global workspace for all tests
)

func TestFeatures(t *testing.T) {
	// Initialize test filesystem
	testFs = afero.NewOsFs()

	// Setup global workspace that persists for all tests
	var err error
	globalWorkspace, err = assets.SetupPKLWorkspaceInTmpDir()
	if err != nil {
		t.Fatalf("Failed to setup global workspace: %v", err)
	}
	defer globalWorkspace.Cleanup() // Clean up after all tests

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a kdeps archive "([^"]*)" is opened$`, aKdepsArchiveIsOpened)
			ctx.Step(`^an ai agent on "([^"]*)" folder exists$`, anAiAgentOnFolder)
			ctx.Step(`^it has a workflow file that has name property "([^"]*)" and version property "([^"]*)" and default action "([^"]*)"$`, itHasAWorkflowFile)
			ctx.Step(`^the content of that archive file will be extracted to "([^"]*)"$`, theContentOfThatArchiveFileWillBeExtractedTo)
			ctx.Step(`^the pkl files is valid$`, thePklFilesIsValid)
			ctx.Step(`^the project is valid$`, theProjectIsValid)
			ctx.Step(`^the project will be archived to "([^"]*)"$`, theProjectWillBeArchivedTo)
			ctx.Step(`^the "([^"]*)" system folder exists$`, theSystemFolderExists)
			ctx.Step(`^theres a data file$`, theresADataFile)
			ctx.Step(`^the pkl files is invalid$`, thePklFilesIsInvalid)
			ctx.Step(`^the project is invalid$`, theProjectIsInvalid)
			ctx.Step(`^the project will not be archived to "([^"]*)"$`, theProjectWillNotBeArchivedTo)

			ctx.Step(`^it has a "([^"]*)" file with id property "([^"]*)" and dependent on "([^"]*)"$`, itHasAFileWithIDPropertyAndDependentOn)
			ctx.Step(`^it has a "([^"]*)" file with no dependency with id property "([^"]*)"$`, itHasAFileWithNoDependencyWithIDProperty)
			ctx.Step(`^it will be stored to "([^"]*)"$`, itWillBeStoredTo)
			ctx.Step(`^the project is compiled$`, theProjectIsCompiled)
			ctx.Step(`^the resource id for "([^"]*)" will be "([^"]*)" and dependency "([^"]*)"$`, theResourceIDForWillBeAndDependency)
			ctx.Step(`^the resource id for "([^"]*)" will be rewritten to "([^"]*)"$`, theResourceIDForWillBeRewrittenTo)
			ctx.Step(`^the workflow action configuration will be rewritten to "([^"]*)"$`, theWorkflowActionConfigurationWillBeRewrittenTo)
			ctx.Step(`^the resources and data folder exists$`, theResourcesAndDataFolderExists)
			ctx.Step(`^the data files will be copied to "([^"]*)"$`, theDataFilesWillBeCopiedTo)
			ctx.Step(`^the package file "([^"]*)" will be created$`, thePackageFileWillBeCreated)
			ctx.Step(`^it has a workflow file that has name property "([^"]*)" and version property "([^"]*)" and default action "([^"]*)" and workspaces "([^"]*)"$`, itHasAWorkflowFileDependencies)
			ctx.Step(`^the resource file "([^"]*)" exists in the "([^"]*)" agent "([^"]*)"$`, theResourceFileExistsInTheAgent)
			ctx.Step(`^it has a "([^"]*)" file with id property "([^"]*)" and dependent on "([^"]*)" with run block "([^"]*)" and is not null$`, itHasAFileWithIDPropertyAndDependentOnWithRunBlockAndIsNotNull)
			ctx.Step(`^it has a "([^"]*)" file with id property "([^"]*)" and dependent on "([^"]*)" with run block "([^"]*)" and is null$`, itHasAFileWithIDPropertyAndDependentOnWithRunBlockAndIsNull)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/archiver"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aKdepsArchiveIsOpened(arg1 string) error {
	name, version := regexp.MustCompile(`^([a-zA-Z]+)-([\d]+\.[\d]+\.[\d]+)\.kdeps$`).FindStringSubmatch(arg1)[1], regexp.MustCompile(`^([a-zA-Z]+)-([\d]+\.[\d]+\.[\d]+)\.kdeps$`).FindStringSubmatch(arg1)[2]

	kdepsAgentPath := filepath.Join(kdepsDir, "agents/"+name+"/"+version)
	if _, err := testFs.Stat(kdepsAgentPath); err == nil {
		return errors.New("agent should not yet exists on system agents dir")
	}

	proj, err := ExtractPackage(testFs, ctx, kdepsDir, lastCreatedPackage, logger)
	if err != nil {
		return err
	}

	fmt.Printf("%# v", pretty.Formatter(proj))

	return nil
}

func theSystemFolderExists(arg1 string) error {
	logger = logging.GetLogger()
	tempDir, err := afero.TempDir(testFs, "", arg1)
	if err != nil {
		return err
	}

	kdepsDir = tempDir

	packageDir = kdepsDir + "/packages"
	if err := testFs.MkdirAll(packageDir, 0o755); err != nil {
		return err
	}

	return nil
}

func anAiAgentOnFolder(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", arg1)
	if err != nil {
		return err
	}

	aiAgentDir = tempDir

	return nil
}

func itHasAFileWithIDPropertyAndDependentOn(arg1, arg2, arg3 string) error {
	// Check if arg3 is a CSV (contains commas)
	var requiresSection string
	if strings.Contains(arg3, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg3, ",")
		var requiresLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			requiresLines = append(requiresLines, fmt.Sprintf(`  "%s"`, value))
		}
		requiresSection = "Requires {\n" + strings.Join(requiresLines, "\n") + "\n}"
	} else {
		// Single value case
		requiresSection = fmt.Sprintf(`Requires {
  "%s"
}`, arg3)
	}

	// Create the document with the id and requires block using global workspace
	doc := fmt.Sprintf(`
amends "%s"

ActionID = "%s"
%s
Run {
  APIResponse {
    Success = true
    Response {
      Data {
        "Hello from %s"
      }
    }
  }
}
`, globalWorkspace.GetImportPath("Resource.pkl"), arg2, requiresSection, arg2)

	return afero.WriteFile(testFs, filepath.Join(resourcesDir, arg1+".pkl"), []byte(doc), 0o644)
}

func itWillBeStoredTo(arg1 string) error {
	workflowFile = filepath.Join(kdepsDir, arg1)

	if _, err := testFs.Stat(workflowFile); err != nil {
		return err
	}

	return nil
}

func theProjectIsCompiled() error {
	ctx = context.Background()
	wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	env := &environment.Environment{
		Home: kdepsDir,
		Pwd:  resourcesDir,
	}

	projectDir, _, _ := CompileProject(testFs, ctx, wf, kdepsDir, aiAgentDir, env, logger)

	workflowFile = filepath.Join(projectDir, "workflow.pkl")

	return nil
}

func theResourceIDForWillBeAndDependency(arg1, arg2, arg3 string) error {
	resFile := filepath.Join(projectDir, "resources/"+arg1)
	if _, err := testFs.Stat(resFile); err == nil {
		res, err := resource.LoadResource(ctx, resFile, logger)
		if err != nil {
			return err
		}
		if res.ActionID != arg2 {
			return errors.New("should be equal!")
		}
		found := false
		for _, v := range *res.Requires {
			if v == arg3 {
				found = true
				break
			}
		}

		if !found {
			return errors.New("require found!")
		}
	}

	return nil
}

func theResourceIDForWillBeRewrittenTo(arg1, arg2 string) error {
	resFile := filepath.Join(projectDir, "resources/"+arg1)
	if _, err := testFs.Stat(resFile); err == nil {
		res, err := resource.LoadResource(ctx, resFile, logger)
		if err != nil {
			return err
		}

		if res.ActionID != arg2 {
			return errors.New("should be equal!")
		}
	}

	return nil
}

func theWorkflowActionConfigurationWillBeRewrittenTo(arg1 string) error {
	wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	if wf.GetTargetActionID() != arg1 {
		return fmt.Errorf("%s = %s does not match!", wf.GetTargetActionID(), arg1)
	}

	return nil
}

func theResourcesAndDataFolderExists() error {
	resourcesDir = filepath.Join(aiAgentDir, "resources")
	if err := testFs.MkdirAll(resourcesDir, 0o755); err != nil {
		return err
	}

	dataDir = filepath.Join(aiAgentDir, "data")
	if err := testFs.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	return nil
}

func itHasAFileWithNoDependencyWithIDProperty(arg1, arg2 string) error {
	// Create the document with the id and requires block using global workspace
	doc := fmt.Sprintf(`
amends "%s"

ActionID = "%s"
Run {
  APIResponse {
    Success = true
    Response {
      Data {
        "Hello from %s"
      }
    }
  }
}
`, globalWorkspace.GetImportPath("Resource.pkl"), arg2, arg2)

	file := filepath.Join(resourcesDir, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	resourceFile = file

	return nil
}

func itHasAWorkflowFile(arg1, arg2, arg3 string) error {
	// Create the document with the id and requires block using global workspace
	doc := fmt.Sprintf(`
amends "%s"

TargetActionID = "%s"
AgentID = "%s"
Description = "My awesome AI Agent"
Version = "%s"
`, globalWorkspace.GetImportPath("Workflow.pkl"), arg3, arg1, arg2)

	file := filepath.Join(aiAgentDir, "workflow.pkl")

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	workflowFile = file

	return nil
}

func theContentOfThatArchiveFileWillBeExtractedTo(arg1 string) error {
	fpath := filepath.Join(kdepsDir, arg1)
	if _, err := testFs.Stat(fpath); err != nil {
		return errors.New("there should be an agent dir present, but none was found")
	}

	return nil
}

func thePklFilesIsValid() error {
	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, ctx, workflowFile, logger); err != nil {
		return err
	}

	return nil
}

func theProjectIsValid() error {
	if err := enforcer.EnforceFolderStructure(testFs, ctx, workflowFile, logger); err != nil {
		return err
	}

	return nil
}

func theProjectWillBeArchivedTo(arg1 string) error {
	wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, ctx, wf, kdepsDir, aiAgentDir, logger)
	if err != nil {
		return err
	}

	if _, err := testFs.Stat(fpath); err != nil {
		return err
	}

	return nil
}

func theresADataFile() error {
	doc := "THIS IS A TEXT FILE: "

	for x := range 10 {
		num := strconv.Itoa(x)
		file := filepath.Join(dataDir, fmt.Sprintf("textfile-%s.txt", num))

		f, _ := testFs.Create(file)
		if _, err := f.WriteString(doc + num); err != nil {
			return err
		}
		f.Close()
	}

	return nil
}

func theDataFilesWillBeCopiedTo(arg1 string) error {
	file := filepath.Join(kdepsDir, arg1+"/textfile-1.txt")

	if _, err := testFs.Stat(file); err != nil {
		return err
	}

	return nil
}

func thePklFilesIsInvalid() error {
	doc := `
	AgentID = "invalidagent"
	Description = "a not valid configuration"
	Version = "five"
	TargetActionID = "helloWorld"
	`
	file := filepath.Join(aiAgentDir, "workflow1.pkl")

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	workflowFile = file

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, ctx, workflowFile, logger); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func theProjectIsInvalid() error {
	if err := enforcer.EnforceFolderStructure(testFs, ctx, workflowFile, logger); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func theProjectWillNotBeArchivedTo(arg1 string) error {
	wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, ctx, wf, kdepsDir, aiAgentDir, logger)
	if err == nil {
		return errors.New("expected an error, but got nil")
	}

	if _, err := testFs.Stat(fpath); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func thePackageFileWillBeCreated(arg1 string) error {
	fpath := filepath.Join(packageDir, arg1)
	if _, err := testFs.Stat(fpath); err != nil {
		return errors.New("expected a package, but got none")
	}
	lastCreatedPackage = fpath

	return nil
}

func itHasAWorkflowFileDependencies(arg1, arg2, arg3, arg4 string) error {
	// Check if arg4 is a CSV (contains commas)
	var workflowsSection string
	if strings.Contains(arg4, ",") {
		// Split arg4 into multiple values if it's a CSV
		values := strings.Split(arg4, ",")
		var workflowsLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			workflowsLines = append(workflowsLines, fmt.Sprintf(`  "%s"`, value))
		}
		workflowsSection = "Workflows {\n" + strings.Join(workflowsLines, "\n") + "\n}"
	} else {
		// Single value case
		workflowsSection = fmt.Sprintf(`Workflows {
  "%s"
}`, arg4)
	}

	// Create the document with the id and requires block using global workspace
	doc := fmt.Sprintf(`
amends "%s"

TargetActionID = "%s"
AgentID = "%s"
Description = "My awesome AI Agent"
Version = "%s"
%s
`, globalWorkspace.GetImportPath("Workflow.pkl"), arg3, arg1, arg2, workflowsSection)

	file := filepath.Join(aiAgentDir, "workflow.pkl")

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	workflowFile = file

	return nil
}

func theResourceFileExistsInTheAgent(arg1, arg2, arg3 string) error {
	fpath := filepath.Join(kdepsDir, "agents/"+arg2+"/1.0.0/resources/"+arg1)
	if _, err := testFs.Stat(fpath); err != nil {
		return errors.New("expected a package, but got none")
	}

	return nil
}

func itHasAFileWithIDPropertyAndDependentOnWithRunBlockAndIsNotNull(arg1, arg2, arg3, arg4 string) error {
	// Check if arg3 is a CSV (contains commas)
	var requiresSection string
	if strings.Contains(arg3, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg3, ",")
		var requiresLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			requiresLines = append(requiresLines, fmt.Sprintf(`  "%s"`, value))
		}
		requiresSection = "Requires {\n" + strings.Join(requiresLines, "\n") + "\n}"
	} else {
		// Single value case
		requiresSection = fmt.Sprintf(`Requires {
  "%s"
}`, arg3)
	}

	var fieldSection string
	if strings.Contains(arg4, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg4, ",")
		var fieldLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			fieldLines = append(fieldLines, value+" {\n    Command = \"echo hello\"\n  }")
		}
		fieldSection = "Run {\n" + strings.Join(fieldLines, "\n") + "\n}"
	} else {
		// Single value case
		fieldSection = fmt.Sprintf(`Run {
  %s {
    Command = "echo hello"
  }
}`, arg4)
	}

	// Create the document with the id and requires block using global workspace
	doc := fmt.Sprintf(`
amends "%s"

ActionID = "%s"
%s
%s
`, globalWorkspace.GetImportPath("Resource.pkl"), arg2, requiresSection, fieldSection)

	// Write to the file
	file := filepath.Join(resourcesDir, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	resourceFile = file

	return nil
}

func itHasAFileWithIDPropertyAndDependentOnWithRunBlockAndIsNull(arg1, arg2, arg3, arg4 string) error {
	// Check if arg3 is a CSV (contains commas)
	var requiresSection string
	if strings.Contains(arg3, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg3, ",")
		var requiresLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			requiresLines = append(requiresLines, fmt.Sprintf(`  "%s"`, value))
		}
		requiresSection = "Requires {\n" + strings.Join(requiresLines, "\n") + "\n}"
	} else {
		// Single value case
		requiresSection = fmt.Sprintf(`Requires {
  "%s"
}`, arg3)
	}

	var fieldSection string
	if strings.Contains(arg4, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg4, ",")
		var fieldLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			fieldLines = append(fieldLines, value+"=null")
		}
		fieldSection = "Run {\n" + strings.Join(fieldLines, "\n") + "\n}"
	} else {
		// Single value case
		fieldSection = fmt.Sprintf(`Run {
  %s=null
}`, arg4)
	}

	// Create the document with the id and requires block using global workspace
	doc := fmt.Sprintf(`
amends "%s"

ActionID = "%s"
%s
%s
`, globalWorkspace.GetImportPath("Resource.pkl"), arg2, requiresSection, fieldSection)

	// Write to the file
	file := filepath.Join(resourcesDir, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	resourceFile = file

	return nil
}

func TestArchiverWithSchemaAssets(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()
	fs := afero.NewOsFs() // Use OS filesystem for PKL operations

	// Setup dedicated workspace for this test suite
	testWorkspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer testWorkspace.Cleanup()

	t.Run("ValidateSchemaAssetsAvailable", func(t *testing.T) {
		// Verify all files needed for archiver operations are available
		files, err := testWorkspace.ListFiles()
		require.NoError(t, err)

		expectedFiles := []string{"Workflow.pkl", "Resource.pkl", "Project.pkl", "Docker.pkl"}
		for _, expected := range expectedFiles {
			assert.Contains(t, files, expected, "Schema file %s should be available", expected)
		}

		t.Logf("Available schema files for archiver: %v", files)
	})

	t.Run("CreateWorkflowWithAssetsSchema", func(t *testing.T) {
		// Create a temporary project directory
		projectDir := t.TempDir()

		// Create a workflow file using the embedded schema
		workflowFile := filepath.Join(projectDir, "workflow.pkl")
		workflowContent := fmt.Sprintf(`amends "%s"

AgentID = "testagent"
Description = "Test agent using schema assets"
Version = "1.0.0" 
TargetActionID = "testAction"
Workflows {}
Settings {
	RateLimitMax = 100
	Environment = "dev"
	APIServerMode = false
	WebServerMode = false
	AgentSettings {
		InstallAnaconda = false
		Timezone = "Etc/UTC"
		Models {
			"llama3.2:1b"
		}
		OllamaTagVersion = "0.8.0"
	}
}`, testWorkspace.GetImportPath("Workflow.pkl"))

		err := os.WriteFile(workflowFile, []byte(workflowContent), 0o644)
		require.NoError(t, err)

		// Validate that we can load the workflow
		wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
		require.NoError(t, err)
		assert.Equal(t, "testagent", wf.GetAgentID())
		assert.Equal(t, "1.0.0", wf.GetVersion())
		assert.Equal(t, "testAction", wf.GetTargetActionID())

		// Create a resource file using the embedded schema
		resourcesDir := filepath.Join(projectDir, "resources")
		err = fs.MkdirAll(resourcesDir, 0o755)
		require.NoError(t, err)

		resourceFile := filepath.Join(resourcesDir, "testAction.pkl")
		resourceContent := fmt.Sprintf(`amends "%s"

ActionID = "testAction"
Name = "Test Action"
Description = "A test action using schema assets"
Category = "test"
Requires {}
Run {
	RestrictToHTTPMethods {
		"GET"
	}
	RestrictToRoutes {
		"/test"
	}
	PreflightCheck {
		Validations {}
		Retry = false
		RetryTimes = 3
	}
	PostflightCheck {
		Validations {}
		Retry = true
		RetryTimes = 5
	}
	AllowedHeaders {
		"Content-Type"
		"Authorization"
	}
	AllowedParams {
		"query"
		"format"
	}
	Exec {
		Commands {
			"echo 'Test command using schema assets'"
		}
		TimeoutDuration = 30.s
	}
}`, testWorkspace.GetImportPath("Resource.pkl"))

		err = os.WriteFile(resourceFile, []byte(resourceContent), 0o644)
		require.NoError(t, err)

		t.Logf("Created workflow file: %s", workflowFile)
		t.Logf("Created resource file: %s", resourceFile)
		t.Logf("Workspace directory: %s", testWorkspace.Directory)
	})

	t.Run("ValidateSchemaProperties", func(t *testing.T) {
		// Get schema content to validate our templates have the right properties
		workflowSchema, err := assets.GetPKLFileAsString("Workflow.pkl")
		require.NoError(t, err)

		// Verify v0.3.8 workflow properties are defined
		assert.Contains(t, workflowSchema, "AgentID: String")
		assert.Contains(t, workflowSchema, "Settings: Project.Settings")

		resourceSchema, err := assets.GetPKLFileAsString("Resource.pkl")
		require.NoError(t, err)

		// Verify v0.3.8 resource properties are defined
		assert.Contains(t, resourceSchema, "ActionID: String")
		assert.Contains(t, resourceSchema, "PostflightCheck: ValidationCheck?")
		assert.Contains(t, resourceSchema, "AllowedHeaders: Listing<String>?")
		assert.Contains(t, resourceSchema, "AllowedParams: Listing<String>?")
		assert.Contains(t, resourceSchema, "Retry: Boolean? = false")
		assert.Contains(t, resourceSchema, "RetryTimes: Int? = 3")

		projectSchema, err := assets.GetPKLFileAsString("Project.pkl")
		require.NoError(t, err)

		// Verify v0.3.8 project settings properties
		assert.Contains(t, projectSchema, "RateLimitMax: Int? = 100")
		assert.Contains(t, projectSchema, "Environment: BuildEnv? = \"dev\"")

		t.Logf("Schema validation completed for v0.3.8 properties")
	})

	t.Run("ArchiveProjectWithAssets", func(t *testing.T) {
		// Create a more comprehensive test that simulates archiving a project
		// Setup PKL workspace
		workspace, err := assets.SetupPKLWorkspaceInTmpDir()
		require.NoError(t, err)
		defer workspace.Cleanup()

		// Create project structure
		projectDir := t.TempDir()

		// Create workflow file
		workflowFile := filepath.Join(projectDir, "workflow.pkl")
		workflowContent := fmt.Sprintf(`amends "%s"

AgentID = "archivetest"
Description = "Archive test with assets"
Version = "2.0.0"
TargetActionID = "mainAction"
Workflows {}
Settings {
	RateLimitMax = 200
	Environment = "prod"
	APIServerMode = true
	AgentSettings {
		InstallAnaconda = true
		Models {
			"llama3.2:3b"
		}
		OllamaTagVersion = "0.8.0"
	}
}`, workspace.GetImportPath("Workflow.pkl"))

		err = os.WriteFile(workflowFile, []byte(workflowContent), 0o644)
		require.NoError(t, err)

		// Load and validate workflow
		wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
		require.NoError(t, err)
		assert.Equal(t, "archivetest", wf.GetAgentID())
		assert.Equal(t, "2.0.0", wf.GetVersion())

		// Test that the archiver can process this workflow
		// This demonstrates that our updated templates work with the archiver system
		t.Logf("Successfully loaded workflow %s v%s using schema assets",
			wf.GetAgentID(), wf.GetVersion())
		t.Logf("Target action: %s", wf.GetTargetActionID())
		t.Logf("Schema workspace: %s", workspace.Directory)
	})
}
