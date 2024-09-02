package cfg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	env "github.com/Netflix/go-env"
	"github.com/charmbracelet/log"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

var (
	ConfigFile           string
	SystemConfigFileName = ".kdeps.pkl"
)

type Environment struct {
	Home   string `env:"HOME"`
	Pwd    string `env:"PWD"`
	Extras env.EnvSet
}

func FindConfiguration(fs afero.Fs, environment *Environment) error {
	var homeConfigFile string
	var cwdConfigFile string

	if len(environment.Home) > 0 {
		ConfigFile = filepath.Join(environment.Home, SystemConfigFileName)
		return nil
	}

	if len(environment.Pwd) > 0 {
		ConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)
		return nil
	}

	es, err := env.UnmarshalFromEnviron(&environment)
	if err != nil {
		return err
	}

	environment.Extras = es

	cwdConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)
	homeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

	if _, err := fs.Stat(cwdConfigFile); err == nil {
		ConfigFile = cwdConfigFile
	} else if _, err = fs.Stat(homeConfigFile); err == nil {
		ConfigFile = homeConfigFile
	} else {
		if os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("Configuration file not found: %s", ConfigFile))
		}
	}

	log.Info("Configuration file found:", "config-file", ConfigFile)

	return nil
}

func LoadConfiguration(fs afero.Fs) error {
	log.Info("Reading config file:", "config-file", ConfigFile)
	_, err := kdeps.LoadFromPath(context.Background(), ConfigFile)
	if err != nil {
		return errors.New(fmt.Sprintf("Error reading config-file '%s': %s", ConfigFile, err))
	}
	return nil
}
