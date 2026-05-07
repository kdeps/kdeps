// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

//go:build !js

// Package cmd provides the command-line interface for KDeps.
// It implements all CLI commands including run, build, validate, and package operations.
package cmd

import (
	"fmt"
	"os"

	"github.com/kdeps/kdeps/v2/pkg/config"
	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	kdepslog "github.com/kdeps/kdeps/v2/pkg/log"

	"github.com/spf13/cobra"
)

const (
	groupDevelop    = "develop"
	groupPackage    = "package"
	groupDistribute = "distribute"
	groupDeploy     = "deploy"
)

// CLIConfig holds the CLI configuration.
type CLIConfig struct {
	version string
	commit  string
}

// NewCLIConfig creates a new CLI configuration.
func NewCLIConfig() *CLIConfig {
	kdeps_debug.Log("enter: NewCLIConfig")
	return &CLIConfig{}
}

// GetRootCommand returns the root cobra command with proper configuration.
func (c *CLIConfig) GetRootCommand() *cobra.Command {
	kdeps_debug.Log("enter: GetRootCommand")
	rootCmd := createRootCommand()
	rootCmd.Version = fmt.Sprintf("%s (commit: %s)", c.version, c.commit)
	return rootCmd
}

// Execute runs the root command.
func Execute(v, c string) error {
	kdeps_debug.Log("enter: Execute")
	config := NewCLIConfig()
	config.version = v
	config.commit = c

	rootCmd := config.GetRootCommand()
	err := rootCmd.Execute()
	kdeps_debug.Flush()
	return err
}

// NewRootCmd returns a new root cobra command for testing.
func NewRootCmd() *cobra.Command {
	kdeps_debug.Log("enter: NewRootCmd")
	return createRootCommand()
}

// createRootCommand creates the root cobra command with all subcommands.
func createRootCommand() *cobra.Command {
	kdeps_debug.Log("enter: createRootCommand")
	rootCmd := &cobra.Command{
		Use:   "kdeps",
		Short: "KDeps - AI Agent Framework",
		Long:  `Build AI agents with YAML configuration.`,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			// Initialize structured logger.
			debugFlag, _ := cmd.Flags().GetBool("debug")
			verboseFlag, _ := cmd.Flags().GetBool("verbose")
			kdepslog.Init(debugFlag, verboseFlag)

			// On first run (no config file), bootstrap interactively.
			// In non-interactive environments Bootstrap falls back to Scaffold.
			if bootErr := config.Bootstrap(os.Stdout); bootErr != nil {
				kdepslog.Warn("bootstrap failed", "error", bootErr)
			}
			if _, loadErr := config.Load(); loadErr != nil {
				kdepslog.Warn("could not load config", "error", loadErr)
			}
			// --instrument enables call-chain instrumentation (pkg/debug).
			if instrFlag, err := cmd.Flags().GetBool("instrument"); err == nil && instrFlag {
				_ = os.Setenv("KDEPS_INSTRUMENT", "true")
			}
			kdepslog.Warn("HIGHLY EXPERIMENTAL SOFTWARE — under active development, expect breaking changes",
				"feedback", "https://github.com/kdeps/kdeps/issues")
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().Bool("instrument", false, "Enable call-chain instrumentation tracing")

	// Add subcommands
	addSubcommands(rootCmd)

	return rootCmd
}

// addSubcommands registers all subcommands to the root command.
func addSubcommands(rootCmd *cobra.Command) {
	kdeps_debug.Log("enter: addSubcommands")

	rootCmd.AddGroup(
		&cobra.Group{ID: groupDevelop, Title: "Develop:"},
		&cobra.Group{ID: groupPackage, Title: "Package:"},
		&cobra.Group{ID: groupDistribute, Title: "Distribute:"},
		&cobra.Group{ID: groupDeploy, Title: "Deploy:"},
	)

	// Develop
	newCmd := newNewCmd()
	newCmd.GroupID = groupDevelop
	rootCmd.AddCommand(newCmd)

	editCmd := newEditCmd()
	editCmd.GroupID = groupDevelop
	rootCmd.AddCommand(editCmd)

	validateCmd := newValidateCmd()
	validateCmd.GroupID = groupDevelop
	rootCmd.AddCommand(validateCmd)

	runCmd := newRunCmd()
	runCmd.GroupID = groupDevelop
	rootCmd.AddCommand(runCmd)

	// Package
	bundleCmd := newBundleCmd()
	bundleCmd.GroupID = groupPackage
	rootCmd.AddCommand(bundleCmd)

	// Distribute
	registryCmd := newRegistryCmd()
	registryCmd.GroupID = groupDistribute
	rootCmd.AddCommand(registryCmd)

	// Deploy
	execCmd := newExecCmd()
	execCmd.GroupID = groupDeploy
	rootCmd.AddCommand(execCmd)

	exportCmd := newExportCmd()
	exportCmd.GroupID = groupDeploy
	rootCmd.AddCommand(exportCmd)

	chatCmd := newChatCmd()
	chatCmd.GroupID = groupDevelop
	rootCmd.AddCommand(chatCmd)

	doctorCmd := newDoctorCmd()
	doctorCmd.GroupID = groupDevelop
	rootCmd.AddCommand(doctorCmd)
}
