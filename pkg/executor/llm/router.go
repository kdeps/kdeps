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

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
)

// routerCounters stores per-router round-robin counters keyed by route fingerprint.
var routerCounters sync.Map //nolint:gochecknoglobals // intentional package-level state for round-robin

const tokensPerK = 1000.0

// Router selects a ModelEntry from a list of models for a given prompt.
type Router struct {
	strategy string
	models   []kdepsconfig.ModelEntry
	logger   *slog.Logger
}

// NewRouter creates a Router for the given strategy and model entries.
func NewRouter(strategy string, models []kdepsconfig.ModelEntry, logger *slog.Logger) *Router {
	return &Router{strategy: strategy, models: models, logger: logger}
}

// Select returns the ModelEntry to use for promptText.
func (r *Router) Select(routerID, promptText string) (*kdepsconfig.ModelEntry, error) {
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
func SortedFallbackRoutes(models []kdepsconfig.ModelEntry) []kdepsconfig.ModelEntry {
	sorted := make([]kdepsconfig.ModelEntry, len(models))
	copy(sorted, models)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}

func (r *Router) selectTokenThreshold(promptText string) (*kdepsconfig.ModelEntry, error) {
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

func (r *Router) selectCostOptimized(promptText string) (*kdepsconfig.ModelEntry, error) {
	model := ""
	if len(r.models) > 0 {
		model = r.models[0].Model
	}
	tokens := CountTokens(model, promptText)

	var cheapest *kdepsconfig.ModelEntry
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

func (r *Router) selectRoundRobin(routerID string) (*kdepsconfig.ModelEntry, error) {
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

func (r *Router) defaultEntry() *kdepsconfig.ModelEntry {
	for i := range r.models {
		if r.models[i].Default {
			return &r.models[i]
		}
	}
	return nil
}

func routerFingerprint(routerID string, models []kdepsconfig.ModelEntry) string {
	h := sha256.New()
	_, _ = fmt.Fprint(h, routerID)
	for _, m := range models {
		_, _ = fmt.Fprintf(h, "|%s:%s", m.Model, m.Backend)
	}
	return hex.EncodeToString(h.Sum(nil))
}
