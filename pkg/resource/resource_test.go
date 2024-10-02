package resource_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"kdeps/pkg/archiver"
	"kdeps/pkg/cfg"
	"kdeps/pkg/docker"
	"kdeps/pkg/enforcer"
	"kdeps/pkg/environment"
	"kdeps/pkg/logging"
	"kdeps/pkg/workflow"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/cucumber/godog"
	"github.com/docker/docker/api/types"
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
	ctx                       context.Context
	packageFile               string
	hostPort                  string = "3000"
	hostIP                    string = "127.0.0.1"
	containerID               string
	logger                    *log.Logger
	runDir                    string
	containerName             string
	apiServerMode             bool
	cName                     string
	pkgProject                *archiver.KdepsPackage
	compiledProjectDir        string
	currentDirPath            string
	systemConfigurationFile   string
	cli                       *client.Client
	systemConfiguration       *kdeps.Kdeps
	workflowConfigurationFile string
	workflowConfiguration     *wfPkl.Workflow
	schemaVersionFilePath     = "../../SCHEMA_VERSION"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a kdeps container with "([^"]*)" endpoint "([^"]*)" API and "([^"]*)"$`, aKdepsContainerWithEndpointAPI)
			ctx.Step(`^I fill in the "([^"]*)" with success "([^"]*)", response data "([^"]*)"$`, iFillInTheWithSuccessResponseData)
			ctx.Step(`^I GET request to "([^"]*)" with data "([^"]*)" and header name "([^"]*)" that maps to "([^"]*)"$`, iGETRequestToWithDataAndHeaderNameThatMapsTo)
			ctx.Step(`^I should see a blank standard template "([^"]*)" in the "([^"]*)" folder$`, iShouldSeeABlankStandardTemplateInTheFolder)
			ctx.Step(`^I should see a "([^"]*)" in the "([^"]*)" folder$`, iShouldSeeAInTheFolder)
			ctx.Step(`^I should see action "([^"]*)", url "([^"]*)", data "([^"]*)", headers "([^"]*)" with values "([^"]*)" and params "([^"]*)" that maps to "([^"]*)"$`, iShouldSeeActionUrlDataHeadersWithValuesAndParamsThatMapsTo)
			ctx.Step(`^it should respond "([^"]*)" in "([^"]*)"$`, itShouldRespondIn)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/resource"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aKdepsContainerWithEndpointAPI(arg1, arg2, arg3 string) error {
	logger = logging.GetLogger()
	ctx = context.Background()

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

	dirPath = filepath.Join(homeDirPath, ".kdeps")

	if err := testFs.MkdirAll(dirPath, 0777); err != nil {
		return err
	}

	kdepsDir = dirPath

	env := &environment.Environment{
		Home:           homeDirPath,
		Pwd:            currentDirPath,
		NonInteractive: "1",
	}

	environ, err := environment.NewEnvironment(testFs, env)
	if err != nil {
		return err
	}

	systemConfigurationContent := `
	amends "package://schema.kdeps.com/core@0.0.48#/Kdeps.pkl"

	runMode = "docker"
	dockerGPU = "cpu"
	`

	systemConfigurationFile = filepath.Join(homeDirPath, ".kdeps.pkl")
	// Write the heredoc content to the file
	err = afero.WriteFile(testFs, systemConfigurationFile, []byte(systemConfigurationContent), 0644)
	if err != nil {
		return err
	}

	systemConfigurationFile, err = cfg.FindConfiguration(testFs, environ, logger)
	if err != nil {
		return err
	}

	if err = enforcer.EnforcePklTemplateAmendsRules(testFs, systemConfigurationFile, logger); err != nil {
		return err
	}

	syscfg, err := cfg.LoadConfiguration(testFs, systemConfigurationFile, logger)
	if err != nil {
		return err
	}

	systemConfiguration = syscfg

	var methodSection string
	if strings.Contains(arg1, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg1, ",")
		var methodLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			methodLines = append(methodLines, fmt.Sprintf(`"%s"`, value))
		}
		methodSection = "methods {\n" + strings.Join(methodLines, "\n") + "\n}"
	} else {
		// Single value case
		methodSection = fmt.Sprintf(`
methods {
  "%s"
}`, arg1)
	}

	workflowConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.50#/Workflow.pkl"

name = "myAIAgentAPI"
description = "AI Agent X API"
action = "helloWorld"
settings {
  apiServerMode = true
  agentSettings {
    packages {}
    models {
      "tinydolphin"
    }
  }
  apiServer {
    routes {
      new {
	path = "/resource1"
	%s
	responseType = "json"
      }
      new {
	path = "/resource2"
	%s
      }
    }
  }
}
`, methodSection, methodSection)
	var filePath string

	filePath = filepath.Join(homeDirPath, "myAgentX")

	if err := testFs.MkdirAll(filePath, 0777); err != nil {
		return err
	}

	agentDir = filePath

	workflowConfigurationFile = filepath.Join(filePath, "workflow.pkl")
	err = afero.WriteFile(testFs, workflowConfigurationFile, []byte(workflowConfigurationContent), 0644)
	if err != nil {
		return err
	}

	resourcesDir := filepath.Join(filePath, "resources")
	if err := testFs.MkdirAll(resourcesDir, 0777); err != nil {
		return err
	}

	resourceConfigurationContent := `
amends "package://schema.kdeps.com/core@0.0.50#/Resource.pkl"

local llmResponse = "@(llm.resource["action2"].response.trim())"
local execResponse = "@(exec.resource["action3"].stdout.trim())"

id = "helloWorld"
name = "default action"
category = "kdepsdockerai"
description = "this is a description for helloWorld @(request.params)"
requires {
  "action1"
  "action2"
  "action3"
}
run {
  preflightCheck {
    validations {
      llmResponse != "hello world"
      1 + 1 == 2
    }
  }
  apiResponse {
    success = true
    response {
      data {
	"@(llmResponse)"
	"@(execResponse)"
      }
    }
  }
  postflightCheck {
    validations {
      1 + 1 == 2
      2 + 2 == 4
    }
  }
}
`

	resourceConfigurationFile := filepath.Join(resourcesDir, "resource1.pkl")
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0644)
	if err != nil {
		return err
	}

	resourceConfigurationContent = `
amends "package://schema.kdeps.com/core@0.0.50#/Resource.pkl"

id = "action1"
category = "kdepsdockerai"
description = "this is a description for action1 - @(request.url)"
requires {
  "action2"
  "helloWorld"
}
name = "default action"
run {
  preflightCheck {
    validations {
      1 + 1 == 2
      2 + 2 == 4
    }
  }
}
`

	resourceConfigurationFile = filepath.Join(resourcesDir, "resource2.pkl")
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0644)
	if err != nil {
		return err
	}

	resourceConfigurationContent = `
amends "package://schema.kdeps.com/core@0.0.50#/Resource.pkl"

id = "action2"
category = "kdepsdockerai"
description = "this is a description for action2 - @(request.method)"
name = "default action"
requires {
  "action1"
  "action3"
  "helloWorld"
}
run {
  chat {
    model = "tinydolphin"
    prompt = """
tell me about this "@(exec.resource["action3"].stdout)"
"""
  }
}
`

	resourceConfigurationFile = filepath.Join(resourcesDir, "resource3.pkl")
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0644)
	if err != nil {
		return err
	}

	resourceConfigurationContent = `
amends "package://schema.kdeps.com/core@0.0.50#/Resource.pkl"

id = "action3"
category = "kdepsdockerai"
description = "this is a description for action3 - @(request.url)"
requires {
  "helloWorld"
  "action2"
  "action1"
}
name = "default action"
run {
  exec {
    env {
      ["NEWVAR"] = "I am a new variable"
      ["NEWVAR2"] = "I am also a new variable"
    }
    command = "echo $NEWVAR && echo $NEWVAR2"
  }
  postflightCheck {
    validations {
      1 + 1 == 2
      2 + 2 == 4
    }
  }
}
`

	resourceConfigurationFile = filepath.Join(resourcesDir, "resource4.pkl")
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0644)
	if err != nil {
		return err
	}

	dataDir := filepath.Join(filePath, "data")
	if err := testFs.MkdirAll(dataDir, 0777); err != nil {
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

	workflowConfiguration = wfconfig

	cDir, pFile, err := archiver.CompileProject(testFs, ctx, workflowConfiguration, kdepsDir, agentDir, logger)
	if err != nil {
		return err
	}

	compiledProjectDir = cDir
	packageFile = pFile

	pkgP, err := archiver.ExtractPackage(testFs, ctx, kdepsDir, packageFile, logger)
	if err != nil {
		return err
	}

	pkgProject = pkgP

	rd, asm, hIP, hPort, err := docker.BuildDockerfile(testFs, ctx, systemConfiguration, kdepsDir, pkgProject, logger)
	if err != nil {
		return err
	}

	runDir = rd
	hostPort = hPort
	hostIP = hIP
	apiServerMode = asm

	cl, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	cli = cl

	cN, conN, err := docker.BuildDockerImage(testFs, ctx, systemConfiguration, cli, runDir, kdepsDir, pkgProject, logger)
	if err != nil {
		return err
	}

	cName = cN
	containerName = conN

	if err := docker.CleanupDockerBuildImages(testFs, ctx, cName, cli); err != nil {
		return err
	}

	dockerClientID, err := docker.CreateDockerContainer(testFs, ctx, cName, containerName, hostIP, hostPort, apiServerMode, cli)
	if err != nil {
		return err
	}

	containerID = dockerClientID

	return nil
}

func iFillInTheWithSuccessResponseData(arg1, arg2, arg3 string) error {
	return godog.ErrPending
}

func iGETRequestToWithDataAndHeaderNameThatMapsTo(arg1, arg2, arg3, arg4 string) error {
	// // Ensure cleanup of the container at the end of the test
	// defer func() {
	//	time.Sleep(30 * time.Second)

	//	err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
	//		Force: true,
	//	})
	//	if err != nil {
	//		log.Printf("Failed to remove container: %v", err)
	//	}
	// }()

	time.Sleep(30 * time.Second)

	// Base URL
	baseURL := fmt.Sprintf("http://%s:%s%s", hostIP, hostPort, arg1)
	reqBody := strings.NewReader(arg2)

	// Create a new GET request
	req, err := http.NewRequest("GET", baseURL, reqBody)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}

	// Set headers
	req.Header.Set(arg3, arg4)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return err
	}

	fmt.Println("Response:", string(body))

	// Return immediately as the request is sent in the background
	return nil
}

func iShouldSeeABlankStandardTemplateInTheFolder(arg1, arg2 string) error {
	return godog.ErrPending
}

func iShouldSeeAInTheFolder(arg1, arg2 string) error {
	execConfig := types.ExecConfig{
		Cmd:          []string{"ls", arg2 + arg1},
		AttachStdout: true,
		AttachStderr: true,
	}
	execIDResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return err
	}

	execID := execIDResp.ID

	// Attach to the exec session to capture the output
	execAttachResp, err := cli.ContainerExecAttach(ctx, execID, types.ExecStartCheck{})
	if err != nil {
		return err
	}
	defer execAttachResp.Close()

	// Capture the command output
	var output bytes.Buffer
	_, err = io.Copy(&output, execAttachResp.Reader)
	if err != nil {
		logger.Fatal("Failed to read exec output: %v", err)
		return err
	}

	// Check the command output
	logger.Info("Output from `ls /` command in container:\n%s", output.String())

	// Optionally, inspect the exec result to check for success/failure
	execInspect, err := cli.ContainerExecInspect(ctx, execID)
	if err != nil {
		logger.Fatal("Failed to inspect exec result: %v", err)
		return err
	}

	if execInspect.ExitCode != 0 {
		logger.Error("Command failed with exit code: %d", execInspect.ExitCode)
		return err
	}

	return nil
}

func iShouldSeeActionUrlDataHeadersWithValuesAndParamsThatMapsTo(arg1, arg2, arg3, arg4, arg5, arg6, arg7 string) error {
	return godog.ErrPending
}

func itShouldRespondIn(arg1, arg2 string) error {
	return godog.ErrPending
}
