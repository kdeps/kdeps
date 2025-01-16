package docker

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/workflow"

	"github.com/charmbracelet/log"
	"github.com/cucumber/godog"
	"github.com/docker/docker/client"
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
	apiServerMode             bool
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
	logger                    *log.Logger
	environ                   *environment.Environment
	cli                       *client.Client
	systemConfiguration       *kdeps.Kdeps
	workflowConfigurationFile string
	workflowConfiguration     *wfPkl.Workflow
	schemaVersionFilePath     = "../../SCHEMA_VERSION"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a "([^"]*)" system configuration file with dockerGPU "([^"]*)" and runMode "([^"]*)" is defined in the "([^"]*)" directory$`, aSystemConfigurationFile)
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
amends "package://schema.kdeps.com/core@0.0.41#/Kdeps.pkl"

runMode = "%s"
dockerGPU = "%s"
`, arg3, arg2)

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

	systemConfigurationFile, err := cfg.FindConfiguration(testFs, environ, logger)
	if err != nil {
		return err
	}

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, systemConfigurationFile, logger); err != nil {
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
amends "package://schema.kdeps.com/core@0.0.41#/Workflow.pkl"

name = "%s"
description = "AI Agent X"
action = "%s"
settings {
  apiServerMode = false
  agentSettings {
    %s
    %s
  }
}
`, arg1, arg1, pkgSection, modelSection)

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
amends "package://schema.kdeps.com/core@0.0.41#/Resource.pkl"

id = "%s"
description = "An action from agent %s"
	`, arg1, arg1)

	resourceConfigurationFile := filepath.Join(resourcesDir, fmt.Sprintf("%s.pkl", arg1))
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	dataDir := filepath.Join(filePath, "data")
	if err := testFs.MkdirAll(dataDir, 0o777); err != nil {
		return err
	}

	doc := "THIS IS A TEXT FILE: "

	for x := 0; x < 10; x++ {
		num := strconv.Itoa(x)
		file := filepath.Join(dataDir, fmt.Sprintf("textfile-%s.txt", num))

		f, _ := testFs.Create(file)
		f.WriteString(doc + num)
		f.Close()
	}

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, workflowConfigurationFile, logger); err != nil {
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
	rd, asm, hIP, hPort, gpu, err := BuildDockerfile(testFs, ctx, systemConfiguration, kdepsDir, pkgProject, logger)
	if err != nil {
		return err
	}

	runDir = rd
	hostPort = hPort
	hostIP = hIP
	apiServerMode = asm
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
	if _, err := CreateDockerContainer(testFs, ctx, cName, containerName, hostIP, hostPort, gpuType, apiServerMode, cli); err != nil {
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
	if _, err := BootstrapDockerSystem(testFs, ctx, environ, logger); err != nil {
		return err
	}

	return nil
}
