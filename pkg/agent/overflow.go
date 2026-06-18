package agent

import (
	"regexp"
	"strings"
)

// overflowPatterns matches provider error messages that indicate context window
// overflow. Mirrors pi's OVERFLOW_PATTERNS from packages/ai/src/utils/overflow.ts.
//
//nolint:gochecknoglobals // compiled regexps are package-level by convention
var overflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(
		`(?i)prompt is too long`,
	), // Anthropic token overflow
	regexp.MustCompile(
		`(?i)request_too_large`,
	), // Anthropic request byte-size overflow (HTTP 413)
	regexp.MustCompile(
		`(?i)input is too long for requested model`,
	), // Amazon Bedrock
	regexp.MustCompile(
		`(?i)exceeds the context window`,
	), // OpenAI (Completions & Responses API)
	regexp.MustCompile(
		`(?i)exceeds (?:the )?(?:model'?s )?maximum context length(?: of [\d,]+ tokens?|\s*\([\d,]+\))`,
	), // OpenAI-compatible proxies (LiteLLM)
	regexp.MustCompile(
		`(?i)input token count.*exceeds the maximum`,
	), // Google (Gemini)
	regexp.MustCompile(
		`(?i)maximum prompt length is \d+`,
	), // xAI (Grok)
	regexp.MustCompile(
		`(?i)reduce the length of the messages`,
	), // Groq
	regexp.MustCompile(
		`(?i)maximum context length is \d+ tokens`,
	), // OpenRouter (most backends)
	regexp.MustCompile(
		`(?i)exceeds (?:the )?maximum allowed input length of [\d,]+ tokens?`,
	), // OpenRouter/Poolside
	regexp.MustCompile(
		`(?i)input \(\d+ tokens\) is longer than the model'?s context length \(\d+ tokens\)`,
	), // Together AI
	regexp.MustCompile(
		`(?i)exceeds the limit of \d+`,
	), // GitHub Copilot
	regexp.MustCompile(
		`(?i)exceeds the available context size`,
	), // llama.cpp server
	regexp.MustCompile(
		`(?i)greater than the context length`,
	), // LM Studio
	regexp.MustCompile(
		`(?i)context window exceeds limit`,
	), // MiniMax
	regexp.MustCompile(
		`(?i)exceeded model token limit`,
	), // Kimi For Coding
	regexp.MustCompile(
		`(?i)too large for model with \d+ maximum context length`,
	), // Mistral
	regexp.MustCompile(
		`(?i)model_context_window_exceeded`,
	), // z.ai non-standard finish_reason
	regexp.MustCompile(
		`(?i)prompt too long; exceeded (?:max )?context length`,
	), // Ollama explicit overflow error
	regexp.MustCompile(
		`(?i)context[_ ]length[_ ]exceeded`,
	), // Generic fallback
	regexp.MustCompile(
		`(?i)too many tokens`,
	), // Generic fallback
	regexp.MustCompile(
		`(?i)token limit exceeded`,
	), // Generic fallback
	regexp.MustCompile(
		`(?i)^4(?:00|13)\s*(?:status code)?\s*\(no body\)`,
	), // Cerebras: 400/413 with no body
}

// nonOverflowPatterns excludes messages that match overflowPatterns but are
// actually rate limits or transient server errors.
//
//nolint:gochecknoglobals // compiled regexps are package-level by convention
var nonOverflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(
		`(?i)^(Throttling error|Service unavailable):`,
	), // AWS Bedrock non-overflow errors
	regexp.MustCompile(`(?i)rate limit`),        // Generic rate limiting
	regexp.MustCompile(`(?i)too many requests`), // Generic HTTP 429
}

// IsContextOverflowError returns true if err represents a provider context
// window overflow. Matches all major providers; mirrors pi's isContextOverflow.
func IsContextOverflowError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, p := range nonOverflowPatterns {
		if p.MatchString(msg) {
			return false
		}
	}
	// Also check for newline-separated error stacks - some wrappers embed
	// the underlying provider message on a later line.
	for line := range strings.SplitSeq(msg, "\n") {
		for _, p := range overflowPatterns {
			if p.MatchString(line) {
				return true
			}
		}
	}
	return false
}
