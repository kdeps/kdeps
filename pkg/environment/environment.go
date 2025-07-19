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
	LocalMode      string `env:"KDEPS_LOCAL_MODE,default=0"`
	NonInteractive string `env:"NON_INTERACTIVE,default=0"`
	TimeoutSec     int    `env:"TIMEOUT,default=60"`
	Extras         env.EnvSet
}

// CheckConfig checks if the .kdeps.pkl file exists in the given directory.
func CheckConfig(fs afero.Fs, baseDir string) (string, error) {
	configFile := filepath.Join(baseDir, SystemConfigFileName)
	if exists, err := afero.Exists(fs, configFile); err == nil && exists {
		return configFile, nil
	} else {
		return "", err
	}
}

// FindKdepsConfig searches for the .kdeps.pkl file in both the Pwd and Home directories.
func FindKdepsConfig(fs afero.Fs, pwd, home string) string {
	// Check for kdeps config in Pwd directory
	if configFile, _ := CheckConfig(fs, pwd); configFile != "" {
		return configFile
	}
	// Check for kdeps config in Home directory
	if configFile, _ := CheckConfig(fs, home); configFile != "" {
		return configFile
	}
	return ""
}

// IsDockerEnvironment checks for the presence of Docker-related indicators.
func IsDockerEnvironment(fs afero.Fs, root string) bool {
	dockerEnvFlag := filepath.Join(root, ".dockerenv")
	if exists, _ := afero.Exists(fs, dockerEnvFlag); exists {
		// Ensure all required Docker environment variables are set
		return AllDockerEnvVarsSet()
	}
	return false
}

// AllDockerEnvVarsSet checks if required Docker environment variables are set.
func AllDockerEnvVarsSet() bool {
	requiredVars := []string{"SCHEMA_VERSION", "OLLAMA_HOST", "KDEPS_HOST"}
	for _, v := range requiredVars {
		if value, exists := os.LookupEnv(v); !exists || value == "" {
			return false
		}
	}
	return true
}

// IsLocalMode checks if local mode is enabled via environment variable.
func IsLocalMode() bool {
	return os.Getenv("KDEPS_LOCAL_MODE") == "1"
}

// NewEnvironment initializes and returns a new Environment based on provided or default settings.
func NewEnvironment(fs afero.Fs, environ *Environment) (*Environment, error) {
	if environ != nil {
		// If an environment is provided, prioritize overriding configurations
		kdepsConfigFile := FindKdepsConfig(fs, environ.Pwd, environ.Home)
		dockerMode := "0"
		if IsDockerEnvironment(fs, environ.Root) {
			dockerMode = "1"
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

	// Find kdepsConfig file and check if running in Docker
	kdepsConfigFile := FindKdepsConfig(fs, environment.Pwd, environment.Home)
	dockerMode := "0"
	if IsDockerEnvironment(fs, environment.Root) {
		dockerMode = "1"
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
