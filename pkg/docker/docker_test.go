package docker

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/workflow"
	"github.com/kdeps/schema/gen/kdeps"
	wfPkl "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

var (
	testFs                    = afero.NewOsFs()
	testingT                  *testing.T
	homeDirPath               string
	kdepsDir                  string
	agentDir                  string
	APIServerMode             bool
	ctx                       context.Context
	packageFile               string
	hostPort                  string
	hostIP                    string
	runDir                    string
	containerName             string
	cName                     string
	pkgProject                *archiver.KdepsPackage
	compiledProjectDir        string
	currentDirPath            string
	systemConfigurationFile   string
	gpuType                   string
	logger                    *logging.Logger
	environ                   *environment.Environment
	cli                       *client.Client
	systemConfiguration       *kdeps.Kdeps
	workflowConfigurationFile string
	workflowConfiguration     *wfPkl.Workflow
	packageDir                string
	aiAgentDir                string
	resourceFile              string
	workflowFile              string
	lastCreatedPackage        string
	resourcesDir              string
	dataDir                   string
	projectDir                string
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a "([^"]*)" system configuration file with dockerGPU "([^"]*)" and Mode "([^"]*)" is defined in the "([^"]*)" directory$`, aSystemConfigurationFile)
			ctx.Step(`^a valid ai-agent "([^"]*)" is present in the "([^"]*)" directory$`, aValidAiagentIsPresentInTheDirectory)
			ctx.Step(`^"([^"]*)" directory exists in the "([^"]*)" directory$`, directoryExistsInTheDirectory)
			ctx.Step(`^it should create the Dockerfile for the agent in the "([^"]*)" directory with package "([^"]*)" and copy the kdeps package to the "([^"]*)" directory$`, itShouldCreateTheDockerfile)
			ctx.Step(`^it should run the container build step for "([^"]*)"$`, itShouldRunTheContainerBuildStepFor)
			ctx.Step(`^it should start the container "([^"]*)"$`, itShouldStartTheContainer)
			ctx.Step(`^kdeps open the package "([^"]*)" and extract it\'s content to the agents directory$`, kdepsOpenThePackage)
			ctx.Step(`^the valid ai-agent "([^"]*)" has been compiled as "([^"]*)" in the packages directory$`, theValidAiagentHas)
			ctx.Step(`^a valid ai-agent "([^"]*)" is present in the "([^"]*)" directory with packages "([^"]*)" and models "([^"]*)"$`, aValidAiagentIsPresentInTheDirectory)
			ctx.Step(`^the command should be run "([^"]*)" action by default$`, theCommandShouldBeRunActionByDefault)
			ctx.Step(`^the Docker entrypoint should be "([^"]*)"$`, theDockerEntrypointShouldBe)
			ctx.Step(`^it will install the model "([^"]*)" defined in the workflow configuration$`, itWillInstallTheModels)
			ctx.Step(`^kdeps will check the presence of the "([^"]*)" file$`, kdepsWillCheckThePresenceOfTheFile)
			ctx.Step(`^the system folder exists "([^"]*)"$`, theSystemFolderExists)
			ctx.Step(`^an ai-agent is present on folder "([^"]*)"$`, anAiAgentOnFolder)
			ctx.Step(`^it has a file with ID property and dependent on "([^"]*)" "([^"]*)" "([^"]*)"$`, itHasAFileWithIDPropertyAndDependentOn)
			ctx.Step(`^it will be stored to "([^"]*)"$`, itWillBeStoredTo)
			ctx.Step(`^it has a file with no dependency with ID property "([^"]*)" "([^"]*)"$`, itHasAFileWithNoDependencyWithIDProperty)
			ctx.Step(`^it has a workflow file "([^"]*)" "([^"]*)" "([^"]*)"$`, itHasAWorkflowFile)
			ctx.Step(`^the content of that archive file will be extracted to "([^"]*)"$`, theContentOfThatArchiveFileWillBeExtractedTo)
			ctx.Step(`^the pkl files is valid$`, thePklFilesIsValid)
			ctx.Step(`^the project is valid$`, theProjectIsValid)
			ctx.Step(`^the project will be archived to "([^"]*)"$`, theProjectWillBeArchivedTo)
			ctx.Step(`^there's a data file$`, theresADataFile)
			ctx.Step(`^the data files will be copied to "([^"]*)"$`, theDataFilesWillBeCopiedTo)
			ctx.Step(`^the pkl files is invalid$`, thePklFilesIsInvalid)
			ctx.Step(`^the project is invalid$`, theProjectIsInvalid)
			ctx.Step(`^the project will not be archived to "([^"]*)"$`, theProjectWillNotBeArchivedTo)
			ctx.Step(`^the package file will be created "([^"]*)"$`, thePackageFileWillBeCreated)
			ctx.Step(`^it has a workflow file dependencies "([^"]*)" "([^"]*)" "([^"]*)" "([^"]*)"$`, itHasAWorkflowFileDependencies)
			ctx.Step(`^the resource file exists in the agent "([^"]*)" "([^"]*)" "([^"]*)"$`, theResourceFileExistsInTheAgent)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/docker"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aSystemConfigurationFile(arg1, arg2, arg3, arg4 string) error {
	ctx = context.Background()
	logger = logging.GetLogger()

	env := &environment.Environment{
		Home:           homeDirPath,
		Pwd:            currentDirPath,
		NonInteractive: "1",
		DockerMode:     "1",
	}

	environ, err := environment.NewEnvironment(testFs, env)
	if err != nil {
		return err
	}

	systemConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Kdeps.pkl"

Mode = "%s"
DockerGPU = "%s"
`, schema.SchemaVersion(ctx), arg3, arg2)

	var filePath string

	if arg4 == "HOME" {
		filePath = homeDirPath
	} else {
		filePath = currentDirPath
	}

	systemConfigurationFile = filepath.Join(filePath, arg1)
	// Write the heredoc content to the file
	err = afero.WriteFile(testFs, systemConfigurationFile, []byte(systemConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	systemConfigurationFile, err := cfg.FindConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return err
	}

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, ctx, systemConfigurationFile, logger); err != nil {
		return err
	}

	syscfg, err := cfg.LoadConfiguration(testFs, ctx, systemConfigurationFile, logger)
	if err != nil {
		return err
	}

	systemConfiguration = syscfg

	return nil
}

func aValidAiagentIsPresentInTheDirectory(arg1, arg2, arg3, arg4 string) error {
	var pkgSection string
	if strings.Contains(arg3, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg3, ",")
		var pkgLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pkgLines = append(pkgLines, fmt.Sprintf(`"%s"`, value))
		}
		pkgSection = "packages {\n" + strings.Join(pkgLines, "\n") + "\n}"
	} else {
		// Single value case
		pkgSection = fmt.Sprintf(`
packages {
  "%s"
}`, arg3)
	}

	var modelSection string
	if strings.Contains(arg4, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg4, ",")
		var modelLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			modelLines = append(modelLines, fmt.Sprintf(`"%s"`, value))
		}
		modelSection = "models {\n" + strings.Join(modelLines, "\n") + "\n}"
	} else {
		// Single value case
		modelSection = fmt.Sprintf(`models {
  "%s"
}`, arg4)
	}

	workflowConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

AgentID = "%s"
Description = "AI Agent X"
TargetActionID = "%s"
Settings {
  APIServerMode = false
  AgentSettings {
    %s
    %s
  }
}
`, schema.SchemaVersion(ctx), arg1, arg1, pkgSection, modelSection)

	var filePath string

	if arg2 == "HOME" {
		filePath = filepath.Join(homeDirPath, arg1)
	} else {
		filePath = filepath.Join(currentDirPath, arg1)
	}

	if err := testFs.MkdirAll(filePath, 0o777); err != nil {
		return err
	}

	agentDir = filePath

	workflowConfigurationFile = filepath.Join(filePath, "workflow.pkl")
	err := afero.WriteFile(testFs, workflowConfigurationFile, []byte(workflowConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	resourcesDir := filepath.Join(filePath, "resources")
	if err := testFs.MkdirAll(resourcesDir, 0o777); err != nil {
		return err
	}

	resourceConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ActionID = "%s"
Description = "An action from agent %s"
	`, schema.SchemaVersion(ctx), arg1, arg1)

	resourceConfigurationFile := filepath.Join(resourcesDir, arg1+".pkl")
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	dataDir := filepath.Join(filePath, "data")
	if err := testFs.MkdirAll(dataDir, 0o777); err != nil {
		return err
	}

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

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, ctx, workflowConfigurationFile, logger); err != nil {
		return err
	}

	wfconfig, err := workflow.LoadWorkflow(ctx, workflowConfigurationFile, logger)
	if err != nil {
		return err
	}

	workflowConfiguration = &wfconfig

	return nil
}

func directoryExistsInTheDirectory(arg1, arg2 string) error {
	tmpHome, err := afero.TempDir(testFs, "", "")
	if err != nil {
		return err
	}

	tmpCurrent, err := afero.TempDir(testFs, "", "")
	if err != nil {
		return err
	}

	var dirPath string

	homeDirPath = tmpHome
	currentDirPath = tmpCurrent

	if arg2 == "HOME" {
		dirPath = filepath.Join(homeDirPath, arg1)
	} else {
		dirPath = filepath.Join(currentDirPath, arg1)
	}

	if err := testFs.MkdirAll(dirPath, 0o777); err != nil {
		return err
	}

	kdepsDir = dirPath

	return nil
}

func searchTextInFile(filePath string, searchText string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), searchText) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func itShouldCreateTheDockerfile(arg1, arg2, arg3 string) error {
	rd, asm, _, hIP, hPort, _, _, gpu, err := BuildDockerfile(testFs, ctx, systemConfiguration, kdepsDir, pkgProject, logger)
	if err != nil {
		return err
	}

	runDir = rd
	hostPort = hPort
	hostIP = hIP
	APIServerMode = asm
	gpuType = gpu

	dockerfile := filepath.Join(runDir, "Dockerfile")

	if strings.Contains(arg2, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg2, ",")
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace

			found, err := searchTextInFile(dockerfile, value)
			if err != nil {
				return err
			}

			if !found {
				return errors.New("package not found!")
			}
		}
	} else {
		found, err := searchTextInFile(dockerfile, arg2)
		if err != nil {
			return err
		}

		if !found {
			return errors.New("package not found!")
		}
	}

	runDirAgentRoot := filepath.Join(kdepsDir, "run/"+arg1)
	if _, err := testFs.Stat(runDirAgentRoot); err != nil {
		return err
	}

	agentDirRoot := filepath.Join(kdepsDir, arg3+"/"+arg1)
	if _, err := testFs.Stat(agentDirRoot); err != nil {
		return err
	}

	return nil
}

func itShouldRunTheContainerBuildStepFor(arg1 string) error {
	cl, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	cli = cl

	cN, conN, err := BuildDockerImage(testFs, ctx, systemConfiguration, cli, runDir, kdepsDir, pkgProject, logger)
	if err != nil {
		return err
	}

	cName = cN
	containerName = conN

	if err := CleanupDockerBuildImages(testFs, ctx, cName, cli); err != nil {
		return err
	}

	return nil
}

func itShouldStartTheContainer(arg1 string) error {
	if _, err := CreateDockerContainer(testFs, ctx, cName, containerName, hostIP, hostPort, "", "", gpuType, APIServerMode, false, cli); err != nil {
		return err
	}

	return nil
}

func kdepsOpenThePackage(arg1 string) error {
	pkgP, err := archiver.ExtractPackage(testFs, ctx, kdepsDir, packageFile, logger)
	if err != nil {
		return err
	}

	pkgProject = pkgP

	return nil
}

func theValidAiagentHas(arg1, arg2 string) error {
	cDir, pFile, err := archiver.CompileProject(testFs, ctx, *workflowConfiguration, kdepsDir, agentDir, environ, logger)
	if err != nil {
		return err
	}

	compiledProjectDir = cDir
	packageFile = pFile

	return nil
}

func theCommandShouldBeRunActionByDefault(arg1 string) error {
	dockerfile := filepath.Join(runDir, "Dockerfile")
	found, err := searchTextInFile(dockerfile, fmt.Sprintf(`CMD ["run", "%s"]`, arg1))
	if err != nil {
		return err
	}

	if !found {
		return errors.New("entrypoint run not found!")
	}

	return nil
}

func theDockerEntrypointShouldBe(arg1 string) error {
	dockerfile := filepath.Join(runDir, "Dockerfile")
	found, err := searchTextInFile(dockerfile, fmt.Sprintf(`ENTRYPOINT ["%s"]`, arg1))
	if err != nil {
		return err
	}

	if !found {
		return errors.New("entrypoint not found!")
	}

	return nil
}

func itWillInstallTheModels(arg1 string) error {
	found, err := searchTextInFile(workflowConfigurationFile, arg1)
	if err != nil {
		return err
	}

	if !found {
		return errors.New("model not found!")
	}

	return nil
}

func kdepsWillCheckThePresenceOfTheFile(arg1 string) error {
	dr, err := resolver.NewGraphResolver(testFs, ctx, environ, nil, logger)
	if err != nil {
		return err
	}

	if _, err := BootstrapDockerSystem(ctx, dr); err != nil {
		return err
	}

	return nil
}

func theSystemFolderExists(arg1 string) error {
	logger = logging.GetLogger()
	tempDir, err := afero.TempDir(testFs, "", arg1)
	if err != nil {
		return err
	}

	kdepsDir = tempDir

	packageDir = filepath.Join(kdepsDir, "packages")
	if err := testFs.MkdirAll(packageDir, 0o755); err != nil {
		return err
	}

	// Create resources directory
	resourcesDir = filepath.Join(kdepsDir, "resources")
	if err := testFs.MkdirAll(resourcesDir, 0o755); err != nil {
		return err
	}

	// Create data directory
	dataDir = filepath.Join(kdepsDir, "data")
	if err := testFs.MkdirAll(dataDir, 0o755); err != nil {
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

	// Create the document with the id and requires block
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ActionID = "%s"
%s
Run {
  Exec {
  ["key"] = """
@(exec.stdout["anAction"])
@(exec.stdin["anAction2"])
@(exec.stderr["anAction2"])
@(http.client["anAction3"].response)
@(llm.chat["anAction4"].response)
"""
  }
}
`, schema.SchemaVersion(ctx), arg2, requiresSection)

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

func itWillBeStoredTo(arg1 string) error {
	workflowFile = filepath.Join(kdepsDir, arg1)

	if _, err := testFs.Stat(workflowFile); err != nil {
		return err
	}

	return nil
}

func itHasAFileWithNoDependencyWithIDProperty(arg1, arg2 string) error {
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ActionID = "%s"
Run {
  Exec {
  ["key"] = """
@(exec.stdout["anAction"])
@(exec.stdin["anAction2"])
@(exec.stderr["anAction2"])
@(http.client["anAction3"].response)
@(llm.chat["anAction4"].response)
"""
  }
}
`, schema.SchemaVersion(ctx), arg2)

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
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

TargetActionID = "%s"
AgentID = "%s"
Description = "My awesome AI Agent"
Version = "%s"
`, schema.SchemaVersion(ctx), arg3, arg1, arg2)

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
	_, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, ctx, *workflowConfiguration, kdepsDir, aiAgentDir, logger)
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
	AgentID = "invalid agent"
	Description = "a not valid configuration"
	Version = "five"
	TargetActionID = "hello World"
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
	_, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, ctx, *workflowConfiguration, kdepsDir, aiAgentDir, logger)
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
	var workflowsSection string
	if strings.Contains(arg4, ",") {
		// Split arg3 into multiple values if it's a CSV
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

	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

TargetActionID = "%s"
AgentID = "%s"
Description = "My awesome AI Agent"
Version = "%s"
%s
`, schema.SchemaVersion(ctx), arg3, arg1, arg2, workflowsSection)

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

// PackageProject is a helper function to package a project
func PackageProject(fs afero.Fs, ctx context.Context, wf wfPkl.Workflow, kdepsDir, aiAgentDir string, logger *logging.Logger) (string, error) {
	// Create package directory if it doesn't exist
	packageDir := filepath.Join(kdepsDir, "packages")
	if err := fs.MkdirAll(packageDir, 0o755); err != nil {
		return "", err
	}

	// Create package file path
	packageFile := filepath.Join(packageDir, fmt.Sprintf("%s-%s.tar.gz", wf.GetAgentID(), wf.GetVersion()))

	// Create package file
	file, err := fs.Create(packageFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Write package content
	if _, err := file.WriteString("package content"); err != nil {
		return "", err
	}

	return packageFile, nil
}

func TestPrintDockerBuildOutputSimple(t *testing.T) {
	successLog := bytes.NewBufferString(`{"stream":"Step 1/2 : FROM alpine\n"}\n{"stream":" ---> 123abc\n"}\n`)
	if err := printDockerBuildOutput(successLog); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Error case should propagate the message
	errBuf := bytes.NewBufferString(`{"error":"build failed"}`)
	if err := printDockerBuildOutput(errBuf); err == nil {
		t.Fatalf("expected error not returned")
	}
}
