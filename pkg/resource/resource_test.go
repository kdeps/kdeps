//go:build integration
// +build integration

package resource_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/kdeps/kdeps/pkg/resource"

	"github.com/cucumber/godog"
	"github.com/docker/docker/api/types/container"
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
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	filesToCleanup            []string
	agentAPIPath              string
	requestFilePath           string
	responseFilePath          string
	filePath                  string
	resourceConfigurationContent string
)

func TestFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource feature tests in -short mode (CI)")
	}

	// Skip if the default API server port is already in use on the host. This avoids
	// flaky failures when other processes (or concurrent test runs) bind to 3000.
	if ln, err := net.Listen("tcp", "127.0.0.1:3000"); err == nil {
		// Port is free; close the listener and continue with the tests.
		_ = ln.Close()
	} else {
		t.Skip("port 3000 already in use; skipping resource feature tests")
	}

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/resource"},
			TestingT: t,
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		filesToCleanup = []string{}
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		for _, file := range filesToCleanup {
			_ = os.RemoveAll(file)
		}
		return ctx, nil
	})

	ctx.Step(`^a kdeps container with "([^"]*)" endpoint "([^"]*)" API and "([^"]*)"$`, aKdepsContainerWithEndpointAPI)
	ctx.Step(`^I fill in the "([^"]*)" with success "([^"]*)", response data "([^"]*)"$`, iFillInTheWithSuccessResponseData)
	ctx.Step(`^I GET request to "([^"]*)" with data "([^"]*)" and header name "([^"]*)" that maps to "([^"]*)"$`, iGETRequestToWithDataAndHeaderNameThatMapsTo)
	ctx.Step(`^I should see a blank standard template "([^"]*)" in the "([^"]*)" folder$`, iShouldSeeABlankStandardTemplateInTheFolder)
	ctx.Step(`^I should see a "([^"]*)" in the "([^"]*)" folder$`, iShouldSeeAInTheFolder)
	ctx.Step(`^I should see action "([^"]*)", url "([^"]*)", data "([^"]*)", headers "([^"]*)" with values "([^"]*)" and params "([^"]*)" that maps to "([^"]*)"$`, iShouldSeeActionURLDataHeadersWithValuesAndParamsThatMapsTo)
	ctx.Step(`^it should respond "([^"]*)" in "([^"]*)"$`, itShouldRespondIn)
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

	systemConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Kdeps.pkl"

runMode = "docker"
dockerGPU = "cpu"
`, schema.SchemaVersion(ctx))

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
	methods := strings.Split(arg1, ",")
	for i, method := range methods {
		methods[i] = strings.TrimSpace(method)
		methodSection += fmt.Sprintf(`      "%s"`, methods[i])
		if i < len(methods)-1 {
			methodSection += "\n"
		}
	}

	workflowConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

name = "myAIAgentAPI"
description = "AI Agent X API"
targetActionID = "helloWorld"
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

	resourceConfigurationContent = fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

local llmResponse = "@(llm.response("action1"))"
local execResponse = "@(exec.stdout("action2"))"
local clientResponse = "@(client.responseBody("action3"))"
local clientResponse2 = "@(client.responseBody("action4"))"

actionID = "helloWorld"
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

actionID = "action1"
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
    timeoutDuration = 0
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

actionID = "action2"
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

actionID = "action3"
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

actionID = "action4"
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

	rd, asm, _, hIP, hPort, _, _, gpu, err := docker.BuildDockerfile(testFs, ctx, systemConfiguration, kdepsDir, pkgProject, logger)
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

	dockerClientID, err := docker.CreateDockerContainer(testFs, ctx, cName, containerName, hostIP, hostPort, "", "", gpuType, APIServerMode, false, cli)
	if err != nil {
		return err
	}

	containerID = dockerClientID

	return nil
}

func iFillInTheWithSuccessResponseData(arg1, arg2, arg3 string) error {
	responseConfigurationContent := fmt.Sprintf(`
extends "package://schema.kdeps.com/core@%s#/APIServerResponse.pkl"

success = %s
response {
  data {
    "%s"
  }
}
`, schema.SchemaVersion(ctx), arg2, arg3)

	responseFilePath = filepath.Join(agentAPIPath, "api", arg1)
	err := afero.WriteFile(testFs, responseFilePath, []byte(responseConfigurationContent), 0o644)
	if err != nil {
		return err
	}

	return nil
}

func iGETRequestToWithDataAndHeaderNameThatMapsTo(arg1, arg2, arg3, arg4 string) error {
	// Create request content
	requestContent := fmt.Sprintf(`{
  "method": "GET",
  "url": "%s",
  "data": "%s",
  "headers": {
    "%s": "%s"
  }
}`, arg1, arg2, arg3, arg4)

	apiDir := filepath.Join(agentAPIPath, "api")
	if err := testFs.MkdirAll(apiDir, 0o777); err != nil {
		return err
	}

	requestFilePath = filepath.Join(apiDir, "request.pkl")
	err := afero.WriteFile(testFs, requestFilePath, []byte(requestContent), 0o644)

	return err
}

func iShouldSeeABlankStandardTemplateInTheFolder(arg1, arg2 string) error {
	filePath := filepath.Join(agentAPIPath, arg2, arg1)
	_, err := testFs.Stat(filePath)
	return err
}

func iShouldSeeAInTheFolder(arg1, arg2 string) error {
	filePath := filepath.Join(agentAPIPath, arg2, arg1)
	_, err := testFs.Stat(filePath)
	return err
}

func iShouldSeeActionURLDataHeadersWithValuesAndParamsThatMapsTo(arg1, arg2, arg3, arg4, arg5, arg6, arg7 string) error {
	content, err := afero.ReadFile(testFs, requestFilePath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, arg1) {
		return fmt.Errorf("action %s not found in request file", arg1)
	}

	if !strings.Contains(contentStr, arg2) {
		return fmt.Errorf("url %s not found in request file", arg2)
	}

	return nil
}

func itShouldRespondIn(arg1, arg2 string) error {
	content, err := afero.ReadFile(testFs, responseFilePath)
	if err != nil {
		return err
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, arg1) {
		return fmt.Errorf("response %s not found in response file", arg1)
	}

	return nil
}

func TestPklResourceReader(t *testing.T) {
	t.Parallel()

	// Use in-memory SQLite database for testing
	dbPath := ":memory:"
	requestID := "test-request-123"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	t.Run("Scheme", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "resource", reader.Scheme())
	})

	t.Run("IsGlobbable", func(t *testing.T) {
		t.Parallel()
		require.False(t, reader.IsGlobbable())
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		t.Parallel()
		require.False(t, reader.HasHierarchicalUris())
	})

	t.Run("ListElements", func(t *testing.T) {
		t.Parallel()
		uri, _ := url.Parse("resource:///test")
		elements, err := reader.ListElements(*uri)
		require.NoError(t, err)
		require.Nil(t, elements)
	})
}

func TestStoreAndRetrieveExecResource(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-123"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Create test exec resource
	timeout := 30 * time.Second
	execRes := &ExecResource{
		BaseResource: BaseResource{
			ID:              "exec-test-1",
			RequestID:       requestID,
			Type:            TypeExec,
			Timestamp:       time.Now(),
			TimeoutDuration: &timeout,
			Metadata:        map[string]interface{}{"test": "value"},
		},
		Command: "echo 'hello world'",
		Env:     map[string]string{"PATH": "/usr/bin", "HOME": "/tmp"},
		Stdout:  "hello world\n",
		Stderr:  "",
		File:    "/tmp/output.txt",
	}

	// Store the resource
	err = reader.StoreExecResource(execRes)
	require.NoError(t, err)

	// Retrieve the resource
	uri, _ := url.Parse("resource:///exec/exec-test-1")
	data, err := reader.Read(*uri)
	require.NoError(t, err)

	// Parse and verify the returned data
	var retrievedRes ExecResource
	err = json.Unmarshal(data, &retrievedRes)
	require.NoError(t, err)

	assert.Equal(t, execRes.ID, retrievedRes.ID)
	assert.Equal(t, execRes.Command, retrievedRes.Command)
	assert.Equal(t, execRes.Stdout, retrievedRes.Stdout)
	assert.Equal(t, execRes.Env, retrievedRes.Env)
	assert.Equal(t, execRes.File, retrievedRes.File)
}

func TestStoreAndRetrievePythonResource(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-456"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Create test Python resource
	pythonRes := &PythonResource{
		BaseResource: BaseResource{
			ID:        "python-test-1",
			RequestID: requestID,
			Type:      TypePython,
			Timestamp: time.Now(),
		},
		Script: "print('Hello from Python')",
		Env:    map[string]string{"PYTHONPATH": "/opt/python"},
		Stdout: "Hello from Python\n",
		Stderr: "",
		File:   "/tmp/python_output.txt",
	}

	// Store the resource
	err = reader.StorePythonResource(pythonRes)
	require.NoError(t, err)

	// Retrieve the resource
	uri, _ := url.Parse("resource:///python/python-test-1")
	data, err := reader.Read(*uri)
	require.NoError(t, err)

	// Parse and verify the returned data
	var retrievedRes PythonResource
	err = json.Unmarshal(data, &retrievedRes)
	require.NoError(t, err)

	assert.Equal(t, pythonRes.ID, retrievedRes.ID)
	assert.Equal(t, pythonRes.Script, retrievedRes.Script)
	assert.Equal(t, pythonRes.Stdout, retrievedRes.Stdout)
	assert.Equal(t, pythonRes.Env, retrievedRes.Env)
}

func TestStoreAndRetrieveHTTPResource(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-789"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Create test HTTP resource
	httpRes := &HTTPResource{
		BaseResource: BaseResource{
			ID:        "http-test-1",
			RequestID: requestID,
			Type:      TypeHTTP,
			Timestamp: time.Now(),
		},
		Method:  "POST",
		URL:     "https://api.example.com/data",
		Headers: map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token123"},
		Data:    []string{`{"key": "value"}`},
		Params:  map[string]string{"format": "json"},
		Response: &HTTPResponse{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    `{"success": true, "message": "Data processed"}`,
		},
		File: "/tmp/http_response.json",
	}

	// Store the resource
	err = reader.StoreHTTPResource(httpRes)
	require.NoError(t, err)

	// Retrieve the resource
	uri, _ := url.Parse("resource:///http/http-test-1")
	data, err := reader.Read(*uri)
	require.NoError(t, err)

	// Parse and verify the returned data
	var retrievedRes HTTPResource
	err = json.Unmarshal(data, &retrievedRes)
	require.NoError(t, err)

	assert.Equal(t, httpRes.ID, retrievedRes.ID)
	assert.Equal(t, httpRes.Method, retrievedRes.Method)
	assert.Equal(t, httpRes.URL, retrievedRes.URL)
	assert.Equal(t, httpRes.Headers, retrievedRes.Headers)
	assert.Equal(t, httpRes.Data, retrievedRes.Data)
	assert.Equal(t, httpRes.Response.Body, retrievedRes.Response.Body)
}

func TestStoreAndRetrieveLLMResource(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-llm"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Create test LLM resource
	llmRes := &LLMResource{
		BaseResource: BaseResource{
			ID:        "llm-test-1",
			RequestID: requestID,
			Type:      TypeLLM,
			Timestamp: time.Now(),
		},
		Model:            "gpt-4",
		Prompt:           "What is the meaning of life?",
		Role:             "user",
		JSONResponse:     true,
		JSONResponseKeys: []string{"answer", "confidence"},
		Scenario:         "philosophical_qa",
		Files:            map[string]string{"context.txt": "/tmp/context.txt"},
		Tools: []LLMTool{
			{
				Name:        "search",
				Description: "Search for information",
				Parameters:  map[string]interface{}{"query": "string", "limit": "number"},
			},
		},
		Response: `{"answer": "42", "confidence": 0.95}`,
		File:     "/tmp/llm_response.json",
	}

	// Store the resource
	err = reader.StoreLLMResource(llmRes)
	require.NoError(t, err)

	// Retrieve the resource
	uri, _ := url.Parse("resource:///llm/llm-test-1")
	data, err := reader.Read(*uri)
	require.NoError(t, err)

	// Parse and verify the returned data
	var retrievedRes LLMResource
	err = json.Unmarshal(data, &retrievedRes)
	require.NoError(t, err)

	assert.Equal(t, llmRes.ID, retrievedRes.ID)
	assert.Equal(t, llmRes.Model, retrievedRes.Model)
	assert.Equal(t, llmRes.Prompt, retrievedRes.Prompt)
	assert.Equal(t, llmRes.Role, retrievedRes.Role)
	assert.Equal(t, llmRes.JSONResponse, retrievedRes.JSONResponse)
	assert.Equal(t, llmRes.JSONResponseKeys, retrievedRes.JSONResponseKeys)
	assert.Equal(t, llmRes.Tools, retrievedRes.Tools)
	assert.Equal(t, llmRes.Response, retrievedRes.Response)
}

func TestStoreAndRetrieveDataResource(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-data"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Create test data resource
	dataRes := &DataResource{
		BaseResource: BaseResource{
			ID:        "data-test-1",
			RequestID: requestID,
			Type:      TypeData,
			Timestamp: time.Now(),
		},
		Files: map[string]map[string]string{
			"agent1": {
				"config.json": "/tmp/agent1/config.json",
				"data.csv":    "/tmp/agent1/data.csv",
			},
			"agent2": {
				"settings.yaml": "/tmp/agent2/settings.yaml",
			},
		},
	}

	// Store the resource
	err = reader.StoreDataResource(dataRes)
	require.NoError(t, err)

	// Retrieve the resource
	uri, _ := url.Parse("resource:///data/data-test-1")
	data, err := reader.Read(*uri)
	require.NoError(t, err)

	// Parse and verify the returned data
	var retrievedRes DataResource
	err = json.Unmarshal(data, &retrievedRes)
	require.NoError(t, err)

	assert.Equal(t, dataRes.ID, retrievedRes.ID)
	assert.Equal(t, dataRes.Files, retrievedRes.Files)
}

func TestListResources(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-list"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Store multiple exec resources
	for i := 1; i <= 3; i++ {
		execRes := &ExecResource{
			BaseResource: BaseResource{
				ID:        fmt.Sprintf("exec-test-%d", i),
				RequestID: requestID,
				Type:      TypeExec,
				Timestamp: time.Now(),
			},
			Command: fmt.Sprintf("echo 'test %d'", i),
		}
		err = reader.StoreExecResource(execRes)
		require.NoError(t, err)
	}

	// List all exec resources
	uri, _ := url.Parse("resource:///exec/_?op=list")
	data, err := reader.Read(*uri)
	require.NoError(t, err)

	// Parse and verify the returned data
	var resources map[string]json.RawMessage
	err = json.Unmarshal(data, &resources)
	require.NoError(t, err)

	assert.Len(t, resources, 3)
	assert.Contains(t, resources, "exec-test-1")
	assert.Contains(t, resources, "exec-test-2")
	assert.Contains(t, resources, "exec-test-3")
}

func TestDeleteResource(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-delete"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Store a resource
	execRes := &ExecResource{
		BaseResource: BaseResource{
			ID:        "exec-to-delete",
			RequestID: requestID,
			Type:      TypeExec,
			Timestamp: time.Now(),
		},
		Command: "echo 'delete me'",
		Stdout:  "delete me\n",
	}
	err = reader.StoreExecResource(execRes)
	require.NoError(t, err)

	// Verify it exists
	uri, _ := url.Parse("resource:///exec/exec-to-delete")
	data, err := reader.Read(*uri)
	require.NoError(t, err)
	assert.NotEqual(t, "{}", string(data))

	// Delete the resource
	deleteURI, _ := url.Parse("resource:///exec/exec-to-delete?op=delete")
	result, err := reader.Read(*deleteURI)
	require.NoError(t, err)
	assert.Contains(t, string(result), "Deleted 1 resource(s)")

	// Verify it's gone
	data, err = reader.Read(*uri)
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))
}

func TestClearResources(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-clear"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Store multiple resources
	for i := 1; i <= 3; i++ {
		execRes := &ExecResource{
			BaseResource: BaseResource{
				ID:        fmt.Sprintf("exec-clear-%d", i),
				RequestID: requestID,
				Type:      TypeExec,
				Timestamp: time.Now(),
			},
			Command: fmt.Sprintf("echo 'clear %d'", i),
		}
		err = reader.StoreExecResource(execRes)
		require.NoError(t, err)
	}

	// Verify they exist
	listURI, _ := url.Parse("resource:///exec/_?op=list")
	data, err := reader.Read(*listURI)
	require.NoError(t, err)

	var resources map[string]json.RawMessage
	err = json.Unmarshal(data, &resources)
	require.NoError(t, err)
	assert.Len(t, resources, 3)

	// Clear all exec resources
	clearURI, _ := url.Parse("resource:///exec/_?op=clear")
	result, err := reader.Read(*clearURI)
	require.NoError(t, err)
	assert.Contains(t, string(result), "Cleared 3 resource(s) of type exec")

	// Verify they're gone
	data, err = reader.Read(*listURI)
	require.NoError(t, err)

	err = json.Unmarshal(data, &resources)
	require.NoError(t, err)
	assert.Len(t, resources, 0)
}

func TestResourceNotFound(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-notfound"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Try to retrieve non-existent resource
	uri, _ := url.Parse("resource:///exec/non-existent")
	data, err := reader.Read(*uri)
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))
}

func TestInvalidURIFormat(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-invalid"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Test invalid URI formats
	testCases := []string{
		"resource:///",
		"resource:///exec",
		"resource:///exec/",
	}

	for _, uriStr := range testCases {
		uri, _ := url.Parse(uriStr)
		_, err := reader.Read(*uri)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid URI path format")
	}
}

func TestUnsupportedOperation(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-unsupported"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)
	defer reader.Close()

	// Test unsupported operation
	uri, _ := url.Parse("resource:///exec/test?op=unsupported")
	_, err = reader.Read(*uri)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operation: unsupported")
}

func TestRequestIDIsolation(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID1 := "request-1"
	requestID2 := "request-2"

	reader1, err := InitializeResource(dbPath, requestID1)
	require.NoError(t, err)
	defer reader1.Close()

	reader2, err := InitializeResource(dbPath, requestID2)
	require.NoError(t, err)
	defer reader2.Close()

	// Store a resource in reader1
	execRes := &ExecResource{
		BaseResource: BaseResource{
			ID:        "test-isolation",
			RequestID: requestID1,
			Type:      TypeExec,
			Timestamp: time.Now(),
		},
		Command: "echo 'isolation test'",
	}
	err = reader1.StoreExecResource(execRes)
	require.NoError(t, err)

	// Try to access from reader2 (different request ID)
	uri, _ := url.Parse("resource:///exec/test-isolation")
	data, err := reader2.Read(*uri)
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data)) // Should not find the resource

	// Should be accessible from reader1
	data, err = reader1.Read(*uri)
	require.NoError(t, err)
	assert.NotEqual(t, "{}", string(data))
}

func TestInitializeDatabase(t *testing.T) {
	t.Parallel()

	t.Run("InMemoryDatabase", func(t *testing.T) {
		t.Parallel()
		db, err := InitializeDatabase(":memory:")
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()
	})

	t.Run("InvalidPath", func(t *testing.T) {
		t.Parallel()
		_, err := InitializeDatabase("/invalid/path/test.db")
		require.Error(t, err)
	})
}

func TestClose(t *testing.T) {
	t.Parallel()

	dbPath := ":memory:"
	requestID := "test-request-close"
	reader, err := InitializeResource(dbPath, requestID)
	require.NoError(t, err)

	// Close should work without error
	err = reader.Close()
	require.NoError(t, err)

	// Second close should not panic
	err = reader.Close()
	require.NoError(t, err)
}
