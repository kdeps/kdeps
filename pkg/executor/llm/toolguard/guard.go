// Package toolguard holds detection heuristics and pacing primitives for LLM
// tool-calling loops that are useful across every backend, not just one. It has
// no dependency on any specific backend or transport, so both the generic
// executor (pkg/executor/llm) and transport-specific packages (e.g.
// pkg/executor/llm/m365) can share it without import cycles.
package toolguard

import "regexp"

const (
	minConfabTextLen = 12
	minHallucTextLen = 8
)

//nolint:gochecknoglobals // compiled pattern set, read-only
var confabulationPatterns = compileAll([]string{
	`(?i)return(?:ing|s|ed)?\s+no\s+(?:output|results?|content)`,
	`(?i)no\s+(?:output|results?|content|data)\s+(?:was\s+|were\s+)?(?:return|provid|present)`,
	`(?i)(?:unable|not able|can.?t|cannot)\s+(?:to\s+)?(?:access|inspect|list|read|run|execute|retrieve|fetch|locate|see|open)`,
	`(?i)don.?t\s+have\s+access`,
	`(?i)no\s+(?:longer\s+have|access\s+to)`,
	`(?i)lost\s+(?:access|my\s+access|the\s+ability)`,
	`(?i)(?:in|use|switch\s+to|need)\s+(?:a\s+)?(?:different|another|proper|coding-?enabled|tool-?enabled|shell-?enabled)\s+(?:session|environment|conversation|mode)`,
	`(?i)(?:can.?t|cannot|not\s+able\s+to|unable\s+to)\s+(?:directly\s+)?(?:edit|modify|write\s+to|change|save|create|open)\s+(?:the\s+|any\s+|to\s+)?files?`,
	`(?i)paste\s+(?:the\s+)?(?:contents?|files?|code|them)`,
	`(?i)provide\s+(?:the\s+)?(?:contents?|files?)`,
	`(?i)no\s+files?\s+(?:in|found|present|visible)`,
	`(?i)(?:file|directory|folder|it)\s+(?:appears?|seems?|looks?)\s+(?:to\s+be\s+)?empty`,
	`(?i)nothing\s+to\s+(?:simplify|fix|do|change|show|read)`,
	`(?i)(?:tool|command|it)\s+returned\s+(?:no|empty|nothing)`,
})

//nolint:gochecknoglobals // compiled pattern set, read-only
var hallucinatedCompletionPatterns = compileAll([]string{
	`(?i)\bI(?:'ve|\s+have|\s+just|\s+now)?\s+(?:created|wrote|written|replaced|updated|saved|applied|added|overwrote|modified|generated|implemented|rewrote)\b`,
	`(?i)\b(?:the\s+)?(?:file|readme|script|config|change|version|content)\s+(?:has|have|is|was|were)\s+(?:been\s+)?(?:created|replaced|updated|saved|written|applied|added|modified|overwritten)\b`,
	`(?i)\bhere'?s\s+(?:the\s+)?(?:updated|new|simplified|replaced|final)\s+(?:file|readme|version|content)\b`,
	`(?i)\b(?:executed|ran|invoked|launched|compiled)\b[^.\n]{0,40}\b(?:it|them|this|the\s+(?:script|program|file|code|command|tests?)|python3?|node)\b`,
})

func compileAll(patterns []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		out = append(out, regexp.MustCompile(p))
	}
	return out
}

// LooksLikeConfabulation reports whether a no-tool-call reply looks like the
// model claiming it cannot act (rather than a genuine final answer). Callers
// typically use this to decide whether to force one more turn instead of
// accepting the reply as the final answer.
func LooksLikeConfabulation(text string) bool {
	t := trimSpace(text)
	if len(t) < minConfabTextLen {
		return false
	}
	for _, re := range confabulationPatterns {
		if re.MatchString(t) {
			return true
		}
	}
	return false
}

// LooksLikeHallucinatedCompletion reports whether a no-tool-call reply claims a
// file mutation it may not have performed. Callers should act on this only when
// no tool actually ran in the conversation so far, keeping false positives low.
func LooksLikeHallucinatedCompletion(text string) bool {
	t := trimSpace(text)
	if len(t) < minHallucTextLen {
		return false
	}
	for _, re := range hallucinatedCompletionPatterns {
		if re.MatchString(t) {
			return true
		}
	}
	return false
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
