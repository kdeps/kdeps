// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

package assets

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// SetInfo says what an asset set is for and where kdeps uses it: the steps
// around it, in order, with the step the set itself drives marked Here.
// kdeps info prints it and the published index carries it, so kdeps.io
// shows the same flow.
type SetInfo struct {
	Purpose string     `json:"purpose"`
	Mode    string     `json:"mode"` // agent, workflow, both or cli
	Flow    []FlowStep `json:"flow"`
	UsedBy  []string   `json:"usedBy"` // the commands and resources that load it
}

// FlowStep is one step of a set's flow.
type FlowStep struct {
	Label string `json:"label"`
	Here  bool   `json:"here,omitempty"`
}

// setPurposes is what each set is for, one sentence. The rest of a set's
// SetInfo - its mode, flow and users - is read from kdeps's call graph by
// tools/assetflow (setflow_gen.go), so it follows the code.
//
//nolint:gochecknoglobals,lll // fixed table
var setPurposes = map[string]string{
	"harness":    "The text kdeps sends the model in agent mode: the sections of the system prompt, and the prompts behind compaction, goals, judges, prompt refinement and nudges.",
	"events":     "Thresholds the agent loop watches - tokens, rounds, bytes, repeated calls - and the action each runs when crossed.",
	"actions":    "The vocabulary of what an event can do when it fires: compact, truncate, block, cap, force an answer, fail a task.",
	"presets":    "Named bundles of event and harness overrides, such as frugal (fewest tokens) or thorough (most context).",
	"themes":     "Colors, prompt and syntax highlighting of the agent loop's terminal.",
	"tools":      "What the model is told about each built-in tool: its description, usage notes and parameters. kdeps runs the tool itself.",
	"commands":   "Slash commands: the help and completion of the built-in ones, and declared commands that run other commands and prompts.",
	"recipes":    "How to build and run an LLM server as a Docker appliance: engine, base image, install steps, ports and health check.",
	"models":     "The model catalogs: downloadable llamafile and GGUF models, and the cloud models of each provider with their limits.",
	"templates":  "Project scaffolds for kdeps new, and the knowledge vault's files (routing table, schema, graph page).",
	"docker":     "The Dockerfile, entrypoint and supervisor templates of a kdeps image.",
	"k8s":        "The Deployment, Service and NetworkPolicy templates of a Kubernetes export.",
	"schemas":    "The JSON schemas a workflow, resource, component and agency must match.",
	"components": "Ready-made workflow steps (sql, browser, email, git, ocr...) a resource calls by name; in agent mode they are also tools.",
	"providers":  "OpenAI-compatible LLM providers: base URL, chat path, API key setting and default model.",
	"defaults":   "The values a resource gets when it sets none: chat, HTTP, python and exec timeouts, context length, embedding store.",
}

// SetInfos describes every asset set: its purpose, and where kdeps uses it
// (from setflow_gen.go).
//
//nolint:gochecknoglobals // built once from the two tables
var SetInfos = func() map[string]SetInfo {
	out := make(map[string]SetInfo, len(setPurposes))
	for set, purpose := range setPurposes {
		info := generatedFlows[set]
		info.Purpose = purpose
		out[set] = info
	}
	return out
}()

// ItemDescription is an item's own one-line purpose, read from its YAML:
// description, metadata.description (components), the first usage's help
// (commands) or label (providers), in a bundle from its component.yaml.
// "" when it has none.
func ItemDescription(data []byte) string {
	var doc map[string]any
	if yaml.Unmarshal(data, &doc) != nil {
		return ""
	}
	if d := bundleDescription(doc); d != "" {
		return d
	}
	if d := lineOf(doc["description"]); d != "" {
		return d
	}
	if meta, ok := doc["metadata"].(map[string]any); ok {
		if d := lineOf(meta["description"]); d != "" {
			return d
		}
	}
	if d := firstUsageHelp(doc); d != "" {
		return d
	}
	return lineOf(doc["label"])
}

// bundleDescription is the description of a bundle's main file, or "".
func bundleDescription(doc map[string]any) string {
	files, ok := doc["files"].(map[string]any)
	if !ok {
		return ""
	}
	for _, f := range []string{"component.yaml", "agency.yaml", "workflow.yaml"} {
		if body, isStr := files[f].(string); isStr {
			if d := ItemDescription([]byte(body)); d != "" {
				return d
			}
		}
	}
	return ""
}

// firstUsageHelp is a command's first usage help, or "".
func firstUsageHelp(doc map[string]any) string {
	usage, ok := doc["usage"].([]any)
	if !ok || len(usage) == 0 {
		return ""
	}
	u, _ := usage[0].(map[string]any)
	return lineOf(u["help"])
}

// lineOf is v's first non-empty line when v is a string.
func lineOf(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(firstLine(strings.TrimSpace(s)))
}
