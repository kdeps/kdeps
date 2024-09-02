package cfg

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/spf13/afero"
)

var (
	testFs         = afero.NewOsFs()
	currentDirPath string
	homeDirPath    string
	testConfigFile string
	fileThatExist  string
	testingT       *testing.T
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a file "([^"]*)" exists in the current directory$`, aFileExistsInTheCurrentDirectory)
			ctx.Step(`^the configuration file is "([^"]*)"$`, theConfigurationFileIs)
			ctx.Step(`^the configuration is loaded$`, theConfigurationIsLoaded)
			ctx.Step(`^the current directory is "([^"]*)"$`, theCurrentDirectoryIs)
			ctx.Step(`^the home directory is "([^"]*)"$`, theHomeDirectoryIs)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aFileExistsInTheCurrentDirectory(arg1 string) error {
	// dir, _ := afero.TempDir(testFs, currentDirPath, "")

	doc := `
amends "package://schema.kdeps.com/core@1.0.0#/Kdeps.pkl"

kdeps = "$HOME/.kdeps"
`
	// f, _ := afero.TempFile(testFs, currentDirPath, arg1)
	file := filepath.Join(currentDirPath, arg1)

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	if _, err := testFs.Stat(file); err != nil {
		return err
	}

	fileThatExist = file

	return nil
}

func theConfigurationFileIs(arg1 string) error {
	if !strings.EqualFold(fileThatExist, arg1) {
		return errors.New(fmt.Sprintf("Configuration file does not match: %s == %s", fileThatExist, arg1))
	}

	return nil
}

func theConfigurationIsLoaded() error {
	env := &Environment{
		Home: homeDirPath,
		Pwd:  fileThatExist,
	}

	if err := FindConfiguration(testFs, env); err != nil {
		return err
	}

	if err := LoadConfiguration(testFs); err != nil {
		return err
	}

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

func theHomeDirectoryIs(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", "")

	if err != nil {
		return err
	}

	homeDirPath = tempDir

	return nil
}
