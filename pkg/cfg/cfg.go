package cfg

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/texteditor"
	"path/filepath"
	"strings"

	env "github.com/Netflix/go-env"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
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
	NonInteractive string `env:"NON_INTERACTIVE,default=0"`
	Extras         env.EnvSet
}

func FindConfiguration(fs afero.Fs, environment *Environment) error {
	evaluator.FindPklBinary()

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
		return nil
	}

	if len(environment.Pwd) > 0 {
		CwdConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)

		ConfigFile = CwdConfigFile
		return nil
	}

	es, err := env.UnmarshalFromEnviron(&environment)
	if err != nil {
		return err
	}

	environment.Extras = es

	CwdConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)
	HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

	if _, err = fs.Stat(CwdConfigFile); err == nil {
		ConfigFile = CwdConfigFile
	} else if _, err = fs.Stat(HomeConfigFile); err == nil {
		ConfigFile = HomeConfigFile
	}

	if _, err = fs.Stat(ConfigFile); err == nil {
		log.Info("Configuration file found:", "config-file", ConfigFile)
	} else {
		log.Warn("Configuration file not found:", "config-file", ConfigFile)
	}

	return nil
}

func GenerateConfiguration(fs afero.Fs, environment *Environment) error {
	var skipPrompts bool = false

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return err
		}

		environment.Extras = es

		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	}

	if environment.NonInteractive == "1" {
		skipPrompts = true
	}

	if _, err := fs.Stat(ConfigFile); err != nil {
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

		// Read the schema version from the SCHEMA_VERSION file
		schemaVersionBytes, err := ioutil.ReadFile("../../SCHEMA_VERSION")
		if err != nil {
			log.Fatalf("Failed to read SCHEMA_VERSION: %v", err)
		}
		schemaVersion := strings.TrimSpace(string(schemaVersionBytes))

		// Create the URL with the schema version
		url := fmt.Sprintf("package://schema.kdeps.com/core@%s#/Kdeps.pkl", schemaVersion)

		// Evaluate the .pkl file and write the result to ConfigFile (append mode)
		result, err := evaluator.EvalPkl(fs, url)
		if err != nil {
			log.Fatalf("Failed to evaluate .pkl file: %v", err)
		}

		content := fmt.Sprintf("amends \"%s\"\n%s", url, result)
		if err = afero.WriteFile(fs, ConfigFile, []byte(content), 0644); err != nil {
			log.Fatalf("Failed to open %s: %v", ConfigFile, err)
		}
	}

	return nil
}

func EditConfiguration(fs afero.Fs, environment *Environment) error {
	var skipPrompts bool = false

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return err
		}

		environment.Extras = es

		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	}

	if environment.NonInteractive == "1" {
		skipPrompts = true
	}

	if _, err := fs.Stat(ConfigFile); err == nil {
		if !skipPrompts {
			if err := texteditor.EditPkl(fs, ConfigFile); err != nil {
				return err
			}
		}
	}

	return nil
}

func ValidateConfiguration(fs afero.Fs, environment *Environment) error {
	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return err
		}

		environment.Extras = es

		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	}

	if _, err := evaluator.EvalPkl(fs, ConfigFile); err != nil {
		return err
	}

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
