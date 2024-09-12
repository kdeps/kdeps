package docker

import (
	"fmt"
	"kdeps/pkg/archiver"
	"kdeps/pkg/cfg"
	"kdeps/pkg/enforcer"
	"kdeps/pkg/workflow"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
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
	packageFile               string
	compiledProjectDir        string
	currentDirPath            string
	systemConfigurationFile   string
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
			ctx.Step(`^it should check if the docker container "([^"]*)" is not running$`, itShouldCheckIfTheDocker)
			ctx.Step(`^it should create the Dockerfile for the agent in the "([^"]*)" directory with model "([^"]*)" and package "([^"]*)" and copy the kdeps package to the "([^"]*)" directory$`, itShouldCreateTheDockerfile)
			ctx.Step(`^it should run the container build step for "([^"]*)"$`, itShouldRunTheContainerBuildStepFor)
			ctx.Step(`^it should start the container "([^"]*)"$`, itShouldStartTheContainer)
			ctx.Step(`^kdeps open the package "([^"]*)" and extract it\'s content to the agents directory$`, kdepsOpenThePackage)
			ctx.Step(`^kdeps should parse the workflow of the "([^"]*)" agent version "([^"]*)" in the agents directory with model "([^"]*)" and packages "([^"]*)"$`, kdepsShouldParseTheWorkflow)
			ctx.Step(`^the valid ai-agent "([^"]*)" has been compiled as "([^"]*)" in the packages directory$`, theValidAiagentHas)

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
	env := &cfg.Environment{
		Home:           homeDirPath,
		Pwd:            currentDirPath,
		NonInteractive: "1",
	}

	systemConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.34#/Kdeps.pkl"

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
	err := afero.WriteFile(testFs, systemConfigurationFile, []byte(systemConfigurationContent), 0644)
	if err != nil {
		return err
	}

	systemConfigurationFile, err := cfg.FindConfiguration(testFs, env)
	if err != nil {
		return err
	}

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, systemConfigurationFile, schemaVersionFilePath); err != nil {
		return err
	}

	syscfg, err := cfg.LoadConfiguration(testFs, systemConfigurationFile)
	if err != nil {
		return err
	}

	systemConfiguration = syscfg

	return nil
}

func aValidAiagentIsPresentInTheDirectory(arg1, arg2 string) error {
	workflowConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.34#/Workflow.pkl"

name = "%s"
description = "AI Agent X"
action = "%s"
`, arg1, arg1)

	var filePath string

	if arg2 == "HOME" {
		filePath = filepath.Join(homeDirPath, arg1)
	} else {
		filePath = filepath.Join(currentDirPath, arg1)
	}

	if err := testFs.MkdirAll(filePath, 0777); err != nil {
		return err
	}

	agentDir = filePath

	workflowConfigurationFile = filepath.Join(filePath, "workflow.pkl")
	err := afero.WriteFile(testFs, workflowConfigurationFile, []byte(workflowConfigurationContent), 0644)
	if err != nil {
		return err
	}

	resourcesDir := filepath.Join(filePath, "resources")
	if err := testFs.MkdirAll(resourcesDir, 0777); err != nil {
		return err
	}

	resourceConfigurationContent := fmt.Sprintf(`
	amends "package://schema.kdeps.com/core@0.0.34#/Resource.pkl"

	id = "%s"
	description = "An action from agent %s"
	`, arg1, arg1)

	resourceConfigurationFile := filepath.Join(resourcesDir, fmt.Sprintf("%s.pkl", arg1))
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0644)
	if err != nil {
		return err
	}

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, workflowConfigurationFile, schemaVersionFilePath); err != nil {
		return err
	}

	wfconfig, err := workflow.LoadWorkflow(workflowConfigurationFile)
	if err != nil {
		return err
	}

	workflowConfiguration = wfconfig

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

	if err := testFs.MkdirAll(dirPath, 0777); err != nil {
		return err
	}

	kdepsDir = dirPath

	return nil
}

func itShouldCheckIfTheDocker(arg1 string) error {
	return godog.ErrPending
}

func itShouldCreateTheDockerfile(arg1, arg2, arg3, arg4 string) error {
	return godog.ErrPending
}

func itShouldRunTheContainerBuildStepFor(arg1 string) error {
	return godog.ErrPending
}

func itShouldStartTheContainer(arg1 string) error {
	return godog.ErrPending
}

func kdepsOpenThePackage(arg1 string) error {
	pkgProject, err := archiver.ExtractPackage(testFs, kdepsDir, packageFile)
	if err != nil {
		return err
	}

	if err := BuildAndRunDockerfile(testFs, systemConfiguration, kdepsDir, pkgProject); err != nil {
		return err
	}

	return nil
}

func kdepsShouldParseTheWorkflow(arg1, arg2, arg3, arg4 string) error {
	return godog.ErrPending
}

func theValidAiagentHas(arg1, arg2 string) error {
	cDir, pFile, err := archiver.CompileProject(testFs, workflowConfiguration, kdepsDir, agentDir)
	if err != nil {
		return err
	}

	compiledProjectDir = cDir
	packageFile = pFile

	return nil
}
