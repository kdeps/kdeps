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

var (
	reactActionRe = regexp.MustCompile(`(?i)Action:\s*(.+?)\s*\nAction\s+Input:\s*(.+)`)
)

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
	if msgs := l.session.rawMessages(); shouldAutoCompact(msgs, l.config.AutoCompactThreshold, l.config.Model) {
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

		// Execute the tool and add the observation.
		observation := l.dispatchReactTool(toolName, toolInput)
		fmt.Fprintf(w, "\nObservation: %s\n", observation)

		step := reactStep{
			thought:     content,
			action:      toolName,
			actionInput: toolInput,
			observation: observation,
		}
		chatCfg = l.appendReactStep(chatCfg, step)
	}

	l.session.Append(input, finalAnswer)
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

	base := fmt.Sprintf(reactSystemTemplate, descs.String(), names.String())

	extra := l.buildSystemPreamble()
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
			Role:   "assistant",
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
		domain.ScenarioItem{Role: "assistant", Prompt: step.thought},
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

// parseReactAction extracts the tool name and input from a ReAct-format LLM output.
func parseReactAction(output string) (string, string, bool) {
	const minMatches = 3
	matches := reactActionRe.FindStringSubmatch(output)
	if len(matches) < minMatches {
		return "", "", false
	}
	return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2]), true
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
	result, err := tool.Execute(args)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return result
}
