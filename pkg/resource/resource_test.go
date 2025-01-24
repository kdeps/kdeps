package resource_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
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
	ctx                       context.Context
	packageFile               string
	hostPort                  string = "3000"
	hostIP                    string = "127.0.0.1"
	containerID               string
	logger                    *logging.Logger
	runDir                    string
	gpuType                   string
	containerName             string
	APIServerMode             bool
	cName                     string
	pkgProject                *archiver.KdepsPackage
	compiledProjectDir        string
	currentDirPath            string
	systemConfigurationFile   string
	cli                       *client.Client
	systemConfiguration       *kdeps.Kdeps
	workflowConfigurationFile string
	workflowConfiguration     *wfPkl.Workflow
)

func TestFeatures(t *testing.T) {
	t.Parallel()
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a kdeps container with "([^"]*)" endpoint "([^"]*)" API and "([^"]*)"$`, aKdepsContainerWithEndpointAPI)
			ctx.Step(`^I fill in the "([^"]*)" with success "([^"]*)", response data "([^"]*)"$`, iFillInTheWithSuccessResponseData)
			ctx.Step(`^I GET request to "([^"]*)" with data "([^"]*)" and header name "([^"]*)" that maps to "([^"]*)"$`, iGETRequestToWithDataAndHeaderNameThatMapsTo)
			ctx.Step(`^I should see a blank standard template "([^"]*)" in the "([^"]*)" folder$`, iShouldSeeABlankStandardTemplateInTheFolder)
			ctx.Step(`^I should see a "([^"]*)" in the "([^"]*)" folder$`, iShouldSeeAInTheFolder)
			ctx.Step(`^I should see action "([^"]*)", url "([^"]*)", data "([^"]*)", headers "([^"]*)" with values "([^"]*)" and params "([^"]*)" that maps to "([^"]*)"$`, iShouldSeeActionURLDataHeadersWithValuesAndParamsThatMapsTo)
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

	if err := testFs.MkdirAll(dirPath, 0o777); err != nil {
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
	amends "package://schema.kdeps.com/core@0.1.9#/Kdeps.pkl"

	runMode = "docker"
	dockerGPU = "cpu"
	`

	systemConfigurationFile = filepath.Join(homeDirPath, ".kdeps.pkl")
	// Write the heredoc content to the file
	err = afero.WriteFile(testFs, systemConfigurationFile, []byte(systemConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	systemConfigurationFile, err = cfg.FindConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return err
	}

	if err = enforcer.EnforcePklTemplateAmendsRules(testFs, ctx, systemConfigurationFile, logger); err != nil {
		return err
	}

	syscfg, err := cfg.LoadConfiguration(testFs, ctx, systemConfigurationFile, logger)
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
amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

name = "myAIAgentAPI"
description = "AI Agent X API"
action = "helloWorld"
settings {
  APIServerMode = true
  agentSettings {
    packages {}
    models {
      "llama3.2"
    }
  }
  APIServer {
    routes {
      new {
	path = "/resource1"
	%s
      }
    }
  }
}
`, schema.SchemaVersion(ctx), methodSection)
	filePath := filepath.Join(homeDirPath, "myAgentX")

	if err := testFs.MkdirAll(filePath, 0o777); err != nil {
		return err
	}

	agentDir = filePath

	workflowConfigurationFile = filepath.Join(filePath, "workflow.pkl")
	err = afero.WriteFile(testFs, workflowConfigurationFile, []byte(workflowConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	resourcesDir := filepath.Join(filePath, "resources")
	if err := testFs.MkdirAll(resourcesDir, 0o777); err != nil {
		return err
	}

	resourceConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

local llmResponse = "@(llm.response("action1"))"
local execResponse = "@(exec.stdout("action2"))"
local clientResponse = "@(client.responseBody("action3"))"
local clientResponse2 = "@(client.responseBody("action4"))"

ID = "helloWorld"
name = "default action"
category = "kdepsdockerai"
description = "this is a description for helloWorld @(request.params)"
requires {
  "action1"
  "action2"
  "action3"
  "action4"
}

run {
  preflightCheck {
    validations {
      llmResponse != "hello world"
      1 + 1 == 2
    }
  }
  APIResponse {
    success = true
    response {
      data {
	"@(llmResponse)"
	"@(execResponse)"
	"@(clientResponse)"
	"@(clientResponse2)"
      }
    }
  }
}
`, schema.SchemaVersion(ctx))

	resourceConfigurationFile := filepath.Join(resourcesDir, "resource1.pkl")
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	resourceConfigurationContent = fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

local clientResponse = "@(client.responseBody("action3"))"

ID = "action1"
category = "kdepsdockerai"
description = "this is a description for action1 - @(request.url)"
requires {
  "action2"
  "helloWorld"
}
name = "default action"
run {
  chat {
    model = "llama3.2"
    prompt = "@(request.data)"
    JSONResponse = true
    JSONResponseKeys {
      "translation"
      "uses"
      "synonyms"
      "antonyms"
    }
    timeoutSeconds = 0
  }
  preflightCheck {
    validations {
      1 + 1 == 2
      2 + 2 == 4
    }
  }
}
`, schema.SchemaVersion(ctx))

	resourceConfigurationFile = filepath.Join(resourcesDir, "resource2.pkl")
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	resourceConfigurationContent = fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ID = "action2"
category = "kdepsdockerai"
description = "this is a description for action2 - @(request.method)"
name = "default action"
requires {
  "action1"
  "action3"
  "helloWorld"
}
run {
  exec {
    env {
      ["RESPONSE"] = "@(client.responseBody("action3"))"
    }
    command = "echo $RESPONSE"
  }
}
`, schema.SchemaVersion(ctx))

	resourceConfigurationFile = filepath.Join(resourcesDir, "resource3.pkl")
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	resourceConfigurationContent = fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ID = "action3"
category = "kdepsdockerai"
description = "this is a description for action3 - @(request.url)"
requires {
  "helloWorld"
  "action2"
  "action1"
}
name = "default action"
run {
  HTTPClient {
    method = "GET"
    url = "https://dog.ceo/api/breeds/list/all"
  }
}
`, schema.SchemaVersion(ctx))

	resourceConfigurationFile = filepath.Join(resourcesDir, "resource4.pkl")
	err = afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	resourceConfigurationContent = fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ID = "action4"
category = "kdepsdockerai"
description = "this is a description for action4 - @(request.url)"
requires {
  "helloWorld"
  "action2"
  "action1"
  "action3"
}
name = "default action"
run {
  HTTPClient {
    method = "GET"
    url = "https://google.com"
  }
}
`, schema.SchemaVersion(ctx))

	resourceConfigurationFile = filepath.Join(resourcesDir, "resource5.pkl")
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

	cDir, pFile, err := archiver.CompileProject(testFs, ctx, *workflowConfiguration, kdepsDir, agentDir, environ, logger)
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

	rd, asm, hIP, hPort, gpu, err := docker.BuildDockerfile(testFs, ctx, systemConfiguration, kdepsDir, pkgProject, logger)
	if err != nil {
		return err
	}

	runDir = rd
	hostPort = hPort
	hostIP = hIP
	APIServerMode = asm
	gpuType = gpu

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

	dockerClientID, err := docker.CreateDockerContainer(testFs, ctx, cName, containerName, hostIP, hostPort, gpuType, APIServerMode, cli)
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
	req, err := http.NewRequest(http.MethodGet, baseURL, reqBody)
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
	body, err := io.ReadAll(resp.Body)
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
		logger.Fatal("failed to read exec output: %v", err)
		return err
	}

	// Check the command output
	logger.Debug("output from `ls /` command in container:\n%s", output.String())

	// Optionally, inspect the exec result to check for success/failure
	execInspect, err := cli.ContainerExecInspect(ctx, execID)
	if err != nil {
		logger.Fatal("failed to inspect exec result: %v", err)
		return err
	}

	if execInspect.ExitCode != 0 {
		logger.Error("command failed with exit code: %d", execInspect.ExitCode)
		return err
	}

	return nil
}

func iShouldSeeActionURLDataHeadersWithValuesAndParamsThatMapsTo(arg1, arg2, arg3, arg4, arg5, arg6, arg7 string) error {
	return godog.ErrPending
}

func itShouldRespondIn(arg1, arg2 string) error {
	return godog.ErrPending
}
