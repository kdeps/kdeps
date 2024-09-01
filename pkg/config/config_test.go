package config

import (
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
	"github.com/spf13/afero"
)

var (
	fs             afero.Fs
	currentDirPath string
	homeDirPath    string
)

func TestFeatures(t *testing.T) {
	fs = afero.NewMemMapFs()

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a file "([^"]*)" exists in the current directory$`, aFileExistsInTheCurrentDirectory)
			ctx.Step(`^the configuration file is "([^"]*)"$`, theConfigurationFileIs)
			ctx.Step(`^the configuration is loaded$`, theConfigurationIsLoaded)
			ctx.Step(`^the current directory is "([^"]*)"$`, theCurrentDirectoryIs)
			ctx.Step(`^the home directory is "([^"]*)"$`, theCurrentDirectoryIs)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aFileExistsInTheCurrentDirectory(arg1 string) error {
	file := filepath.Join(currentDirPath, arg1)

	f, _ := fs.Create(file)
	f.WriteString("mock content")
	f.Close()

	if _, err := fs.Stat(file); err != nil {
		return err
	}
	return nil
}

func theConfigurationFileIs(arg1 string) error {
	return godog.ErrPending
}

func theConfigurationIsLoaded() error {
	return godog.ErrPending
}

func theCurrentDirectoryIs(arg1 string) error {
	currentDirPath = arg1

	if err := fs.MkdirAll(currentDirPath, 0755); err != nil {
		return err
	}
	return nil
}

func theHomeDirectoryIs(arg1 string) error {
	homeDirPath = arg1

	if err := fs.MkdirAll(homeDirPath, 0755); err != nil {
		return err
	}
	return nil
}
