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

package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	kdeps_debug "github.com/kdeps/kdeps/v3/pkg/debug"
)

// shortHelp is what kdeps, kdeps help and kdeps --help print.
const shortHelp = `Example usage:
  kdeps search TEXT|/REGEX/
  kdeps info [NAME...]
  kdeps install NAME...
  kdeps update
  kdeps upgrade [NAME...]
  kdeps uninstall NAME...
  kdeps list [NAME...]

Build and run:
  kdeps new DIR
  kdeps run [workflow.yaml]
  kdeps validate | test
  kdeps bundle build|package|prepackage
  kdeps export iso|k8s

Troubleshooting:
  kdeps --version | --upgrade
  kdeps config
  kdeps doctor
  kdeps install --verbose --debug NAME

Contributing:
  kdeps create URL [--no-fetch]
  kdeps edit NAME...
  kdeps verify [DIR]

Further help:
  kdeps commands
  kdeps help [COMMAND]
  https://kdeps.com
`

// setShortHelp makes the root help the short list; subcommands keep the
// full cobra help.
func setShortHelp(root *cobra.Command) {
	cobraHelp := root.HelpFunc()
	root.SetHelpFunc(func(c *cobra.Command, args []string) {
		if c != root {
			cobraHelp(c, args)
			return
		}
		fmt.Fprint(c.OutOrStdout(), shortHelp)
	})
}

// newCommandsCmd is kdeps commands: every command with its summary.
func newCommandsCmd() *cobra.Command {
	kdeps_debug.Log("enter: newCommandsCmd")
	return &cobra.Command{
		Use:   "commands",
		Short: "List every kdeps command with its summary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			printCommands(cmd.OutOrStdout(), cmd.Root())
			return nil
		},
	}
}

func printCommands(w io.Writer, root *cobra.Command) {
	for _, c := range root.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		fmt.Fprintf(w, "%-12s %s\n", c.Name(), strings.TrimSuffix(c.Short, "."))
	}
}
