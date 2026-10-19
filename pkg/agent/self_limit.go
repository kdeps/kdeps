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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package agent

import "strings"

// selfLimitMarkers are fictional capability claims. A text-only reply that
// contains one is a product bug: the tools are registered, and saying
// otherwise invents a limit between the user's prompt and the outcome.
//
// Kept separate from refusalMarkers. That list also settles a goal task, and
// widening it would record a task failed while this nudge is still pushing.
//
//nolint:gochecknoglobals // static lookup table
var selfLimitMarkers = []string{
	"i can't", "i cannot", "i can not",
	"i'm unable", "i am unable", "i'm not able", "i am not able",
	"not able to", "unable to", "cannot complete", "can't complete",
	"i don't have", "i do not have", "i dont have",
	"no access", "don't have access", "do not have access",
	"not present in the tool", "not in the tool list", "no tools",
	"can't emit", "cannot emit", "can't execute", "cannot execute",
	"can't perform", "cannot perform", "can't run", "cannot run",
	"can't access", "cannot access", "can't use", "cannot use",
	"beyond my capability", "beyond my capabilities", "outside my capability",
	"not capable", "i'm just a language", "i am just a language",
	"as an ai", "as a language model",
	"i have no way", "don't have a way", "do not have a way",
	"no way for me", "no way to",
	"won't be able", "will not be able", "impossible for me",
	// Paraphrases that dodge the phrases above: handing the work back,
	// claiming a tool or shell is missing, or stopping on a negative vibe.
	"not something i can", "isn't possible", "is not possible", "not possible",
	"i lack", "lack the ability", "lack permission",
	"i am limited", "i'm limited",
	"don't think i can", "do not think i can",
	"command yourself", "do it yourself", "run it yourself",
	"do this manually", "on your machine",
	"no filesystem", "has no shell",
	"restricted from", "not permitted", "outside what i",
	"wasn't built", "was not built", "not built to", "not designed",
	"not sure the tools", "tool use isn't", "tools aren't",
	"i don't support", "i do not support", "out of scope",
	"i'm only able", "i am only able",
	"for you to paste", "best i can do", "no can do",
	"part of my setup", "without tools", "nothing to execute",
	"hands are tied", "instead of me", "exceeds what",
	"don't see a way", "do not see a way",
	"isn't a tool", "is not a tool", "include a terminal",
	"is on you", "have to stop", "i must decline", "shouldn't touch",
	"sandbox i have",
}

// safetyStopMarkers are a real refusal of a destructive or abusive action.
// Those replies are not fictional limits and must not be pushed.
//
//nolint:gochecknoglobals // static lookup table
var safetyStopMarkers = []string{
	"rm -rf", "rm -fr",
	"force-push", "force push", "git push -f", "git push --force",
	"drop database", "drop table",
	"delete all", "wipe all", "wipe the",
	"i refuse", "refusing to",
	"i will not delete", "i won't delete",
	"i will not destroy", "i won't destroy",
	"i will not wipe", "i won't wipe",
	"i will not drop", "i won't drop",
	"i will not force", "i won't force",
	"i will not overwrite", "i won't overwrite",
	"credential", "exfiltrat", "malware",
}

// limitFillerWords are dropped before marker checks so "I don't actually
// have" and "I can't really emit" match the same phrases as the bare form.
//
//nolint:gochecknoglobals // static lookup table
var limitFillerWords = []string{"actually", "really", "currently", "simply"}

// realFailureAdmissionWords mark a reply that reports a tool call that
// already failed. "can't" and "cannot" are not here: those are the fictional
// limits the nudge pushes back on, even after a failed call.
//
//nolint:gochecknoglobals // static lookup table
var realFailureAdmissionWords = []string{
	"fail", "error", "did not", "didn't", "wasn't able", "was not able",
}

// admitsToolFailure reports whether the reply states that a step failed,
// rather than inventing a limit on what the agent can do.
func admitsToolFailure(content string) bool {
	return containsAny(strings.ToLower(content), realFailureAdmissionWords...)
}

// providerThrottlePhrases are the service's own rate-limit text. "unable to"
// is a self-limit marker, and that sentence contains it. Pushing back on a
// throttle stacks another invoke example and makes the next call, which is
// how one turn reached dozens of requests.
//
//nolint:gochecknoglobals // static lookup table
var providerThrottlePhrases = []string{
	"temporarily unable to respond",
	"volume of requests",
	"too many requests",
	"rate limit",
}

// looksLikeSelfLimit reports whether a text-only reply invents a limit on
// what the agent can do. A genuine safety stop, or the provider's own
// throttle text, returns false.
func looksLikeSelfLimit(content string) bool {
	folded := foldLimitText(content)
	if folded == "" || containsAny(folded, safetyStopMarkers...) ||
		containsAny(folded, providerThrottlePhrases...) {
		return false
	}
	return containsAny(folded, selfLimitMarkers...)
}

// foldLimitText lowercases, straightens apostrophes, drops filler adverbs,
// and collapses whitespace so marker checks see one stable form.
func foldLimitText(s string) string {
	s = strings.NewReplacer(
		"\u2019", "'",
		"\u2018", "'",
		"\u201c", `"`,
		"\u201d", `"`,
	).Replace(s)
	fields := strings.Fields(strings.ToLower(s))
	if len(fields) == 0 {
		return ""
	}
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if !isLimitFiller(f) {
			out = append(out, f)
		}
	}
	return strings.Join(out, " ")
}

func isLimitFiller(word string) bool {
	for _, f := range limitFillerWords {
		if word == f {
			return true
		}
	}
	return false
}
