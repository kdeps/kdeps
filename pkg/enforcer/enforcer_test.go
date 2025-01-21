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
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

var (
	testFs                = afero.NewOsFs()
	homeDirPath           string
	currentDirPath        string
	ctx                   = context.Background()
	systemConfiguration   *kdeps.Kdeps
	fileThatExist         string
	logger                *logging.Logger
	agentPath             string
	doc                   string
	schemaVersionFilePath = "../../SCHEMA_VERSION"
	workflowAmendsLine    = `amends "package://schema.kdeps.com/core@0.0.32#/Workflow.pkl"`
	configAmendsLine      = `amends "package://schema.kdeps.com/core@0.0.32#/Kdeps.pkl"`
	resourceAmendsLine    = `amends "package://schema.kdeps.com/core@0.0.32#/Resource.pkl"`
	resourceValues        = `
id = "helloWorld"

name = null
description = null
category = null
requires = null
run = null
`
	configValues = `
runMode = "docker"
dockerGPU = "cpu"
`
	workflowValues = `
settings {
  apiServerMode = false
  apiServerSettings {
    serverPort = 3000
    routes {
      new {
	path = "/api"
	methods {
	  "POST"
	}
	requestParams = "ENV:API_SERVER_REQUEST_PARAMS"
	request = "ENV:API_SERVER_REQUEST"
	response = "ENV:API_SERVER_RESPONSE"
      }
    }
  }
}
name = "myAgent"
description = "My awesome AI Agent"
version = "1.0.0"
action = "helloWorld"
`
	testingT *testing.T
)

func TestFeatures(t *testing.T) {
	t.Parallel()
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Configuration steps
			ctx.Step(`^we have a blank config file$`, weHaveABlankFile)
			ctx.Step(`^a file "([^"]*)" exists in the current directory$`, aFileExistsInTheCurrentDirectory)
			ctx.Step(`^a system configuration is defined$`, aSystemConfigurationIsDefined)
			ctx.Step(`^it does not have a config amends line on top of the file$`, itDoesNotHaveAConfigAmendsLineOnTopOfTheFile)
			ctx.Step(`^it have a "([^"]*)" amends url line on top of the file$`, itHaveAAmendsUrlLineOnTopOfTheFile)
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
	f.WriteString(doc)
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

	environ, err := environment.NewEnvironment(testFs, ctx, env)
	if err != nil {
		return err
	}

	cfgFile, err := cfg.GenerateConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return err
	}

	scfg, err := cfg.LoadConfiguration(testFs, ctx, cfgFile, logger)
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

func itHaveAAmendsUrlLineOnTopOfTheFile(arg1 string) error {
	doc = strings.Replace(doc, "kdeps.com", arg1, -1)

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
	if err := EnforcePklTemplateAmendsRules(testFs, ctx, fileThatExist, logger); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func itIsAValidPklFile() error {
	if err := EnforcePklTemplateAmendsRules(testFs, ctx, fileThatExist, logger); err != nil {
		return err
	}

	if _, err := evaluator.EvalPkl(testFs, ctx, fileThatExist, "", logger); err != nil {
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
	fmt.Printf("Creating %s file!", file)

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	fileThatExist = file

	return nil
}

func anAgentFolderExistsInTheCurrentDirectory(arg1 string) error {
	agentPath = currentDirPath + "/my-agent"
	if err := testFs.MkdirAll(agentPath, 0o755); err != nil {
		return err
	}
	fmt.Printf("Agent path %s created!", agentPath)

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
	fmt.Printf("Agent path %s created!", subfolderPath)

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
