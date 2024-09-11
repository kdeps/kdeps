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

var SystemconfigFileName = ".kdeps.pkl"

type Environment struct {
	Home           string `env:"HOME"`
	Pwd            string `env:"PWD"`
	NonInteractive string `env:"NON_INTERACTIVE,default=0"`
	Extras         env.EnvSet
}

func FindConfiguration(fs afero.Fs, environment *Environment) (configFile string, err error) {
	logging.Info("Finding configuration...")

	// Ensure PKL binary exists before proceeding
	if err := evaluator.EnsurePklBinaryExists(); err != nil {
		return "", err
	}

	// Check if configuration exists in the current directory (override)
	if len(environment.Pwd) > 0 {
		configFile = filepath.Join(environment.Pwd, SystemconfigFileName)
		if _, err = fs.Stat(configFile); err == nil {
			logging.Info("Configuration file found in current directory (override)", "config-file", configFile)
			return configFile, nil
		}
	}

	// Check if configuration exists in the home directory (override)
	if len(environment.Home) > 0 {
		configFile = filepath.Join(environment.Home, SystemconfigFileName)
		if _, err = fs.Stat(configFile); err == nil {
			logging.Info("Configuration file found in home directory (override)", "config-file", configFile)
			return configFile, nil
		}
	}

	// Load the environment and extra settings if overrides don't exist
	es, err := env.UnmarshalFromEnviron(environment)
	if err != nil {
		return "", err
	}
	environment.Extras = es

	// Check configuration file in the environment's current directory
	configFilePwd := filepath.Join(environment.Pwd, SystemconfigFileName)
	if _, err = fs.Stat(configFilePwd); err == nil {
		logging.Info("Configuration file found in environment's current directory", "config-file", configFilePwd)
		return configFilePwd, nil
	}

	// Check configuration file in the environment's home directory
	configFileHome := filepath.Join(environment.Home, SystemconfigFileName)
	if _, err = fs.Stat(configFileHome); err == nil {
		logging.Info("Configuration file found in environment's home directory", "config-file", configFileHome)
		return configFileHome, nil
	}

	// No configuration file found
	logging.Warn("Configuration file not found in any location", "config-file", SystemconfigFileName)
	return "", nil
}

func GenerateConfiguration(fs afero.Fs, environment *Environment) (configFile string, err error) {
	logging.Info("Generating configuration...")

	if len(environment.Home) > 0 {
		configFile = filepath.Join(environment.Home, SystemconfigFileName)
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return "", err
		}
		environment.Extras = es

		configFile = filepath.Join(environment.Home, SystemconfigFileName)
	}

	skipPrompts := environment.NonInteractive == "1"

	if _, err := fs.Stat(configFile); err != nil {
		var confirm bool
		if !skipPrompts {
			if err := huh.Run(
				huh.NewConfirm().
					Title("Configuration file not found. Do you want to generate one?").
					Description("The configuration will be validated. This will require the `pkl` package to be installed. Please refer to https://pkl-lang.org for more details.").
					Value(&confirm),
			); err != nil {
				return "", fmt.Errorf("could not create a configuration file: %w", err)
			}

			if !confirm {
				return "", errors.New("aborted by user")
			}
		}

		// Read the schema version from the SCHEMA_VERSION file
		schemaVersionBytes, err := ioutil.ReadFile("../../SCHEMA_VERSION")
		if err != nil {
			return "", fmt.Errorf("failed to read SCHEMA_VERSION: %w", err)
		}
		schemaVersion := strings.TrimSpace(string(schemaVersionBytes))

		// Create the URL with the schema version
		url := fmt.Sprintf("package://schema.kdeps.com/core@%s#/Kdeps.pkl", schemaVersion)

		// Evaluate the .pkl file and write the result to configFile
		result, err := evaluator.EvalPkl(fs, url)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate .pkl file: %w", err)
		}

		content := fmt.Sprintf("amends \"%s\"\n%s", url, result)
		if err = afero.WriteFile(fs, configFile, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write to %s: %w", configFile, err)
		}

		logging.Info("Configuration file generated", "config-file", configFile)
	}

	return configFile, nil
}

func EditConfiguration(fs afero.Fs, environment *Environment) (configFile string, err error) {
	logging.Info("Editing configuration...")

	if len(environment.Home) > 0 {
		configFile = filepath.Join(environment.Home, SystemconfigFileName)
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return "", err
		}
		environment.Extras = es
		configFile = filepath.Join(environment.Home, SystemconfigFileName)
	}

	skipPrompts := environment.NonInteractive == "1"

	if _, err := fs.Stat(configFile); err == nil {
		if !skipPrompts {
			if err := texteditor.EditPkl(fs, configFile); err != nil {
				return configFile, fmt.Errorf("failed to edit configuration file: %w", err)
			}
		}
	} else {
		logging.Warn("Configuration file does not exist", "config-file", configFile)
	}

	return configFile, nil
}

func ValidateConfiguration(fs afero.Fs, environment *Environment) (configFile string, err error) {
	logging.Info("Validating configuration...")

	if len(environment.Home) > 0 {
		configFile = filepath.Join(environment.Home, SystemconfigFileName)
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return "", err
		}
		environment.Extras = es
		configFile = filepath.Join(environment.Home, SystemconfigFileName)
	}

	if _, err := evaluator.EvalPkl(fs, configFile); err != nil {
		return configFile, fmt.Errorf("configuration validation failed: %w", err)
	}

	logging.Info("Configuration validated successfully", "config-file", configFile)
	return configFile, nil
}

func LoadConfiguration(fs afero.Fs, configFile string) (konfig *kdeps.Kdeps, err error) {
	logging.Info("Loading configuration file", "config-file", configFile)

	konfig, err = kdeps.LoadFromPath(context.Background(), configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config-file '%s': %w", configFile, err)
	}

	return konfig, nil
}
