package agent

import (
	"errors"
	"testing"
)

func TestIsContextOverflowError(t *testing.T) {
	overflow := []string{
		"prompt is too long: 213462 tokens > 200000 maximum",
		"413 {\"error\":{\"type\":\"request_too_large\",\"message\":\"Request exceeds the maximum size\"}}",
		"input is too long for requested model",
		"Your input exceeds the context window of this model",
		"Requested token count exceeds the model's maximum context length of 131072 tokens",
		"Input length (265330) exceeds model's maximum context length (262144).",
		"The input token count (1196265) exceeds the maximum number of tokens allowed (1048575)",
		"This model's maximum prompt length is 131072 but the request contains 537812 tokens",
		"Please reduce the length of the messages or completion",
		"This endpoint's maximum context length is 100000 tokens. However, you requested about 200000 tokens",
		"Input length 200000 exceeds the maximum allowed input length of 131072 tokens.",
		"The input (265330 tokens) is longer than the model's context length (131072 tokens).",
		"prompt token count of 200000 exceeds the limit of 100000",
		"the request exceeds the available context size, try increasing it",
		"tokens to keep from the initial prompt is greater than the context length",
		"invalid params, context window exceeds limit",
		"Your request exceeded model token limit: 200000 (requested: 300000)",
		"400 status code (no body)",
		"413 (no body)",
		"model_context_window_exceeded",
		"prompt too long; exceeded max context length by 5000 tokens",
		"context_length_exceeded",
		"too many tokens in your prompt",
		"token limit exceeded, please shorten your input",
		"too large for model with 128000 maximum context length",
		// Wrapped errors with overflow in second line
		"outer error\nprompt is too long: 100000 tokens > 50000 maximum",
	}

	for _, msg := range overflow {
		t.Run(msg[:min(len(msg), 60)], func(t *testing.T) {
			if !IsContextOverflowError(errors.New(msg)) {
				t.Errorf("expected overflow detection for: %q", msg)
			}
		})
	}

	nonOverflow := []string{
		"Throttling error: Too many tokens, please wait before trying again.",
		"Service unavailable: Too many tokens queued",
		"rate limit exceeded, please retry after 60 seconds",
		"too many requests, slow down",
		"connection timeout",
		"invalid API key",
		"model not found",
		"",
	}

	for _, msg := range nonOverflow {
		t.Run("non_overflow_"+msg[:min(len(msg), 40)], func(t *testing.T) {
			if msg == "" {
				if IsContextOverflowError(nil) {
					t.Error("expected false for nil error")
				}
				return
			}
			if IsContextOverflowError(errors.New(msg)) {
				t.Errorf("false positive overflow detection for: %q", msg)
			}
		})
	}
}
