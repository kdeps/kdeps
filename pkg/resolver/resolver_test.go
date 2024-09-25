package resolver

import (
	"context"
	"fmt"
	"kdeps/pkg/archiver"
	"kdeps/pkg/cfg"
	"kdeps/pkg/enforcer"
	"kdeps/pkg/logging"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

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
			ctx.Step(`^an ai agent with "([^"]*)" resources$`, anAiAgentWithResources)
			ctx.Step(`^each resource are reloaded when opened$`, eachResourceAreReloadedWhenOpened)
			ctx.Step(`^I load the workflow resources$`, iLoadTheWorkflowResources)
			ctx.Step(`^I was able to see the "([^"]*)" top-down dependencies$`, iWasAbleToSeeTheTopdownDependencies)
			ctx.Step(`^an ai agent with "([^"]*)" resources that was configured differently$`, anAiAgentWithResources2)

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

	env := &cfg.Environment{
		Home:           homeDirPath,
		Pwd:            currentDirPath,
		NonInteractive: "1",
	}

	systemConfigurationContent := `
	amends "package://schema.kdeps.com/core@0.0.44#/Kdeps.pkl"

	runMode = "docker"
	dockerGPU = "cpu"
	`

	systemConfigurationFile = filepath.Join(homeDirPath, ".kdeps.pkl")
	// Write the heredoc content to the file
	err = afero.WriteFile(testFs, systemConfigurationFile, []byte(systemConfigurationContent), 0644)
	if err != nil {
		return err
	}

	systemConfigurationFile, err = cfg.FindConfiguration(testFs, env)
	if err != nil {
		return err
	}

	if err = enforcer.EnforcePklTemplateAmendsRules(testFs, systemConfigurationFile); err != nil {
		return err
	}

	syscfg, err := cfg.LoadConfiguration(testFs, systemConfigurationFile)
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
  "%s"
}`, arg1)
	}

	workflowConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.44#/Workflow.pkl"

name = "myAIAgentAPI1"
description = "AI Agent X API"
action = "helloWorld99"
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

	// Convert totalResources from string to int
	totalResourcesInt, err := strconv.Atoi(arg1)
	if err != nil {
		return fmt.Errorf("failed to convert totalResources to int: %w", err)
	}

	for num := totalResourcesInt; num >= 1; num-- {
		// Define the content of the resource configuration file
		resourceConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.44#/Resource.pkl"

id = "helloWorld%d"
name = "default action %d"
description = "default action {{request.method}}}"
category = "category"
requires {
  "helloWorld%d"
}
`, num, num, num-1)

		// Skip the "requires" for the first resource (num 1)
		if num == 1 {
			resourceConfigurationContent = fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.44#/Resource.pkl"

id = "helloWorld%d"
name = "default action %d"
description = "default action {{request.url}}"
category = "category"
`, num, num)
		}

		// Define the file path
		resourceConfigurationFile := filepath.Join(resourcesDir, fmt.Sprintf("resource%d.pkl", num))

		// Write the file content using afero
		err := afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0644)
		if err != nil {
			return err
		}
	}

	logger := logging.GetLogger()
	ctx := context.Background()

	dr, err := NewGraphResolver(testFs, logger, ctx, agentDir)
	if err != nil {
		log.Fatal(err)
	}

	dr.HandleRunAction()

	return nil
}

func eachResourceAreReloadedWhenOpened() error {
	return godog.ErrPending
}

func iLoadTheWorkflowResources() error {
	return godog.ErrPending
}

func iWasAbleToSeeTheTopdownDependencies(arg1 string) error {
	return godog.ErrPending
}

func anAiAgentWithResources2(arg1 string) error {
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

	env := &cfg.Environment{
		Home:           homeDirPath,
		Pwd:            currentDirPath,
		NonInteractive: "1",
	}

	systemConfigurationContent := `
	amends "package://schema.kdeps.com/core@0.0.44#/Kdeps.pkl"

	runMode = "docker"
	dockerGPU = "cpu"
	`

	systemConfigurationFile = filepath.Join(homeDirPath, ".kdeps.pkl")
	// Write the heredoc content to the file
	err = afero.WriteFile(testFs, systemConfigurationFile, []byte(systemConfigurationContent), 0644)
	if err != nil {
		return err
	}

	systemConfigurationFile, err = cfg.FindConfiguration(testFs, env)
	if err != nil {
		return err
	}

	if err = enforcer.EnforcePklTemplateAmendsRules(testFs, systemConfigurationFile); err != nil {
		return err
	}

	syscfg, err := cfg.LoadConfiguration(testFs, systemConfigurationFile)
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
amends "package://schema.kdeps.com/core@0.0.44#/Workflow.pkl"

name = "myAIAgentAPI2"
description = "AI Agent X API"
action = "helloWorld100"
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

	filePath = filepath.Join(homeDirPath, "myAgentX2")

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

	// Convert totalResources from string to int
	totalResourcesInt, err := strconv.Atoi(arg1)
	if err != nil {
		return fmt.Errorf("failed to convert totalResources to int: %w", err)
	}

	// Iterate and create resources starting from totalResourcesInt down to 1
	for num := totalResourcesInt; num >= 1; num-- {
		// Prepare the dependencies for the current resource
		var requiresContent string
		if num > 1 {
			// Create a list of dependencies from "action1" to "action(num-1)"
			var dependencies []string
			for i := 1; i < num; i++ {
				dependencies = append(dependencies, fmt.Sprintf(`"helloWorld%d"`, i))
			}
			// Join the dependencies into a requires block
			requiresContent = fmt.Sprintf(`requires {
  %s
}`, strings.Join(dependencies, "\n  "))
		}

		// Define the content of the resource configuration file
		resourceConfigurationContent := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.44#/Resource.pkl"

id = "helloWorld%d"
name = "default action %d"
description = "default action"
category = "category"
%s
`, num, num, requiresContent)

		// Define the file path
		resourceConfigurationFile := filepath.Join(resourcesDir, fmt.Sprintf("resource%d.pkl", num))

		// Write the file content using afero
		err := afero.WriteFile(testFs, resourceConfigurationFile, []byte(resourceConfigurationContent), 0644)
		if err != nil {
			return err
		}

		fmt.Println("config 2: ", resourceConfigurationFile)
	}

	return nil
}
