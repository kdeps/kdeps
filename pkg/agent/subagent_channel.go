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

package agent

import (
	"errors"
	"strings"

	executorLLM "github.com/kdeps/kdeps/v3/pkg/executor/llm"
)

// Subagent channel: knowledge-graph extraction, prompt refinement, and
// judges. The session model stays on its own calls (the turn, compaction,
// handshake, branch summaries).

const (
	subagentCurrent = "current"
	subagentFree    = "free"
)

// ResolveSubagent is the model, backend, and base URL for one subagent call.
// spec is llm.subagent or the /subagent setting. An empty spec or "current" keeps
// curModel, curBackend, and curBase. overridden is true when spec named a
// model or the large-random pool and a model was chosen. A pin whose
// provider matches the session keeps curBase. Any other pin clears it.
func ResolveSubagent(spec, curModel, curBackend, curBase string) (string, string, string, bool) {
	model, backend, ok := selectSubagent(spec)
	if !ok {
		return curModel, curBackend, curBase, false
	}
	if backend == "" {
		backend = curBackend
	}
	base := ""
	if backend == curBackend {
		base = curBase
	}
	return model, backend, base, true
}

func normalizeSubagent(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// selectSubagent resolves a non-current spec. ok is false for current, empty,
// or a free pool with no catalog. Any other id is a pin. An empty backend
// means the session backend.
func selectSubagent(spec string) (string, string, bool) {
	switch normalizeSubagent(spec) {
	case "", subagentCurrent:
		return "", "", false
	case subagentFree:
		id := pickLargeFreeModel()
		if id == "" {
			return "", "", false
		}
		return id, openrouterBackend, true
	default:
		id, backend := resolvePinnedSubagent(spec)
		if id == "" {
			return "", "", false
		}
		return id, backend, true
	}
}

// resolvePinnedSubagent is the model id and backend for a pin. A catalog id uses
// that entry. A free-catalog id uses OpenRouter. A slash id that is not a
// file path is an OpenRouter model that is not in the catalog. Anything else
// is a name on the session backend, so the backend is empty.
func resolvePinnedSubagent(spec string) (string, string) {
	spec = strings.TrimSpace(spec)
	switch normalizeSubagent(spec) {
	case "", subagentCurrent, subagentFree:
		return "", ""
	}
	if id, backend, ok := canonicalCatalogID(spec); ok {
		return id, backend
	}
	if m, ok := executorLLM.ModelByKey(spec); ok {
		return m.ID, m.Backend
	}
	if id, ok := canonicalFreeID(spec); ok {
		return id, openrouterBackend
	}
	if localModelSpec(spec) {
		return spec, ""
	}
	if strings.Contains(spec, "/") {
		return spec, openrouterBackend
	}
	return spec, ""
}

func canonicalCatalogID(spec string) (string, string, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", "", false
	}
	for _, m := range executorLLM.KnownCloudModels {
		if m.ID != "" && strings.EqualFold(m.ID, spec) {
			return m.ID, m.Backend, true
		}
	}
	return "", "", false
}

func localModelSpec(spec string) bool {
	lower := strings.ToLower(spec)
	if strings.HasPrefix(lower, "/") || strings.HasPrefix(lower, "./") ||
		strings.HasPrefix(lower, "../") || strings.HasPrefix(lower, "~/") {
		return true
	}
	return strings.HasSuffix(lower, ".gguf") || strings.HasSuffix(lower, ".llamafile")
}

// subagentOverrides reports whether spec will leave the session model, without
// picking a random id.
func (l *Loop) subagentOverrides() bool {
	if l == nil {
		return false
	}
	switch normalizeSubagent(l.config.Subagent) {
	case "", subagentCurrent:
		return false
	case subagentFree:
		return len(preferLargeFreeModels(freeModelIDs())) > 0
	default:
		id, _ := resolvePinnedSubagent(l.config.Subagent)
		return id != ""
	}
}

// subagentChat is the model this subagent call should use.
func (l *Loop) subagentChat() (string, string, string) {
	if l == nil {
		return "", "", ""
	}
	model, backend, baseURL, _ := ResolveSubagent(l.config.Subagent, l.config.Model, l.config.Backend, l.config.BaseURL)
	return model, backend, baseURL
}

// prepareSubagentCall is subagentChat plus the guards a side call already had: an
// engine, and a served local model when the call stays on the session's
// local backend. An empty model name on a cloud backend still runs.
func (l *Loop) prepareSubagentCall() (string, string, string, bool) {
	if l == nil || l.engine == nil {
		return "", "", "", false
	}
	model, backend, baseURL := l.subagentChat()
	if !l.subagentOverrides() && localModelNotServed(l) {
		return "", "", "", false
	}
	return model, backend, baseURL, true
}

// SetSubagent stores the subagent channel. current uses the session model.
// free picks a large OpenRouter free model per call. A catalog model is
// stored as its model_id. Any other name is stored as given.
func (l *Loop) SetSubagent(v string) error {
	if l == nil {
		return errors.New("no session")
	}
	switch normalizeSubagent(v) {
	case "", subagentCurrent:
		l.config.Subagent = subagentCurrent
	case subagentFree:
		l.config.Subagent = subagentFree
	default:
		id, backend := resolvePinnedSubagent(v)
		if id == "" {
			return errors.New("empty model")
		}
		// A catalog model is stored as its model_id. Calls and /subagent resolve
		// that back to the API id.
		l.config.Subagent = internalModelID(id, backend)
	}
	l.config.SubagentSet = true
	return nil
}

func canonicalFreeID(spec string) (string, bool) {
	spec = normalizeSubagent(spec)
	if spec == "" || spec == subagentCurrent || spec == subagentFree {
		return "", false
	}
	for _, id := range freeModelIDs() {
		if strings.EqualFold(id, spec) {
			return id, true
		}
	}
	return "", false
}

// subagentSettingState is what /subagent prints. It does not pick a random model:
// free stays a mode, and a pinned id is shown as itself.
func subagentSettingState(l *Loop) map[string]any {
	raw, model, backend := "", "", ""
	if l != nil {
		raw = strings.TrimSpace(l.config.Subagent)
		model, backend = l.config.Model, l.config.Backend
	}
	switch normalizeSubagent(raw) {
	case subagentFree:
		model, backend = "", openrouterBackend
		raw = subagentFree
	case "", subagentCurrent:
		raw = subagentCurrent
	default:
		id, b := resolvePinnedSubagent(raw)
		if id != "" {
			model = id
		}
		if b != "" {
			backend = b
		}
		raw = model
	}
	return map[string]any{"value": raw, "model": model, "backend": backend}
}

// internalModelID is the catalog model_id for this API id and backend.
// A name that is not in the catalog is stored unchanged.
func internalModelID(id, backend string) string {
	for _, m := range executorLLM.KnownCloudModels {
		if strings.EqualFold(m.ID, id) && m.Backend == backend && m.ModelID != "" {
			return m.ModelID
		}
	}
	return id
}

// pickLargeFreeModel chooses one id from the larger half of the free catalog.
func pickLargeFreeModel() string {
	ids := preferLargeFreeModels(freeModelIDs())
	if len(ids) == 0 {
		return ""
	}
	n := pickFreeModelN(len(ids))
	if n < 0 {
		n = 0
	}
	return ids[n%len(ids)]
}

// localExtractUnserved reports a local backend that has no server URL yet.
// Ollama is called without one. A cloud backend is never unserved here.
func localExtractUnserved(backend, baseURL string) bool {
	if baseURL != "" || backend == "ollama" {
		return false
	}
	return backend == "" || backend == "file" || backend == "gguf"
}
