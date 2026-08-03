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
	"regexp"
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

// refreshREPLModelLists repopulates the model lists on repl from one unified,
// collision-qualified pass over every local + cloud model (see
// buildUnifiedEntries). Called at startup and inside the SetRefreshModelsFn
// closure.
func refreshREPLModelLists(repl *agent.REPL) {
	// Refresh registries to pick up any embedded data changes since last build.
	llm.ReloadGGUFRegistry()
	llm.ReloadRegistry()
	entries := buildUnifiedEntries()
	names := make([]string, 0, len(entries))
	types := make(map[string]string, len(entries))
	repos := make(map[string]string, len(entries))
	downloaded := make(map[string]bool, len(entries))
	cloudBackends := make(map[string]string, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
		types[e.Name] = e.Type
		if e.Repo != "" {
			repos[e.Name] = e.Repo
		}
		if e.Downloaded {
			downloaded[e.Name] = true
		}
		if e.Type == "" { // cloud
			cloudBackends[e.Name] = e.Backend
		}
	}
	repl.SetModelNames(names)
	repl.SetDownloadedModels(downloaded)
	repl.SetModelTypes(types)
	repl.SetModelRepos(repos)
	repl.SetCloudModelBackends(cloudBackends)
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
	// Index llmfit results by exact GGUF source repo, normalized base name,
	// and a looser key (instruct/chat/it stripped). Most llmfit entries have
	// no gguf_sources, and kdeps repos are quantizer/packaging repos while
	// llmfit names are base model ids — so multi-key matching is required.
	repoMap := make(map[string]llamaFitScoreEntry)
	nameMap := make(map[string]llamaFitScoreEntry)
	looseMap := make(map[string]llamaFitScoreEntry)
	record := func(m map[string]llamaFitScoreEntry, key string, e llamaFitScoreEntry) {
		if key == "" {
			return
		}
		if existing, ok := m[key]; !ok || e.score > existing.score {
			m[key] = e
		}
	}
	for _, m := range result.Models {
		entry := llamaFitScoreEntry{m.Score, m.FitLevel}
		record(nameMap, normalizeModelKey(m.Name), entry)
		record(looseMap, normalizeModelKeyLoose(m.Name), entry)
		for _, src := range m.GGUFSrcs {
			record(repoMap, strings.ToLower(src.Repo), entry)
			record(nameMap, normalizeModelKey(src.Repo), entry)
			record(looseMap, normalizeModelKeyLoose(src.Repo), entry)
		}
	}

	// Candidates per alias: HF repo, filename, and description. Filename is
	// essential for multi-model packaging repos (e.g. mozilla-ai/llamafile_0.10)
	// where the repo name is not the model name.
	candidates := buildLlamaFitMatchCandidates()

	scores := make(map[string]float64)
	fitLevels := make(map[string]string)
	for _, alias := range repl.ModelNames() {
		cands := candidates[alias]
		if repo := repl.ModelRepos()[alias]; repo != "" {
			cands = append([]string{repo}, cands...)
		}
		if entry, ok := matchLlamaFitScore(cands, repoMap, nameMap, looseMap); ok {
			scores[alias] = entry.score
			fitLevels[alias] = entry.fit
		}
	}
	repl.SetLlamaFitScores(scores, fitLevels)
}

// llamaFitScoreEntry is the score + fit level for one matched model.
type llamaFitScoreEntry struct {
	score float64
	fit   string
}

// buildLlamaFitMatchCandidates returns alias -> filename/description strings
// from the GGUF and llamafile registries for name-based llmfit matching.
func buildLlamaFitMatchCandidates() map[string][]string {
	out := make(map[string][]string)
	add := func(alias, filename, desc string) {
		if alias == "" {
			return
		}
		var cands []string
		if filename != "" {
			cands = append(cands, filename)
		}
		if desc != "" {
			cands = append(cands, desc)
		}
		if len(cands) == 0 {
			return
		}
		out[alias] = append(out[alias], cands...)
	}
	for _, e := range llm.ListLlamafileMappings() {
		add(e.Alias, e.Filename, e.Description)
	}
	for _, e := range llm.ListGGUFMappings() {
		add(e.Alias, e.Filename, e.Description)
	}
	return out
}

// matchLlamaFitScore tries exact repo, then normalized name, then loose name
// against the candidate strings (repo / filename / description).
func matchLlamaFitScore(
	cands []string,
	repoMap, nameMap, looseMap map[string]llamaFitScoreEntry,
) (llamaFitScoreEntry, bool) {
	for _, c := range cands {
		if c == "" {
			continue
		}
		if entry, ok := repoMap[strings.ToLower(c)]; ok {
			return entry, true
		}
	}
	for _, c := range cands {
		if entry, ok := nameMap[normalizeModelKey(c)]; ok {
			return entry, true
		}
	}
	for _, c := range cands {
		if entry, ok := looseMap[normalizeModelKeyLoose(c)]; ok {
			return entry, true
		}
	}
	return llamaFitScoreEntry{}, false
}

// trailingQuantRE matches a trailing GGUF/llamafile quant suffix on a stem,
// e.g. "-Q4_K_M", ".Q8_0", "-UD-Q2_K_XL". Applied after lowercasing.
var trailingQuantRE = regexp.MustCompile(`(?i)[._-](?:ud-)?(?:q|iq)\d+(?:_[a-z0-9]+)*$`)

// normalizeModelKey reduces a HuggingFace repo id, filename, or model name to a
// comparable key: the part after the owner, lowercased, with quantizer and
// packaging suffixes dropped, Meta- prefix stripped, trailing quant markers
// removed, and all non-alphanumerics removed.
//
//	bartowski/Llama-3.2-1B-Instruct-GGUF
//	mozilla-ai/Meta-Llama-3.1-8B-Instruct-llamafile
//	Meta-Llama-3.1-8B-Instruct.Q4_K_M.llamafile
//	alpindale/Llama-3.2-1B-Instruct
//
// all normalize toward the same family of keys (meta- stripped so Meta-Llama
// and Llama share a key with llmfit's meta-llama/Llama-… names).
func normalizeModelKey(id string) string {
	if id == "" {
		return ""
	}
	// Descriptions may end with " [default]".
	if i := strings.Index(id, " ["); i >= 0 {
		id = id[:i]
	}
	if i := strings.LastIndex(id, "/"); i >= 0 {
		id = id[i+1:]
	}
	id = strings.ToLower(id)
	for _, suffix := range []string{".gguf", ".llamafile", "-gguf", "-llamafile", "-hf"} {
		id = strings.TrimSuffix(id, suffix)
	}
	for {
		next := trailingQuantRE.ReplaceAllString(id, "")
		if next == id {
			break
		}
		id = next
	}
	// llmfit uses meta-llama/Llama-3.1-…; mozilla-ai ships Meta-Llama-3.1-….
	id = strings.TrimPrefix(id, "meta-")
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalizeModelKeyLoose is normalizeModelKey plus stripping of role suffixes
// that differ across registries: "instruct", "chat", and gemma's trailing "it".
// Used as a fallback so Llama-3.2-3B-Instruct matches llmfit's Llama-3.2-3B.
func normalizeModelKeyLoose(id string) string {
	k := normalizeModelKey(id)
	for _, s := range []string{"instruct", "chat"} {
		k = strings.ReplaceAll(k, s, "")
	}
	// gemma-4-12b-it -> gemma412bit -> gemma412b. The size unit "b" sits
	// between the digit and the role suffix "it".
	if len(k) > 4 && strings.HasSuffix(k, "bit") {
		if prev := k[len(k)-4]; prev >= '0' && prev <= '9' {
			k = k[:len(k)-2] // drop "it", keep trailing "b"
		}
	}
	return k
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

// agentModelTypeLlamafile is the display type for llamafile-backed models.
const agentModelTypeLlamafile = "llamafile"

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
	// Fill empty model/backend the same way the agent loop does (file default
	// llama3.2:1b, cloud keys, ollama, cached local models).
	startModel, startBackend = agent.ResolveModelAndBackend(startModel, startBackend)

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
	// Provide model name suggestions for /model <tab> completion. Cloud
	// backends are populated as part of the same unified pass now, not a
	// separate call (see refreshREPLModelLists / buildUnifiedEntries).
	refreshREPLModelLists(repl)
	runLlamaFit(repl)
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
	// A backend-qualified value (e.g. "gguf:qwen3:30b", saved once a
	// previously-ambiguous bare name was disambiguated) fully determines
	// both fields; skip the bare-name inference chain below entirely so it
	// never mis-parses the qualifier as part of the model name.
	if qualBackend, bareModel, ok := agent.SplitQualifiedModelName(m); ok {
		m = bareModel
		if b == "" {
			b = qualBackend
		}
		return m, b
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

// buildUnifiedEntries collects one agent.LocalModelEntry per llamafile, GGUF,
// Ollama, and cloud-catalog model, with same-named entries from different
// backends (e.g. "qwen3:30b" registered in both the llamafile and GGUF
// registries as distinct downloadable artifacts) qualified to
// "<type>:<name>" so neither silently shadows the other. Non-colliding
// entries keep their bare alias name unchanged. See agent.BuildUnifiedModelEntries.
func buildUnifiedEntries() []agent.LocalModelEntry {
	modelsDir, dirErr := llm.DefaultModelsDir()
	llamafileCached := func(alias string) bool {
		if dirErr != nil {
			return false
		}
		p, ok := llm.LlamafileCachedPath(alias, modelsDir)
		if !ok {
			return false
		}
		_, statErr := os.Stat(p)
		return statErr == nil
	}
	ggufCached := func(alias string) bool {
		if dirErr != nil {
			return false
		}
		p, ok := llm.GGUFCachedPath(alias, modelsDir)
		if !ok {
			return false
		}
		_, statErr := os.Stat(p)
		return statErr == nil
	}
	return agent.BuildUnifiedModelEntries(
		llm.ListLlamafileMappings(), llm.ListGGUFMappings(), llm.ListOllamaModels(),
		llamafileCached, ggufCached, true,
	)
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
			_, bareName, _ := agent.SplitQualifiedModelName(name)
			sizeGB := ""
			switch types[name] {
			case agentModelTypeLlamafile:
				sizeGB = tui.FormatSizeGB(llm.LlamafileSizeBytes(bareName))
			case agentBackendGGUF:
				sizeGB = tui.FormatSizeGB(llm.GGUFSizeBytes(bareName))
			}
			entries = append(entries, tui.ModelEntry{
				Name:      name,
				ModelType: types[name],
				Backend:   backend,
				Repo:      repos[name],
				Cached:    downloaded[name],
				Enabled:   enabled,
				SizeGB:    sizeGB,
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
