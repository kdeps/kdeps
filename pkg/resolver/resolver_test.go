package resolver_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/charmbracelet/log"
	"github.com/cucumber/godog"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	"github.com/kdeps/schema/gen/kdeps"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	pklRes "github.com/kdeps/schema/gen/resource"
	"github.com/spf13/afero"
)

var (
	testFs                    = afero.NewOsFs()
	testingT                  *testing.T
	homeDirPath               string
	logger                    *logging.Logger
	kdepsDir                  string
	agentDir                  string
	ctx                       context.Context
	environ                   *environment.Environment
	currentDirPath            string
	systemConfigurationFile   string
	systemConfiguration       *kdeps.Kdeps
	visited                   map[string]bool
	actionID                  string
	graphResolver             *resolver.DependencyResolver
	workflowConfigurationFile string
)

func TestFeatures(t *testing.T) {
	t.Parallel()
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^an ai agent with "([^"]*)" resources$`, anAiAgentWithResources)
			ctx.Step(`^each resource are reloaded when opened$`, eachResourceAreReloadedWhenOpened)
			ctx.Step(`^I load the workflow resources$`, iLoadTheWorkflowResources)
			ctx.Step(`^I was able to see the "([^"]*)" top-down dependencies$`, iWasAbleToSeeTheTopdownDependencies)
			// ctx.Step(`^an ai agent with "([^"]*)" resources that was configured differently$`, anAiAgentWithResources2)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/resolver"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func anAiAgentWithResources(arg1 string) error {
	logger = logging.GetLogger()

	tmpRoot, err := afero.TempDir(testFs, "", "")
	if err != nil {
		return err
	}

	if err = docker.CreateFlagFile(testFs, ctx, filepath.Join(tmpRoot, ".dockerenv")); err != nil {
		return err
	}

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

	envStruct := &environment.Environment{
		Root:           tmpRoot,
		Home:           homeDirPath,
		Pwd:            currentDirPath,
		NonInteractive: "1",
		DockerMode:     "1",
	}

	env, err := environment.NewEnvironment(testFs, envStruct)
	if err != nil {
		return err
	}

	environ = env

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

	methods := "POST, GET"
	var methodSection string
	if strings.Contains(methods, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(methods, ",")
		var methodLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			methodLines = append(methodLines, fmt.Sprintf(`"%s"`, value))
		}
		methodSection = "methods {\n" + strings.Join(methodLines, "\n") + "\n}"
	} else {
		// Single value case
		methodSection = `
methods {
  "GET"
}`
	}

	workflowConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

name = "TestWorkflow"
description = "AI Agent X API"
targetActionID = "TestAction"
settings {
  APIServerMode = true
  agentSettings {
    packages {}
    models {
      "tinydolphin"
    }
  }
  APIServer {
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
`, schema.SchemaVersion(ctx), methodSection, methodSection)
	filePath := filepath.Join(homeDirPath, "myAgentX1")

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

	apiDir := filepath.Join(filePath, "/actions/api/")
	if err := testFs.MkdirAll(apiDir, 0o777); err != nil {
		return err
	}

	projectDir := filepath.Join(filePath, "/project/")
	if err := testFs.MkdirAll(projectDir, 0o777); err != nil {
		return err
	}

	llmDir := filepath.Join(filePath, "/actions/llm/")
	if err := testFs.MkdirAll(llmDir, 0o777); err != nil {
		return err
	}

	llmResponsesContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/LLM.pkl"

chat {
  ["Hello"] {
    model = "llama3.2"
    prompt = "prompt"
    response = """
response
"""
  }
}
`, schema.SchemaVersion(ctx))

	llmDirFile := filepath.Join(llmDir, "llm_output.pkl")
	err = afero.WriteFile(testFs, llmDirFile, []byte(llmResponsesContent), 0o644)
	if err != nil {
		return err
	}

	clientDir := filepath.Join(filePath, "/actions/client/")
	if err := testFs.MkdirAll(clientDir, 0o777); err != nil {
		return err
	}

	execDir := filepath.Join(filePath, "/actions/exec/")
	if err := testFs.MkdirAll(execDir, 0o777); err != nil {
		return err
	}

	// Convert totalResources from string to int
	totalResourcesInt, err := strconv.Atoi(arg1)
	if err != nil {
		return fmt.Errorf("failed to convert totalResources to int: %w", err)
	}

	for num := totalResourcesInt; num >= 1; num-- {
		// Define the content of the resource configuration file
		resourceConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

actionID = "helloWorld%d"
name = "default action %d"
description = """
  default action
"""
category = "category"
requires {
  "helloWorld%d"
}
run {
  chat {
    model = "tinydolphin"
    prompt = "who was "
  }
}
`, schema.SchemaVersion(ctx), num, num, num-1)

		// Skip the "requires" for the first resource (num 1)
		//		if num == 1 {
		//			resourceConfigurationContent = fmt.Sprintf(`
		// amends "package://schema.kdeps.com/core@0.1.0#/Resource.pkl"

		// actionID = "helloWorld%d"
		// name = "default action %d"
		// description = "default action @(request.url)"
		// category = "category"
		// requires {}
		// run {}
		// `, num, num)
		//		}

		// Define the file path
		resourceConfigurationFile := filepath.Join(resourcesDir, fmt.Sprintf("resource%d.pkl", num))

		// Write the file content using afero
		err := afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}

func eachResourceAreReloadedWhenOpened() error {
	actionID = "helloWorld9"
	visited = make(map[string]bool)

	stack := graphResolver.Graph.BuildDependencyStack(actionID, visited)
	for _, resNode := range stack {
		for _, res := range graphResolver.Resources {
			if res.ActionID == resNode {
				logger.Debug("executing resource: ", res.ActionID)

				rsc, err := pklRes.LoadFromPath(graphResolver.Context, res.File)
				if err != nil {
					logger.Debug(err)
					// return graphResolver.HandleAPIErrorResponse(500, err.Error())
				}

				logger.Debug(rsc.Description)

				// runBlock := rsc.Run
				// if runBlock != nil {

				//	// Check Skip Condition
				//	if runBlock.SkipCondition != nil {
				//		if resolver.ShouldSkip(runBlock.SkipCondition) {
				//			logger.Debug("skip condition met, skipping:", res.ActionID)
				//			continue
				//		}
				//	}

				//	// Handle Preflight Check
				//	if runBlock.PreflightCheck != nil && runBlock.PreflightCheck.Validations != nil {
				//		if !resolver.AllConditionsMet(runBlock.PreflightCheck.Validations) {
				//			logger.Error("preflight check not met, failing:", res.ActionID)
				//			if runBlock.PreflightCheck.Error != nil {
				//				logger.Debug(err)

				//				//	return graphResolver.HandleAPIErrorResponse(
				//				//		runBlock.PreflightCheck.Error.Code,
				//				//		fmt.Sprintf("%s: %s", runBlock.PreflightCheck.Error.Message, res.ActionID))
				//			}

				//			// return graphResolver.HandleAPIErrorResponse(500, "Preflight
				//			// check failed for resource: "+res.ActionID)
				//			logger.Debug(err)

				//		}
				//	}

				//	// API Response
				//	if graphResolver.APIServerMode && runBlock.APIResponse != nil {
				//		if err := graphResolver.CreateResponsePklFile(runBlock.APIResponse); err != nil {
				//			logger.Debug(err)

				//			// return graphResolver.HandleAPIErrorResponse(500, err.Error())
				//		}
				//	}
				// }
			}
		}
	}

	return nil
}

func iLoadTheWorkflowResources() error {
	logger := logging.GetLogger()
	ctx = context.Background()

	dr, err := resolver.NewGraphResolver(testFs, ctx, environ, nil, logger)
	if err != nil {
		log.Fatal(err)
	}

	graphResolver = dr

	return nil
}

func iWasAbleToSeeTheTopdownDependencies(arg1 string) error {
	// Load resource entries using graphResolver
	if err := graphResolver.LoadResourceEntries(); err != nil {
		return err
	}

	actionID = "helloWorld9"
	visited = make(map[string]bool)
	// Build the dependency stack
	stack := graphResolver.Graph.BuildDependencyStack(actionID, visited)

	// Convert arg1 (string) to an integer for comparison with len(stack)
	arg1Int, err := strconv.Atoi(arg1) // Convert string to int
	if err != nil {
		return fmt.Errorf("invalid argument: %s is not a valid number", arg1)
	}

	// Compare the converted integer value with the length of the stack
	if arg1Int != len(stack) {
		return fmt.Errorf("stack not equal, expected %d but got %d", arg1Int, len(stack))
	}

	return nil
}

// func anAiAgentWithResources2(arg1 string) error {
//	tmpRoot, err := afero.TempDir(testFs, "", "")
//	if err != nil {
//		return err
//	}

//	if err = docker.CreateFlagFile(testFs, filepath.Join(tmpRoot, ".dockerenv")); err != nil {
//		return err
//	}

//	tmpHome, err := afero.TempDir(testFs, "", "")
//	if err != nil {
//		return err
//	}

//	tmpCurrent, err := afero.TempDir(testFs, "", "")
//	if err != nil {
//		return err
//	}

//	var dirPath string

//	homeDirPath = tmpHome
//	currentDirPath = tmpCurrent

//	dirPath = filepath.Join(homeDirPath, ".kdeps")

//	if err := testFs.MkdirAll(dirPath, 0777); err != nil {
//		return err
//	}

//	kdepsDir = dirPath

//	env := &environment.Environment{
//		Root:           tmpRoot,
//		Home:           homeDirPath,
//		Pwd:            currentDirPath,
//		NonInteractive: "1",
//		DockerMode:     "1",
//	}

//	environ, err := environment.NewEnvironment(testFs, env)
//	if err != nil {
//		return err
//	}

//	systemConfigurationContent := `
//	amends "package://schema.kdeps.com/core@0.0.44#/Kdeps.pkl"

//	runMode = "docker"
//	dockerGPU = "cpu"
//	`

//	systemConfigurationFile = filepath.Join(homeDirPath, ".kdeps.pkl")
//	// Write the heredoc content to the file
//	err = afero.WriteFile(testFs, systemConfigurationFile, []byte(systemConfigurationContent), 0644)
//	if err != nil {
//		return err
//	}

//	systemConfigurationFile, err = cfg.FindConfiguration(testFs, environ)
//	if err != nil {
//		return err
//	}

//	if err = enforcer.EnforcePklTemplateAmendsRules(testFs, systemConfigurationFile); err != nil {
//		return err
//	}

//	syscfg, err := cfg.LoadConfiguration(testFs, systemConfigurationFile)
//	if err != nil {
//		return err
//	}

//	systemConfiguration = syscfg

//	var methodSection string
//	if strings.Contains(arg1, ",") {
//		// Split arg3 into multiple values if it's a CSV
//		values := strings.Split(arg1, ",")
//		var methodLines []string
//		for _, value := range values {
//			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
//			methodLines = append(methodLines, fmt.Sprintf(`"%s"`, value))
//		}
//		methodSection = "methods {\n" + strings.Join(methodLines, "\n") + "\n}"
//	} else {
//		// Single value case
//		methodSection = fmt.Sprintf(`
// methods {
//   "%s"
// }`, arg1)
//	}

//	workflowConfigurationContent := fmt.Sprintf(`
// amends "package://schema.kdeps.com/core@0.0.44#/Workflow.pkl"

// name = "myAIAgentAPI2"
// description = "AI Agent X API"
// targetActionID = "helloWorld100"
// settings {
//   APIServerMode = true
//   agentSettings {
//     packages {}
//     models {
//       "tinydolphin"
//     }
//   }
//   APIServer {
//     routes {
//       new {
//	path = "/resource1"
//	%s
//	responseType = "json"
//       }
//       new {
//	path = "/resource2"
//	%s
//       }
//     }
//   }
// }
// `, methodSection, methodSection)
//	var filePath string

//	filePath = filepath.Join(homeDirPath, "myAgentX2")

//	if err := testFs.MkdirAll(filePath, 0777); err != nil {
//		return err
//	}

//	agentDir = filePath

//	workflowConfigurationFile = filepath.Join(filePath, "workflow.pkl")
//	err = afero.WriteFile(testFs, workflowConfigurationFile, []byte(workflowConfigurationContent), 0644)
//	if err != nil {
//		return err
//	}

//	resourcesDir := filepath.Join(filePath, "resources")
//	if err := testFs.MkdirAll(resourcesDir, 0777); err != nil {
//		return err
//	}

//	// Convert totalResources from string to int
//	totalResourcesInt, err := strconv.Atoi(arg1)
//	if err != nil {
//		return fmt.Errorf("failed to convert totalResources to int: %w", err)
//	}

//	// Iterate and create resources starting from totalResourcesInt down to 1
//	for num := totalResourcesInt; num >= 1; num-- {
//		// Prepare the dependencies for the current resource
//		var requiresContent string
//		if num > 1 {
//			// Create a list of dependencies from "action1" to "action(num-1)"
//			var dependencies []string
//			for i := 1; i < num; i++ {
//				dependencies = append(dependencies, fmt.Sprintf(`"helloWorld%d"`, i))
//			}
//			// Join the dependencies into a requires block
//			requiresContent = fmt.Sprintf(`requires {
//   %s
// }`, strings.Join(dependencies, "\n  "))
//		}

//		// Define the content of the resource configuration file
//		resourceConfigurationContent := fmt.Sprintf(`
// amends "package://schema.kdeps.com/core@0.0.44#/Resource.pkl"

// actionID = "helloWorld%d"
// name = "default action %d"
// description = "default action"
// category = "category"
// %s
// `, num, requiresContent)

//		// Define the file path
//		resourceConfigurationFile := filepath.Join(resourcesDir, fmt.Sprintf("resource%d.pkl", num))

//		// Write the file content using afero
//		err := afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0644)
//		if err != nil {
//			return err
//		}

//		fmt.Println("config 2: ", resourceConfigurationFile)
//	}

//	return nil
// }

// Helper to set up context and directories for NewGraphResolver
func setupResolverTestEnv(fs afero.Fs, agentDir, graphID, actionDir string) (context.Context, error) {
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)

	workflowDir := filepath.Join(agentDir, "workflow")
	projectDir := filepath.Join(agentDir, "project")
	filesDir := filepath.Join(actionDir, "files")
	dirs := []string{workflowDir, projectDir, actionDir, filesDir}
	for _, dir := range dirs {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return nil, nil // ctx is not used by the caller
}

// MockPklLoader is a mock implementation of the Pkl loader for testing
type MockPklLoader struct {
	LoadFunc func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error)
}

func (m *MockPklLoader) Load(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
	return m.LoadFunc(ctx, path, resourceType)
}

func setupTestResolver(t *testing.T) (*resolver.DependencyResolver, *MockPklLoader) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-resolver-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	// Create necessary subdirectories
	agentDir := filepath.Join(tempDir, "agent")
	workflowDir := filepath.Join(tempDir, "workflow")
	actionDir := filepath.Join(tempDir, "action")
	filesDir := filepath.Join(tempDir, "files")

	dirs := []string{agentDir, workflowDir, actionDir, filesDir}
	if err := utils.CreateDirectories(afero.NewOsFs(), context.Background(), dirs); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	// Write a minimal valid workflow.pkl file
	workflowSubdir := filepath.Join(agentDir, "workflow")
	if err := os.MkdirAll(workflowSubdir, 0o755); err != nil {
		t.Fatalf("Failed to create workflow subdir: %v", err)
	}
	workflowContent := `amends "package://schema.kdeps.com/core@0.2.30#/Workflow.pkl"
name = "TestWorkflow"
description = "Test workflow"
version = "1.0.0"
targetActionID = "TestAction"
`
	if err := os.WriteFile(filepath.Join(workflowSubdir, "workflow.pkl"), []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow.pkl: %v", err)
	}

	// Set up environment
	os.Setenv("KDEPS_DIR", tempDir)

	// Create mock loader
	mockLoader := &MockPklLoader{
		LoadFunc: func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
			// Return a fixed timestamp value for all resource types
			timestamp := &pkl.Duration{
				Value: 1234567890,
				Unit:  pkl.Second,
			}

			switch resourceType {
			case resolver.ExecResource:
				return &pklExec.ExecImpl{
					Resources: &map[string]*pklExec.ResourceExec{
						"test-exec": {
							Timestamp: timestamp,
						},
					},
				}, nil
			case resolver.PythonResource:
				return &pklPython.PythonImpl{
					Resources: &map[string]*pklPython.ResourcePython{
						"test-python": {
							Timestamp: timestamp,
						},
					},
				}, nil
			case resolver.LLMResource:
				return &pklLLM.LLMImpl{
					Resources: &map[string]*pklLLM.ResourceChat{
						"test-llm": {
							Timestamp: timestamp,
						},
					},
				}, nil
			case resolver.HTTPResource:
				return &pklHTTP.HTTPImpl{
					Resources: &map[string]*pklHTTP.ResourceHTTPClient{
						"test-client": {
							Timestamp: timestamp,
						},
					},
				}, nil
			default:
				return nil, fmt.Errorf("unsupported resourceType %s provided", resourceType)
			}
		},
	}

	// Create a new resolver instance
	fs := afero.NewOsFs()
	ctx := context.Background()
	env := &environment.Environment{
		Root:           tempDir,
		Home:           tempDir,
		Pwd:            tempDir,
		NonInteractive: "1",
		DockerMode:     "1",
	}
	logger := logging.GetLogger()
	req := &gin.Context{}

	// Set up context with required values
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, "test-graph")
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)

	resolver, err := resolver.NewGraphResolver(fs, ctx, env, req, logger)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	resolver.LoadResourceFunc = mockLoader.LoadFunc

	return resolver, mockLoader
}

func TestNewGraphResolver(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (*resolver.DependencyResolver, error)
		wantErr bool
	}{
		{
			name: "successful_initialization",
			setup: func(t *testing.T) (*resolver.DependencyResolver, error) {
				_, resolver := setupTestWorkflow(t)
				return resolver, nil
			},
			wantErr: false,
		},
		{
			name: "missing_workflow_file",
			setup: func(t *testing.T) (*resolver.DependencyResolver, error) {
				fs := afero.NewOsFs()
				ctx := context.Background()
				env := &environment.Environment{
					Root:           "/nonexistent",
					Home:           "/nonexistent",
					Pwd:            "/nonexistent",
					NonInteractive: "1",
					DockerMode:     "1",
				}
				logger := logging.GetLogger()
				return resolver.NewGraphResolver(fs, ctx, env, nil, logger)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := tt.setup(t)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGraphResolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && resolver == nil {
				t.Error("NewGraphResolver() returned nil resolver when no error expected")
			}
		})
	}
}

func TestValidateRequestParams(t *testing.T) {
	testDir, resolver := setupTestWorkflow(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name           string
		fileContent    string
		allowedParams  []string
		expectedResult error
	}{
		{
			name:           "valid_params",
			fileContent:    "request.params(\"param1\") request.params(\"param2\")",
			allowedParams:  []string{"param1", "param2"},
			expectedResult: nil,
		},
		{
			name:           "invalid_param",
			fileContent:    "request.params(\"param1\") request.params(\"param3\")",
			allowedParams:  []string{"param1", "param2"},
			expectedResult: fmt.Errorf("invalid parameter: param3"),
		},
		{
			name:           "empty_allowed_params",
			fileContent:    "request.params(\"param1\")",
			allowedParams:  []string{},
			expectedResult: fmt.Errorf("no parameters allowed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ValidateRequestParams(tt.fileContent, tt.allowedParams)
			if (result == nil) != (tt.expectedResult == nil) {
				t.Errorf("ValidateRequestParams() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestValidateRequestHeaders(t *testing.T) {
	testDir, resolver := setupTestWorkflow(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name           string
		fileContent    string
		allowedHeaders []string
		expectedResult error
	}{
		{
			name:           "valid_headers",
			fileContent:    "request.header(\"header1\") request.header(\"header2\")",
			allowedHeaders: []string{"header1", "header2"},
			expectedResult: nil,
		},
		{
			name:           "invalid_header",
			fileContent:    "request.header(\"header1\") request.header(\"header3\")",
			allowedHeaders: []string{"header1", "header2"},
			expectedResult: fmt.Errorf("invalid header: header3"),
		},
		{
			name:           "empty_allowed_headers",
			fileContent:    "request.header(\"header1\")",
			allowedHeaders: []string{},
			expectedResult: fmt.Errorf("no headers allowed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ValidateRequestHeaders(tt.fileContent, tt.allowedHeaders)
			if (result == nil) != (tt.expectedResult == nil) {
				t.Errorf("ValidateRequestHeaders() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestValidateRequestPath(t *testing.T) {
	testDir, resolver := setupTestWorkflow(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name           string
		path           string
		allowedPaths   []string
		expectedResult error
	}{
		{
			name:           "valid_path",
			path:           "/resource1",
			allowedPaths:   []string{"/resource1", "/resource2"},
			expectedResult: nil,
		},
		{
			name:           "invalid_path",
			path:           "/resource3",
			allowedPaths:   []string{"/resource1", "/resource2"},
			expectedResult: fmt.Errorf("path /resource3 not in the allowed routes"),
		},
		{
			name:           "empty_allowed_paths",
			path:           "/resource1",
			allowedPaths:   []string{},
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", tt.path, nil)
			result := resolver.ValidateRequestPath(c, tt.allowedPaths)
			if (result == nil) != (tt.expectedResult == nil) {
				t.Errorf("ValidateRequestPath() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestValidateRequestMethod(t *testing.T) {
	testDir, resolver := setupTestWorkflow(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name           string
		method         string
		allowedMethods []string
		expectedResult error
	}{
		{
			name:           "valid_method",
			method:         "GET",
			allowedMethods: []string{"GET", "POST"},
			expectedResult: nil,
		},
		{
			name:           "invalid_method",
			method:         "PUT",
			allowedMethods: []string{"GET", "POST"},
			expectedResult: fmt.Errorf("method PUT not in the allowed HTTP methods"),
		},
		{
			name:           "empty_allowed_methods",
			method:         "GET",
			allowedMethods: []string{},
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tt.method, "/test", nil)
			result := resolver.ValidateRequestMethod(c, tt.allowedMethods)
			if (result == nil) != (tt.expectedResult == nil) {
				t.Errorf("ValidateRequestMethod() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestLoadResource(t *testing.T) {
	tests := []struct {
		name          string
		resourceType  resolver.ResourceType
		setup         func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader)
		expectError   bool
		expectedError string
	}{
		{
			name:         "successful exec resource load",
			resourceType: resolver.ExecResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklExec.ExecImpl{}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "successful python resource load",
			resourceType: resolver.PythonResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklPython.PythonImpl{}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "successful llm resource load",
			resourceType: resolver.LLMResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklLLM.LLMImpl{}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "successful http resource load",
			resourceType: resolver.HTTPResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklHTTP.HTTPImpl{
						Resources: &map[string]*pklHTTP.ResourceHTTPClient{
							"test-client": {
								Timestamp: &pkl.Duration{
									Value: 1234567890,
									Unit:  pkl.Second,
								},
							},
						},
					}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "load error",
			resourceType: resolver.ExecResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return nil, fmt.Errorf("mock load error")
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError:   true,
			expectedError: "mock load error",
		},
		{
			name:         "invalid resource type",
			resourceType: resolver.ResourceType("invalid"),
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError:   true,
			expectedError: "unsupported resourceType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr, mockLoader := setupTestResolver(t)
			tt.setup(dr, mockLoader)

			// Test resource loading
			_, err := dr.LoadResource(dr.Context, "/test/path", tt.resourceType)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetResourceFilePath(t *testing.T) {
	tests := []struct {
		name         string
		resourceType resolver.ResourceType
		expectError  bool
	}{
		{
			name:         "valid llm type",
			resourceType: resolver.LLMResource,
			expectError:  false,
		},
		{
			name:         "valid client type",
			resourceType: resolver.HTTPResource,
			expectError:  false,
		},
		{
			name:         "valid exec type",
			resourceType: resolver.ExecResource,
			expectError:  false,
		},
		{
			name:         "valid python type",
			resourceType: resolver.PythonResource,
			expectError:  false,
		},
		{
			name:         "invalid type",
			resourceType: resolver.ResourceType("invalid"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr, _ := setupTestResolver(t)
			dr.RequestID = "test123"

			path, err := dr.GetResourceFilePath(tt.resourceType)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Verify the path structure without checking the exact temporary directory
				expectedSuffix := fmt.Sprintf("%s/test123__%s_output.pkl", tt.resourceType, tt.resourceType)
				if tt.resourceType == resolver.HTTPResource {
					expectedSuffix = "client/test123__client_output.pkl"
				}
				if !strings.HasSuffix(path, expectedSuffix) {
					t.Errorf("path %q does not end with expected suffix %q", path, expectedSuffix)
				}
			}
		})
	}
}

func TestLoadPKLFile(t *testing.T) {
	tests := []struct {
		name          string
		resourceType  resolver.ResourceType
		setup         func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader)
		expectError   bool
		expectedError string
	}{
		{
			name:         "successful exec resource load",
			resourceType: resolver.ExecResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklExec.ExecImpl{}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "successful python resource load",
			resourceType: resolver.PythonResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklPython.PythonImpl{}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "successful llm resource load",
			resourceType: resolver.LLMResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklLLM.LLMImpl{}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "successful client resource load",
			resourceType: resolver.HTTPResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklHTTP.HTTPImpl{
						Resources: &map[string]*pklHTTP.ResourceHTTPClient{
							"test-client": {
								Timestamp: &pkl.Duration{
									Value: 1234567890,
									Unit:  pkl.Second,
								},
							},
						},
					}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "load error",
			resourceType: resolver.ExecResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return nil, fmt.Errorf("mock load error")
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError:   true,
			expectedError: "mock load error",
		},
		{
			name:         "invalid resource type",
			resourceType: resolver.ResourceType("invalid"),
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError:   true,
			expectedError: "unsupported resourceType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr, mockLoader := setupTestResolver(t)
			tt.setup(dr, mockLoader)

			// Get the actual path for the resource type
			path, err := dr.GetResourceFilePath(tt.resourceType)
			if err != nil {
				if tt.expectError {
					// If we expect an error, check the error message and return
					if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
						t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
					}
					return
				} else {
					t.Fatalf("Failed to get resource file path: %v", err)
				}
			}

			// Only create the directory and file if we are not testing invalid resource type
			if !tt.expectError || (tt.expectError && tt.name == "load error") {
				dir := filepath.Dir(path)
				if err := dr.Fs.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}

				// Create an empty PKL file
				if err := afero.WriteFile(dr.Fs, path, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create PKL file: %v", err)
				}
			}

			// Test PKL file loading
			_, err = dr.LoadPKLFile(tt.resourceType, path)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetCurrentTimestamp(t *testing.T) {
	tests := []struct {
		name          string
		resourceID    string
		resourceType  resolver.ResourceType
		setup         func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader)
		expectError   bool
		expectedError string
	}{
		{
			name:         "successful exec timestamp",
			resourceID:   "test-exec",
			resourceType: resolver.ExecResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklExec.ExecImpl{
						Resources: &map[string]*pklExec.ResourceExec{
							"test-exec": {
								Timestamp: &pkl.Duration{
									Value: 1234567890,
									Unit:  pkl.Second,
								},
							},
						},
					}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "successful python timestamp",
			resourceID:   "test-python",
			resourceType: resolver.PythonResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklPython.PythonImpl{
						Resources: &map[string]*pklPython.ResourcePython{
							"test-python": {
								Timestamp: &pkl.Duration{
									Value: 1234567890,
									Unit:  pkl.Second,
								},
							},
						},
					}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "successful llm timestamp",
			resourceID:   "test-llm",
			resourceType: resolver.LLMResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklLLM.LLMImpl{
						Resources: &map[string]*pklLLM.ResourceChat{
							"test-llm": {
								Timestamp: &pkl.Duration{
									Value: 1234567890,
									Unit:  pkl.Second,
								},
							},
						},
					}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "successful client timestamp",
			resourceID:   "test-client",
			resourceType: resolver.HTTPResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklHTTP.HTTPImpl{
						Resources: &map[string]*pklHTTP.ResourceHTTPClient{
							"test-client": {
								Timestamp: &pkl.Duration{
									Value: 1234567890,
									Unit:  pkl.Second,
								},
							},
						},
					}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError: false,
		},
		{
			name:         "load error",
			resourceID:   "test-exec",
			resourceType: resolver.ExecResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return nil, fmt.Errorf("mock load error")
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError:   true,
			expectedError: "failed to load ExecResource PKL file: mock load error",
		},
		{
			name:         "invalid resource type",
			resourceID:   "test-invalid",
			resourceType: resolver.ResourceType("invalid"),
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError:   true,
			expectedError: "unsupported resourceType",
		},
		{
			name:         "resource not found",
			resourceID:   "nonexistent",
			resourceType: resolver.ExecResource,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				mockLoader.LoadFunc = func(ctx context.Context, path string, resourceType resolver.ResourceType) (interface{}, error) {
					return &pklExec.ExecImpl{
						Resources: &map[string]*pklExec.ResourceExec{
							"test-exec": {
								Timestamp: &pkl.Duration{
									Value: 1234567890,
									Unit:  pkl.Second,
								},
							},
						},
					}, nil
				}
				dr.LoadResourceFunc = mockLoader.Load
			},
			expectError:   true,
			expectedError: "resource ID nonexistent does not exist in the file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr, mockLoader := setupTestResolver(t)
			tt.setup(dr, mockLoader)

			// Test getting current timestamp
			_, err := dr.GetCurrentTimestamp(tt.resourceID, tt.resourceType)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestProcessResourceStep(t *testing.T) {
	tests := []struct {
		name          string
		resourceID    string
		step          string
		timeout       *pkl.Duration
		setup         func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader)
		handler       func() error
		expectError   bool
		expectedError string
	}{
		{
			name:       "successful resource processing",
			resourceID: "test-resource",
			step:       "test-step",
			timeout: &pkl.Duration{
				Value: 30,
				Unit:  pkl.Second,
			},
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				// Mock timestamp retrieval
				dr.GetCurrentTimestampFunc = func(resourceID, step string) (string, error) {
					return "1234567890", nil
				}
				// Mock timestamp change wait
				dr.WaitForTimestampChangeFunc = func(resourceID, timestamp string, timeout time.Duration, step string) error {
					return nil
				}
			},
			handler: func() error {
				return nil
			},
			expectError: false,
		},
		{
			name:       "timestamp retrieval error",
			resourceID: "test-resource",
			step:       "test-step",
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				dr.GetCurrentTimestampFunc = func(resourceID, step string) (string, error) {
					return "", fmt.Errorf("timestamp retrieval error")
				}
			},
			handler: func() error {
				return nil
			},
			expectError:   true,
			expectedError: "test-step error: timestamp retrieval error",
		},
		{
			name:       "handler execution error",
			resourceID: "test-resource",
			step:       "test-step",
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				dr.GetCurrentTimestampFunc = func(resourceID, step string) (string, error) {
					return "1234567890", nil
				}
			},
			handler: func() error {
				return fmt.Errorf("handler execution error")
			},
			expectError:   true,
			expectedError: "test-step error: handler execution error",
		},
		{
			name:       "timestamp change wait error",
			resourceID: "test-resource",
			step:       "test-step",
			timeout: &pkl.Duration{
				Value: 30,
				Unit:  pkl.Second,
			},
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				dr.GetCurrentTimestampFunc = func(resourceID, step string) (string, error) {
					return "1234567890", nil
				}
				dr.WaitForTimestampChangeFunc = func(resourceID, timestamp string, timeout time.Duration, step string) error {
					return fmt.Errorf("timestamp change wait error")
				}
			},
			handler: func() error {
				return nil
			},
			expectError:   true,
			expectedError: "test-step timeout awaiting for output: timestamp change wait error",
		},
		{
			name:       "default timeout",
			resourceID: "test-resource",
			step:       "test-step",
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				dr.GetCurrentTimestampFunc = func(resourceID, step string) (string, error) {
					return "1234567890", nil
				}
				dr.WaitForTimestampChangeFunc = func(resourceID, timestamp string, timeout time.Duration, step string) error {
					if timeout != 60*time.Second {
						return fmt.Errorf("unexpected timeout duration: %v", timeout)
					}
					return nil
				}
			},
			handler: func() error {
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr, _ := setupTestResolver(t)
			tt.setup(dr, nil)

			err := dr.ProcessResourceStep(tt.resourceID, tt.step, tt.timeout, tt.handler)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestHandleExec(t *testing.T) {
	tests := []struct {
		name          string
		actionID      string
		execBlock     *pklExec.ResourceExec
		setup         func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader)
		expectError   bool
		expectedError string
	}{
		{
			name:     "successful exec handling",
			actionID: "test-action",
			execBlock: &pklExec.ResourceExec{
				Command: "echo test",
				Env: &map[string]string{
					"TEST_ENV": "test_value",
				},
				Stdout: func() *string {
					s := utils.EncodeValue("test output")
					return &s
				}(),
				Stderr: func() *string {
					s := utils.EncodeValue("")
					return &s
				}(),
			},
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				// Mock successful decoding and processing
				dr.DecodeExecBlockFunc = func(block *pklExec.ResourceExec) error {
					return nil
				}
				dr.ProcessExecBlockFunc = func(actionID string, block *pklExec.ResourceExec) error {
					return nil
				}
			},
			expectError: false,
		},
		{
			name:     "decode error",
			actionID: "test-action",
			execBlock: &pklExec.ResourceExec{
				Command: "echo test",
			},
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				dr.DecodeExecBlockFunc = func(block *pklExec.ResourceExec) error {
					return fmt.Errorf("decode error")
				}
			},
			expectError:   true,
			expectedError: "decode error",
		},
		{
			name:     "process error",
			actionID: "test-action",
			execBlock: &pklExec.ResourceExec{
				Command: "echo test",
			},
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				dr.DecodeExecBlockFunc = func(block *pklExec.ResourceExec) error {
					return nil
				}
				dr.ProcessExecBlockFunc = func(actionID string, block *pklExec.ResourceExec) error {
					return fmt.Errorf("process error")
				}
			},
			expectError:   true,
			expectedError: "process error",
		},
		{
			name:      "nil exec block",
			actionID:  "test-action",
			execBlock: nil,
			setup: func(dr *resolver.DependencyResolver, mockLoader *MockPklLoader) {
				// No setup needed
			},
			expectError:   true,
			expectedError: "nil exec block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr, mockLoader := setupTestResolver(t)
			tt.setup(dr, mockLoader)

			err := dr.HandleExec(tt.actionID, tt.execBlock)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func setupTestWorkflow(t *testing.T) (string, *resolver.DependencyResolver) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "test-workflow-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set KDEPS_DIR environment variable
	os.Setenv("KDEPS_DIR", tmpDir)

	// Create a temporary .kdeps directory
	// Create the agent directory structure
	agentDir := filepath.Join(tmpDir, "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}

	// Create the workflow directory
	workflowDir := filepath.Join(agentDir, "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("Failed to create workflow dir: %v", err)
	}

	// Write a minimal valid workflow.pkl file
	workflowSubdir := filepath.Join(agentDir, "workflow")
	if err := os.MkdirAll(workflowSubdir, 0o755); err != nil {
		t.Fatalf("Failed to create workflow subdir: %v", err)
	}
	workflowContent := `amends "package://schema.kdeps.com/core@0.2.30#/Workflow.pkl"
name = "TestWorkflow"
description = "Test workflow"
version = "1.0.0"
targetActionID = "TestAction"
`
	if err := os.WriteFile(filepath.Join(workflowSubdir, "workflow.pkl"), []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow.pkl: %v", err)
	}

	// Create test environment
	fs := afero.NewOsFs()
	ctx := context.Background()
	env := &environment.Environment{
		Root:           tmpDir,
		Home:           tmpDir,
		Pwd:            tmpDir,
		NonInteractive: "1",
		DockerMode:     "1",
	}
	logger := logging.GetLogger()

	// Set up context with required values
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, "test-graph")
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, filepath.Join(agentDir, "action"))

	// Create resolver
	resolver, err := resolver.NewGraphResolver(fs, ctx, env, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	return tmpDir, resolver
}
