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
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

func newTestLogger() *slog.Logger {
	return slog.Default()
}

// --- token_threshold ---

func TestRouterTokenThreshold_MatchesSmallRoute(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "gpt-4o-mini", Backend: "openai", MaxTokens: intPtr(50), Default: true},
		{Model: "gpt-4o", Backend: "openai", MinTokens: intPtr(51)},
	}
	r := NewRouter("token_threshold", models, newTestLogger())
	route, err := r.Select("", "hi")
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, "gpt-4o-mini", route.Model)
}

func TestRouterTokenThreshold_MatchesLargeRoute(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "gpt-4o-mini", Backend: "openai", MaxTokens: intPtr(5)},
		{Model: "gpt-4o", Backend: "openai", MinTokens: intPtr(6), Default: false},
	}
	r := NewRouter("token_threshold", models, newTestLogger())
	longPrompt := "The quick brown fox jumps over the lazy dog and the cat sat on the mat."
	route, err := r.Select("", longPrompt)
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, "gpt-4o", route.Model)
}

func TestRouterTokenThreshold_FallsBackToDefault(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "gpt-4o-mini", Backend: "openai", MaxTokens: intPtr(1)},
		{Model: "fallback-model", Backend: "openai", Default: true},
	}
	r := NewRouter("token_threshold", models, newTestLogger())
	route, err := r.Select("", "hello world how are you doing today")
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, "fallback-model", route.Model)
}

func TestRouterTokenThreshold_NoMatch_NoDefault_ReturnsNil(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "gpt-4o-mini", Backend: "openai", MaxTokens: intPtr(1)},
	}
	r := NewRouter("token_threshold", models, newTestLogger())
	route, err := r.Select("", "hello world this is a longer prompt")
	require.NoError(t, err)
	assert.Nil(t, route)
}

// --- cost_optimized ---

func TestRouterCostOptimized_PicksCheapest(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "gpt-4o", Backend: "openai", CostPerInputToken: floatPtr(0.0025)},
		{Model: "gpt-4o-mini", Backend: "openai", CostPerInputToken: floatPtr(0.00015)},
	}
	r := NewRouter("cost_optimized", models, newTestLogger())
	route, err := r.Select("", "some prompt text")
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, "gpt-4o-mini", route.Model)
}

func TestRouterCostOptimized_NilCostTreatedAsZero(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "expensive", Backend: "openai", CostPerInputToken: floatPtr(1.0)},
		{Model: "free-local", Backend: "ollama"},
	}
	r := NewRouter("cost_optimized", models, newTestLogger())
	route, err := r.Select("", "any prompt")
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, "free-local", route.Model)
}

// --- round_robin ---

func TestRouterRoundRobin_DistributesEvenly(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "model-a", Backend: "openai"},
		{Model: "model-b", Backend: "openai"},
		{Model: "model-c", Backend: "openai"},
	}
	r := NewRouter("round_robin", models, newTestLogger())

	id := "test-rr-distributes-evenly"
	seen := map[string]int{}
	for range 9 {
		route, err := r.Select(id, "prompt")
		require.NoError(t, err)
		require.NotNil(t, route)
		seen[route.Model]++
	}
	assert.Equal(t, 3, seen["model-a"])
	assert.Equal(t, 3, seen["model-b"])
	assert.Equal(t, 3, seen["model-c"])
}

// --- fallback sorted routes ---

func TestSortedFallbackRoutes_SortsByPriority(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "c", Priority: 3},
		{Model: "a", Priority: 1},
		{Model: "b", Priority: 2},
	}
	sorted := SortedFallbackRoutes(models)
	assert.Equal(t, "a", sorted[0].Model)
	assert.Equal(t, "b", sorted[1].Model)
	assert.Equal(t, "c", sorted[2].Model)
}

func TestSortedFallbackRoutes_StableOnEqualPriority(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "first", Priority: 0},
		{Model: "second", Priority: 0},
	}
	sorted := SortedFallbackRoutes(models)
	assert.Equal(t, "first", sorted[0].Model)
	assert.Equal(t, "second", sorted[1].Model)
}

func TestSortedFallbackRoutes_DoesNotMutateOriginal(t *testing.T) {
	models := []config.ModelEntry{
		{Model: "z", Priority: 9},
		{Model: "a", Priority: 1},
	}
	original := make([]config.ModelEntry, len(models))
	copy(original, models)

	SortedFallbackRoutes(models)

	assert.Equal(t, original[0].Model, models[0].Model)
	assert.Equal(t, original[1].Model, models[1].Model)
}

// --- unknown strategy ---

func TestRouter_UnknownStrategy_ReturnsError(t *testing.T) {
	models := []config.ModelEntry{{Model: "m", Backend: "openai"}}
	r := NewRouter("bogus", models, newTestLogger())
	_, err := r.Select("", "prompt")
	assert.Error(t, err)
}

// --- empty config ---

func TestRouter_NilConfig_ReturnsNil(t *testing.T) {
	r := NewRouter("", nil, newTestLogger())
	route, err := r.Select("", "prompt")
	require.NoError(t, err)
	assert.Nil(t, route)
}

func TestRouter_EmptyRoutes_ReturnsNil(t *testing.T) {
	r := NewRouter("round_robin", nil, newTestLogger())
	route, err := r.Select("", "prompt")
	require.NoError(t, err)
	assert.Nil(t, route)
}
