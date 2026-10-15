// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/knowledge"
)

// newKnowledgeCmd is "kdeps knowledge": the vault the agent recalls from.
func newKnowledgeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "knowledge",
		Short: "Knowledge vault: routing cards, session notes and graph recall for the agent",
		Long: `The knowledge vault is a folder of markdown notes the agent knows without
reading: each turn, kdeps walks the vault's link graph from the words of the
request and names the few notes worth reading. It is on by default for agent
sessions, in ~/.kdeps/notes (set up on first use); turn it off with
knowledge.enabled: false in config.yaml, or /knowledge off in a session.

  kdeps knowledge init [folder]     scaffold KDEPS.md, records/ and the PARA folders (default ~/.kdeps/notes)
  kdeps knowledge build             (re)build the link graph, changed notes only
  kdeps knowledge check             one home per fact: duplicates, dangling links, missing summaries
  kdeps knowledge recall "query"    show what recall would give the agent
  kdeps knowledge report            print GRAPH_REPORT.md: god nodes, communities, surprising connections

One home per fact: status lives in a card (records/projects/<name>.md), the
story in a note, the connections in the graph (kept under ~/.kdeps).`,
	}
	c.PersistentFlags().
		String("vault", "", "vault folder (default: knowledge.vault in ~/.kdeps/config.yaml, else ~/.kdeps/notes)")
	c.AddCommand(newKnowledgeInitCmd(), newKnowledgeBuildCmd(), newKnowledgeCheckCmd(), newKnowledgeRecallCmd(),
		newKnowledgeReportCmd())
	return c
}

// knowledgeTarget resolves the vault and workspace from --vault or config.
func knowledgeTarget(cmd *cobra.Command) (knowledge.Vault, knowledge.Workspace, *config.KnowledgeConfig, error) {
	kc := &config.KnowledgeConfig{}
	if cfg, err := config.LoadStruct(); err == nil && cfg.Knowledge != nil {
		kc = cfg.Knowledge
	}
	vault, _ := cmd.Flags().GetString("vault")
	if vault == "" {
		vault = kc.Vault
	}
	home, _ := os.UserHomeDir()
	v := knowledge.Vault{Root: knowledge.DefaultVault()}
	if vault != "" {
		v.Root = knowledge.ExpandPath(vault, home)
	}
	ws := knowledge.Workspace(v)
	if kc.Workspace != "" {
		ws.Root = knowledge.ExpandPath(kc.Workspace, home)
	}
	if info, err := os.Stat(v.Root); err != nil || !info.IsDir() {
		return v, ws, kc, fmt.Errorf("vault %s is not a folder (run kdeps knowledge init)", v.Root)
	}
	return v, ws, kc, nil
}

func newKnowledgeInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [folder]",
		Short: "Scaffold a knowledge workspace in a vault folder (default ~/.kdeps/notes)",
		Long: `Creates what the folder lacks - the PARA folders (Inbox, Projects, Areas,
Resources, Archives; existing ones with any number prefix are reused),
records/projects, records/clients, references/, KDEPS.md and schema.md - and
sets knowledge.vault in ~/.kdeps/config.yaml when a folder other than the
default is given and none is set. Existing files are never changed, so an
existing notes folder works as is.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := knowledge.DefaultVault()
			if len(args) == 1 {
				root = knowledge.ExpandPath(args[0], mustGetwd())
			}
			if err := os.MkdirAll(root, 0o750); err != nil {
				return err
			}
			v := knowledge.Vault{Root: root}
			ws := knowledge.Workspace(v)
			created, err := knowledge.Init(v, ws)
			out := cmd.OutOrStdout()
			for _, p := range created {
				fmt.Fprintf(out, "  created %s\n", p)
			}
			if err != nil {
				return err
			}
			if len(created) == 0 {
				fmt.Fprintf(out, "%s already has everything.\n", root)
			}
			if cfg, loadErr := config.LoadStruct(); loadErr == nil && root != knowledge.DefaultVault() &&
				(cfg.Knowledge == nil || cfg.Knowledge.Vault == "") {
				if saveErr := config.SaveKnowledgeVault(root); saveErr != nil {
					return saveErr
				}
				fmt.Fprintf(out, "Set knowledge.vault: %s in ~/.kdeps/config.yaml\n", root)
			}
			fmt.Fprintln(
				out,
				"Next: add a card per project under records/projects (see schema.md), then kdeps knowledge build.",
			)
			return nil
		},
	}
}

func mustGetwd() string {
	wd, _ := os.Getwd()
	return wd
}

func newKnowledgeBuildCmd() *cobra.Command {
	var extract bool
	var model string
	var limit int
	c := &cobra.Command{
		Use:   "build",
		Short: "Build the vault's link graph (only changed notes and code files are re-read)",
		Long: `Builds the vault's graph from what the notes and code write down (links,
tags, imports), groups it into communities and writes the report.

With --extract, a local model first reads every note that changed since its
last extraction and lists what it is about; its findings join the graph as
inferred edges (with the model's confidence) or ambiguous ones (below 0.5).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			v, ws, kc, err := knowledgeTarget(cmd)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			dir := knowledge.GraphDir(v.Root)
			if extract {
				if extractErr := runKnowledgeExtract(cmd, v, dir, kc, model, limit); extractErr != nil {
					return extractErr
				}
			}
			start := time.Now()
			g, st, err := knowledge.Build(cmd.Context(), v, ws, dir, time.Now())
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "%d notes (%d changed), %d code files -> %d nodes, %d edges, %d communities in %s\n",
				st.Notes, st.Changed, st.CodeFiles, st.Nodes, st.Edges, st.Communities,
				time.Since(start).Round(time.Millisecond))
			if len(g.Missing) > 0 {
				fmt.Fprintf(
					out,
					"%d links point to notes that do not exist (kdeps knowledge check lists them)\n",
					len(g.Missing),
				)
			}
			fmt.Fprintf(out, "Graph:  %s\nReport: %s\n", knowledge.GraphPath(dir), knowledge.ReportPath(dir))
			if _, statErr := os.Stat(knowledge.HTMLPath(dir)); statErr == nil {
				fmt.Fprintf(out, "Explore: %s\n", knowledge.HTMLPath(dir))
			}
			return nil
		},
	}
	c.Flags().BoolVar(&extract, "extract", false, "have a local model read changed notes first (inferred edges)")
	c.Flags().
		StringVar(&model, "model", "", "model for --extract (default: knowledge.extract.model, else the detected local model)")
	c.Flags().IntVar(&limit, "limit", 0, "most notes --extract reads (0: all changed)")
	return c
}

// runKnowledgeExtract runs the model pass with a fresh engine per call.
func runKnowledgeExtract(
	cmd *cobra.Command,
	v knowledge.Vault,
	dir string,
	kc *config.KnowledgeConfig,
	model string,
	limit int,
) error {
	backend := ""
	if model == "" {
		model, backend = kc.Extract.Model, kc.Extract.Backend
	}
	if model == "" {
		model, backend = agent.ResolveModelAndBackend("", "")
	}
	if model == "" {
		return errors.New("no model for --extract: pass --model or set knowledge.extract.model")
	}
	if backend == "" {
		backend = agent.BackendForModel(model)
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Reading changed notes with %s...\n", model)
	ask := agent.KnowledgeAsk(func() *executor.Engine { return setupEngine(nil, false) }, model, backend, "")
	st, err := knowledge.Extract(cmd.Context(), v, dir, ask, knowledge.ExtractOptions{
		Model: model, Workers: kc.Extract.Workers, Limit: limit, System: agent.HarnessText("knowledge-extract-system"),
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%d notes read (%d failed), %d were pending\n", st.Done+st.Failed, st.Failed, st.Pending)
	return nil
}

func newKnowledgeCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Report facts with more than one home, dangling links and notes recall cannot use",
		RunE: func(cmd *cobra.Command, _ []string) error {
			v, ws, _, err := knowledgeTarget(cmd)
			if err != nil {
				return err
			}
			findings, err := knowledge.Check(v, ws)
			if err != nil {
				return err
			}
			printFindings(cmd.OutOrStdout(), v.Root, findings)
			if len(findings) > 0 {
				return fmt.Errorf("%d findings", len(findings))
			}
			return nil
		},
	}
}

func printFindings(w io.Writer, root string, findings []knowledge.Finding) {
	if len(findings) == 0 {
		fmt.Fprintln(w, "No findings: every fact has one home.")
		return
	}
	for _, f := range findings {
		rel, err := filepath.Rel(root, f.File)
		if err != nil || strings.HasPrefix(rel, "..") {
			rel = f.File
		}
		fmt.Fprintf(w, "  %s: %s\n", rel, f.Problem)
	}
}

func newKnowledgeRecallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "recall <query>",
		Short: "Show the notes recall would give the agent for a request",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, _, kc, err := knowledgeTarget(cmd)
			if err != nil {
				return err
			}
			g, err := knowledge.Load(knowledge.GraphDir(v.Root))
			if err != nil {
				return errors.New("no graph yet: run kdeps knowledge build")
			}
			r := knowledge.NewIndex(g).Recall(strings.Join(args, " "), kc.Recall.Hops, kc.Recall.BudgetTokens)
			out := cmd.OutOrStdout()
			if len(r.Hits) == 0 {
				fmt.Fprintln(out, "No notes relate to that.")
				return nil
			}
			fmt.Fprintf(out, "seeds: %s\n", strings.Join(r.Seeds, ", "))
			for _, h := range r.Hits {
				via := ""
				if h.Via != "" {
					via = fmt.Sprintf("  (%d hop from %s)", h.Hops, h.Via)
				}
				fmt.Fprintf(out, "  %-50s %.2f%s\n", h.Note, h.Score, via)
				if h.Summary != "" {
					fmt.Fprintf(out, "      %s\n", h.Summary)
				}
			}
			fmt.Fprintf(out, "%d notes, ~%d tokens\n", len(r.Hits), r.Tokens)
			return nil
		},
	}
}

func newKnowledgeReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Print the graph report: god nodes, communities, surprising connections, dangling links",
		Long: `Prints GRAPH_REPORT.md from the last build: the most connected notes (god
nodes), the communities Louvain found, single links between communities that
otherwise never touch (surprising connections) and links to missing notes.
The report and graph.html (an explorer page) live next to graph.json under
~/.kdeps, never in the vault.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			v, _, _, err := knowledgeTarget(cmd)
			if err != nil {
				return err
			}
			dir := knowledge.GraphDir(v.Root)
			data, err := os.ReadFile(knowledge.ReportPath(dir))
			if err != nil {
				return errors.New("no report yet: run kdeps knowledge build")
			}
			out := cmd.OutOrStdout()
			fmt.Fprint(out, string(data))
			fmt.Fprintf(out, "\n(%s", knowledge.ReportPath(dir))
			if _, statErr := os.Stat(knowledge.HTMLPath(dir)); statErr == nil {
				fmt.Fprintf(out, "; explore: %s", knowledge.HTMLPath(dir))
			}
			fmt.Fprintln(out, ")")
			return nil
		},
	}
}
