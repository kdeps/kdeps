package cfg

import (
	"context"
	"errors"
	"fmt"
	"kdeps/pkg/download"
	"os"
	"path/filepath"

	env "github.com/Netflix/go-env"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/editor"
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
	NonInteractive string `env:"NON_INTERACTIVE"`
	Extras         env.EnvSet
}

func FindConfiguration(fs afero.Fs, environment *Environment) error {
	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = filepath.Join(environment.Home, SystemConfigFileName)
		return nil
	}

	if len(environment.Pwd) > 0 {
		CwdConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)

		ConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)
		return nil
	}

	es, err := env.UnmarshalFromEnviron(&environment)
	if err != nil {
		return err
	}

	environment.Extras = es

	CwdConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)
	HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

	if _, err := fs.Stat(CwdConfigFile); err == nil {
		ConfigFile = CwdConfigFile
	} else if _, err = fs.Stat(HomeConfigFile); err == nil {
		ConfigFile = HomeConfigFile
	}

	if _, err = fs.Stat(ConfigFile); err == nil {
		log.Info("Configuration file found:", "config-file", ConfigFile)
	}

	return nil
}

func DownloadConfiguration(fs afero.Fs, environment *Environment) error {
	var skipPrompts bool
	if len(environment.NonInteractive) > 0 {
		skipPrompts = true
	}

	es, err := env.UnmarshalFromEnviron(&environment)
	if err != nil {
		return err
	}

	environment.Extras = es

	if _, err := fs.Stat(ConfigFile); err != nil {
		ConfigFile = HomeConfigFile

		var confirm bool
		if !skipPrompts {
			if err := huh.Run(
				huh.NewConfirm().
					Title("Configuration file not found. Do you want to generate one?").
					Description("The configuration will be validated. This will require the `pkl` package to be installed. Please refer to https://pkl-lang.org for more details.").
					Value(&confirm),
			); err != nil {
				return errors.New(fmt.Sprintln("Could not create a configuration file:", ConfigFile))
			}

			if !confirm {
				return errors.New("Aborted by user")
			}
		}

		download.DownloadFile(fs, "https://github.com/kdeps/schema/releases/latest/download/kdeps.pkl", ConfigFile)

		if !skipPrompts {
			c, err := editor.Cmd("kdeps", ConfigFile)
			if err != nil {
				return errors.New(fmt.Sprintln("Config file does not exist!"))
			}

			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr

			if err := c.Run(); err != nil {
				return errors.New(fmt.Sprintf("Missing %s.", "$EDITOR"))
			}
		}

	}

	ConfigFile = HomeConfigFile

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
