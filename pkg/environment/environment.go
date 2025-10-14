package environment

import (
	"os"
	"path/filepath"

	env "github.com/Netflix/go-env"
	"github.com/spf13/afero"
)

const SystemConfigFileName = ".kdeps.pkl"

// Environment holds environment configurations loaded from the OS or defaults.
type Environment struct {
	Root           string `env:"ROOT_DIR,default=/"`
	Home           string `env:"HOME"`
	Pwd            string `env:"PWD"`
	KdepsConfig    string `env:"KDEPS_CONFIG,default=$HOME/.kdeps.pkl"`
	DockerMode     string `env:"DOCKER_MODE,default=0"`
	NonInteractive string `env:"NON_INTERACTIVE,default=0"`
	TimeoutSec     int    `env:"TIMEOUT,default=60"`
	Extras         env.EnvSet
}

// checkConfig checks if the .kdeps.pkl file exists in the given directory.
func checkConfig(fs afero.Fs, baseDir string) (string, error) {
	configFile := filepath.Join(baseDir, SystemConfigFileName)
	exists, err := afero.Exists(fs, configFile)
	if err == nil && exists {
		return configFile, nil
	}
	return "", err
}

// findKdepsConfig searches for the .kdeps.pkl file in both the Pwd and Home directories.
func findKdepsConfig(fs afero.Fs, pwd, home string) string {
	// Check for kdeps config in Pwd directory
	if configFile, _ := checkConfig(fs, pwd); configFile != "" {
		return configFile
	}
	// Check for kdeps config in Home directory
	if configFile, _ := checkConfig(fs, home); configFile != "" {
		return configFile
	}
	return ""
}

// isDockerEnvironment checks for the presence of Docker-related indicators.
func isDockerEnvironment(fs afero.Fs, root string) bool {
	dockerEnvFlag := filepath.Join(root, ".dockerenv")
	if exists, _ := afero.Exists(fs, dockerEnvFlag); exists {
		// Ensure all required Docker environment variables are set
		return allDockerEnvVarsSet()
	}
	return false
}

// allDockerEnvVarsSet checks if required Docker environment variables are set.
func allDockerEnvVarsSet() bool {
	requiredVars := []string{"SCHEMA_VERSION", "OLLAMA_HOST", "KDEPS_HOST"}
	for _, v := range requiredVars {
		if value, exists := os.LookupEnv(v); !exists || value == "" {
			return false
		}
	}
	return true
}

// NewEnvironment initializes and returns a new Environment based on provided or default settings.
func NewEnvironment(fs afero.Fs, environ *Environment) (*Environment, error) {
	if environ != nil {
		// If an environment is provided, prioritize overriding configurations
		// Use OR condition: check env var first, then auto-detect
		dockerMode := environ.DockerMode
		if dockerMode == "" {
			dockerMode = "0" // Default to not in Docker
		}
		if dockerMode != "1" && isDockerEnvironment(fs, environ.Root) {
			dockerMode = "1"
		}

		// Only search for config file if NOT in Docker mode
		kdepsConfigFile := ""
		if dockerMode != "1" {
			kdepsConfigFile = findKdepsConfig(fs, environ.Pwd, environ.Home)
		}

		return &Environment{
			Root:           environ.Root,
			Home:           environ.Home,
			Pwd:            environ.Pwd,
			KdepsConfig:    kdepsConfigFile,
			NonInteractive: "1", // Prioritize non-interactive mode for overridden environments
			DockerMode:     dockerMode,
			TimeoutSec:     environ.TimeoutSec,
		}, nil
	}

	// Load environment variables into a new Environment struct
	environment := &Environment{}
	extras, err := env.UnmarshalFromEnviron(environment)
	if err != nil {
		return nil, err
	}
	environment.Extras = extras

	// Ensure NonInteractive is set from the environment variable
	environment.NonInteractive = os.Getenv("NON_INTERACTIVE")

	// Use OR condition: check env var first (already loaded), then auto-detect
	dockerMode := environment.DockerMode
	if dockerMode != "1" && isDockerEnvironment(fs, environment.Root) {
		dockerMode = "1"
	}

	// Only search for config file if NOT in Docker mode
	kdepsConfigFile := ""
	if dockerMode != "1" {
		kdepsConfigFile = findKdepsConfig(fs, environment.Pwd, environment.Home)
	}

	return &Environment{
		Root:           environment.Root,
		Home:           environment.Home,
		Pwd:            environment.Pwd,
		KdepsConfig:    kdepsConfigFile,
		DockerMode:     dockerMode,
		Extras:         environment.Extras,
		NonInteractive: environment.NonInteractive,
		TimeoutSec:     environment.TimeoutSec,
	}, nil
}
