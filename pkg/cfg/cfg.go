package cfg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/apple/pkl-go/pkl"
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

func FindConfiguration(_ context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("finding configuration...")

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

func GenerateConfiguration(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("generating configuration...")

	// Set configFile path in Home directory
	configFile := filepath.Join(env.Home, environment.SystemConfigFileName)

	// Always create the configuration file if it doesn't exist
	if _, err := fs.Stat(configFile); err != nil {
		// Generate configuration without asking the user for confirmation
		url := fmt.Sprintf("package://schema.kdeps.com/core@%s#/Kdeps.pkl", schema.Version(ctx))
		headerSection := fmt.Sprintf("amends \"%s\"\n", url)

		// Use the singleton evaluator directly to avoid writing back to package URL
		eval, err := evaluator.GetEvaluator()
		if err != nil {
			return "", fmt.Errorf("failed to get evaluator: %w", err)
		}

		// Create a ModuleSource using UriSource for the package URL
		moduleSource := pkl.UriSource(url)

		// Evaluate the Pkl file
		result, err := eval.EvaluateOutputText(ctx, moduleSource)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate .pkl file: %w", err)
		}

		// Format the result by prepending the headerSection
		formattedResult := fmt.Sprintf("%s\n%s", headerSection, result)

		if err = afero.WriteFile(fs, configFile, []byte(formattedResult), 0o644); err != nil {
			return "", fmt.Errorf("failed to write to %s: %w", configFile, err)
		}

		logger.Debug("configuration file generated", "config-file", configFile)
	}

	return configFile, nil
}

func EditConfiguration(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("editing configuration...")

	configFile := filepath.Join(env.Home, environment.SystemConfigFileName)
	skipPrompts := env.NonInteractive == "1"

	if _, err := fs.Stat(configFile); err == nil {
		var confirm bool
		if !skipPrompts {
			if err := huh.Run(
				huh.NewConfirm().
					Title("Do you want to edit the configuration file now?").
					Description("This will open the file in your default text editor.").
					Value(&confirm),
			); err != nil {
				return configFile, fmt.Errorf("could not prompt for editing configuration file: %w", err)
			}
		}

		if confirm || skipPrompts {
			// In non-interactive mode (skipPrompts) we skip the prompt; in that case, we follow previous behavior and DO NOT edit automatically.
			// Only edit automatically if user explicitly confirmed.
			if confirm {
				if err := texteditor.EditPkl(fs, ctx, configFile, logger); err != nil {
					return configFile, fmt.Errorf("failed to edit configuration file: %w", err)
				}
			}
		}
	} else {
		logger.Warn("configuration file does not exist", "config-file", configFile)
	}

	return configFile, nil
}

func ValidateConfiguration(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("validating configuration...")

	configFile := filepath.Join(env.Home, environment.SystemConfigFileName)

	if _, err := evaluator.EvalPkl(fs, ctx, configFile, "", nil, logger); err != nil {
		return configFile, fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.Debug("configuration validated successfully", "config-file", configFile)
	return configFile, nil
}

func LoadConfiguration(ctx context.Context, _ afero.Fs, configFile string, logger *logging.Logger) (*kdeps.Kdeps, error) {
	logger.Debug("loading configuration", "config-file", configFile)

	konfig, err := kdeps.LoadFromPath(ctx, configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file '%s': %w", configFile, err)
	}

	return konfig, nil
}

func GetKdepsPath(_ context.Context, kdepsCfg kdeps.Kdeps) (string, error) {
	kdepsDir := kdepsCfg.KdepsDir
	p := kdepsCfg.KdepsPath

	// Handle nil pointers with defaults
	if kdepsDir == nil {
		defaultDir := ".kdeps"
		kdepsDir = &defaultDir
	}
	if p == nil {
		defaultPath := path.User
		p = &defaultPath
	}

	switch *p {
	case path.User:
		// Use the user's home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, *kdepsDir), nil

	case path.Project:
		// Use the current working directory (project dir)
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, *kdepsDir), nil

	case path.Xdg:
		// Use the XDG config home directory
		return filepath.Join(xdg.ConfigHome, *kdepsDir), nil

	default:
		return "", fmt.Errorf("unknown path type: %s", *p)
	}
}
