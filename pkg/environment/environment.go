package environment

import (
	"fmt"
	"os"
	"path/filepath"

	env "github.com/Netflix/go-env"
	"github.com/spf13/afero"
)

var SystemconfigFileName = ".kdeps.pkl"

type Environment struct {
	Root           string `env:"ROOT_DIR,default=/"`
	Home           string `env:"HOME"`
	Pwd            string `env:"PWD"`
	KdepsConfig    string `env:"KDEPS_CONFIG,default=$HOME/.kdeps.pkl"`
	DockerMode     string `env:"DOCKER_MODE,default=0"`
	NonInteractive string `env:"NON_INTERACTIVE,default=0"`
	Extras         env.EnvSet
}

// Helper to check and set kdepsConfig if the file exists in the given path
func checkConfig(fs afero.Fs, baseDir string) (string, error) {
	configFile := filepath.Join(baseDir, SystemconfigFileName)
	if _, err := fs.Stat(configFile); err == nil {
		return configFile, nil
	}
	return "", nil
}

func NewEnvironment(fs afero.Fs, environ *Environment) (*Environment, error) {
	// If an environment is provided, prioritize overriding configurations
	if environ != nil {
		var kdepsConfigFile, dockerMode string

		// Check for kdeps config in Pwd directory
		if configFile, _ := checkConfig(fs, environ.Pwd); configFile != "" {
			kdepsConfigFile = configFile
		}

		// Check for kdeps config in Home directory
		if configFile, _ := checkConfig(fs, environ.Home); configFile != "" {
			kdepsConfigFile = configFile
		}

		// Check if running in Docker by detecting .dockerenv
		dockerEnvFlag := filepath.Join(environ.Root, ".dockerenv")
		if _, err := fs.Stat(dockerEnvFlag); err == nil {
			fmt.Println("Hello 2")
			dockerMode = "1"
		}

		return &Environment{
			Root:           environ.Root,
			Home:           environ.Home,
			Pwd:            environ.Pwd,
			KdepsConfig:    kdepsConfigFile,
			NonInteractive: "1",
			DockerMode:     dockerMode,
		}, nil
	}

	environment := &Environment{}

	// Otherwise, load environment variables and extra settings
	es, err := env.UnmarshalFromEnviron(environment)
	if err != nil {
		return nil, err
	}
	environment.Extras = es

	// Set defaults for paths and docker mode
	kdepsConfigFile, dockerMode := "", "0"

	// Check for kdeps config in Pwd directory
	if configFile, _ := checkConfig(fs, environment.Pwd); configFile != "" {
		kdepsConfigFile = configFile
	}

	// Check for kdeps config in Home directory
	if configFile, _ := checkConfig(fs, environment.Home); configFile != "" {
		kdepsConfigFile = configFile
	}

	// Check if running in Docker by detecting .dockerenv
	dockerEnvFlag := filepath.Join(environment.Root, ".dockerenv")
	if _, err := fs.Stat(dockerEnvFlag); err == nil {
		if allDockerEnvVarsSet() {
			dockerMode = "1"
		}
	}

	return &Environment{
		Root:        environment.Root,
		Home:        environment.Home,
		Pwd:         environment.Pwd,
		KdepsConfig: kdepsConfigFile,
		DockerMode:  dockerMode,
		Extras:      environment.Extras,
	}, nil
}

func allDockerEnvVarsSet() bool {
	vars := []string{"SCHEMA_VERSION", "OLLAMA_HOST", "KDEPS_HOST"}

	for _, v := range vars {
		if value, exists := os.LookupEnv(v); !exists || value == "" {
			return false
		}
	}
	return true
}
