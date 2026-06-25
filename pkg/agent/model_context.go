// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package agent

import "strings"

// modelContextWindows maps model names to their known context window sizes in tokens.
// Sourced from provider docs, OpenRouter, and litellm/model_prices_and_context_window.
// When a model is not listed here, contextFromParams provides a parameter-based estimate.
//
//nolint:gochecknoglobals,mnd // read-only lookup table; values are well-known published token limits
var modelContextWindows = map[string]int{
	// === OpenAI ===
	"gpt-4o":                 128000,
	"gpt-4o-mini":            128000,
	"gpt-4o-2024-08-06":      128000,
	"gpt-4o-2024-05-13":      128000,
	"gpt-4-turbo":            128000,
	"gpt-4-turbo-2024-04-09": 128000,
	"gpt-4-turbo-preview":    128000,
	"gpt-4-0125-preview":     128000,
	"gpt-4-1106-preview":     128000,
	"gpt-4":                  8192,
	"gpt-4-32k":              32768,
	"gpt-4-0613":             8192,
	"gpt-3.5-turbo":          16385,
	"gpt-3.5-turbo-0125":     16385,
	"gpt-3.5-turbo-1106":     16385,
	"gpt-3.5-turbo-16k":      16385,
	"gpt-3.5-turbo-instruct": 4096,
	"o1":                     200000,
	"o1-mini":                128000,
	"o1-preview":             128000,
	"o3":                     200000,
	"o3-mini":                200000,
	"o4-mini":                200000,

	// === Anthropic ===
	"claude-3-5-sonnet-20241022": 200000,
	"claude-3-5-sonnet-20240620": 200000,
	"claude-3-5-sonnet-latest":   200000,
	"claude-3-5-haiku-20241022":  200000,
	"claude-3-5-haiku-latest":    200000,
	"claude-3-opus-20240229":     200000,
	"claude-3-opus-latest":       200000,
	"claude-3-sonnet-20240229":   200000,
	"claude-3-haiku-20240307":    200000,
	"claude-2.1":                 200000,
	"claude-2.0":                 100000,
	"claude-instant-1.2":         100000,

	// === Google ===
	"gemini-2.5-pro":        1048576,
	"gemini-2.5-flash":      1048576,
	"gemini-2.0-flash":      1048576,
	"gemini-2.0-flash-lite": 1048576,
	"gemini-2.0-pro":        2097152,
	"gemini-1.5-pro":        2097152,
	"gemini-1.5-flash":      1048576,
	"gemini-1.5-flash-8b":   1048576,
	"gemini-1.0-pro":        32768,
	"gemini-pro":            32768,
	"gemma-2-27b":           8192,
	"gemma-2-9b":            8192,

	// === DeepSeek ===
	"deepseek-reasoner": 65536,
	"deepseek-chat":     65536,
	"deepseek-coder":    65536,
	"deepseek-v3":       65536,
	"deepseek-r1":       65536,

	// === Groq ===
	"llama-3.3-70b-versatile":      131072,
	"llama-3.1-8b-instant":         131072,
	"llama-3.2-90b-vision-preview": 131072,
	"llama-3.2-11b-vision-preview": 131072,
	"llama-3.2-3b-preview":         131072,
	"llama-3.2-1b-preview":         131072,
	"mixtral-8x7b-32768":           32768,
	"gemma2-9b-it":                 8192,

	// === Mistral ===
	"mistral-large-latest":  131072,
	"mistral-large-2411":    131072,
	"mistral-medium-latest": 32768,
	"mistral-small-latest":  32768,
	"mistral-tiny":          32768,
	"codestral-latest":      32768,
	"ministral-3b-latest":   131072,
	"ministral-8b-latest":   131072,
	"pixtral-large-latest":  131072,

	// === Cohere ===
	"command-r-plus": 128000,
	"command-r":      128000,
	"command":        4096,
	"command-light":  4096,

	// === xAI / Grok ===
	"grok-2":        131072,
	"grok-2-vision": 131072,
	"grok-beta":     131072,

	// === OpenRouter / others ===
	"qwen2.5-72b-instruct":          131072,
	"qwen2.5-32b-instruct":          131072,
	"qwen2.5-14b-instruct":          131072,
	"qwen2.5-7b-instruct":           131072,
	"qwen2.5-coder-32b":             131072,
	"nvidia/llama-3.1-nemotron-70b": 131072,
	"meta-llama/llama-3.1-405b":     131072,
	"meta-llama/llama-3.1-70b":      131072,
	"meta-llama/llama-3.1-8b":       131072,
	"meta-llama/llama-3-70b":        8192,
	"meta-llama/llama-3-8b":         8192,
	"google/gemini-2.5-pro":         1048576,
	"google/gemini-2.5-flash":       1048576,

	// === Cerebras ===
	"llama3.1-8b":  8192,
	"llama3.1-70b": 8192,

	// === Perplexity ===
	"llama-3.1-sonar-small-128k": 131072,
	"llama-3.1-sonar-large-128k": 131072,
	"llama-3.1-sonar-huge-128k":  131072,
	"sonar-reasoning":            131072,
	"sonar-pro":                  200000,
}

// ContextWindowForModel returns the known context window size in tokens for a model.
// Returns 0 if the model is unknown (caller should fall back to defaults).
func ContextWindowForModel(model string) int {
	if ctx, ok := modelContextWindows[model]; ok {
		return ctx
	}
	// Check without version suffixes
	for name, ctx := range modelContextWindows {
		if strings.HasPrefix(model, name) {
			return ctx
		}
	}
	return 0
}
