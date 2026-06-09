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
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func applyLLMRouter(logger *slog.Logger, cfg *domain.ChatConfig, promptStr string) []kdepsconfig.ModelEntry {
	routerJSON := os.Getenv("KDEPS_LLM_ROUTER")
	if routerJSON == "" {
		return nil
	}
	var uc kdepsconfig.UnifiedModelsConfig
	if err := json.Unmarshal([]byte(routerJSON), &uc); err != nil {
		return nil
	}
	if uc.Strategy == "" || len(uc.Models) == 0 {
		return nil
	}
	if uc.Strategy == "fallback" {
		entries := SortedFallbackRoutes(uc.Models)
		if len(entries) > 0 {
			applyRoute(cfg, &entries[0])
		}
		return entries
	}
	if entry, err := NewRouter(uc.Strategy, uc.Models, logger).Select("", promptStr); err == nil && entry != nil {
		applyRoute(cfg, entry)
	}
	return nil
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
