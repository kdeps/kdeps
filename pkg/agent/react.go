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

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/jsonutil"
)

const (
	reactFinalAnswerPrefix = "Final Answer:"

	reactSystemTemplate = `Answer the following questions as best you can. You have access to the following tools:

%s

Use the following format:

Question: the input question you must answer
Thought: you should always think about what to do
Action: the action to take, should be one of [ %s ]
Action Input: the input to the action as a JSON object
Observation: the result of the action
... (this Thought/Action/Action Input/Observation can repeat N times)
Thought: I now know the final answer
Final Answer: the final answer to the original input question

Begin!`
)

// reactActionPrefixRe locates the action name and the start of the Action
// Input value. It intentionally does NOT capture the value itself: the
// original regex captured it with a single-line (.+) group, which the
// framing prompt's "input to the action as a JSON object" doesn't forbid
// pretty-printing across multiple lines (a common model formatting habit) --
// that truncated the value to just its opening "{", which then fails
// json.Unmarshal in dispatchReactTool and silently falls back to wrapping
// the truncated "{" as a bogus {"input": "{"} argument, corrupting the real
// tool call. Same root cause as the m365 fenced/invoke parsing bugs:
// assuming a narrower shape (single-line) than what a real reply produces.
// parseReactAction below uses jsonutil.ScanBalancedObject to recover the
// full value regardless of how it's formatted.
var reactActionPrefixRe = regexp.MustCompile(`(?i)Action:\s*(.+?)\s*\nAction\s+Input:\s*`)

// reactStep is one tool-call/observation pair in the ReAct scratchpad.
type reactStep struct {
	thought     string
	action      string
	actionInput string
	observation string
}

// RunReact runs the ReAct (Thought/Action/Observation) loop for models that
// do not support native function calling. Tool descriptions are injected into
// the system prompt; the model responds in the structured text format and we
// parse tool calls from the output manually.
func (l *Loop) RunReact(ctx context.Context, input string, w io.Writer) (string, error) {
	if msgs := l.session.RawMessages(); shouldAutoCompact(msgs, l.config.AutoCompactThreshold, l.config.Model) {
		if summary, err := l.CompactWithLLM(ctx); err == nil && summary != "" {
			if l.onAutoCompact != nil {
				l.onAutoCompact(summary)
			}
		}
	}

	systemPreamble := l.buildReactSystemPreamble()
	chatCfg := l.buildReactChatConfig(input, systemPreamble, nil)

	var finalAnswer string
	for i := range l.config.MaxToolRounds {
		content, _, err := l.streamer.StreamChat(ctx, chatCfg, w)
		if err != nil {
			return "", fmt.Errorf("agent react stream: %w", err)
		}
		fmt.Fprintln(w)

		// Final Answer terminates the loop.
		if answer, ok := extractFinalAnswer(content); ok {
			finalAnswer = answer
			break
		}

		// Try to parse an Action/Action Input pair.
		toolName, toolInput, ok := parseReactAction(content)
		if !ok || i == l.config.MaxToolRounds-1 {
			// No action found or max rounds: treat whole output as final answer.
			finalAnswer = content
			break
		}

		// Execute the tool and add the observation. The user sees the raw tool
		// output; the copy fed back to the LLM is routed through turo (when
		// active) so tool output is reduced like everything else in the prompt.
		observation := l.dispatchReactTool(toolName, toolInput)
		fmt.Fprintf(w, "\nObservation: %s\n", observation)

		step := reactStep{
			thought:     content,
			action:      toolName,
			actionInput: toolInput,
			observation: turoReduce(ctx, observation),
		}
		chatCfg = l.appendReactStep(chatCfg, step)
	}

	l.session.Append(input, finalAnswer)

	// Auto-extract facts from the turn into persistent memory.
	if l.memoryStore != nil {
		l.memoryStore.ExtractTurn(input, finalAnswer)
	}

	return finalAnswer, nil
}

// buildReactSystemPreamble builds the MRKL system prompt with tool descriptions.
func (l *Loop) buildReactSystemPreamble() string {
	tools := l.registry.List()
	var descs, names strings.Builder
	for _, t := range tools {
		fmt.Fprintf(&descs, "- %s: %s\n", t.Name, t.Description)
		if names.Len() > 0 {
			names.WriteString(", ")
		}
		names.WriteString(t.Name)
	}

	base := turoReduce(context.Background(), fmt.Sprintf(reactSystemTemplate, descs.String(), names.String()))

	extra := l.buildSystemPreamble("")
	if extra != "" {
		return extra + "\n\n" + base
	}
	return base
}

// buildReactChatConfig builds a ChatConfig for the ReAct loop.
// The scratchpad (prior steps) is passed as extra system messages.
func (l *Loop) buildReactChatConfig(
	input, systemPreamble string, steps []reactStep,
) *domain.ChatConfig {
	chatCfg := &domain.ChatConfig{
		Model:   l.config.Model,
		Backend: l.config.Backend,
		BaseURL: l.config.BaseURL,
		Role:    l.config.Role,
		Prompt:  "Question: " + input,
		// No native tools: ReAct relies on text parsing, not function calling.
	}

	scenario := []domain.ScenarioItem{
		{Role: "system", Prompt: systemPreamble},
	}

	for _, s := range steps {
		scenario = append(scenario, domain.ScenarioItem{
			Role:   RoleAssistant,
			Prompt: s.thought,
		})
		scenario = append(scenario, domain.ScenarioItem{
			Role:   "system",
			Prompt: "Observation: " + s.observation,
		})
	}

	chatCfg.Scenario = scenario

	if history := l.session.BuildMessagesJSON(); history != "" {
		chatCfg.Messages = history
	}

	return chatCfg
}

// appendReactStep appends a tool-call/observation step and returns an updated config.
func (l *Loop) appendReactStep(cfg *domain.ChatConfig, step reactStep) *domain.ChatConfig {
	updated := *cfg
	updated.Scenario = append(updated.Scenario,
		domain.ScenarioItem{Role: RoleAssistant, Prompt: step.thought},
		domain.ScenarioItem{Role: "system", Prompt: "Observation: " + step.observation},
	)
	updated.Prompt = "" // already in scratchpad
	return &updated
}

// extractFinalAnswer checks if the output contains "Final Answer:" and returns the answer.
func extractFinalAnswer(output string) (string, bool) {
	idx := strings.Index(output, reactFinalAnswerPrefix)
	if idx < 0 {
		// Case-insensitive fallback.
		lower := strings.ToLower(output)
		for _, variant := range []string{"final answer:", "the final answer is:", "the answer is:"} {
			if i := strings.Index(lower, variant); i >= 0 {
				return strings.TrimSpace(output[i+len(variant):]), true
			}
		}
		return "", false
	}
	return strings.TrimSpace(output[idx+len(reactFinalAnswerPrefix):]), true
}

// parseReactAction extracts the tool name and input from a ReAct-format LLM
// output. When the Action Input value is a JSON object, its true extent is
// found with jsonutil.ScanBalancedObject rather than assumed to fit on one
// line, so a model that pretty-prints multi-line JSON still round-trips.
// Non-JSON input (a plain string, per the model ignoring the "JSON object"
// instruction) falls back to the rest of the first line, matching the prior
// single-line behavior for that case.
func parseReactAction(output string) (string, string, bool) {
	loc := reactActionPrefixRe.FindStringSubmatchIndex(output)
	if loc == nil {
		return "", "", false
	}
	action := strings.TrimSpace(output[loc[2]:loc[3]])
	valueStart := loc[1]

	if valueStart < len(output) && output[valueStart] == '{' {
		if end, ok := jsonutil.ScanBalancedObject(output, valueStart); ok {
			return action, output[valueStart:end], true
		}
	}

	rest := output[valueStart:]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	return action, strings.TrimSpace(rest), true
}

// dispatchReactTool executes a tool by name with the given JSON input string.
func (l *Loop) dispatchReactTool(toolName, toolInput string) string {
	tool := l.registry.Get(toolName)
	if tool == nil {
		return fmt.Sprintf(`{"error":"tool %q not found"}`, toolName)
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolInput), &args); err != nil {
		// Non-JSON input: treat as a single "input" argument.
		args = map[string]interface{}{"input": toolInput}
	}
	canonical := l.registry.ResolveAlias(toolName)
	normalizeToolArgs(canonical, args)
	coerceToolArgTypes(tool.Parameters, args)
	if denyReason, blocked := l.checkPathBoundary(args); blocked {
		return fmt.Sprintf(`{"error":"%s"}`, denyReason)
	}
	result, err := tool.Execute(args)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return result
}
