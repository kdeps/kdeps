package llm

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
)

// resetLlamaFitIndexCache clears the process-lifetime llmfit index cache so
// each test that exercises it (cachedLlamaFitIndex/RunLlamaFitCached/
// ScoreModelEntries) starts from a clean slate and doesn't leak into
// whichever test runs next in this package.
func resetLlamaFitIndexCache(t *testing.T) {
	t.Helper()
	reset := func() {
		llamaFitIndexOnce = sync.Once{}
		llamaFitIndexRepoMap, llamaFitIndexNameMap, llamaFitIndexLooseMap = nil, nil, nil
		llamaFitIndexOK = false
	}
	reset()
	t.Cleanup(reset)
}

func TestNormalizeModelKey(t *testing.T) {
	// Quantizer repos, llamafile repos, filenames, and base repos of the same
	// model all normalize to the same key. Meta- is stripped so Meta-Llama
	// matches llmfit's meta-llama/Llama-… names.
	want := "llama321binstruct"
	for _, in := range []string{
		"bartowski/Llama-3.2-1B-Instruct-GGUF",
		"unsloth/Llama-3.2-1B-Instruct-GGUF",
		"alpindale/Llama-3.2-1B-Instruct",
		"mozilla-ai/Llama-3.2-1B-Instruct-llamafile",
		"Llama-3.2-1B-Instruct-Q4_K_M.llamafile",
		"Llama-3.2-1B-Instruct.Q4_K_M.gguf",
	} {
		assert.Equal(t, want, normalizeModelKey(in), "input: %s", in)
	}
	assert.Equal(
		t,
		"llama318binstruct",
		normalizeModelKey("mozilla-ai/Meta-Llama-3.1-8B-Instruct-llamafile"),
	)
	assert.Equal(
		t,
		"llama318binstruct",
		normalizeModelKey("Meta-Llama-3.1-8B-Instruct.Q4_K_M.llamafile"),
	)
	assert.Equal(t, "llama318binstruct", normalizeModelKey("meta-llama/Llama-3.1-8B-Instruct"))
	assert.Equal(t, "qwen2515binstruct", normalizeModelKey("Qwen/Qwen2.5-1.5B-Instruct"))
	assert.Equal(t, "starcoder23b", normalizeModelKey("starcoder2-3b.Q4_K_M.llamafile"))
	assert.Equal(t, "qwen2515binstruct", normalizeModelKey("Qwen2.5-1.5B-Instruct [default]"))
	assert.Empty(t, normalizeModelKey(""))
}

func TestNormalizeModelKeyLoose(t *testing.T) {
	// Instruct/chat/it stripped so base and instruct variants share a key.
	assert.Equal(
		t,
		"llama323b",
		normalizeModelKeyLoose("mozilla-ai/Llama-3.2-3B-Instruct-llamafile"),
	)
	assert.Equal(t, "llama323b", normalizeModelKeyLoose("meta-llama/Llama-3.2-3B"))
	assert.Equal(t, "gemma412b", normalizeModelKeyLoose("gemma-4-12b-it-Q4_K_M.gguf"))
	assert.Equal(t, "gemma412b", normalizeModelKeyLoose("google/gemma-4-12b"))
}

func TestMatchLlamaFitScore(t *testing.T) {
	repoMap := map[string]FitScore{
		"bartowski/qwen2.5-1.5b-instruct-gguf": {66.1, "Too Tight"},
	}
	nameMap := map[string]FitScore{
		"llama321binstruct": {78.5, "Good"},
		"starcoder23b":      {55.0, "Marginal"},
	}
	looseMap := map[string]FitScore{
		"llama323b": {67.3, "Marginal"},
	}

	// Exact repo wins; a leading empty candidate is skipped, not matched.
	e, ok := matchLlamaFitScore(
		[]string{"", "bartowski/Qwen2.5-1.5B-Instruct-GGUF"},
		repoMap,
		nameMap,
		looseMap,
	)
	require.True(t, ok)
	assert.InDelta(t, 66.1, e.Score, 0.001)

	// Filename-based name match (packaging repo).
	e, ok = matchLlamaFitScore(
		[]string{"mozilla-ai/llamafile_0.10", "starcoder2-3b.Q4_K_M.llamafile"},
		repoMap, nameMap, looseMap,
	)
	require.True(t, ok)
	assert.InDelta(t, 55.0, e.Score, 0.001)
	assert.Equal(t, "Marginal", e.FitLevel)

	// Loose match: Instruct filename vs base llmfit name.
	e, ok = matchLlamaFitScore(
		[]string{
			"mozilla-ai/Llama-3.2-3B-Instruct-llamafile",
			"Llama-3.2-3B-Instruct-Q4_K_M.llamafile",
		},
		repoMap,
		nameMap,
		looseMap,
	)
	require.True(t, ok)
	assert.InDelta(t, 67.3, e.Score, 0.001)

	_, ok = matchLlamaFitScore([]string{"unknown/model"}, repoMap, nameMap, looseMap)
	assert.False(t, ok)
}

// --- ListInstalledLocalCandidates -------------------------------------------

func TestListInstalledLocalCandidates_NoneCached(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())
	t.Setenv("PATH", t.TempDir()) // no ollama on PATH either
	got := ListInstalledLocalCandidates(afero.NewMemMapFs())
	assert.Empty(t, got)
}

func TestResolveCachedPath_UnknownAlias(t *testing.T) {
	fakeCachedPath := func(string, string) (string, bool) { return "", false }
	path, ok := resolveCachedPath(afero.NewMemMapFs(), fakeCachedPath, "unknown", "/models")
	assert.False(t, ok)
	assert.Empty(t, path)
}

func TestResolveCachedPath_KnownButNotOnDisk(t *testing.T) {
	fakeCachedPath := func(string, string) (string, bool) { return "/models/x.gguf", true }
	path, ok := resolveCachedPath(afero.NewMemMapFs(), fakeCachedPath, "x", "/models")
	assert.False(t, ok)
	assert.Empty(t, path)
}

func TestResolveCachedPath_Found(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "/models/x.gguf", []byte("data"), 0o600))
	fakeCachedPath := func(string, string) (string, bool) { return "/models/x.gguf", true }
	path, ok := resolveCachedPath(fs, fakeCachedPath, "x", "/models")
	assert.True(t, ok)
	assert.Equal(t, "/models/x.gguf", path)
}

func TestListInstalledLocalCandidates_LlamafileCached(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	modelsDir, err := DefaultModelsDir()
	require.NoError(t, err)

	entries := ListLlamafileMappings()
	require.NotEmpty(t, entries, "embedded llamafile registry must not be empty")
	var alias, path string
	for _, e := range entries {
		if p, ok := LlamafileCachedPath(e.Alias, modelsDir); ok {
			alias, path = e.Alias, p
			break
		}
	}
	require.NotEmpty(t, alias, "no llamafile registry entry resolved to a cache path")

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, path, []byte("fake"), 0o600))

	got := ListInstalledLocalCandidates(fs)
	var found bool
	for _, c := range got {
		if c.Alias == alias {
			found = true
			assert.Equal(t, BackendFile, c.Backend)
			assert.True(t, c.Installed)
		}
	}
	assert.True(t, found, "cached llamafile alias %q not reported as installed", alias)
}

func TestListInstalledLocalCandidates_GGUFCachedAndLoadable(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	modelsDir, err := DefaultModelsDir()
	require.NoError(t, err)

	entries := ListGGUFMappings()
	require.NotEmpty(t, entries, "embedded GGUF registry must not be empty")
	var alias, path string
	for _, e := range entries {
		if p, ok := GGUFCachedPath(e.Alias, modelsDir); ok {
			alias, path = e.Alias, p
			break
		}
	}
	require.NotEmpty(t, alias, "no GGUF registry entry resolved to a cache path")

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, path, ggufTestHeader(3), 0o600))

	got := ListInstalledLocalCandidates(fs)
	var found bool
	for _, c := range got {
		if c.Alias == alias {
			found = true
			assert.Equal(t, BackendGGUF, c.Backend)
			assert.True(t, c.Installed)
		}
	}
	assert.True(t, found, "cached, loadable GGUF alias %q not reported as installed", alias)
}

func TestListInstalledLocalCandidates_GGUFCachedButNotLoadable(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	modelsDir, err := DefaultModelsDir()
	require.NoError(t, err)

	entries := ListGGUFMappings()
	require.NotEmpty(t, entries)
	var alias, path string
	for _, e := range entries {
		if p, ok := GGUFCachedPath(e.Alias, modelsDir); ok {
			alias, path = e.Alias, p
			break
		}
	}
	require.NotEmpty(t, alias)

	fs := afero.NewMemMapFs()
	// GGUFv1 header: present but not loadable by current llama-server builds.
	require.NoError(t, afero.WriteFile(fs, path, ggufTestHeader(1), 0o600))

	got := ListInstalledLocalCandidates(fs)
	for _, c := range got {
		assert.NotEqual(t, alias, c.Alias, "non-loadable GGUFv1 file must not be reported as installed")
	}
}

func TestListInstalledLocalCandidates_OllamaPulled(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())

	orig := execCommandContext
	t.Cleanup(func() { execCommandContext = orig })
	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		output := "NAME\tID\tSIZE\nllama3:latest\tabc123\t4.5GB\n"
		return exec.CommandContext(ctx, "echo", output)
	}

	got := ListInstalledLocalCandidates(afero.NewMemMapFs())
	var found bool
	for _, c := range got {
		if c.Alias == "llama3:latest" {
			found = true
			assert.Equal(t, "ollama", c.Backend)
			assert.True(t, c.Installed)
		}
	}
	assert.True(t, found, "pulled ollama model not reported as installed")
}

func TestListInstalledLocalCandidates_ModelsDirUnresolvable(t *testing.T) {
	// HOME unset and no KDEPS_MODELS_DIR override -> DefaultModelsDir fails.
	t.Setenv("KDEPS_MODELS_DIR", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "") // Windows home var
	got := ListInstalledLocalCandidates(afero.NewMemMapFs())
	assert.Nil(t, got)
}

// --- BestInstalled -----------------------------------------------------------

func TestBestInstalled_PicksHighestScore(t *testing.T) {
	candidates := []ScoredModelCandidate{
		{Alias: "a", Backend: BackendFile, Installed: true},
		{Alias: "b", Backend: BackendFile, Installed: true},
	}
	scores := map[string]FitScore{
		"a": {Score: 50, FitLevel: "Marginal"},
		"b": {Score: 90, FitLevel: "Good"},
	}
	alias, backend, ok := BestInstalled(candidates, scores, nil)
	require.True(t, ok)
	assert.Equal(t, "b", alias)
	assert.Equal(t, BackendFile, backend)
}

func TestBestInstalled_TiesBreakByAlias(t *testing.T) {
	candidates := []ScoredModelCandidate{
		{Alias: "z", Backend: BackendFile, Installed: true},
		{Alias: "a", Backend: BackendFile, Installed: true},
	}
	scores := map[string]FitScore{
		"z": {Score: 50},
		"a": {Score: 50},
	}
	alias, _, ok := BestInstalled(candidates, scores, nil)
	require.True(t, ok)
	assert.Equal(t, "a", alias)
}

func TestBestInstalled_SkipsNotInstalled(t *testing.T) {
	candidates := []ScoredModelCandidate{
		{Alias: "a", Backend: BackendFile, Installed: false},
	}
	scores := map[string]FitScore{"a": {Score: 99}}
	_, _, ok := BestInstalled(candidates, scores, nil)
	assert.False(t, ok)
}

func TestBestInstalled_SkipsUnscored(t *testing.T) {
	candidates := []ScoredModelCandidate{
		{Alias: "a", Backend: BackendFile, Installed: true},
	}
	_, _, ok := BestInstalled(candidates, map[string]FitScore{}, nil)
	assert.False(t, ok)
}

func TestBestInstalled_FiltersByAllowedBackend(t *testing.T) {
	candidates := []ScoredModelCandidate{
		{Alias: "a", Backend: "ollama", Installed: true},
		{Alias: "b", Backend: BackendFile, Installed: true},
	}
	scores := map[string]FitScore{
		"a": {Score: 99},
		"b": {Score: 10},
	}
	alias, backend, ok := BestInstalled(candidates, scores, []string{BackendFile})
	require.True(t, ok)
	assert.Equal(t, "b", alias)
	assert.Equal(t, BackendFile, backend)
}

func TestBestInstalled_NoCandidates(t *testing.T) {
	_, _, ok := BestInstalled(nil, map[string]FitScore{}, nil)
	assert.False(t, ok)
}

// --- RunLlamaFit ---------------------------------------------------------

// fakeLlamaFitBinary writes a fake `llmfit` shim that prints fixture and
// prepends its directory to PATH -- prepend, not replace, so the shim's own
// `cat` (used to emit the fixture) still resolves via the rest of PATH.
func fakeLlamaFitBinary(t *testing.T, fixture string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake llmfit shim is a #!/bin/sh heredoc script; not runnable on Windows")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\ncat <<'EOF'\n" + fixture + "\nEOF\n"
	fake := filepath.Join(dir, "llmfit")
	require.NoError(t, os.WriteFile(fake, []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestRunLlamaFit_MissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	got := RunLlamaFit(context.Background(), []ScoredModelCandidate{{Alias: "a", MatchNames: []string{"x"}}})
	assert.Empty(t, got)
}

func TestRunLlamaFit_MalformedJSON(t *testing.T) {
	fakeLlamaFitBinary(t, "not json")
	got := RunLlamaFit(context.Background(), []ScoredModelCandidate{{Alias: "a", MatchNames: []string{"x"}}})
	assert.Empty(t, got)
}

func TestRunLlamaFit_MatchesAndScores(t *testing.T) {
	fixture := `{"models":[
		{"name":"alpindale/Llama-3.2-1B-Instruct","score":78.5,"fit_level":"Good","gguf_sources":[]},
		{"name":"Qwen/Qwen2.5-1.5B-Instruct","score":66.1,"fit_level":"Too Tight",
		 "gguf_sources":[{"repo":"bartowski/Qwen2.5-1.5B-Instruct-GGUF"}]},
		{"name":"","score":10,"fit_level":"Marginal","gguf_sources":[{"repo":""}]}
	]}`
	fakeLlamaFitBinary(t, fixture)

	candidates := []ScoredModelCandidate{
		{Alias: "llama3.2:1b", MatchNames: []string{"alpindale/Llama-3.2-1B-Instruct"}},
		{Alias: "qwen2.5:1.5b", MatchNames: []string{"bartowski/Qwen2.5-1.5B-Instruct-GGUF"}},
		{Alias: "unmatched", MatchNames: []string{"nonexistent/model"}},
	}
	got := RunLlamaFit(context.Background(), candidates)
	require.Contains(t, got, "llama3.2:1b")
	assert.InDelta(t, 78.5, got["llama3.2:1b"].Score, 0.001)
	assert.Equal(t, "Good", got["llama3.2:1b"].FitLevel)
	require.Contains(t, got, "qwen2.5:1.5b")
	assert.InDelta(t, 66.1, got["qwen2.5:1.5b"].Score, 0.001)
	assert.NotContains(t, got, "unmatched")
}

// --- BestInstalledModelByFit ----------------------------------------------

func TestBestInstalledModelByFit_NoCandidates(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	alias, backend, ok := BestInstalledModelByFit(context.Background(), afero.NewMemMapFs(), nil)
	assert.False(t, ok)
	assert.Empty(t, alias)
	assert.Empty(t, backend)
}

func TestBestInstalledModelByFit_InstalledButUnscored(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())
	modelsDir, err := DefaultModelsDir()
	require.NoError(t, err)
	entries := ListLlamafileMappings()
	require.NotEmpty(t, entries)
	var path string
	for _, e := range entries {
		if p, ok := LlamafileCachedPath(e.Alias, modelsDir); ok {
			path = p
			break
		}
	}
	require.NotEmpty(t, path)

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, path, []byte("fake"), 0o600))
	// No llmfit on PATH -> RunLlamaFit returns no scores -> nothing to pick.
	t.Setenv("PATH", t.TempDir())

	_, _, ok := BestInstalledModelByFit(context.Background(), fs, nil)
	assert.False(t, ok)
}

// --- cachedLlamaFitIndex / RunLlamaFitCached ------------------------------

func TestCachedLlamaFitIndex_MissingBinary(t *testing.T) {
	resetLlamaFitIndexCache(t)
	t.Setenv("PATH", t.TempDir())

	_, _, _, ok := cachedLlamaFitIndex(context.Background())
	assert.False(t, ok)
}

func TestRunLlamaFitCached_MemoizesAcrossCalls(t *testing.T) {
	resetLlamaFitIndexCache(t)
	if runtime.GOOS == "windows" {
		t.Skip("fake llmfit shim is a #!/bin/sh heredoc script; not runnable on Windows")
	}

	dir := t.TempDir()
	counterFile := filepath.Join(dir, "calls")
	fixture := `{"models":[{"name":"alpindale/Llama-3.2-1B-Instruct","score":78.5,"fit_level":"Good","gguf_sources":[]}]}`
	script := "#!/bin/sh\necho x >> " + counterFile + "\ncat <<'EOF'\n" + fixture + "\nEOF\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "llmfit"), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	candidates := []ScoredModelCandidate{
		{Alias: "llama3.2:1b", MatchNames: []string{"alpindale/Llama-3.2-1B-Instruct"}},
	}
	got1 := RunLlamaFitCached(context.Background(), candidates)
	got2 := RunLlamaFitCached(context.Background(), candidates)
	assert.Equal(t, got1, got2)
	require.Contains(t, got1, "llama3.2:1b")
	assert.InDelta(t, 78.5, got1["llama3.2:1b"].Score, 0.001)

	data, err := os.ReadFile(counterFile)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(data), "x"), "llmfit must only be invoked once, cached thereafter")
}

// --- ScoreModelEntries -----------------------------------------------------

func TestScoreModelEntries_NoLocalBackendEntries(t *testing.T) {
	resetLlamaFitIndexCache(t)
	entries := []kdepsconfig.ModelEntry{
		{Model: "gpt-4o", Backend: "openai"},
		{Model: "claude-3", Backend: "anthropic"},
	}
	best, ok := ScoreModelEntries(context.Background(), entries)
	assert.False(t, ok)
	assert.Nil(t, best)
}

func TestScoreModelEntries_EmptyEntries(t *testing.T) {
	resetLlamaFitIndexCache(t)
	best, ok := ScoreModelEntries(context.Background(), nil)
	assert.False(t, ok)
	assert.Nil(t, best)
}

func TestScoreModelEntries_LlmfitUnavailable(t *testing.T) {
	resetLlamaFitIndexCache(t)
	t.Setenv("PATH", t.TempDir())

	entries := []kdepsconfig.ModelEntry{{Model: "llama3.2:1b", Backend: BackendFile}}
	best, ok := ScoreModelEntries(context.Background(), entries)
	assert.False(t, ok)
	assert.Nil(t, best)
}

func TestScoreModelEntries_PicksBestAmongLocalOnly(t *testing.T) {
	resetLlamaFitIndexCache(t)
	fixture := `{"models":[
		{"name":"alpindale/Llama-3.2-1B-Instruct","score":40,"fit_level":"Marginal","gguf_sources":[]},
		{"name":"Qwen/Qwen2.5-1.5B-Instruct","score":90,"fit_level":"Good",
		 "gguf_sources":[{"repo":"bartowski/Qwen2.5-1.5B-Instruct-GGUF"}]}
	]}`
	fakeLlamaFitBinary(t, fixture)

	entries := []kdepsconfig.ModelEntry{
		{Model: "alpindale/Llama-3.2-1B-Instruct", Backend: BackendFile},
		{Model: "bartowski/Qwen2.5-1.5B-Instruct-GGUF", Backend: BackendGGUF},
		{Model: "gpt-4o", Backend: "openai"}, // never eligible, no local backend
	}
	best, ok := ScoreModelEntries(context.Background(), entries)
	require.True(t, ok)
	assert.Equal(t, "bartowski/Qwen2.5-1.5B-Instruct-GGUF", best.Model)
}

func TestScoreModelEntries_UnmatchedEntriesExcluded(t *testing.T) {
	resetLlamaFitIndexCache(t)
	fixture := `{"models":[]}`
	fakeLlamaFitBinary(t, fixture)

	entries := []kdepsconfig.ModelEntry{{Model: "unknown-model", Backend: BackendFile}}
	best, ok := ScoreModelEntries(context.Background(), entries)
	assert.False(t, ok)
	assert.Nil(t, best)
}

// --- AutoRouterPick --------------------------------------------------------

// clearCloudProviderEnvVars blanks every known cloud provider's API key env
// var, so cloud-fallback tests are deterministic regardless of what's set in
// the environment actually running the test suite (e.g. a developer's shell).
func clearCloudProviderEnvVars(t *testing.T) {
	t.Helper()
	for _, p := range kdepsconfig.CloudLLMProviders() {
		t.Setenv(p.EnvVar, "")
	}
}

func TestAutoRouterPick_NothingAvailable(t *testing.T) {
	resetLlamaFitIndexCache(t)
	clearCloudProviderEnvVars(t)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())

	model, backend, ok := AutoRouterPick(context.Background(), afero.NewMemMapFs())
	assert.False(t, ok)
	assert.Empty(t, model)
	assert.Empty(t, backend)
}

func TestAutoRouterPick_CloudFallbackWhenNoLlmfit(t *testing.T) {
	resetLlamaFitIndexCache(t)
	clearCloudProviderEnvVars(t)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "sk-test")

	model, backend, ok := AutoRouterPick(context.Background(), afero.NewMemMapFs())
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", model)
	assert.Equal(t, "openai", backend)
}

func TestAutoRouterPick_CloudFallbackSkipsProvidersWithNoDefaultModel(t *testing.T) {
	resetLlamaFitIndexCache(t)
	clearCloudProviderEnvVars(t)
	t.Setenv("PATH", t.TempDir())
	// huggingface has no DefaultModel (many models, no single canonical
	// pick); only openai (later in the list) should ever be reachable.
	t.Setenv("HF_TOKEN", "hf-test")
	t.Setenv("OPENAI_API_KEY", "sk-test")

	model, backend, ok := AutoRouterPick(context.Background(), afero.NewMemMapFs())
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", model)
	assert.Equal(t, "openai", backend)
}

func TestAutoRouterPick_LocalWinsOverCloud(t *testing.T) {
	resetLlamaFitIndexCache(t)
	clearCloudProviderEnvVars(t)
	t.Setenv("OPENAI_API_KEY", "sk-test") // present but must lose to a local match
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())

	modelsDir, err := DefaultModelsDir()
	require.NoError(t, err)
	entries := ListLlamafileMappings()
	require.NotEmpty(t, entries)
	var alias, repo, path string
	for _, e := range entries {
		if p, ok := LlamafileCachedPath(e.Alias, modelsDir); ok && e.Repo != "" {
			alias, repo, path = e.Alias, e.Repo, p
			break
		}
	}
	require.NotEmpty(t, alias)
	require.NotEmpty(t, repo)

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, path, []byte("fake"), 0o600))
	fakeLlamaFitBinary(t, `{"models":[{"name":"x","score":90,"fit_level":"Good",`+
		`"gguf_sources":[{"repo":"`+repo+`"}]}]}`)

	model, backend, ok := AutoRouterPick(context.Background(), fs)
	require.True(t, ok)
	assert.Equal(t, alias, model)
	assert.Equal(t, BackendFile, backend)
}

func TestAutoRouterPick_LlmfitPresentButNothingScoresFallsToCloud(t *testing.T) {
	resetLlamaFitIndexCache(t)
	clearCloudProviderEnvVars(t)
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "sk-test")
	fakeLlamaFitBinary(t, `{"models":[]}`)

	model, backend, ok := AutoRouterPick(context.Background(), afero.NewMemMapFs())
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", model)
	assert.Equal(t, "openai", backend)
}
