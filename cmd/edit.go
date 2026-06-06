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

//go:build !js

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

func newEditCmd() *cobra.Command {
	kdeps_debug.Log("enter: newEditCmd")
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit the global kdeps configuration file (~/.kdeps/config.yaml)",
		Long: `Open ~/.kdeps/config.yaml in your editor.

The editor is chosen from, in order:
  1. $KDEPS_EDITOR
  2. $VISUAL
  3. $EDITOR
  4. vi (fallback)`,
		RunE: runEdit,
	}
}

func runEdit(_ *cobra.Command, _ []string) error {
	kdeps_debug.Log("enter: runEdit")

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
	cmd := exec.CommandContext(context.Background(), editor, path)
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
