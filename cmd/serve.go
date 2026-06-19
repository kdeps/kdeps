package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
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

// agentBackendFile is the default LLM backend (llamafile).
const agentBackendFile = "file"

// agentBackendGGUF is the llama.cpp/llama-server backend for GGUF model files.
const agentBackendGGUF = "gguf"

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
//
// Discovered items from ~/.kdeps are registered according to persisted settings
// (default: all enabled). Use /settings inside the REPL to change selections.
func runAgentLoopCmd(path string, flags *agentLoopFlags) error {
	registry := tools.NewRegistry()
	tools.RegisterFFormatTools(registry)
	agent.RegisterBuiltinTools(context.Background(), registry)

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

	// Load persisted settings and register discovered items accordingly.
	// Default (SelectAll: true) registers everything found in ~/.kdeps.
	settings, _ := tui.LoadSettings()
	applySettingsToRegistry(settings, registry, flags, flags.Debug)

	skillPaths := resolveSkillPaths(flags.SkillPaths)

	eng := setupEngine(nil, flags.Debug)
	llmAdapter := llm.NewAdapter(flags.BaseURL)

	cfg := agent.Config{
		Model:        flags.Model,
		Backend:      flags.Backend,
		BaseURL:      flags.BaseURL,
		SystemPrompt: flags.SystemPrompt,
		SkillPaths:   skillPaths,
		Streamer:     llmAdapter,
		ModelService: llm.NewModelService(nil),
	}

	if flags.Resume != "" {
		store := agent.NewSessionStore("")
		saved, loadErr := store.Load(flags.Resume)
		if loadErr != nil {
			return fmt.Errorf("agent loop: failed to load session %q: %w", flags.Resume, loadErr)
		}
		cfg.ResumeSession = saved
	}

	// Start model download in background so it is ready before the first prompt.
	prefetchModel(resolveAgentBackend(flags.Backend), flags.Model)

	loop := agent.New(eng, hostWorkflow, registry, cfg)
	repl := agent.NewREPL(loop)
	defer llm.ShutdownLocalServers()

	// Provide model name suggestions for /model <tab> completion.
	repl.SetModelNames(buildAllModelNames())
	repl.SetDownloadedModels(llm.DownloadedModelAliases())
	repl.SetModelTypes(buildModelTypes())
	repl.SetCloudModelBackends(buildCloudBackends())
	repl.SetProviderStatus(agent.BuildProviderStatus())

	// Wire model picker TUI.
	repl.SetModelPickerFn(buildModelPickerFn(repl))

	// Wire /settings TUI when running interactively.
	if isTerminal(os.Stdout) && isTerminal(os.Stdin) {
		repl.SetTUIRunner(buildTUIRunner(registry, flags))
	}

	err = repl.Run()
	return err
}

// resolveAgentBackend returns the effective LLM backend, applying the same
// fallback order as the LLM executor: flag -> env var -> "file" (llamafile).
func resolveAgentBackend(flagBackend string) string {
	if flagBackend != "" {
		return flagBackend
	}
	if env := os.Getenv("KDEPS_DEFAULT_BACKEND"); env != "" {
		return env
	}
	return agentBackendFile
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

// buildTUIRunner returns an agent.TUIRunner that opens the settings TUI,
// saves the result, and reports what changed.
func buildTUIRunner(registry *tools.Registry, flags *agentLoopFlags) agent.TUIRunner {
	return func() ([]string, bool, error) {
		prevSel := tui.SelectionFromSettings(func() tui.Settings {
			s, _ := tui.LoadSettings()
			return s
		}())

		sel, _, tuiErr := tui.Run()
		if tuiErr != nil {
			return nil, false, tuiErr
		}

		skillPaths := make([]string, 0, len(sel.Skills))
		for _, it := range sel.Skills {
			skillPaths = append(skillPaths, it.Path)
		}

		// Detect if tool selections changed (requires restart to take effect).
		toolsChanged := !selectionsEqual(prevSel, sel)

		// Register newly selected tools immediately (best-effort; duplicates are safe).
		for _, it := range sel.Workflows {
			_, _ = registerServeTools(it.Path, registry, flags.Debug)
		}
		for _, it := range sel.Agencies {
			_, _ = registerServeTools(it.Path, registry, flags.Debug)
		}
		for _, it := range sel.Components {
			_, _ = registerServeTools(it.Path, registry, flags.Debug)
		}

		return skillPaths, toolsChanged, nil
	}
}

// selectionsEqual returns true when the workflow/agency/component sets are identical.
func selectionsEqual(a, b tui.Selection) bool {
	return namesEqual(a.Workflows, b.Workflows) &&
		namesEqual(a.Agencies, b.Agencies) &&
		namesEqual(a.Components, b.Components)
}

func namesEqual(a, b []tui.Item) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name {
			return false
		}
	}
	return true
}

// buildModelNames returns local model alias names from llamafile, gguf, and ollama.
func buildModelNames() []string {
	names := append(llm.LlamafileAliasNames(), llm.GGUFAliasNames()...)
	for _, o := range llm.ListOllamaModels() {
		names = append(names, o.Name)
	}
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// buildAllModelNames returns local model aliases followed by all known cloud model IDs.
// Local models sort first so they appear first in /model <tab> completion.
func buildAllModelNames() []string {
	local := buildModelNames()
	cloud := agent.CloudModelIDs()
	seen := make(map[string]bool, len(local)+len(cloud))
	out := make([]string, 0, len(local)+len(cloud))
	for _, n := range local {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	for _, n := range cloud {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// buildModelTypes returns a map of model name → type used by /model completion.
// Types: "" (cloud), "llamafile", "gguf". Used for visual prefix in tab completion:
//
//	(no prefix) = cloud / ollama
//	~ = llamafile (not downloaded)
//	# = GGUF (not downloaded)
//	* = downloaded (any type, overrides)
func buildModelTypes() map[string]string {
	types := make(map[string]string)
	for _, a := range llm.ListLlamafileMappings() {
		types[a.Alias] = "llamafile"
	}
	for _, a := range llm.ListGGUFMappings() {
		types[a.Alias] = "gguf"
	}
	for _, o := range llm.ListOllamaModels() {
		types[o.Name] = chatBackendOllama
	}
	return types
}

// buildCloudBackends returns a map from cloud model name → backend for /model
// completion. Used to show [deepseek] instead of [cloud] when the API key is set.
func buildCloudBackends() map[string]string {
	m := make(map[string]string)
	for _, cm := range agent.KnownCloudModels {
		m[cm.ID] = cm.Backend
	}
	return m
}

// buildModelPickerFn returns a function that opens the TUI model picker with
// data from the agent REPL's model catalog.
func buildModelPickerFn(repl *agent.REPL) func(filter string) (string, error) {
	return func(filter string) (string, error) {
		entries := make([]tui.ModelEntry, 0)
		names := repl.ModelNames()
		downloaded := repl.DownloadedModels()
		types := repl.ModelTypes()
		backends := repl.CloudModelBackends()
		status := repl.ProviderStatus()
		for _, name := range names {
			backend, enabled := backends[name]
			if !enabled && types[name] == "" {
				// Only check provider status for cloud models without type data.
				for _, cm := range agent.KnownCloudModels {
					if cm.ID == name && status[cm.Backend] {
						enabled = true
						backend = cm.Backend
						break
					}
				}
			}
			entries = append(entries, tui.ModelEntry{
				Name:      name,
				ModelType: types[name],
				Backend:   backend,
				Cached:    downloaded[name],
				Enabled:   enabled,
			})
		}
		return tui.RunModelPicker(entries, repl.CurrentModel(), filter)
	}
}

// applySettingsToRegistry discovers items from ~/.kdeps and registers those
// permitted by settings. When SelectAll is true (the default), everything is
// registered. Otherwise only items whose names appear in the enabled lists are
// registered.
func applySettingsToRegistry(settings tui.Settings, registry *tools.Registry, flags *agentLoopFlags, debug bool) {
	sel := tui.SelectionFromSettings(settings)
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
