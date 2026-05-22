package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

type serveFlags struct {
	Model        string
	Backend      string
	BaseURL      string
	SystemPrompt string
	Debug        bool
}

func newServeCmd() *cobra.Command {
	flags := &serveFlags{}

	cmd := &cobra.Command{
		Use:   "serve <path>",
		Short: "Start agent mode (interactive LLM loop with workflows as tools)",
		Long: `Start agent mode -- an interactive LLM loop where whole workflows are
registered as callable tools.

Pass a single workflow file to expose one tool, or a directory to expose
every workflow and agency found inside as separate tools. The tool name
for each workflow is its metadata.name field. Each tool call runs the
full workflow DAG so requires: dependencies always resolve.

Examples:
  # One tool from a single workflow
  kdeps serve workflow.yaml

  # All workflows in a folder become separate tools
  kdeps serve ./agents/

  # Override the model
  kdeps serve workflow.yaml --model mistral

  # Provide a system prompt
  kdeps serve workflow.yaml --system "You are a helpful assistant."

Environment variables (override defaults):
  KDEPS_AGENT_MODEL      LLM model name (default: llama3.2)
  KDEPS_AGENT_BACKEND    LLM backend (default: ollama)
  KDEPS_AGENT_BASE_URL   LLM API base URL`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			debugMode, _ := cmd.Flags().GetBool("debug")
			flags.Debug = debugMode
			return runServeCmd(args[0], flags)
		},
	}

	cmd.Flags().StringVar(&flags.Model, "model", "", "LLM model to use (default: KDEPS_AGENT_MODEL env or llama3.2)")
	cmd.Flags().StringVar(&flags.Backend, "backend", "", "LLM backend (default: KDEPS_AGENT_BACKEND env or ollama)")
	cmd.Flags().StringVar(&flags.BaseURL, "base-url", "", "LLM API base URL (default: KDEPS_AGENT_BASE_URL env)")
	cmd.Flags().StringVar(
		&flags.SystemPrompt, "system", "",
		"System prompt injected at the start of every conversation",
	)

	return cmd
}

func runServeCmd(path string, flags *serveFlags) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("serve: invalid path %q: %w", path, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("serve: path not found %q: %w", path, err)
	}

	eng := setupEngine(nil, flags.Debug)
	registry := tools.NewRegistry()
	tools.RegisterFFormatTools(registry)

	var hostWorkflow = newMinimalHostWorkflow()

	if info.IsDir() {
		wfPaths := findServeWorkflowFiles(absPath)
		if len(wfPaths) == 0 {
			return fmt.Errorf("serve: no workflow or agency files found under %s", absPath)
		}
		for _, p := range wfPaths {
			wf, loadErr := ParseWorkflowFile(p)
			if loadErr != nil {
				return fmt.Errorf("serve: failed to load %s: %w", p, loadErr)
			}
			registry.Register(tools.AgentToolDef(wf, eng))
			registerComponentTools(registry, wf, eng)
		}
	} else {
		wf, loadErr := ParseWorkflowFile(absPath)
		if loadErr != nil {
			return fmt.Errorf("serve: failed to load workflow: %w", loadErr)
		}
		hostWorkflow = wf
		registry.Register(tools.AgentToolDef(wf, eng))
		registerComponentTools(registry, wf, eng)
	}

	cfg := agent.Config{
		Model:        flags.Model,
		Backend:      flags.Backend,
		BaseURL:      flags.BaseURL,
		SystemPrompt: flags.SystemPrompt,
	}
	loop := agent.New(eng, hostWorkflow, registry, cfg)
	return runREPL(loop)
}

func runREPL(loop *agent.Loop) error {
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Fprintln(os.Stdout, "kdeps agent mode — type your message and press Enter. Ctrl+D to exit.")
	for {
		fmt.Fprint(os.Stdout, "> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		if input == "" {
			continue
		}
		resp, err := loop.Run(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Fprintln(os.Stdout, resp)
	}
	return scanner.Err()
}

// newMinimalHostWorkflow returns a bare workflow used as the agent loop host
// when no single workflow is the canonical entry point (e.g. folder mode).
func newMinimalHostWorkflow() *domain.Workflow {
	return &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata: domain.WorkflowMetadata{
			Name:    "agent",
			Version: "1.0.0",
		},
	}
}

// registerComponentTools registers each component from wf as a callable tool.
func registerComponentTools(registry *tools.Registry, wf *domain.Workflow, eng *executor.Engine) {
	if len(wf.Components) == 0 {
		return
	}
	comps := make([]*domain.Component, 0, len(wf.Components))
	for _, c := range wf.Components {
		comps = append(comps, c)
	}
	for _, t := range tools.ComponentToolDefs(comps, wf, eng) {
		registry.Register(t)
	}
}

// findServeWorkflowFiles walks root recursively and returns one workflow or
// agency file per directory. Agency files take precedence over workflow files.
func findServeWorkflowFiles(root string) []string {
	var paths []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if p := FindAgencyFile(path); p != "" {
			paths = append(paths, p)
			return nil
		}
		if p := FindWorkflowFile(path); p != "" {
			paths = append(paths, p)
		}
		return nil
	})
	return paths
}
