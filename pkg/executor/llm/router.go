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

package llm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

// routerCounters stores per-router round-robin counters keyed by route fingerprint.
var routerCounters sync.Map //nolint:gochecknoglobals // intentional package-level state for round-robin

const tokensPerK = 1000.0 // tokens per pricing unit used in cost calculations

// Router selects a ModelEntry from a list of models for a given prompt.
type Router struct {
	strategy string
	models   []config.ModelEntry
	logger   *slog.Logger
}

// NewRouter creates a Router for the given strategy and model entries.
func NewRouter(strategy string, models []config.ModelEntry, logger *slog.Logger) *Router {
	return &Router{strategy: strategy, models: models, logger: logger}
}

// Select returns the ModelEntry to use for promptText.
// routerID is used as the round-robin counter key (pass the resource actionID).
// Returns (nil, nil) when no route matches and no default is configured — callers
// treat nil as "use the existing model selection unchanged".
func (r *Router) Select(routerID, promptText string) (*config.ModelEntry, error) {
	if len(r.models) == 0 {
		return nil, nil //nolint:nilnil // nil route = no match, caller falls through to default behaviour
	}
	switch r.strategy {
	case "token_threshold":
		return r.selectTokenThreshold(promptText)
	case "cost_optimized":
		return r.selectCostOptimized(promptText)
	case "round_robin":
		return r.selectRoundRobin(routerID)
	default:
		return nil, fmt.Errorf("unknown router strategy: %q", r.strategy)
	}
}

// SortedFallbackRoutes returns models sorted by priority ascending (lower = tried first).
// Models with equal priority preserve their original order.
func SortedFallbackRoutes(models []config.ModelEntry) []config.ModelEntry {
	sorted := make([]config.ModelEntry, len(models))
	copy(sorted, models)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}

// selectTokenThreshold routes by estimated prompt token count.
// Matches the first model where minTokens <= tokens <= maxTokens (nil bound = open).
// Falls through to the first model with default:true if no range matches.
func (r *Router) selectTokenThreshold(promptText string) (*config.ModelEntry, error) {
	model := ""
	if len(r.models) > 0 {
		model = r.models[0].Model
	}
	tokens := CountTokens(model, promptText)

	for i := range r.models {
		entry := &r.models[i]
		minOK := entry.MinTokens == nil || tokens >= *entry.MinTokens
		maxOK := entry.MaxTokens == nil || tokens <= *entry.MaxTokens
		if minOK && maxOK {
			return entry, nil
		}
	}
	return r.defaultEntry(), nil
}

// selectCostOptimized picks the entry with the lowest estimated input cost.
// Entries with nil CostPerInputToken are treated as zero cost.
// Falls through to the first entry with default:true on tie or missing costs.
func (r *Router) selectCostOptimized(promptText string) (*config.ModelEntry, error) {
	model := ""
	if len(r.models) > 0 {
		model = r.models[0].Model
	}
	tokens := CountTokens(model, promptText)

	var cheapest *config.ModelEntry
	minCost := -1.0

	for i := range r.models {
		entry := &r.models[i]
		cost := 0.0
		if entry.CostPerInputToken != nil {
			cost = float64(tokens) * (*entry.CostPerInputToken) / tokensPerK
		}
		if cheapest == nil || cost < minCost {
			cheapest = entry
			minCost = cost
		}
	}
	if cheapest != nil {
		return cheapest, nil
	}
	return r.defaultEntry(), nil
}

// selectRoundRobin distributes requests evenly across models using an atomic counter.
// The counter is keyed by a fingerprint of the model names so that different
// router configs maintain independent counters.
func (r *Router) selectRoundRobin(routerID string) (*config.ModelEntry, error) {
	if len(r.models) == 0 {
		return nil, nil //nolint:nilnil // nil route = no routes configured, caller falls through
	}
	key := routerFingerprint(routerID, r.models)
	val, _ := routerCounters.LoadOrStore(key, new(atomic.Uint64))
	counter, ok := val.(*atomic.Uint64)
	if !ok {
		return &r.models[0], nil
	}
	idx := int(counter.Add(1)-1) % len(r.models)
	return &r.models[idx], nil
}

// defaultEntry returns the first entry marked default:true, or nil.
func (r *Router) defaultEntry() *config.ModelEntry {
	for i := range r.models {
		if r.models[i].Default {
			return &r.models[i]
		}
	}
	return nil
}

// routerFingerprint returns a stable key for the given routerID + model list.
func routerFingerprint(routerID string, models []config.ModelEntry) string {
	h := sha256.New()
	_, _ = fmt.Fprint(h, routerID)
	for _, m := range models {
		_, _ = fmt.Fprintf(h, "|%s:%s", m.Model, m.Backend)
	}
	return hex.EncodeToString(h.Sum(nil))
}
