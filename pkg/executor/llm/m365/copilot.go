// Package m365 implements a client for the M365 Copilot chat backend, which
// speaks SignalR over a WebSocket rather than an OpenAI-style HTTP endpoint.
//
// The package is transport-only: it converts a text prompt plus an access token
// into a streamed answer with per-turn metadata (throttle counters, classifier
// scores, content origin). Higher layers wrap it as a kdeps LLM Backend so it
// works in both workflow (DAG) and agent-loop modes.
package m365

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// modelTones maps a caller-facing model name to the server-side "tone" the chat
// request must carry. An unknown tone makes the server reject the turn, so every
// value here is one the live service has been observed to accept.
//
//nolint:gochecknoglobals // static lookup table, read-only after init
var modelTones = map[string]string{
	// Default: let the service pick.
	"m365-copilot": "auto", //nolint:goconst // JSON/wire literal, not worth a shared const

	// Reasoning depth toggles for the default model.
	"quick":        "Gpt_Quick",
	"think-deeper": "Gpt_Reasoning",

	// Anthropic-hosted models exposed through the service.
	"Claude_Sonnet":              "Claude_Sonnet",
	"claude-sonnet":              "Claude_Sonnet",
	"claude-sonnet-4.5":          "Claude_Sonnet",
	"claude-sonnet-think-deeper": "Claude_Sonnet_Reasoning",
	"claude-opus":                "Claude_Opus",

	// GPT-5.5.
	"gpt-5.5":              "Gpt_5_5_Chat",
	"gpt-5.5-quick":        "Gpt_5_5_Chat",
	"gpt-5.5-think-deeper": "Gpt_5_5_Reasoning",

	// GPT-5.4.
	"gpt-5.4":              "Gpt_5_4_Reasoning",
	"gpt-5.4-think-deeper": "Gpt_5_4_Reasoning",
	"gpt-5.4-quick":        "Gpt_5_4_Quick",

	// GPT-5.3.
	"gpt-5.3":              "Gpt_5_3_Quick",
	"gpt-5.3-quick":        "Gpt_5_3_Quick",
	"gpt-5.3-think-deeper": "Gpt_5_3_Reasoning",

	// GPT-5.2.
	"gpt-5.2":              "Gpt_5_2_Quick",
	"gpt-5.2-quick":        "Gpt_5_2_Quick",
	"gpt-5.2-think-deeper": "Gpt_5_2_Reasoning",
}

var claudePrefix = regexp.MustCompile(`(?i)^claude`)

// GetToneForModel resolves a model name to a server tone. Unmapped names that
// still look Claude-flavoured fall back to the Claude chat tone; anything else
// falls back to the default model's tone.
func GetToneForModel(model string) string {
	kdeps_debug.Log("enter: GetToneForModel")
	if tone, ok := modelTones[model]; ok {
		return tone
	}
	if claudePrefix.MatchString(model) {
		return "Claude_Sonnet"
	}
	return modelTones["m365-copilot"]
}

// AvailableModels returns the set of model names with an explicit tone mapping.
func AvailableModels() []string {
	kdeps_debug.Log("enter: AvailableModels")
	names := make([]string, 0, len(modelTones))
	for name := range modelTones {
		names = append(names, name)
	}
	return names
}

// DecodeJWT extracts the claims from a JWT access token without verifying its
// signature. Only the middle (payload) segment is read; the oid and tid claims
// identify the tenant/user pair that the chat WebSocket URL embeds.
func DecodeJWT(token string) (JWTClaims, error) {
	kdeps_debug.Log("enter: DecodeJWT")
	var claims JWTClaims

	const minJWTSegments = 2 // header.payload is enough to read claims
	parts := splitJWT(token)
	if len(parts) < minJWTSegments {
		return claims, fmt.Errorf("m365: malformed token: want 3 segments, got %d", len(parts))
	}

	// JWT uses base64url without padding; RawURLEncoding handles that directly.
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, fmt.Errorf("m365: decode token payload: %w", err)
	}

	if err = json.Unmarshal(payload, &claims); err != nil {
		return claims, fmt.Errorf("m365: parse token claims: %w", err)
	}
	if claims.OID == "" || claims.TID == "" {
		return claims, errors.New("m365: token missing oid/tid claims")
	}
	return claims, nil
}

// splitJWT splits a token on '.' into its dot-separated segments.
func splitJWT(token string) []string {
	var out []string
	start := 0
	for i := range len(token) {
		if token[i] == '.' {
			out = append(out, token[start:i])
			start = i + 1
		}
	}
	out = append(out, token[start:])
	return out
}
