package cfg

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"kdeps/pkg/environment"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/schema"
	"kdeps/pkg/texteditor"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

func FindConfiguration(fs afero.Fs, env *environment.Environment, logger *log.Logger) (string, error) {
	logger.Info("Finding configuration...")

	// Ensure PKL binary exists before proceeding
	if err := evaluator.EnsurePklBinaryExists(logger); err != nil {
		return "", err
	}

	// Use the initialized environment's Pwd directory
	configFilePwd := filepath.Join(env.Pwd, environment.SystemconfigFileName)
	if _, err := fs.Stat(configFilePwd); err == nil {
		logger.Info("Configuration file found in Pwd directory", "config-file", configFilePwd)
		return configFilePwd, nil
	}

	// Use the initialized environment's Home directory
	configFileHome := filepath.Join(env.Home, environment.SystemconfigFileName)
	if _, err := fs.Stat(configFileHome); err == nil {
		logger.Info("Configuration file found in Home directory", "config-file", configFileHome)
		return configFileHome, nil
	}

	logger.Warn("Configuration file not found", "config-file", environment.SystemconfigFileName)
	return "", nil
}

func GenerateConfiguration(fs afero.Fs, env *environment.Environment, logger *log.Logger) (string, error) {
	logger.Info("Generating configuration...")

	// Set configFile path in Home directory
	configFile := filepath.Join(env.Home, environment.SystemconfigFileName)
	skipPrompts := env.NonInteractive == "1"

	if _, err := fs.Stat(configFile); err != nil {
		var confirm bool
		if !skipPrompts {
			if err := huh.Run(
				huh.NewConfirm().
					Title("Configuration file not found. Do you want to generate one?").
					Description("The configuration will be validated using the `pkl` package.").
					Value(&confirm),
			); err != nil {
				return "", fmt.Errorf("could not create a configuration file: %w", err)
			}
			if !confirm {
				return "", errors.New("aborted by user")
			}
		}

		// Generate configuration
		url := fmt.Sprintf("package://schema.kdeps.com/core@%s#/Kdeps.pkl", schema.SchemaVersion)
		result, err := evaluator.EvalPkl(fs, url, logger)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate .pkl file: %w", err)
		}

		content := fmt.Sprintf("amends \"%s\"\n%s", url, result)
		if err = afero.WriteFile(fs, configFile, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write to %s: %w", configFile, err)
		}

		logger.Info("Configuration file generated", "config-file", configFile)
	}

	return configFile, nil
}

func EditConfiguration(fs afero.Fs, env *environment.Environment, logger *log.Logger) (string, error) {
	logger.Info("Editing configuration...")

	configFile := filepath.Join(env.Home, environment.SystemconfigFileName)
	skipPrompts := env.NonInteractive == "1"

	if _, err := fs.Stat(configFile); err == nil {
		if !skipPrompts {
			if err := texteditor.EditPkl(fs, configFile, logger); err != nil {
				return configFile, fmt.Errorf("failed to edit configuration file: %w", err)
			}
		}
	} else {
		logger.Warn("Configuration file does not exist", "config-file", configFile)
	}

	return configFile, nil
}

func ValidateConfiguration(fs afero.Fs, env *environment.Environment, logger *log.Logger) (string, error) {
	logger.Info("Validating configuration...")

	configFile := filepath.Join(env.Home, environment.SystemconfigFileName)

	if _, err := evaluator.EvalPkl(fs, configFile, logger); err != nil {
		return configFile, fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.Info("Configuration validated successfully", "config-file", configFile)
	return configFile, nil
}

func LoadConfiguration(fs afero.Fs, configFile string, logger *log.Logger) (*kdeps.Kdeps, error) {
	logger.Info("Loading configuration", "config-file", configFile)

	konfig, err := kdeps.LoadFromPath(context.Background(), configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file '%s': %w", configFile, err)
	}

	return konfig, nil
}
