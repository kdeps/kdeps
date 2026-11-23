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
	"os"
	"os/exec"

	kdeps_debug "github.com/kdeps/kdeps/v3/pkg/debug"
	"github.com/kdeps/kdeps/v3/pkg/formulas"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v3/pkg/config"
)

func newEditCmd() *cobra.Command {
	kdeps_debug.Log("enter: newEditCmd")
	return &cobra.Command{
		Use:   "edit NAME...",
		Short: "Open a formula in your editor",
		Long: `Open each formula NAME (name or owner/repo/name) in your editor, in the
local copy of its registry under ~/.kdeps/registries. kdeps install reads the
changed formula at once, so you can test it. kdeps update replaces the copy;
publish the change on GitHub (kdeps prints the link).

The editor is $KDEPS_EDITOR, $VISUAL, $EDITOR, then vi. kdeps config --edit
opens ~/.kdeps/config.yaml.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runEdit,
	}
}

func runEdit(cmd *cobra.Command, args []string) error {
	kdeps_debug.Log("enter: runEdit")
	for _, ref := range args {
		f, err := findFormula(processParent(cmd.Context()), ref)
		if err != nil {
			return err
		}
		root, err := formulas.Root()
		if err != nil {
			return err
		}
		if err = launchEditorFunc(formulas.File(root, f.Registry, f.Name)); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "To publish the change, edit https://github.com/%s/blob/HEAD/formulas/%s.yaml\n",
			f.Registry, f.Name)
	}
	return nil
}

// editConfig opens ~/.kdeps/config.yaml (kdeps config --edit).
func editConfig() error {
	path, err := prepareConfigForEdit()
	if err != nil {
		return err
	}
	return launchEditorFunc(path)
}

// configScaffoldFunc is overridable in tests for prepareConfigForEdit error paths.
//
//nolint:gochecknoglobals // test-replaceable hook
var configScaffoldFunc = config.Scaffold

// configPathFunc is overridable in tests for prepareConfigForEdit error paths.
//
//nolint:gochecknoglobals // test-replaceable hook
var configPathFunc = config.Path

// prepareConfigForEdit ensures the config file exists and returns its path.
func prepareConfigForEdit() (string, error) {
	if err := configScaffoldFunc(); err != nil {
		return "", fmt.Errorf("create config: %w", err)
	}
	path, err := configPathFunc()
	if err != nil {
		return "", fmt.Errorf("locate config: %w", err)
	}
	return path, nil
}

// launchEditorFunc opens path in the user's preferred editor (overridable in tests).
//
//nolint:gochecknoglobals // test-replaceable hook
var launchEditorFunc = launchEditor

// launchEditor opens path in the user's preferred editor.
func launchEditor(path string) error {
	editor := resolveEditor()
	cmd := exec.CommandContext(processRoot(), editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// resolveEditor picks the editor binary to use.
func resolveEditor() string {
	for _, env := range []string{"KDEPS_EDITOR", "VISUAL", "EDITOR"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return "vi"
}
