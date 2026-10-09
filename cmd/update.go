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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/agent"
)

// newUpdateCmd is "kdeps update": update the versioned assets (harness,
// events, actions, presets, themes, tool definitions, LLM server recipes,
// project templates). The kdeps binary itself is updated by --upgrade.
func newUpdateCmd() *cobra.Command {
	var opts agent.UpdateOptions
	c := &cobra.Command{
		Use:   "update [item[@version]...]",
		Short: "Update harness, tool definitions, themes, recipes and templates",
		Long: `Update kdeps' versioned assets from kdeps.io.

  kdeps update                          every item to its newest version
  kdeps update harness                  every item in one set
  kdeps update harness/safety           one item to its newest version (unpins it)
  kdeps update harness/safety@1.2.0     exactly that version, pinned
  kdeps update --check                  list available updates, change nothing
  kdeps update --list                   every item, its version and state
  kdeps update --remove themes/dracula  remove items (restore with kdeps update <item>)

Sets: harness, events, actions, presets, themes, tools, recipes, templates.
The kdeps binary itself is updated with kdeps --upgrade.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Items = args
			return agent.RunAssetUpdate(cmd.Context(), cmd.OutOrStdout(), opts)
		},
	}
	c.Flags().BoolVar(&opts.Check, "check", false, "list available updates without installing them")
	c.Flags().
		BoolVar(&opts.List, "list", false, "list every asset, its version and whether it is built-in, downloaded, pinned or removed")
	c.Flags().
		StringSliceVar(&opts.Remove, "remove", nil, "remove assets (set/name); restore one with kdeps update <set/name>")
	return c
}
