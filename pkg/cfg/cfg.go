package cfg

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"kdeps/pkg/evaluator"
	"kdeps/pkg/logging"
	"kdeps/pkg/texteditor"

	env "github.com/Netflix/go-env"
	"github.com/charmbracelet/huh"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

var (
	SystemConfigFileName = ".kdeps.pkl"
	ConfigFile           string
	HomeConfigFile       string
	CwdConfigFile        string
)

type Environment struct {
	Home           string `env:"HOME"`
	Pwd            string `env:"PWD"`
	NonInteractive string `env:"NON_INTERACTIVE,default=0"`
	Extras         env.EnvSet
}

func FindConfiguration(fs afero.Fs, environment *Environment) error {
	logging.Info("Finding configuration...")

	if err := evaluator.EnsurePklBinaryExists(); err != nil {
		return err
	}

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)
		ConfigFile = HomeConfigFile
		logging.Info("Configuration file set to home directory", "config-file", ConfigFile)
		return nil
	}

	if len(environment.Pwd) > 0 {
		CwdConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)
		ConfigFile = CwdConfigFile
		logging.Info("Configuration file set to current directory", "config-file", ConfigFile)
		return nil
	}

	es, err := env.UnmarshalFromEnviron(&environment)
	if err != nil {
		return err
	}
	environment.Extras = es

	CwdConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)
	HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

	if _, err = fs.Stat(CwdConfigFile); err == nil {
		ConfigFile = CwdConfigFile
		logging.Info("Configuration file found in current directory", "config-file", ConfigFile)
	} else if _, err = fs.Stat(HomeConfigFile); err == nil {
		ConfigFile = HomeConfigFile
		logging.Info("Configuration file found in home directory", "config-file", ConfigFile)
	} else {
		logging.Warn("Configuration file not found", "config-file", ConfigFile)
	}

	return nil
}

func GenerateConfiguration(fs afero.Fs, environment *Environment) error {
	logging.Info("Generating configuration...")

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)
		ConfigFile = HomeConfigFile
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return err
		}
		environment.Extras = es
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)
		ConfigFile = HomeConfigFile
	}

	skipPrompts := environment.NonInteractive == "1"

	if _, err := fs.Stat(ConfigFile); err != nil {
		var confirm bool
		if !skipPrompts {
			if err := huh.Run(
				huh.NewConfirm().
					Title("Configuration file not found. Do you want to generate one?").
					Description("The configuration will be validated. This will require the `pkl` package to be installed. Please refer to https://pkl-lang.org for more details.").
					Value(&confirm),
			); err != nil {
				return fmt.Errorf("could not create a configuration file: %w", err)
			}

			if !confirm {
				return errors.New("aborted by user")
			}
		}

		// Read the schema version from the SCHEMA_VERSION file
		schemaVersionBytes, err := ioutil.ReadFile("../../SCHEMA_VERSION")
		if err != nil {
			return fmt.Errorf("failed to read SCHEMA_VERSION: %w", err)
		}
		schemaVersion := strings.TrimSpace(string(schemaVersionBytes))

		// Create the URL with the schema version
		url := fmt.Sprintf("package://schema.kdeps.com/core@%s#/Kdeps.pkl", schemaVersion)

		// Evaluate the .pkl file and write the result to ConfigFile
		result, err := evaluator.EvalPkl(fs, url)
		if err != nil {
			return fmt.Errorf("failed to evaluate .pkl file: %w", err)
		}

		content := fmt.Sprintf("amends \"%s\"\n%s", url, result)
		if err = afero.WriteFile(fs, ConfigFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write to %s: %w", ConfigFile, err)
		}

		logging.Info("Configuration file generated", "config-file", ConfigFile)
	}

	return nil
}

func EditConfiguration(fs afero.Fs, environment *Environment) error {
	logging.Info("Editing configuration...")

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)
		ConfigFile = HomeConfigFile
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return err
		}
		environment.Extras = es
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)
		ConfigFile = HomeConfigFile
	}

	skipPrompts := environment.NonInteractive == "1"

	if _, err := fs.Stat(ConfigFile); err == nil {
		if !skipPrompts {
			if err := texteditor.EditPkl(fs, ConfigFile); err != nil {
				return fmt.Errorf("failed to edit configuration file: %w", err)
			}
		}
	} else {
		logging.Warn("Configuration file does not exist", "config-file", ConfigFile)
	}

	return nil
}

func ValidateConfiguration(fs afero.Fs, environment *Environment) error {
	logging.Info("Validating configuration...")

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)
		ConfigFile = HomeConfigFile
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return err
		}
		environment.Extras = es
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)
		ConfigFile = HomeConfigFile
	}

	if _, err := evaluator.EvalPkl(fs, ConfigFile); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	logging.Info("Configuration validated successfully", "config-file", ConfigFile)
	return nil
}

func LoadConfiguration(fs afero.Fs) (konfig *kdeps.Kdeps, err error) {
	logging.Info("Loading configuration file", "config-file", ConfigFile)

	konfig, err = kdeps.LoadFromPath(context.Background(), ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config-file '%s': %w", ConfigFile, err)
	}

	return konfig, nil
}
