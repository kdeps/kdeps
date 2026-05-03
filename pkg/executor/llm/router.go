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

// Router selects a RouteEntry from a RouterConfig for a given prompt.
type Router struct {
	cfg    *config.RouterConfig
	logger *slog.Logger
}

// NewRouter creates a Router for the given config.
func NewRouter(cfg *config.RouterConfig, logger *slog.Logger) *Router {
	return &Router{cfg: cfg, logger: logger}
}

// Select returns the RouteEntry to use for promptText.
// routerID is used as the round-robin counter key (pass the resource actionID).
// Returns (nil, nil) when no route matches and no default is configured — callers
// treat nil as "use the existing model selection unchanged".
func (r *Router) Select(routerID, promptText string) (*config.RouteEntry, error) {
	if r.cfg == nil || len(r.cfg.Routes) == 0 {
		return nil, nil //nolint:nilnil // nil route = no match, caller falls through to default behaviour
	}
	switch r.cfg.Strategy {
	case "token_threshold":
		return r.selectTokenThreshold(promptText)
	case "cost_optimized":
		return r.selectCostOptimized(promptText)
	case "round_robin":
		return r.selectRoundRobin(routerID)
	default:
		return nil, fmt.Errorf("unknown router strategy: %q", r.cfg.Strategy)
	}
}

// SortedFallbackRoutes returns routes sorted by priority ascending (lower = tried first).
// Routes with equal priority preserve their original order.
func SortedFallbackRoutes(routes []config.RouteEntry) []config.RouteEntry {
	sorted := make([]config.RouteEntry, len(routes))
	copy(sorted, routes)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}

// selectTokenThreshold routes by estimated prompt token count.
// Matches the first route where minTokens <= tokens <= maxTokens (nil bound = open).
// Falls through to the first route with default:true if no range matches.
func (r *Router) selectTokenThreshold(promptText string) (*config.RouteEntry, error) {
	// Use first route model as proxy for encoding (any route's model works here)
	model := ""
	if len(r.cfg.Routes) > 0 {
		model = r.cfg.Routes[0].Model
	}
	tokens := CountTokens(model, promptText)

	for i := range r.cfg.Routes {
		route := &r.cfg.Routes[i]
		minOK := route.MinTokens == nil || tokens >= *route.MinTokens
		maxOK := route.MaxTokens == nil || tokens <= *route.MaxTokens
		if minOK && maxOK {
			return route, nil
		}
	}
	return r.defaultRoute(), nil
}

// selectCostOptimized picks the route with the lowest estimated input cost.
// Routes with nil CostPerInputToken are treated as zero cost.
// Falls through to the first route with default:true on tie or missing costs.
func (r *Router) selectCostOptimized(promptText string) (*config.RouteEntry, error) {
	model := ""
	if len(r.cfg.Routes) > 0 {
		model = r.cfg.Routes[0].Model
	}
	tokens := CountTokens(model, promptText)

	var cheapest *config.RouteEntry
	minCost := -1.0

	for i := range r.cfg.Routes {
		route := &r.cfg.Routes[i]
		cost := 0.0
		if route.CostPerInputToken != nil {
			cost = float64(tokens) * (*route.CostPerInputToken) / tokensPerK
		}
		if cheapest == nil || cost < minCost {
			cheapest = route
			minCost = cost
		}
	}
	if cheapest != nil {
		return cheapest, nil
	}
	return r.defaultRoute(), nil
}

// selectRoundRobin distributes requests evenly across routes using an atomic counter.
// The counter is keyed by a fingerprint of the route model names so that different
// router configs maintain independent counters.
func (r *Router) selectRoundRobin(routerID string) (*config.RouteEntry, error) {
	if len(r.cfg.Routes) == 0 {
		return nil, nil //nolint:nilnil // nil route = no routes configured, caller falls through
	}
	key := routerFingerprint(routerID, r.cfg.Routes)
	val, _ := routerCounters.LoadOrStore(key, new(atomic.Uint64))
	counter, ok := val.(*atomic.Uint64)
	if !ok {
		return &r.cfg.Routes[0], nil
	}
	idx := int(counter.Add(1)-1) % len(r.cfg.Routes)
	return &r.cfg.Routes[idx], nil
}

// defaultRoute returns the first route marked default:true, or nil.
func (r *Router) defaultRoute() *config.RouteEntry {
	for i := range r.cfg.Routes {
		if r.cfg.Routes[i].Default {
			return &r.cfg.Routes[i]
		}
	}
	return nil
}

// routerFingerprint returns a stable key for the given routerID + route list.
func routerFingerprint(routerID string, routes []config.RouteEntry) string {
	h := sha256.New()
	_, _ = fmt.Fprint(h, routerID)
	for _, r := range routes {
		_, _ = fmt.Fprintf(h, "|%s:%s", r.Model, r.Backend)
	}
	return hex.EncodeToString(h.Sum(nil))
}
