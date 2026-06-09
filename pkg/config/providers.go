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

package config

// LLMProvider describes a supported cloud LLM provider.
type LLMProvider struct {
	Name    string // backend name, e.g. "openai"
	YAMLKey string // config yaml field, e.g. "openai_api_key"
	EnvVar  string // environment variable, e.g. "OPENAI_API_KEY"
}

// CloudLLMProviders returns supported cloud LLM providers in registry order.
func CloudLLMProviders() []LLMProvider {
	providers := make([]LLMProvider, len(cloudProvidersList))
	for i, p := range cloudProvidersList {
		providers[i] = LLMProvider{Name: p.name, YAMLKey: p.yamlKey, EnvVar: p.envVar}
	}
	return providers
}

// AllLLMProviderNames returns bootstrap menu provider names (ollama first, then cloud).
func AllLLMProviderNames() []string {
	names := make([]string, len(allProviderNames))
	copy(names, allProviderNames)
	return names
}
