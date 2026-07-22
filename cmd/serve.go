package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

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

// refreshREPLModelLists repopulates the four model lists on repl.
// Called at startup and inside the SetRefreshModelsFn closure.
func refreshREPLModelLists(repl *agent.REPL) {
	// Refresh registries to pick up any embedded data changes since last build.
	llm.ReloadGGUFRegistry()
	llm.ReloadRegistry()
	repl.SetModelNames(buildAllModelNames())
	repl.SetDownloadedModels(llm.DownloadedModelAliases())
	repl.SetModelTypes(buildModelTypes())
	repl.SetModelRepos(buildModelRepos())
}

// runLlamaFit runs llmfit and populates the REPL's score maps.
// Non-fatal: if llmfit is not installed or fails, scores are simply absent.
// Synchronous at startup since llmfit returns in <1s; the scores should
// be present before the user opens the model picker for the first time.
// "fit" (not "recommend") is used because it returns every model llmfit
// knows including Too Tight ones — recommend cuts at min-fit=marginal,
// leaving most catalog models unscored.
func runLlamaFit(repl *agent.REPL) {
	//nolint:noctx // llmfit is a local sub-second CLI call; no context needed.
	cmd := exec.Command("llmfit", "fit", "--json")
	out, execErr := cmd.Output()
	if execErr != nil {
		return
	}
	var result struct {
		Models []struct {
			Name     string  `json:"name"`
			Score    float64 `json:"score"`
			FitLevel string  `json:"fit_level"`
			GGUFSrcs []struct {
				Repo string `json:"repo"`
			} `json:"gguf_sources"`
		} `json:"models"`
	}
	if unmarshalErr := json.Unmarshal(out, &result); unmarshalErr != nil {
		return
	}
	// Index llmfit results two ways: by exact GGUF source repo, and by the
	// normalized base model name. Most llmfit entries have no gguf_sources,
	// and llmfit's "name" is the base repo (Qwen/Qwen2.5-1.5B-Instruct) while
	// kdeps registry repos are quantizer repos (bartowski/…-GGUF), so exact
	// repo matching alone leaves most aliases unscored.
	type scoreEntry struct {
		score float64
		fit   string
	}
	repoMap := make(map[string]scoreEntry)
	nameMap := make(map[string]scoreEntry)
	record := func(m map[string]scoreEntry, key string, e scoreEntry) {
		if key == "" {
			return
		}
		if existing, ok := m[key]; !ok || e.score > existing.score {
			m[key] = e
		}
	}
	for _, m := range result.Models {
		entry := scoreEntry{m.Score, m.FitLevel}
		record(nameMap, normalizeModelKey(m.Name), entry)
		for _, src := range m.GGUFSrcs {
			record(repoMap, strings.ToLower(src.Repo), entry)
			record(nameMap, normalizeModelKey(src.Repo), entry)
		}
	}
	// Map each alias to its score via its HuggingFace repo: exact repo match
	// first, then normalized base-name match.
	scores := make(map[string]float64)
	fitLevels := make(map[string]string)
	for _, alias := range repl.ModelNames() {
		repo := repl.ModelRepos()[alias]
		if repo == "" {
			continue
		}
		entry, ok := repoMap[strings.ToLower(repo)]
		if !ok {
			entry, ok = nameMap[normalizeModelKey(repo)]
		}
		if ok {
			scores[alias] = entry.score
			fitLevels[alias] = entry.fit
		}
	}
	repl.SetLlamaFitScores(scores, fitLevels)
}

// normalizeModelKey reduces a HuggingFace repo id or model name to a
// comparable key: the part after the owner, lowercased, with quantizer
// suffixes ("-GGUF", "-llamafile") dropped and all non-alphanumerics removed.
// "bartowski/Llama-3.2-1B-Instruct-GGUF", "unsloth/Llama-3.2-1B-Instruct-GGUF",
// and "alpindale/Llama-3.2-1B-Instruct" all normalize to "llama321binstruct".
func normalizeModelKey(id string) string {
	if i := strings.LastIndex(id, "/"); i >= 0 {
		id = id[i+1:]
	}
	id = strings.ToLower(id)
	for _, suffix := range []string{"-gguf", "-llamafile", ".gguf", ".llamafile"} {
		id = strings.TrimSuffix(id, suffix)
	}
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// optionalToolNotices returns install suggestions for optional tools that
// improve the agent experience but are not required. Only missing tools
// produce a notice.
func optionalToolNotices() []string {
	var notices []string
	if _, err := exec.LookPath("aria2c"); err != nil {
		notices = append(notices,
			"aria2c not installed — model downloads use the slower built-in downloader"+
				" (brew install aria2)")
	}
	if _, err := exec.LookPath("llmfit"); err != nil {
		notices = append(notices,
			"llmfit not installed — /model can't show which models fit your hardware"+
				" (brew install AlexsJones/llmfit/llmfit)")
	}
	return notices
}

// runCronScheduler polls the cron registry every 60s and creates tasks for
// due cron jobs. Runs in a background goroutine; returns when ctx is cancelled.
func runCronScheduler(ctx context.Context) {
	const cronTickInterval = 60 * time.Second
	ticker := time.NewTicker(cronTickInterval)
	defer ticker.Stop()
	// Use robfig/cron/v3 for NextRun calculation.
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			due := agent.GlobalCronRegistry.Tick(now)
			for _, c := range due {
				_ = agent.GlobalTaskRegistry.Create(
					c.TaskPrompt, c.TaskDescription,
				)
				var nextRun time.Time
				if sched, parseErr := parser.Parse(c.Expression); parseErr == nil {
					nextRun = sched.Next(now)
				}
				agent.GlobalCronRegistry.MarkRun(c.CronID, nextRun)
			}
		}
	}
}

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
	// Single root context for the entire agent loop session lifetime.
	// All derived contexts (REPL, model prefetch, tool execution) derive from this.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// The REPL owns SIGINT (Ctrl+C cancels the current turn/tool without
	// exiting); tell the llm package's local-server shutdown hook not to also
	// kill the running local model server on every Ctrl+C. Graceful shutdown
	// is instead handled by the deferred llm.ShutdownLocalServers() below.
	llm.SetInteractiveSignalOwner(true)

	registry := tools.NewRegistry()
	tools.RegisterFFormatTools(registry)

	// Lean mode: KDEPS_LEAN_MODE=1 removes bash/network tools, reducing the
	// tool surface for CI/automation use cases.
	// KDEPS_AGENT_PRESET=audit|explain|implement automatically selects a
	// permission mode and filters tools accordingly.
	initLeanTools(rootCtx, registry)

	hostWorkflow, err := resolveHostWorkflow(path, registry, flags)
	if err != nil {
		return err
	}

	// Load persisted settings and register discovered items accordingly.
	// Default (SelectAll: true) registers everything found in ~/.kdeps.
	settings, _ := tui.LoadSettings()
	applySettingsToRegistry(settings, registry, flags, flags.Debug)

	skillPaths := resolveSkillPaths(flags.SkillPaths)

	eng := setupEngine(nil, flags.Debug)
	llmAdapter := llm.NewAdapter(flags.BaseURL)

	store := agent.NewSessionStore("")
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		store.SetCwd(cwd)
	}

	memStore := agent.NewMemoryStore("")
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		memStore.SetCwd(cwd)
		_ = memStore.Load()
	}

	startModel, startBackend := resolveStartModel(flags, settings)

	cfg := agent.Config{
		Model:        startModel,
		Backend:      startBackend,
		BaseURL:      flags.BaseURL,
		SystemPrompt: flags.SystemPrompt,
		SkillPaths:   skillPaths,
		Streamer:     llmAdapter,
		ModelService: llm.NewModelService(nil),
		Store:        store,
		MemoryStore:  memStore,
	}

	// Restore full LLM config from persistent session memory. Only sets fields
	// the user hasn't explicitly overridden via CLI flags.
	if flags.Model == "" && flags.Backend == "" && flags.BaseURL == "" {
		agent.RestoreSessionConfig(memStore, &cfg)
	}

	if flags.Resume != "" {
		saved, loadErr := store.Load(flags.Resume)
		if loadErr != nil {
			return fmt.Errorf("agent loop: failed to load session %q: %w", flags.Resume, loadErr)
		}
		cfg.ResumeSession = saved
	}

	// Prefetch the model so it is ready before the first prompt. Ctrl+C during
	// this startup phase cancels the download/load and exits; the signal watch
	// is released before repl.Run() so the REPL owns SIGINT afterwards.
	prefetchCtx, stopPrefetchSignals := signal.NotifyContext(
		rootCtx, os.Interrupt, syscall.SIGTERM,
	)
	prefetchModel(prefetchCtx, resolveAgentBackend(flags.Backend, startModel), startModel)
	interrupted := prefetchCtx.Err() != nil
	stopPrefetchSignals()
	if interrupted {
		return errors.New("agent loop: interrupted during model startup")
	}

	loop := agent.New(eng, hostWorkflow, registry, cfg)
	repl := agent.NewREPL(rootCtx, loop)
	defer llm.ShutdownLocalServers()

	wireREPL(repl, registry, flags)

	// Start cron background scheduler: polls every 60s for due cron jobs.
	// Each due job creates a task and advances its NextRun.
	ctx, cancelCron := context.WithCancel(rootCtx)
	defer cancelCron()
	go runCronScheduler(ctx)

	err = repl.Run()
	return err
}

// resolveHostWorkflow loads and registers the workflow/agency at the given path.
func resolveHostWorkflow(path string, registry *tools.Registry, flags *agentLoopFlags) (*domain.Workflow, error) {
	if path == "" {
		return newMinimalHostWorkflow(), nil
	}
	absPath, absErr := filepathAbsAgentLoopFunc(path)
	if absErr != nil {
		return nil, fmt.Errorf("agent loop: invalid path %q: %w", path, absErr)
	}
	info, statErr := os.Stat(absPath)
	if statErr != nil {
		return nil, fmt.Errorf("agent loop: path not found %q: %w", path, statErr)
	}
	return loadAndRegisterAll(absPath, info.IsDir(), registry, flags.Debug)
}

// wireREPL connects model lists, llmfit scores, pickers, and TUI runners to
// a freshly created REPL.
func wireREPL(repl *agent.REPL, registry *tools.Registry, flags *agentLoopFlags) {
	// Provide model name suggestions for /model <tab> completion.
	refreshREPLModelLists(repl)
	runLlamaFit(repl)
	repl.SetCloudModelBackends(buildCloudBackends())
	repl.SetProviderStatus(agent.BuildProviderStatus())
	// Merge optional-tool notices + preflight warnings for low-limit backends.
	notices := optionalToolNotices()
	notices = append(notices, agent.RequestSizePreflightWarnings()...)
	repl.SetStartupNotices(notices)

	// Refresh in-memory model lists after /model hff download registers a new GGUF.
	repl.SetRefreshModelsFn(func() { refreshREPLModelLists(repl) })

	// Load persisted custom OpenAI-compatible endpoints and let new ones
	// registered via "/model <base-url>" persist back.
	if s, err := tui.LoadSettings(); err == nil {
		endpoints := make(map[string]string, len(s.CustomOpenAIModels))
		for _, m := range s.CustomOpenAIModels {
			endpoints[m.Alias] = m.BaseURL
		}
		repl.SetCustomEndpoints(endpoints, tui.AddCustomOpenAIModel)
		repl.SetFavorites(s.FavoriteModels, tui.SetFavoriteModel)
	}

	// Wire default-model persistence for /model default <name>.
	repl.SetSaveDefaultFn(tui.SaveDefaultModel)

	// Persist /model tool settings across sessions, and apply any saved ones at
	// startup. tui.AgentLoopTuning and agent.ToolTuning have identical fields, so
	// they convert directly.
	repl.SetSaveTuningFn(func(t agent.ToolTuning) error {
		return tui.SaveAgentLoopTuning(tui.AgentLoopTuning(t))
	})
	if s, err := tui.LoadSettings(); err == nil && s.AgentLoop != nil {
		repl.SetPersistedTuning(agent.ToolTuning(*s.AgentLoop))
	}

	// Wire model picker TUI.
	repl.SetModelPickerFn(buildModelPickerFn(repl))

	// Wire /settings TUI when running interactively.
	if isTerminal(os.Stdout) && isTerminal(os.Stdin) {
		repl.SetTUIRunner(buildTUIRunner(registry, flags))
	}
}

// resolveStartModel returns the model and backend to use at startup.
// Falls back to settings.DefaultModel when --model is not given.
// Auto-selects BackendGGUF when the model is a GGUF alias or path.
func resolveStartModel(flags *agentLoopFlags, settings tui.Settings) (string, string) {
	m := flags.Model
	b := flags.Backend
	if m == "" && settings.DefaultModel != "" {
		m = settings.DefaultModel
	}
	// .gguf suffix always means GGUF backend regardless of env vars.
	if b == "" && agent.IsGGUFModelName(m) {
		b = llm.BackendGGUF
	}
	// Use config/env backend as fallback.
	if b == "" {
		if v := os.Getenv("KDEPS_DEFAULT_BACKEND"); v != "" {
			b = v
		}
	}
	// Resolve backend from model name when nothing else is set (e.g., model
	// restored from memory on resume).
	if b == "" {
		if backend := agent.BackendForModel(m); backend != "" {
			b = backend
		}
	}
	return m, b
}

// resolveAgentBackend returns the effective LLM backend, applying the same
// fallback order as the LLM executor: flag -> env var -> model catalog -> "file" (llamafile).
func resolveAgentBackend(flagBackend, model string) string {
	if flagBackend != "" {
		return flagBackend
	}
	if env := os.Getenv("KDEPS_DEFAULT_BACKEND"); env != "" {
		return env
	}
	if backend := agent.BackendForModel(model); backend != "" {
		return backend
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
			Version: defaultVersion,
		},
	}
}

// initLeanTools handles tool registration with lean/preset mode filtering.
func initLeanTools(ctx context.Context, reg *tools.Registry) {
	if !agent.ResolveLeanMode() && !agent.IsLeanOrPreseted() {
		agent.RegisterBuiltinTools(ctx, reg)
		return
	}
	agent.RegisterBuiltinTools(ctx, reg)
	mode, applied := agent.ApplyPresetIfConfigured(reg)
	if !applied {
		// Lean mode without a preset: restrict tools but keep default permission.
		allTools := reg.List()
		kept := agent.LeanModeToolFilter(extractToolNames(allTools))
		keptSet := make(map[string]bool, len(kept))
		for _, n := range kept {
			keptSet[n] = true
		}
		for _, t := range allTools {
			if !keptSet[t.Name] {
				reg.Unregister(t.Name)
			}
		}
		return
	}
	_ = mode // permission mode applied; consumed below
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

// extractToolNames returns the Name field from each tool in the slice.
func extractToolNames(tools []*tools.Tool) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
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
	// First pass: llamafile aliases.
	for _, a := range llm.ListLlamafileMappings() {
		types[a.Alias] = "llamafile"
	}
	// Second pass: GGUF aliases may overwrite. When an alias is in both
	// registries, check what's actually on disk so the user sees the right
	// type (e.g. "llamafile cached" vs "gguf cached").
	modelsDir, dirErr := llm.DefaultModelsDir()
	for _, a := range llm.ListGGUFMappings() {
		if types[a.Alias] == "llamafile" && dirErr == nil {
			if p, ok := llm.LlamafileCachedPath(a.Alias, modelsDir); ok {
				if _, err := os.Stat(p); err == nil {
					// The .llamafile file exists — keep "llamafile" type.
					continue
				}
			}
		}
		types[a.Alias] = "gguf"
	}
	for _, o := range llm.ListOllamaModels() {
		types[o.Name] = chatBackendOllama
	}
	return types
}

// buildModelRepos returns a map of model alias → HuggingFace repo id (e.g. "googleai/gemma4")
// for llamafile and gguf models. Shown in /models next to each local model alias.
func buildModelRepos() map[string]string {
	repos := make(map[string]string)
	for _, a := range llm.ListLlamafileMappings() {
		if a.Repo != "" {
			repos[a.Alias] = a.Repo
		}
	}
	for _, a := range llm.ListGGUFMappings() {
		if a.Repo != "" {
			repos[a.Alias] = a.Repo
		}
	}
	return repos
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
		repos := repl.ModelRepos()
		backends := repl.CloudModelBackends()
		status := repl.ProviderStatus()
		for _, name := range names {
			backend := backends[name]
			// enabled = API key is set for this model's provider.
			// Map existence alone does not mean enabled; check providerStatus.
			enabled := backend != "" && status[backend]
			entries = append(entries, tui.ModelEntry{
				Name:      name,
				ModelType: types[name],
				Backend:   backend,
				Repo:      repos[name],
				Cached:    downloaded[name],
				Enabled:   enabled,
				Score:     repl.LlamaFitScore(name),
				FitLevel:  repl.LlamaFitFitLevel(name),
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
