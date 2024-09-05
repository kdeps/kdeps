package docker

import (
	"testing"

	"github.com/cucumber/godog"
	"github.com/spf13/afero"
)

var (
	testFs   = afero.NewOsFs()
	testingT *testing.T
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a system configuration is defined$`, aSystemConfigurationIsDefined)
			ctx.Step(`^copy the necessary files to make it ready to be used$`, copyTheNecessaryFilesToMakeItReadyToBeUsed)
			ctx.Step(`^it initialize the "([^"]*)" docker subsystem$`, itInitializeTheDockerSubsystem)
			ctx.Step(`^the docker image "([^"]*)" is defined$`, theDockerImageIsDefined)
			ctx.Step(`^the docker subsystem is invoked$`, theDockerSubsystemIsInvoked)
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
	return godog.ErrPending
}

func copyTheNecessaryFilesToMakeItReadyToBeUsed() error {
	return godog.ErrPending
}

func itInitializeTheDockerSubsystem(arg1 string) error {
	return godog.ErrPending
}

func theDockerImageIsDefined(arg1 string) error {
	return godog.ErrPending
}

func theDockerSubsystemIsInvoked() error {
	return godog.ErrPending
}
