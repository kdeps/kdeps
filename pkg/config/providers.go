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

import "strings"

// LLMProvider describes a supported cloud LLM provider.
type LLMProvider struct {
	Name    string // backend name, e.g. "openai"
	YAMLKey string // config yaml field, e.g. "openai_api_key"
	EnvVar  string // environment variable, e.g. "OPENAI_API_KEY"
	// DefaultModel is a representative model id for this provider (e.g.
	// "gpt-4o"), or "" for providers with no single canonical default.
	// Used only as a last-resort cloud pick when no local model is found --
	// see pkg/executor/llm.AutoRouterPick.
	DefaultModel string
	// The rest is set for OpenAI-compatible providers (the providers asset set).
	Label      string   // display name, e.g. "Groq"
	URL        string   // default API host, ${VAR}s expanded, e.g. https://api.groq.com
	ChatPath   string   // chat completions path under URL, e.g. /openai/v1/chat/completions
	Embeddings bool     // also serves OpenAI-compatible /embeddings
	Hosts      []string // URL substrings that identify the provider, e.g. groq.com
}

// OpenAICompatible reports whether the provider comes from the providers
// asset set (an OpenAI-compatible chat endpoint).
func (p LLMProvider) OpenAICompatible() bool { return p.ChatPath != "" }

// BaseURL is the OpenAI SDK base URL (URL plus ChatPath without
// /chat/completions), or "" when ChatPath is not OpenAI-shaped.
func (p LLMProvider) BaseURL() string {
	base, ok := strings.CutSuffix(p.ChatPath, "/chat/completions")
	if !ok || p.URL == "" {
		return ""
	}
	return p.URL + base
}

// CloudLLMProviders returns supported cloud LLM providers in provider order:
// the built-in ones and the providers asset set.
func CloudLLMProviders() []LLMProvider {
	list := cloudProviderList()
	providers := make([]LLMProvider, len(list))
	for i, p := range list {
		providers[i] = p.public()
	}
	return providers
}

// LookupProvider returns the cloud provider called name.
func LookupProvider(name string) (LLMProvider, bool) {
	p, ok := providerByName(name)
	return p.public(), ok
}

func (p cloudProvider) public() LLMProvider {
	u := p.url
	if u != "" {
		u = expandProviderURL(u)
	}
	return LLMProvider{
		Name: p.name, YAMLKey: p.yamlKey, EnvVar: p.envVar, DefaultModel: p.defaultModel,
		Label: p.label, URL: u, ChatPath: p.chatPath, Embeddings: p.embeddings, Hosts: p.hosts,
	}
}

// EmbeddingBaseURL is the OpenAI-compatible embeddings base URL of the
// provider called name, or "" when it does not serve embeddings.
func EmbeddingBaseURL(name string) string {
	p, ok := LookupProvider(name)
	if !ok || !p.Embeddings {
		return ""
	}
	return p.BaseURL()
}
