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
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func applyLLMRouter(
	ctx context.Context, logger *slog.Logger, cfg *domain.ChatConfig, promptStr string,
) []kdepsconfig.ModelEntry {
	uc, ok := configuredModels()
	if !ok || uc.Strategy == "" || len(uc.Models) == 0 {
		return nil
	}
	if uc.Strategy == "fallback" {
		entries := SortedFallbackRoutes(uc.Models)
		if len(entries) > 0 {
			applyRoute(cfg, &entries[0])
		}
		return entries
	}
	if entry, err := NewRouter(uc.Strategy, uc.Models, logger).Select(ctx, "", promptStr); err == nil && entry != nil {
		applyRoute(cfg, entry)
	}
	return nil
}

// configuredModels parses KDEPS_LLM_ROUTER into a UnifiedModelsConfig.
// ok=false when the env var is unset or doesn't parse.
func configuredModels() (kdepsconfig.UnifiedModelsConfig, bool) {
	routerJSON := os.Getenv("KDEPS_LLM_ROUTER")
	if routerJSON == "" {
		return kdepsconfig.UnifiedModelsConfig{}, false
	}
	var uc kdepsconfig.UnifiedModelsConfig
	if err := json.Unmarshal([]byte(routerJSON), &uc); err != nil {
		return kdepsconfig.UnifiedModelsConfig{}, false
	}
	return uc, true
}

// ConfiguredModelEntries returns the model entries configured via
// KDEPS_LLM_ROUTER (i.e. llm.models in ~/.kdeps/config.yaml), regardless of
// strategy. Used as the shared candidate pool for the "auto" strategy in
// both the workflow router (this file) and agent-loop mode's --model auto
// sentinel (cmd/serve.go). Returns nil when nothing is configured.
func ConfiguredModelEntries() []kdepsconfig.ModelEntry {
	uc, ok := configuredModels()
	if !ok {
		return nil
	}
	return uc.Models
}

func applyRoute(cfg *domain.ChatConfig, r *kdepsconfig.ModelEntry) {
	if r.Model != "" {
		cfg.Model = r.Model
	}
	if r.Backend != "" {
		cfg.Backend = r.Backend
	}
	if r.BaseURL != "" {
		cfg.BaseURL = r.BaseURL
	}
}

func allowedModelsFromEnv() []string {
	v := os.Getenv("KDEPS_LLM_MODELS")
	if v == "" {
		return nil
	}
	return strings.Split(v, ",")
}

func resolveAllowedModel(model string, allowed []string) string {
	if len(allowed) == 0 {
		return model
	}
	for _, m := range allowed {
		if m == model {
			return model
		}
	}
	return allowed[0]
}
