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

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/llmserver/catalog"

	"github.com/spf13/cobra"
)

func newLLMListCmd() *cobra.Command {
	kdeps_debug.Log("enter: newLLMListCmd")
	return &cobra.Command{
		Use:   "list",
		Short: "List available LLM server recipes",
		Long: `List stock and user/project LLM server recipes.

Stock recipes ship with kdeps. Override or add recipes in:
  ~/.kdeps/llm-servers/*.yaml
  ./llm-servers/*.yaml

No workflow path is required.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runLLMList()
		},
	}
}

const (
	llmListPad     = 2
	llmListDescMax = 60
)

func runLLMList() error {
	kdeps_debug.Log("enter: runLLMList")
	entries, err := catalog.LoadDefault()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, llmListPad, ' ', 0)
	fmt.Fprintln(w, "ID\tENGINE\tPORT\tMODEL\tSOURCE\tDESCRIPTION")
	for _, e := range entries {
		r := e.Recipe
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			r.ID,
			r.Engine.Kind,
			r.API.Port,
			r.Models.Strategy,
			e.Source,
			truncate(r.Description, llmListDescMax),
		)
	}
	return w.Flush()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
