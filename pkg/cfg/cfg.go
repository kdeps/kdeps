package cfg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/huh"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/texteditor"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/kdeps/schema/gen/kdeps/path"
	"github.com/spf13/afero"
)

func FindConfiguration(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("finding configuration...")

	// Ensure PKL binary exists before proceeding
	if err := evaluator.EnsurePklBinaryExists(ctx, logger); err != nil {
		return "", err
	}

	// Use the initialized environment's Pwd directory
	configFilePwd := filepath.Join(env.Pwd, environment.SystemConfigFileName)
	if _, err := fs.Stat(configFilePwd); err == nil {
		logger.Debug("configuration file found in Pwd directory", "config-file", configFilePwd)
		return configFilePwd, nil
	}

	// Use the initialized environment's Home directory
	configFileHome := filepath.Join(env.Home, environment.SystemConfigFileName)
	if _, err := fs.Stat(configFileHome); err == nil {
		logger.Debug("configuration file found in Home directory", "config-file", configFileHome)
		return configFileHome, nil
	}

	logger.Warn("configuration file not found", "config-file", environment.SystemConfigFileName)
	return "", nil
}

func GenerateConfiguration(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("generating configuration...")

	// Set configFile path in Home directory
	configFile := filepath.Join(env.Home, environment.SystemConfigFileName)
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
		url := fmt.Sprintf("package://schema.kdeps.com/core@%s#/Kdeps.pkl", schema.SchemaVersion(ctx))
		headerSection := fmt.Sprintf("amends \"%s\"\n", url)

		content, err := evaluator.EvalPkl(fs, ctx, url, headerSection, logger)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate .pkl file: %w", err)
		}

		if err = afero.WriteFile(fs, configFile, []byte(content), 0o644); err != nil {
			return "", fmt.Errorf("failed to write to %s: %w", configFile, err)
		}

		logger.Debug("configuration file generated", "config-file", configFile)
	}

	return configFile, nil
}

func EditConfiguration(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("editing configuration...")

	configFile := filepath.Join(env.Home, environment.SystemConfigFileName)
	skipPrompts := env.NonInteractive == "1"

	if _, err := fs.Stat(configFile); err == nil {
		if !skipPrompts {
			if err := texteditor.EditPkl(fs, ctx, configFile, logger); err != nil {
				return configFile, fmt.Errorf("failed to edit configuration file: %w", err)
			}
		}
	} else {
		logger.Warn("configuration file does not exist", "config-file", configFile)
	}

	return configFile, nil
}

func ValidateConfiguration(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("validating configuration...")

	configFile := filepath.Join(env.Home, environment.SystemConfigFileName)

	if _, err := evaluator.EvalPkl(fs, ctx, configFile, "", logger); err != nil {
		return configFile, fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.Debug("configuration validated successfully", "config-file", configFile)
	return configFile, nil
}

func LoadConfiguration(fs afero.Fs, ctx context.Context, configFile string, logger *logging.Logger) (*kdeps.Kdeps, error) {
	logger.Debug("loading configuration", "config-file", configFile)

	konfig, err := kdeps.LoadFromPath(ctx, configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file '%s': %w", configFile, err)
	}

	return konfig, nil
}

func GetKdepsPath(ctx context.Context, kdepsCfg kdeps.Kdeps) (string, error) {
	kdepsDir := kdepsCfg.KdepsDir
	p := kdepsCfg.KdepsPath

	switch p {
	case path.User:
		// Use the user's home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, kdepsDir), nil

	case path.Project:
		// Use the current working directory (project dir)
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, kdepsDir), nil

	case path.Xdg:
		// Use the XDG config home directory
		return filepath.Join(xdg.ConfigHome, kdepsDir), nil

	default:
		return "", fmt.Errorf("unknown path type: %s", p)
	}
}
