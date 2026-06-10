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

// filepathAbsServeFunc resolves serve paths (overridable in tests).
//
//nolint:gochecknoglobals // test-replaceable hook
var filepathAbsServeFunc = filepath.Abs

// registerAgencyTargetParseFunc parses agency target workflows (overridable in tests).
//
//nolint:gochecknoglobals // test-replaceable hook
var registerAgencyTargetParseFunc = ParseWorkflowFile

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
		Long: `Start agent mode -- an interactive LLM loop where whole workflows and
components are registered as callable tools.

Pass a directory to expose every workflow and agency found inside as
separate tools. Each workflow becomes one tool. Each agency becomes one
tool (the agency's entry point runs; internal agents are not exposed
individually). Pass a single workflow file or agency file to expose just
that one tool.

The tool name for each workflow or agency is its metadata.name field.
Each tool call runs the full workflow DAG so requires: dependencies
always resolve.

Examples:
  # All workflows and agencies in a folder -- each becomes one tool
  kdeps serve ./agents/

  # One tool from a single workflow directory
  kdeps serve ./my-agent/

  # One tool from an agency file (entry point runs when called)
  kdeps serve agency.yaml

  # Override the model
  kdeps serve ./agents/ --model mistral

  # Provide a system prompt
  kdeps serve ./agents/ --system "You are a helpful assistant."

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
	absPath, err := filepathAbsServeFunc(path)
	if err != nil {
		return fmt.Errorf("serve: invalid path %q: %w", path, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("serve: path not found %q: %w", path, err)
	}

	registry := tools.NewRegistry()
	tools.RegisterFFormatTools(registry)

	hostWorkflow, err := loadAndRegisterAll(absPath, info.IsDir(), registry, flags.Debug)
	if err != nil {
		return err
	}

	cfg := agent.Config{
		Model:        flags.Model,
		Backend:      flags.Backend,
		BaseURL:      flags.BaseURL,
		SystemPrompt: flags.SystemPrompt,
	}
	eng := setupEngine(nil, flags.Debug)
	loop := agent.New(eng, hostWorkflow, registry, cfg)
	return runREPL(loop)
}

// loadAndRegisterAll loads workflow/agency files from path and registers tools.
// If isDir, walks the directory; otherwise loads the single file.
// Returns the first workflow loaded for use as the agent loop host.
func loadAndRegisterAll(absPath string, isDir bool, registry *tools.Registry, debug bool) (*domain.Workflow, error) {
	hostWorkflow := newMinimalHostWorkflow()

	paths := []string{absPath}
	if isDir {
		paths = findServeWorkflowFiles(absPath)
		if len(paths) == 0 {
			return nil, fmt.Errorf("serve: no workflow or agency files found under %s", absPath)
		}
	}

	for _, p := range paths {
		first, err := registerServeTools(p, registry, debug)
		if err != nil {
			return nil, err
		}
		if first != nil && hostWorkflow.Metadata.Name == "agent" {
			hostWorkflow = first
		}
	}
	return hostWorkflow, nil
}

// registerServeTools loads a workflow or agency file and registers tools.
//
// Workflow: registers one tool (metadata.name) + its component tools.
// Agency: registers one tool (agency metadata.name) whose Execute runs
// the agency entry-point workflow. Internal agents are NOT exposed as
// individual tools. A dedicated engine with the agency's agentPaths is
// created so agent: resources inside the entry-point workflow resolve.
func registerServeTools(p string, registry *tools.Registry, debug bool) (*domain.Workflow, error) {
	if isAgencyFile(p) {
		return registerAgencyTool(p, registry, debug)
	}
	return registerWorkflowTool(p, registry, debug)
}

func serveLoadError(kind, path string, err error) error {
	return fmt.Errorf("serve: failed to load %s %s: %w", kind, path, err)
}

func registerWorkflowTool(p string, registry *tools.Registry, debug bool) (*domain.Workflow, error) {
	wf, err := ParseWorkflowFile(p)
	if err != nil {
		return nil, serveLoadError("workflow", p, err)
	}
	eng := setupEngine(nil, debug)
	registry.Register(tools.AgentToolDef(wf, eng))
	registerComponentTools(registry, wf, eng)
	return wf, nil
}

func registerAgencyTool(p string, registry *tools.Registry, debug bool) (*domain.Workflow, error) {
	agency, agentPaths, err := ParseAgencyFile(p)
	if err != nil {
		return nil, serveLoadError("agency", p, err)
	}
	nameMap, targetPath, err := buildAgentNameMap(agentPaths, agency.Metadata.TargetAgentID)
	if err != nil {
		return nil, fmt.Errorf("serve: agency %s: %w", p, err)
	}
	targetWF, err := registerAgencyTargetParseFunc(targetPath)
	if err != nil {
		return nil, fmt.Errorf("serve: agency %s target: %w", p, err)
	}

	// Give the agency its own engine so AgentPaths is scoped correctly.
	agencyEng := setupEngine(nil, debug)
	agencyEng.SetNewExecutionContextForAgency(nameMap)

	// Register one tool named after the agency (not the individual agents).
	agencyTool := agencyToolDef(agency, targetWF, agencyEng)
	registry.Register(agencyTool)
	return targetWF, nil
}

// agencyToolNameAndDesc returns the display name and description for an agency tool.
func agencyToolNameAndDesc(agency *domain.Agency) (string, string) {
	name := agency.Metadata.Name
	if name == "" {
		name = "agency"
	}
	desc := agency.Metadata.Description
	if desc == "" {
		desc = fmt.Sprintf("Agency: %s v%s", name, agency.Metadata.Version)
	}
	return name, desc
}

// agencyToolDef wraps a whole agency as a single callable tool.
// The tool name is the agency's metadata.name. Execute runs the entry-point workflow.
func agencyToolDef(agency *domain.Agency, entryWorkflow *domain.Workflow, eng *executor.Engine) *tools.Tool {
	name, desc := agencyToolNameAndDesc(agency)
	return tools.AgentToolDefWithName(name, desc, entryWorkflow, eng)
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

func findServeManifestInDir(dir string) string {
	if p := FindAgencyFile(dir); p != "" {
		return p
	}
	return FindWorkflowFile(dir)
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
		if p := findServeManifestInDir(path); p != "" {
			paths = append(paths, p)
		}
		return nil
	})
	return paths
}
