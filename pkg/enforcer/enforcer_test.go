package enforcer

import (
	"errors"
	"fmt"
	"kdeps/pkg/cfg"
	"kdeps/pkg/evaluator"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

var (
	testFs                = afero.NewOsFs()
	homeDirPath           string
	currentDirPath        string
	systemConfiguration   *kdeps.Kdeps
	fileThatExist         string
	agentPath             string
	doc                   string
	schemaVersionFilePath = "../../SCHEMA_VERSION"
	workflowAmendsLine    = `amends "package://schema.kdeps.com/core@0.0.29#/Workflow.pkl"`
	configAmendsLine      = `amends "package://schema.kdeps.com/core@0.0.29#/Kdeps.pkl"`
	resourceAmendsLine    = `amends "package://schema.kdeps.com/core@0.0.29#/Resource.pkl"`
	resourceValues        = `resources {}`
	configValues          = `
runMode = "docker"
dockerGPU = "cpu"
llmSettings {
  llmAPIKeys {
    openai_api_key = null
    mistral_api_key = null
    huggingface_api_token = null
    groq_api_key = null
  }
  llmFallbackBackend = "local"
  llmFallbackModel = "llama3.1"
}
`
	workflowValues = `
settings {
  runTimeout = 15.min
  llmSettings {
    llmAPIKeys {
      openai_api_key = null
      mistral_api_key = null
      huggingface_api_token = null
      groq_api_key = null
    }
    llmFallbackBackend = "local"
    llmFallbackModel = "llama3.1"
  }
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
version = "1.0.0"
action = "helloWorld"
workflows {}
args = null
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
	env := &cfg.Environment{
		Home:           homeDirPath,
		Pwd:            "",
		NonInteractive: "1",
	}

	if err := cfg.GenerateConfiguration(testFs, env); err != nil {
		return err
	}

	scfg, err := cfg.LoadConfiguration(testFs)
	if err != nil {
		return err
	}

	systemConfiguration = scfg

	return nil
}

func itDoesNotHaveAConfigAmendsLineOnTopOfTheFile() error {
	doc = fmt.Sprintf("%s", configValues)

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
	if err := EnforceFolderStructure(testFs, agentPath); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func itIsAValidAgent() error {
	if err := EnforceFolderStructure(testFs, agentPath); err != nil {
		return err
	}

	return nil
}

func itIsAnInvalidPklFile() error {
	if err := EnforcePklTemplateAmendsRules(testFs, fileThatExist, schemaVersionFilePath); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func itIsAValidPklFile() error {
	if err := EnforcePklTemplateAmendsRules(testFs, fileThatExist, schemaVersionFilePath); err != nil {
		return err
	}

	if _, err := evaluator.EvalPkl(testFs, fileThatExist); err != nil {
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
	if err := testFs.MkdirAll(agentPath, 0755); err != nil {
		return err
	}
	fmt.Printf("Agent path %s created!", agentPath)

	return nil
}

func itDoesNotHaveAWorkflowAmendsLineOnTopOfTheFile() error {
	doc = fmt.Sprintf("%s", workflowValues)

	return nil
}

func itHaveAWorkflowAmendsLineOnTopOfTheFile() error {
	doc = fmt.Sprintf("%s\n%s", workflowAmendsLine, workflowValues)

	return nil
}

func aFolderNamedExistsInThe(arg1, arg2 string) error {
	agentPath = currentDirPath + "/my-agent"
	subfolderPath := agentPath + "/" + arg1
	if err := testFs.MkdirAll(subfolderPath, 0755); err != nil {
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
	doc = fmt.Sprintf("%s", resourceValues)

	return nil
}
