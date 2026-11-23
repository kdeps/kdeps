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
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v3/pkg/assets"
	"github.com/kdeps/kdeps/v3/pkg/config"
	kdeps_debug "github.com/kdeps/kdeps/v3/pkg/debug"
	executorLLM "github.com/kdeps/kdeps/v3/pkg/executor/llm"
	"github.com/kdeps/kdeps/v3/pkg/formulas"
	"github.com/kdeps/kdeps/v3/pkg/version"
)

// newConfigCmd is "kdeps config": where kdeps keeps things and what it uses.
func newConfigCmd() *cobra.Command {
	kdeps_debug.Log("enter: newConfigCmd")
	var edit bool
	c := &cobra.Command{
		Use:   "config",
		Short: "Show the kdeps setup: version, backend, model, folders and registries",
		Long: `Show what kdeps uses: its version and platform, the LLM backend and models
from ~/.kdeps/config.yaml, the folders it keeps assets, models and registries
in, the asset profile, and each registry with when it was last downloaded.

--edit opens ~/.kdeps/config.yaml in your editor ($KDEPS_EDITOR, $VISUAL,
$EDITOR, then vi).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if edit {
				return runEdit(cmd, args)
			}
			return printConfig(cmd.OutOrStdout())
		},
	}
	c.Flags().BoolVar(&edit, "edit", false, "open ~/.kdeps/config.yaml in your editor")
	return c
}

// printConfig writes the setup, one "Name: value" line each.
func printConfig(w io.Writer) error {
	line := func(k, v string) { fmt.Fprintf(w, "%-11s %s\n", k+":", v) }
	line("kdeps", fmt.Sprintf("%s (commit %s) %s/%s", version.Version, version.Commit, runtime.GOOS, runtime.GOARCH))

	path, err := config.Path()
	if err != nil {
		return err
	}
	cfg, loadErr := config.LoadStruct()
	switch {
	case loadErr != nil:
		line("Config", path+" (cannot read: "+loadErr.Error()+")")
	case fileExists(path):
		line("Config", path)
	default:
		line("Config", path+" (not created yet)")
	}
	backend, models := "file", "(none: a chat with no model runs on pybot)"
	if cfg != nil {
		if cfg.LLM.Backend != "" {
			backend = cfg.LLM.Backend
		}
		if names := modelNames(cfg); len(names) > 0 {
			models = strings.Join(names, ", ")
		}
	}
	line("Backend", backend)
	line("Models", models)

	line("Assets", assets.Root())
	if lock, lockErr := assets.ReadLock(assets.Root()); lockErr == nil && lock.Profile != "" {
		line("Profile", lock.Profile)
	}
	if dir, dirErr := executorLLM.DefaultModelsDir(); dirErr == nil {
		line("Models dir", dir)
	}

	src, srcErr := formulas.Configured()
	if srcErr != nil {
		line("Registries", "("+srcErr.Error()+")")
		return nil //nolint:nilerr // a bad registries: list is shown, not fatal
	}
	line("Registries", src.Root)
	for _, reg := range src.Registries {
		state := "not downloaded yet"
		if at := formulas.FetchedAt(src.Root, reg); at != "" {
			state = "downloaded " + at
		}
		fmt.Fprintf(w, "  %-28s %s\n", reg, state)
	}
	return nil
}

// modelNames are the model names config.yaml lists under llm.models.
func modelNames(cfg *config.Config) []string {
	var out []string
	for _, m := range cfg.LLM.Models {
		if m.Model != "" {
			out = append(out, m.Model)
		}
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
