package enforcer

import (
	"errors"
	"fmt"
	"kdeps/pkg/cfg"
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
	doc                   string
	schemaVersionFilePath = "../../SCHEMA_VERSION"
	amendsLine            = `amends "package://schema.kdeps.com/core@0.0.26#/Kdeps.pkl"`
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
  llmFallbackBackend = "ollama"
  llmFallbackModel = "llama3.1"
  modelFile = null
}
`
	testingT *testing.T
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^we have a blank config file$`, weHaveABlankConfigFile)
			ctx.Step(`^a file "([^"]*)" also exists in the "([^"]*)" folder$`, aFileAlsoExistsInTheFolder)
			ctx.Step(`^a file "([^"]*)" exists in the current directory$`, aFileExistsInTheCurrentDirectory)
			ctx.Step(`^a file "([^"]*)" exists in the "([^"]*)" folder$`, aFileExistsInTheFolder)
			ctx.Step(`^a system configuration is defined$`, aSystemConfigurationIsDefined)
			ctx.Step(`^all resources have a valid amends line$`, allResourcesHaveAValidAmendsLine)
			ctx.Step(`^all resources have a valid domain$`, allResourcesHaveAValidDomain)
			ctx.Step(`^all resources have an amends line$`, allResourcesHaveAnAmendsLine)
			ctx.Step(`^an agent folder "([^"]*)" exists in the current directory$`, anAgentFolderExistsInTheCurrentDirectory)
			ctx.Step(`^"([^"]*)" does not have an amends line$`, doesNotHaveAnAmendsLine)
			ctx.Step(`^"([^"]*)" have a "([^"]*)" file in the amends$`, haveAFileInTheAmends)
			ctx.Step(`^"([^"]*)" have a "([^"]*)" url in the amends$`, haveAUrlInTheAmends)
			ctx.Step(`^it does not have a config amends line on top of the file$`, itDoesNotHaveAConfigAmendsLineOnTopOfTheFile)
			ctx.Step(`^it does not have an workflow amends line on top of the file$`, itDoesNotHaveAnWorkflowAmendsLineOnTopOfTheFile)
			ctx.Step(`^it have a "([^"]*)" amends url line on top of the file$`, itHaveAAmendsUrlLineOnTopOfTheFile)
			ctx.Step(`^it have a config amends line on top of the file$`, itHaveAConfigAmendsLineOnTopOfTheFile)
			ctx.Step(`^it have a other amends line on top of the file$`, itHaveAOtherAmendsLineOnTopOfTheFile)
			ctx.Step(`^it have a workflow amends line on top of the file$`, itHaveAWorkflowAmendsLineOnTopOfTheFile)
			ctx.Step(`^it is a invalid agent$`, itIsAInvalidAgent)
			ctx.Step(`^it is a invalid configuration file$`, itIsAInvalidConfigurationFile)
			ctx.Step(`^it is a valid agent$`, itIsAValidAgent)
			ctx.Step(`^it is a valid agent with a warning$`, itIsAValidAgentWithAWarning)
			ctx.Step(`^it is a valid agent without a warning$`, itIsAValidAgentWithoutAWarning)
			ctx.Step(`^it is a valid configuration file$`, itIsAValidConfigurationFile)
			ctx.Step(`^it is an invalid agent$`, itIsAnInvalidAgent)
			ctx.Step(`^the current directory is "([^"]*)"$`, theCurrentDirectoryIs)
			ctx.Step(`^the home directory is "([^"]*)"$`, theHomeDirectoryIs)
			ctx.Step(`^the valid workflow does not have a resources$`, theValidWorkflowDoesNotHaveAResources)
			ctx.Step(`^the valid workflow have the following <resources>$`, theValidWorkflowHaveTheFollowingResources)
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

func weHaveABlankConfigFile() error {
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

func aFileAlsoExistsInTheFolder(arg1, arg2 string) error {
	return godog.ErrPending
}

func aFileExistsInTheFolder(arg1, arg2 string) error {
	return godog.ErrPending
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

func allResourcesHaveAValidAmendsLine() error {
	return godog.ErrPending
}

func allResourcesHaveAValidDomain() error {
	return godog.ErrPending
}

func allResourcesHaveAnAmendsLine() error {
	return godog.ErrPending
}

func anAgentFolderExistsInTheCurrentDirectory(arg1 string) error {
	return godog.ErrPending
}

func doesNotHaveAnAmendsLine(arg1 string) error {
	return godog.ErrPending
}

func haveAFileInTheAmends(arg1, arg2 string) error {
	return godog.ErrPending
}

func haveAUrlInTheAmends(arg1, arg2 string) error {
	return godog.ErrPending
}

func itDoesNotHaveAConfigAmendsLineOnTopOfTheFile() error {
	doc = fmt.Sprintf("%s", configValues)

	return nil
}

func itDoesNotHaveAnWorkflowAmendsLineOnTopOfTheFile() error {
	return godog.ErrPending
}

func itHaveAAmendsUrlLineOnTopOfTheFile(arg1 string) error {
	doc = strings.Replace(doc, "kdeps.com", arg1, -1)

	return nil
}

func itHaveAConfigAmendsLineOnTopOfTheFile() error {
	doc = fmt.Sprintf("%s\n%s", amendsLine, configValues)

	return nil
}

func itHaveAOtherAmendsLineOnTopOfTheFile() error {
	return godog.ErrPending
}

func itHaveAWorkflowAmendsLineOnTopOfTheFile() error {
	return godog.ErrPending
}

func itIsAInvalidAgent() error {
	return godog.ErrPending
}

func itIsAInvalidConfigurationFile() error {
	if err := ValidateAmendsLine(fileThatExist, schemaVersionFilePath); err != nil {
		return nil
	} else {
		return errors.New("expected an error, but got none")
	}

	return nil
}

func itIsAValidAgent() error {
	return godog.ErrPending
}

func itIsAValidAgentWithAWarning() error {
	return godog.ErrPending
}

func itIsAValidAgentWithoutAWarning() error {
	return godog.ErrPending
}

func itIsAValidConfigurationFile() error {
	if err := ValidateAmendsLine(fileThatExist, schemaVersionFilePath); err != nil {
		return err
	}

	return nil
}

func itIsAnInvalidAgent() error {
	return godog.ErrPending
}

func theValidWorkflowDoesNotHaveAResources() error {
	return godog.ErrPending
}

func theValidWorkflowHaveTheFollowingResources(arg1 *godog.Table) error {
	return godog.ErrPending
}
