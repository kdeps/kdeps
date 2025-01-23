package cfg

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

var (
	testFs         = afero.NewOsFs()
	currentDirPath string
	homeDirPath    string
	fileThatExist  string
	ctx            = context.Background()
	logger         *logging.Logger
	testingT       *testing.T
)

func TestFeatures(t *testing.T) {
	t.Parallel()
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a file "([^"]*)" exists in the current directory$`, aFileExistsInTheCurrentDirectory)
			ctx.Step(`^a file "([^"]*)" exists in the home directory$`, aFileExistsInTheHomeDirectory)
			ctx.Step(`^the configuration file is "([^"]*)"$`, theConfigurationFileIs)
			ctx.Step(`^the configuration is loaded in the current directory$`, theConfigurationIsLoadedInTheCurrentDirectory)
			ctx.Step(`^the configuration is loaded in the home directory$`, theConfigurationIsLoadedInTheHomeDirectory)
			ctx.Step(`^the current directory is "([^"]*)"$`, theCurrentDirectoryIs)
			ctx.Step(`^the home directory is "([^"]*)"$`, theHomeDirectoryIs)
			ctx.Step(`^a file "([^"]*)" does not exists in the home or current directory$`, aFileDoesNotExistsInTheHomeOrCurrentDirectory)
			ctx.Step(`^the configuration fails to load any configuration$`, theConfigurationFailsToLoadAnyConfiguration)
			ctx.Step(`^the configuration file will be generated to "([^"]*)"$`, theConfigurationFileWillBeGeneratedTo)
			ctx.Step(`^the configuration will be edited$`, theConfigurationWillBeEdited)
			ctx.Step(`^the configuration will be validated$`, theConfigurationWillBeValidated)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/cfg"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aFileExistsInTheCurrentDirectory(arg1 string) error {
	logger = logging.GetLogger()

	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Kdeps.pkl"

runMode = "docker"
dockerGPU = "cpu"
`, schema.SchemaVersion(ctx))
	file := filepath.Join(currentDirPath, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	fileThatExist = file

	return nil
}

func aFileExistsInTheHomeDirectory(arg1 string) error {
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Kdeps.pkl"

runMode = "docker"
dockerGPU = "cpu"
`, schema.SchemaVersion(ctx))
	file := filepath.Join(homeDirPath, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	fileThatExist = file

	return nil
}

func theConfigurationFileIs(arg1 string) error {
	if _, err := testFs.Stat(fileThatExist); err != nil {
		return err
	}

	return nil
}

func theConfigurationIsLoadedInTheCurrentDirectory() error {
	env := &environment.Environment{
		Home: "",
		Pwd:  currentDirPath,
	}

	environ, err := environment.NewEnvironment(testFs, ctx, env)
	if err != nil {
		return err
	}

	cfgFile, err := FindConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return err
	}

	if _, err := LoadConfiguration(testFs, ctx, cfgFile, logger); err != nil {
		return err
	}

	return nil
}

func theConfigurationIsLoadedInTheHomeDirectory() error {
	env := &environment.Environment{
		Home: homeDirPath,
		Pwd:  "",
	}

	environ, err := environment.NewEnvironment(testFs, ctx, env)
	if err != nil {
		return err
	}

	cfgFile, err := FindConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return err
	}

	if _, err := LoadConfiguration(testFs, ctx, cfgFile, logger); err != nil {
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

func aFileDoesNotExistsInTheHomeOrCurrentDirectory(arg1 string) error {
	fileThatExist = ""

	return nil
}

func theConfigurationFailsToLoadAnyConfiguration() error {
	env := &environment.Environment{
		Home: homeDirPath,
		Pwd:  currentDirPath,
	}

	environ, err := environment.NewEnvironment(testFs, ctx, env)
	if err != nil {
		return err
	}

	cfgFile, err := FindConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return fmt.Errorf("An error occurred while finding configuration: %w", err)
	}
	if cfgFile != "" {
		return errors.New("expected not finding configuration file, but found")
	}

	return nil
}

func theConfigurationFileWillBeGeneratedTo(arg1 string) error {
	env := &environment.Environment{
		Home:           homeDirPath,
		Pwd:            "",
		NonInteractive: "1",
	}

	environ, err := environment.NewEnvironment(testFs, ctx, env)
	if err != nil {
		return err
	}

	cfgFile, err := GenerateConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return err
	}

	if _, err := LoadConfiguration(testFs, ctx, cfgFile, logger); err != nil {
		return err
	}

	return nil
}

func theConfigurationWillBeEdited() error {
	env := &environment.Environment{
		Home:           homeDirPath,
		Pwd:            "",
		NonInteractive: "1",
	}

	environ, err := environment.NewEnvironment(testFs, ctx, env)
	if err != nil {
		return err
	}

	if _, err := EditConfiguration(testFs, ctx, environ, logger); err != nil {
		return err
	}

	return nil
}

func theConfigurationWillBeValidated() error {
	env := &environment.Environment{
		Home: homeDirPath,
		Pwd:  "",
	}

	environ, err := environment.NewEnvironment(testFs, ctx, env)
	if err != nil {
		return err
	}

	if _, err := ValidateConfiguration(testFs, ctx, environ, logger); err != nil {
		return err
	}

	return nil
}
