package resolver_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"

	"github.com/charmbracelet/log"
	"github.com/cucumber/godog"
	"github.com/docker/docker/client"
	"github.com/kdeps/schema/gen/kdeps"
	pklRes "github.com/kdeps/schema/gen/resource"
	wfPkl "github.com/kdeps/schema/gen/workflow"
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
	packageFile               string
	hostPort                  string = "3000"
	hostIP                    string = "127.0.0.1"
	containerID               string
	runDir                    string
	containerName             string
	apiServerMode             bool
	cName                     string
	pkgProject                *archiver.KdepsPackage
	compiledProjectDir        string
	environ                   *environment.Environment
	currentDirPath            string
	systemConfigurationFile   string
	cli                       *client.Client
	systemConfiguration       *kdeps.Kdeps
	visited                   map[string]bool
	actionId                  string
	graphResolver             *resolver.DependencyResolver
	workflowConfigurationFile string
	workflowConfiguration     *wfPkl.Workflow
	schemaVersionFilePath     = "../../SCHEMA_VERSION"
)

func TestFeatures(t *testing.T) {
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

	if err = docker.CreateFlagFile(testFs, filepath.Join(tmpRoot, ".dockerenv")); err != nil {
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

	systemConfigurationFile, err = cfg.FindConfiguration(testFs, environ, logger)
	if err != nil {
		return err
	}

	if err = enforcer.EnforcePklTemplateAmendsRules(testFs, systemConfigurationFile, logger); err != nil {
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
		methodSection = fmt.Sprintf(`
methods {
  "GET"
}`)
	}

	workflowConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.1.1#/Workflow.pkl"

name = "myAIAgentAPI1"
description = "AI Agent X API"
action = "helloWorld9"
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

	filePath = filepath.Join(homeDirPath, "myAgentX1")

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
amends "package://schema.kdeps.com/core@0.1.1#/LLM.pkl"

chat {
  ["Hello"] {
    model = "llama3.2"
    prompt = "prompt"
    response = """
response
"""
  }
}
`)

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
amends "package://schema.kdeps.com/core@0.1.0#/Resource.pkl"

id = "helloWorld%d"
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
`, num, num, num-1)

		// Skip the "requires" for the first resource (num 1)
		//		if num == 1 {
		//			resourceConfigurationContent = fmt.Sprintf(`
		// amends "package://schema.kdeps.com/core@0.1.0#/Resource.pkl"

		// id = "helloWorld%d"
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
	actionId = "helloWorld9"
	visited = make(map[string]bool)

	stack := graphResolver.Graph.BuildDependencyStack(actionId, visited)
	for _, resNode := range stack {
		for _, res := range graphResolver.Resources {
			if res.Id == resNode {
				logger.Debug("Executing resource: ", res.Id)

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
				//			logger.Debug("Skip condition met, skipping:", res.Id)
				//			continue
				//		}
				//	}

				//	// Handle Preflight Check
				//	if runBlock.PreflightCheck != nil && runBlock.PreflightCheck.Validations != nil {
				//		if !resolver.AllConditionsMet(runBlock.PreflightCheck.Validations) {
				//			logger.Error("Preflight check not met, failing:", res.Id)
				//			if runBlock.PreflightCheck.Error != nil {
				//				logger.Debug(err)

				//				//	return graphResolver.HandleAPIErrorResponse(
				//				//		runBlock.PreflightCheck.Error.Code,
				//				//		fmt.Sprintf("%s: %s", runBlock.PreflightCheck.Error.Message, res.Id))
				//			}

				//			// return graphResolver.HandleAPIErrorResponse(500, "Preflight
				//			// check failed for resource: "+res.Id)
				//			logger.Debug(err)

				//		}
				//	}

				//	// API Response
				//	if graphResolver.ApiServerMode && runBlock.ApiResponse != nil {
				//		if err := graphResolver.CreateResponsePklFile(runBlock.ApiResponse); err != nil {
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

	dr, err := resolver.NewGraphResolver(testFs, logger, ctx, environ, agentDir)
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

	actionId = "helloWorld9"
	visited = make(map[string]bool)
	// Build the dependency stack
	stack := graphResolver.Graph.BuildDependencyStack(actionId, visited)

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
// action = "helloWorld100"
// settings {
//   apiServerMode = true
//   agentSettings {
//     packages {}
//     models {
//       "tinydolphin"
//     }
//   }
//   apiServer {
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

// id = "helloWorld%d"
// name = "default action %d"
// description = "default action"
// category = "category"
// %s
// `, num, num, requiresContent)

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
