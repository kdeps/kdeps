package llm

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// runLlamaFitTimeout bounds the llmfit subprocess. It normally returns in
// <1s; this is a backstop against a slow or hung first invocation (e.g.
// building its own hardware/model database) blocking startup indefinitely.
const runLlamaFitTimeout = 10 * time.Second

// ScoredModelCandidate is one model llmfit-scoring can be attempted against:
// a bare alias plus every string worth matching against llmfit's own model
// names (HF repo, filename, description), and whether it's already
// installed (downloaded/pulled) on this machine.
type ScoredModelCandidate struct {
	Alias      string
	Backend    string
	Installed  bool
	MatchNames []string // repo, filename, description -- tried in this order
}

// FitScore is one model's llmfit hardware-fit result.
type FitScore struct {
	Score    float64
	FitLevel string
}

// ListInstalledLocalCandidates enumerates only already-downloaded/pulled
// local models: cached llamafiles, loadable GGUF files, and Ollama's pulled
// tags. Cloud models are excluded -- llmfit scores hardware fit, which is
// meaningless for a remote API model. Returns nil (not an error) when the
// models directory can't be resolved.
func ListInstalledLocalCandidates(fs afero.Fs) []ScoredModelCandidate {
	modelsDir, err := DefaultModelsDir()
	if err != nil {
		return nil
	}
	var out []ScoredModelCandidate
	for _, e := range ListLlamafileMappings() {
		if _, ok := resolveCachedPath(fs, LlamafileCachedPath, e.Alias, modelsDir); !ok {
			continue
		}
		out = append(out, ScoredModelCandidate{
			Alias: e.Alias, Backend: BackendFile, Installed: true,
			MatchNames: []string{e.Repo, e.Filename, e.Description},
		})
	}
	for _, e := range ListGGUFMappings() {
		path, ok := resolveCachedPath(fs, GGUFCachedPath, e.Alias, modelsDir)
		if !ok || !GGUFLoadable(fs, path) {
			continue
		}
		out = append(out, ScoredModelCandidate{
			Alias: e.Alias, Backend: BackendGGUF, Installed: true,
			MatchNames: []string{e.Repo, e.Filename, e.Description},
		})
	}
	out = append(out, listInstalledOllamaCandidates()...)
	return out
}

// resolveCachedPath resolves cachedPath(alias, modelsDir) and, if it names a
// path that exists on fs, returns it. Shared by the llamafile and GGUF
// branches of ListInstalledLocalCandidates so "not cached" has one
// implementation instead of two parallel copies.
func resolveCachedPath(
	fs afero.Fs,
	cachedPath func(alias, modelsDir string) (string, bool),
	alias, modelsDir string,
) (string, bool) {
	path, ok := cachedPath(alias, modelsDir)
	if !ok {
		return "", false
	}
	if _, err := fs.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

// RunLlamaFit shells out to `llmfit fit --json` and matches each candidate
// against the results by exact repo, then normalized model name, then a
// looser normalized name (role suffixes like "instruct"/"chat" stripped).
// Returns an empty, non-nil map -- never an error -- when llmfit is
// missing, times out, or its output doesn't parse: callers always fall back
// to their own existing behavior when a candidate has no score.
func RunLlamaFit(ctx context.Context, candidates []ScoredModelCandidate) map[string]FitScore {
	scores := make(map[string]FitScore)
	repoMap, nameMap, looseMap, ok := fetchLlamaFitIndex(ctx)
	if !ok {
		return scores
	}
	for _, c := range candidates {
		if entry, matched := matchLlamaFitScore(c.MatchNames, repoMap, nameMap, looseMap); matched {
			scores[c.Alias] = entry
		}
	}
	return scores
}

// BestInstalled picks the highest-scored Installed candidate from
// candidates, restricted to allowedBackends when non-nil. Ties break by
// alias for deterministic output. ok=false when nothing qualifies.
func BestInstalled(
	candidates []ScoredModelCandidate,
	scores map[string]FitScore,
	allowedBackends []string,
) (string, string, bool) {
	allowed := func(b string) bool {
		return allowedBackends == nil || slices.Contains(allowedBackends, b)
	}
	var bestAlias, bestBackend string
	var bestScore float64
	found := false
	for _, c := range candidates {
		if !c.Installed || !allowed(c.Backend) {
			continue
		}
		score, scored := scores[c.Alias]
		if !scored {
			continue
		}
		if !found || score.Score > bestScore || (score.Score == bestScore && c.Alias < bestAlias) {
			bestAlias, bestBackend, bestScore = c.Alias, c.Backend, score.Score
			found = true
		}
	}
	return bestAlias, bestBackend, found
}

// BestInstalledModelByFit is the convenience path for callers (like the
// workflow-mode model resolver) that just want "the best-fit model already
// installed on this machine," without needing the full scored candidate
// list back. ok=false whenever there's nothing to recommend: no installed
// candidates, llmfit unavailable, or none of them matched a score.
func BestInstalledModelByFit(
	ctx context.Context,
	fs afero.Fs,
	allowedBackends []string,
) (string, string, bool) {
	candidates := ListInstalledLocalCandidates(fs)
	if len(candidates) == 0 {
		return "", "", false
	}
	scores := RunLlamaFit(ctx, candidates)
	return BestInstalled(candidates, scores, allowedBackends)
}

// fetchLlamaFitIndex runs llmfit and indexes its results by exact GGUF
// source repo, normalized base name, and a looser key (instruct/chat/it
// stripped). Most llmfit entries have no gguf_sources, and kdeps repos are
// quantizer/packaging repos while llmfit names are base model ids -- so
// multi-key matching is required. ok=false on any failure (missing binary,
// timeout, unparseable output).
func fetchLlamaFitIndex(
	ctx context.Context,
) (map[string]FitScore, map[string]FitScore, map[string]FitScore, bool) {
	ctx, cancel := context.WithTimeout(ctx, runLlamaFitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "llmfit", "fit", "--json")
	out, execErr := cmd.Output()
	if execErr != nil {
		return nil, nil, nil, false
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
		return nil, nil, nil, false
	}
	repoMap := make(map[string]FitScore)
	nameMap := make(map[string]FitScore)
	looseMap := make(map[string]FitScore)
	record := func(m map[string]FitScore, key string, e FitScore) {
		if key == "" {
			return
		}
		if existing, exists := m[key]; !exists || e.Score > existing.Score {
			m[key] = e
		}
	}
	for _, m := range result.Models {
		entry := FitScore{m.Score, m.FitLevel}
		record(nameMap, normalizeModelKey(m.Name), entry)
		record(looseMap, normalizeModelKeyLoose(m.Name), entry)
		for _, src := range m.GGUFSrcs {
			record(repoMap, strings.ToLower(src.Repo), entry)
			record(nameMap, normalizeModelKey(src.Repo), entry)
			record(looseMap, normalizeModelKeyLoose(src.Repo), entry)
		}
	}
	return repoMap, nameMap, looseMap, true
}

// matchLlamaFitScore tries exact repo, then normalized name, then loose name
// against the candidate strings (repo / filename / description, in order).
func matchLlamaFitScore(
	cands []string,
	repoMap, nameMap, looseMap map[string]FitScore,
) (FitScore, bool) {
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
	return FitScore{}, false
}

// trailingQuantRE matches a trailing GGUF/llamafile quant suffix on a stem,
// e.g. "-Q4_K_M", ".Q8_0", "-UD-Q2_K_XL". Applied after lowercasing.
var trailingQuantRE = regexp.MustCompile(`(?i)[._-](?:ud-)?(?:q|iq)\d+(?:_[a-z0-9]+)*$`)

// normalizeModelKey reduces a HuggingFace repo id, filename, or model name to
// a comparable key: the part after the owner, lowercased, with quantizer and
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
