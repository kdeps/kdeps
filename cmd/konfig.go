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

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// defaultKonfigPath mirrors pkg/agent's REPL /konfig default.
const defaultKonfigPath = "./konfig.yaml"

func newKonfigCmd() *cobra.Command {
	kdeps_debug.Log("enter: newKonfigCmd")
	cmd := &cobra.Command{
		Use:   "konfig",
		Short: "Export or import the full agent-loop config (tuning, harness, themes, skills)",
		Long: `konfig is one YAML file that is a total, self-contained, declarative
description of a kdeps agent's behavior -- tuning, harness, themes, and
skills -- exportable from the current effective state (including pure
compiled-in defaults, with nothing customized yet) and importable to fully
configure another agent.`,
	}
	cmd.AddCommand(newKonfigExportCmd())
	return cmd
}

func newKonfigExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [path]",
		Short: "Export the current effective config to a konfig file",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runKonfigExportCmd,
	}
}

func runKonfigExportCmd(cmd *cobra.Command, args []string) error {
	kdeps_debug.Log("enter: runKonfigExportCmd")
	path := defaultKonfigPath
	if len(args) == 1 {
		path = args[0]
	}

	tuning, err := agent.PersistedOrDefaultTuning()
	if err != nil {
		return fmt.Errorf("konfig export: %w", err)
	}
	skills := agent.LoadSkillSlice(nil)

	k, err := agent.ExportKonfigWithSkills(tuning, skills)
	if err != nil {
		return fmt.Errorf("konfig export: %w", err)
	}
	if writeErr := agent.WriteKonfig(k, path); writeErr != nil {
		return fmt.Errorf("konfig export: %w", writeErr)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Exported %d harness section(s), %d theme(s), %d skill(s) to %s\n",
		len(k.Harness), len(k.Themes), len(k.Skills), path)
	return nil
}
