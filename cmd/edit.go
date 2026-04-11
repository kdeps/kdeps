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

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

func newEditCmd() *cobra.Command {
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
	// Ensure the config file exists before opening it.
	if err := config.Scaffold(); err != nil {
		return fmt.Errorf("create config: %w", err)
	}

	path, err := config.Path()
	if err != nil {
		return fmt.Errorf("locate config: %w", err)
	}

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
