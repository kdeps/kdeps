package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
	"github.com/kdeps/kdeps/v2/pkg/tui"
)

// filepathAbsAgentLoopFunc resolves agent loop paths (overridable in tests).
//
//nolint:gochecknoglobals // test-replaceable hook
var filepathAbsAgentLoopFunc = filepath.Abs

// registerAgencyTargetParseFunc parses agency target workflows (overridable in tests).
//
//nolint:gochecknoglobals // test-replaceable hook
var registerAgencyTargetParseFunc = ParseWorkflowFile

type agentLoopFlags struct {
	Model        string
	Backend      string
	BaseURL      string
	SystemPrompt string
	Debug        bool
	SkillPaths   []string
	Resume       string
}

// runAgentLoopCmd starts the interactive agent loop. When path is empty the
// loop starts with no workflow tools (model-only mode). When path is provided
// every workflow and agency found at that path is registered as a tool.
func runAgentLoopCmd(path string, flags *agentLoopFlags) error {
	registry := tools.NewRegistry()
	tools.RegisterFFormatTools(registry)

	var (
		hostWorkflow *domain.Workflow
		err          error
	)

	if path != "" {
		absPath, absErr := filepathAbsAgentLoopFunc(path)
		if absErr != nil {
			return fmt.Errorf("agent loop: invalid path %q: %w", path, absErr)
		}
		info, statErr := os.Stat(absPath)
		if statErr != nil {
			return fmt.Errorf("agent loop: path not found %q: %w", path, statErr)
		}
		hostWorkflow, err = loadAndRegisterAll(absPath, info.IsDir(), registry, flags.Debug)
		if err != nil {
			return err
		}
	} else {
		hostWorkflow = newMinimalHostWorkflow()
	}

	// Show TUI selector when running in an interactive terminal.
	// A TUI failure is non-fatal — we fall through to the REPL without selections.
	if isTerminal(os.Stdout) && isTerminal(os.Stdin) {
		if sel, tuiErr := tui.Run(); tuiErr == nil {
			applyTUISelection(sel, registry, flags, flags.Debug)
		}
	}

	skillPaths := resolveSkillPaths(flags.SkillPaths)

	cfg := agent.Config{
		Model:        flags.Model,
		Backend:      flags.Backend,
		BaseURL:      flags.BaseURL,
		SystemPrompt: flags.SystemPrompt,
		SkillPaths:   skillPaths,
	}
	eng := setupEngine(nil, flags.Debug)

	if flags.Resume != "" {
		store := agent.NewSessionStore("")
		saved, loadErr := store.Load(flags.Resume)
		if loadErr != nil {
			return fmt.Errorf("agent loop: failed to load session %q: %w", flags.Resume, loadErr)
		}
		cfg.ResumeSession = saved
	}

	loop := agent.New(eng, hostWorkflow, registry, cfg)
	repl := agent.NewREPL(loop)
	return repl.Run()
}

// resolveSkillPaths converts relative skill paths to absolute paths.
func resolveSkillPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	resolved := make([]string, 0, len(paths))
	for _, p := range paths {
		if abs, err := filepath.Abs(p); err == nil {
			resolved = append(resolved, abs)
		}
	}
	return resolved
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

	agencyEng := setupEngine(nil, debug)
	agencyEng.SetNewExecutionContextForAgency(nameMap)

	agencyTool := agencyToolDef(agency, targetWF, agencyEng)
	registry.Register(agencyTool)
	return targetWF, nil
}

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

func agencyToolDef(agency *domain.Agency, entryWorkflow *domain.Workflow, eng *executor.Engine) *tools.Tool {
	name, desc := agencyToolNameAndDesc(agency)
	return tools.AgentToolDefWithName(name, desc, entryWorkflow, eng)
}

// newMinimalHostWorkflow returns a bare workflow used as the agent loop host.
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

// isTerminal returns true when f is connected to an interactive terminal.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// applyTUISelection registers tools and skill paths from the TUI selection.
func applyTUISelection(sel tui.Selection, registry *tools.Registry, flags *agentLoopFlags, debug bool) {
	for _, it := range sel.Workflows {
		if _, regErr := registerServeTools(it.Path, registry, debug); regErr != nil {
			continue
		}
	}
	for _, it := range sel.Agencies {
		if _, regErr := registerServeTools(it.Path, registry, debug); regErr != nil {
			continue
		}
	}
	for _, it := range sel.Components {
		if _, regErr := registerServeTools(it.Path, registry, debug); regErr != nil {
			continue
		}
	}
	for _, it := range sel.Skills {
		flags.SkillPaths = append(flags.SkillPaths, it.Path)
	}
}
