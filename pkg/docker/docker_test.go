package docker

import (
	"context"
	"errors"
	"fmt"
	"kdeps/pkg/cfg"
	"testing"

	"github.com/cucumber/godog"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/kdeps/schema/gen/kdeps/gpu"
	"github.com/spf13/afero"
)

var (
	testFs              = afero.NewOsFs()
	testingT            *testing.T
	homeDirPath         string
	systemConfiguration *kdeps.Kdeps
	dockerName          string
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a system configuration is defined$`, aSystemConfigurationIsDefined)
			ctx.Step(`^it should copy the necessary files to make it ready to be used$`, copyTheNecessaryFilesToMakeItReadyToBeUsed)
			ctx.Step(`^it should initialize the "([^"]*)" docker subsystem$`, itInitializeTheDockerSubsystem)
			ctx.Step(`^the docker gpu "([^"]*)" is defined$`, theDockerGPUIsDefined)
			ctx.Step(`^the docker subsystem "([^"]*)" is invoked$`, theDockerSubsystemIsInvoked)
			ctx.Step(`^existing docker container "([^"]*)" system is deleted$`, existingDockerContainerSystemIsDeleted)
			ctx.Step(`^copy the necessary files to make it ready to be used$`, copyTheNecessaryFilesToMakeItReadyToBeUsed)
			ctx.Step(`^custom <packages> has been defined$`, customPackagesHasBeenDefined)
			ctx.Step(`^llm <models> has been installed$`, llmModelsHasBeenInstalled)
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

func aSystemConfigurationIsDefined() error {
	homeDirPath, err := afero.TempDir(testFs, "", "")
	if err != nil {
		return err
	}

	env := &cfg.Environment{
		Home:           homeDirPath,
		Pwd:            "",
		NonInteractive: "1",
	}

	if err := cfg.GenerateConfiguration(testFs, env); err != nil {
		return err
	}

	cfg, err := cfg.LoadConfiguration(testFs)
	if err != nil {
		return err
	}

	systemConfiguration = cfg

	return nil
}

func copyTheNecessaryFilesToMakeItReadyToBeUsed() error {
	return godog.ErrPending
}

func itInitializeTheDockerSubsystem(arg1 string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	// Fetch running containers
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return err
	}

	// Check if container with specified name exists
	containerName := arg1
	var found bool
	for _, container := range containers {
		if container.Names[0] == "/"+containerName {
			found = true
			break
		}
	}
	if !found {
		return errors.New("Docker container not found!")
	}

	// Check if image exists
	_, _, err = cli.ImageInspectWithRaw(context.Background(), "ollama/ollama")
	if err != nil {
		return err
	}

	return nil
}

func theDockerGPUIsDefined(arg1 string) error {
	configGPU := systemConfiguration.DockerGPU
	expectedGPU := gpu.GPU(arg1)

	if configGPU != expectedGPU {
		return errors.New(fmt.Sprintf("Docker GPU '%s' is not defined: %s", expectedGPU, configGPU))
	}

	return nil
}

func theDockerSubsystemIsInvoked(arg1 string) error {
	name, err := LoadDockerSystem(systemConfiguration, arg1)

	if err != nil {
		return err
	}

	dockerName = name

	return nil
}

func deleteContainerIfExists(containerName string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	// List all containers (including stopped ones)
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return err
	}

	// Check if container with the specified name exists and delete it
	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+containerName {
				fmt.Printf("Deleting container: %s\n", c.ID)
				err := cli.ContainerRemove(context.Background(), c.ID, container.RemoveOptions{Force: true})
				return err
			}
		}
	}

	fmt.Printf("Container %s not found\n", containerName)
	return nil
}

func existingDockerContainerSystemIsDeleted(arg1 string) error {
	if err := deleteContainerIfExists("kdeps-cpu-test"); err != nil {
		return err
	}

	return nil
}

func customPackagesHasBeenDefined(arg1 *godog.Table) error {
	var packages []string

	for _, r := range arg1.Rows {
		for _, c := range r.Cells {
			packages = append(packages, c.Value)
		}
	}

	return godog.ErrPending
}

func llmModelsHasBeenInstalled(arg1 *godog.Table) error {
	var models []string

	for _, r := range arg1.Rows {
		for _, c := range r.Cells {
			models = append(models, c.Value)
		}
	}

	return godog.ErrPending
}
