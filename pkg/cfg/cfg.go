package cfg

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/assets"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/texteditor"
	schemaAssets "github.com/kdeps/schema/assets"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/kdeps/schema/gen/kdeps/path"
	"github.com/spf13/afero"
)

// simpleConfirm provides a simple Yes/No prompt without TUI complications
func simpleConfirm(title, description string) (bool, error) {
	fmt.Printf("\n%s\n", title)
	if description != "" {
		fmt.Printf("%s\n", description)
	}
	fmt.Print("Do you want to continue? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

func FindConfiguration(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("finding configuration...")

	// No need to ensure PKL CLI; we use the SDK now

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

	// Always create the configuration file if it doesn't exist
	if _, err := fs.Stat(configFile); err != nil {
		// Generate configuration without asking the user for confirmation
		url := fmt.Sprintf("package://schema.kdeps.com/core@%s#/Kdeps.pkl", schema.SchemaVersion(ctx))
		headerSection := fmt.Sprintf("amends \"%s\"\n", url)

		content, err := evaluator.EvalPkl(fs, ctx, url, headerSection, nil, logger)
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
		var confirm bool
		if !skipPrompts {
			var err error
			confirm, err = simpleConfirm(
				"Do you want to edit the configuration file now?",
				"This will open the file in your default text editor.",
			)
			if err != nil {
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

func ValidateConfiguration(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
	logger.Debug("validating configuration...")

	configFile := filepath.Join(env.Home, environment.SystemConfigFileName)

	if _, err := evaluator.EvalPkl(fs, ctx, configFile, "", nil, logger); err != nil {
		return configFile, fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.Debug("configuration validated successfully", "config-file", configFile)
	return configFile, nil
}

func LoadConfiguration(fs afero.Fs, ctx context.Context, configFile string, logger *logging.Logger) (*kdeps.Kdeps, error) {
	logger.Debug("loading configuration", "config-file", configFile)

	// Check if we should use embedded assets
	if assets.ShouldUseEmbeddedAssets() {
		return loadConfigurationFromEmbeddedAssets(ctx, configFile, logger)
	}

	return loadConfigurationFromFile(ctx, configFile, logger)
}

// loadConfigurationFromEmbeddedAssets loads configuration using embedded PKL assets
func loadConfigurationFromEmbeddedAssets(ctx context.Context, configFile string, logger *logging.Logger) (*kdeps.Kdeps, error) {
	logger.Debug("loading configuration from embedded assets", "config-file", configFile)

	// Use GetPKLFileWithFullConversion to get the embedded Kdeps.pkl template
	_, err := schemaAssets.GetPKLFileWithFullConversion("Kdeps.pkl")
	if err != nil {
		logger.Error("error reading embedded kdeps template", "error", err)
		return nil, fmt.Errorf("error reading embedded kdeps template: %w", err)
	}

	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		logger.Error("error creating pkl evaluator", "config-file", configFile, "error", err)
		return nil, fmt.Errorf("error creating pkl evaluator for config file '%s': %w", configFile, err)
	}
	defer evaluator.Close()

	// Use the user's config file but with embedded asset support
	source := pkl.FileSource(configFile)
	var conf *kdeps.Kdeps
	err = evaluator.EvaluateModule(ctx, source, &conf)
	if err != nil {
		logger.Error("error reading config file", "config-file", configFile, "error", err)
		return nil, fmt.Errorf("error reading config file '%s': %w", configFile, err)
	}

	logger.Debug("successfully read and parsed config file from embedded assets", "config-file", configFile)
	return conf, nil
}

// loadConfigurationFromFile loads configuration using direct file evaluation (original method)
func loadConfigurationFromFile(ctx context.Context, configFile string, logger *logging.Logger) (*kdeps.Kdeps, error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		logger.Error("error creating pkl evaluator", "config-file", configFile, "error", err)
		return nil, fmt.Errorf("error creating pkl evaluator for config file '%s': %w", configFile, err)
	}
	defer evaluator.Close()

	source := pkl.FileSource(configFile)
	var conf *kdeps.Kdeps
	err = evaluator.EvaluateModule(ctx, source, &conf)
	if err != nil {
		logger.Error("error reading config file", "config-file", configFile, "error", err)
		return nil, fmt.Errorf("error reading config file '%s': %w", configFile, err)
	}

	logger.Debug("successfully read and parsed config file", "config-file", configFile)
	return conf, nil
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
