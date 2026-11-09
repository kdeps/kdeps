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

package domain

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// layout places a struct's flat YAML keys under named groups in the file:
// flat key -> group path. A path of one element maps the whole group value to
// the flat key.
type layout struct {
	paths   map[string][]string
	groups  []string          // group order on output
	renamed map[string]string // old top-level key -> where it went
}

func newLayout(groups []string, entries map[string]string) layout {
	l := layout{paths: make(map[string][]string, len(entries)), groups: groups}
	for flat, path := range entries {
		l.paths[flat] = strings.Split(path, ".")
	}
	return l
}

func (l layout) isGroup(key string) bool {
	for _, g := range l.groups {
		if g == key {
			return true
		}
	}
	return false
}

// flatKey returns the flat key stored at path, if any.
func (l layout) flatKey(path []string) (string, bool) {
	want := strings.Join(path, ".")
	for flat, p := range l.paths {
		if strings.Join(p, ".") == want {
			return flat, true
		}
	}
	return "", false
}

// hasPrefix reports whether some path continues below path.
func (l layout) hasPrefix(path []string) bool {
	want := strings.Join(path, ".") + "."
	for _, p := range l.paths {
		if strings.HasPrefix(strings.Join(p, "."), want) {
			return true
		}
	}
	return false
}

// ungroup turns a grouped mapping node into the flat mapping the struct
// decodes. A flat key written at the top level is an error naming its group.
func (l layout) ungroup(node *yaml.Node, where string) (*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return node, nil
	}
	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Line: node.Line, Column: node.Column}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if l.isGroup(k.Value) {
			if err := l.collect(out, []string{k.Value}, v, where); err != nil {
				return nil, err
			}
			continue
		}
		if to, ok := l.renamed[k.Value]; ok {
			return nil, fmt.Errorf("line %d: %s.%s is now %s", k.Line, where, k.Value, to)
		}
		if p, moved := l.paths[k.Value]; moved {
			return nil, fmt.Errorf("line %d: %s.%s moved to %s.%s", k.Line, where, k.Value, where, strings.Join(p, "."))
		}
		out.Content = append(out.Content, k, v)
	}
	return out, nil
}

func (l layout) collect(out *yaml.Node, path []string, v *yaml.Node, where string) error {
	if flat, ok := l.flatKey(path); ok {
		out.Content = append(out.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: flat}, v)
		return nil
	}
	name := where + "." + strings.Join(path, ".")
	if v.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: %s must be a mapping", v.Line, name)
	}
	for i := 0; i+1 < len(v.Content); i += 2 {
		k, sub := v.Content[i], v.Content[i+1]
		next := append(append([]string{}, path...), k.Value)
		if _, ok := l.flatKey(next); !ok && !l.hasPrefix(next) {
			return fmt.Errorf("line %d: unknown key %s.%s", k.Line, name, k.Value)
		}
		if err := l.collect(out, next, sub, where); err != nil {
			return err
		}
	}
	return nil
}

// group turns the flat mapping node a struct encodes to into the grouped form.
func (l layout) group(node *yaml.Node) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return node
	}
	groups := map[string]*yaml.Node{}
	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	var rest []*yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		path, ok := l.paths[k.Value]
		if !ok {
			rest = append(rest, k, v)
			continue
		}
		if len(path) == 1 {
			groups[path[0]] = v
			continue
		}
		g := groups[path[0]]
		if g == nil {
			g = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			groups[path[0]] = g
		}
		setPath(g, path[1:], v)
	}
	out.Content = append(out.Content, rest...)
	for _, name := range l.groups {
		if g := groups[name]; g != nil {
			out.Content = append(out.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}, g)
		}
	}
	return out
}

func setPath(m *yaml.Node, path []string, v *yaml.Node) {
	if len(path) == 1 {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: path[0]}, v)
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == path[0] {
			setPath(m.Content[i+1], path[1:], v)
			return
		}
	}
	sub := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: path[0]}, sub)
	setPath(sub, path[1:], v)
}

//nolint:gochecknoglobals // static key layouts
var (
	chatLayout = newLayout(
		[]string{"model", "context", "harness", "tools", "rag", "guardrails"},
		map[string]string{
			"model":                   "model.name",
			"backend":                 "model.backend",
			"timeout":                 "model.timeout",
			"streaming":               "model.streaming",
			"thinking":                "model.thinking",
			"promptCaching":           "model.promptCaching",
			"useCache":                "model.useCache",
			"contextSize":             "model.contextSize",
			"temperature":             "model.temperature",
			"maxTokens":               "model.maxTokens",
			"topP":                    "model.topP",
			"topK":                    "model.topK",
			"seed":                    "model.seed",
			"frequencyPenalty":        "model.frequencyPenalty",
			"presencePenalty":         "model.presencePenalty",
			"repetitionPenalty":       "model.repetitionPenalty",
			"stopWords":               "model.stopWords",
			"candidateCount":          "model.candidateCount",
			"n":                       "model.n",
			"minLength":               "model.minLength",
			"maxLength":               "model.maxLength",
			"googleCachedContent":     "model.googleCachedContent",
			"googleHarmThreshold":     "model.googleHarmThreshold",
			"googleCloudProject":      "model.googleCloudProject",
			"googleCloudLocation":     "model.googleCloudLocation",
			"googleCredentialsJSON":   "model.googleCredentialsJSON",
			"googleCredentialsFile":   "model.googleCredentialsFile",
			"anthropicExtendedOutput": "model.anthropicExtendedOutput",
			"anthropicBetaHeaders":    "model.anthropicBetaHeaders",
			"openAILegacyMaxTokens":   "model.openAILegacyMaxTokens", "openAIWebSearch": "model.openAIWebSearch",
			"ollamaThink":       "model.ollamaThink",
			"ollamaKeepAlive":   "model.ollamaKeepAlive",
			"ollamaPullModel":   "model.ollamaPullModel",
			"ollamaPullTimeout": "model.ollamaPullTimeout",
			"ollamaNumCtx":      "model.ollamaNumCtx",
			"prompt":            "context.prompt",
			"role":              "context.role",
			"messages":          "context.messages",
			"scenario":          "context.scenario",
			"promptVars":        "context.promptVars",
			"goTemplate":        "context.goTemplate",
			"files":             "context.files",
			"contextLength":     "context.contextLength",
			"fewShot":           "context.fewShot.examples", "fewShotSelectK": "context.fewShot.selectK",
			"fewShotMaxTokens":        "context.fewShot.maxTokens",
			"fewShotEmbeddingModel":   "context.fewShot.embeddingModel",
			"fewShotEmbeddingBackend": "context.fewShot.embeddingBackend",
			"harness":                 "harness.sections", "refine": "harness.refine",
			"chainOfThought": "harness.chainOfThought",
			"jsonResponse":   "harness.output.json", "jsonResponseKeys": "harness.output.keys",
			"jsonSchema":                "harness.output.schema",
			"outputParser":              "harness.output.parser",
			"tools":                     "tools.list",
			"componentTools":            "tools.components",
			"toolChoice":                "tools.choice",
			"maxToolRounds":             "tools.maxRounds",
			"skills":                    "tools.skills",
			"knowledge":                 "rag.knowledge",
			"retrieverContext":          "rag.retriever.context",
			"retrieverContextTopK":      "rag.retriever.topK",
			"retrieverContextMaxTokens": "rag.retriever.maxTokens",
			"chunkSize":                 "rag.chunk.size", "chunkOverlap": "rag.chunk.overlap",
			"chunkSplitter": "rag.chunk.splitter",
			"judges":        "guardrails.judges", "judgePreset": "guardrails.judgePreset",
		},
	)
	resourceLayout = func() layout {
		l := newLayout([]string{"workflow"}, map[string]string{
			"requires": "workflow.requires",
			"items":    "workflow.items",
			"loop":     "workflow.loop",
			"onError":  "workflow.onError",
		})
		l.renamed = map[string]string{"validations": "guardrails"}
		return l
	}()
	workflowLayout = func() layout {
		l := newLayout([]string{"deploy", "memory", "agent"}, map[string]string{
			"apiServer":     "deploy.apiServer",
			"webServer":     "deploy.webServer",
			"certFile":      "deploy.certFile",
			"keyFile":       "deploy.keyFile",
			"letsEncrypt":   "deploy.letsEncrypt",
			"input":         "deploy.input",
			"agentSettings": "deploy.runtime",
			"session":       "memory",
			"llm":           "agent",
		})
		l.renamed = map[string]string{"settings": "deploy, memory and agent", "tests": "evals"}
		return l
	}()
)

type chatConfigPlain ChatConfig

// UnmarshalYAML reads the grouped chat layout (model, context, harness,
// tools, rag, guardrails).
func (c *ChatConfig) UnmarshalYAML(node *yaml.Node) error {
	flat, err := chatLayout.ungroup(node, "chat")
	if err != nil {
		return err
	}
	return flat.Decode((*chatConfigPlain)(c))
}

// MarshalYAML writes the grouped chat layout.
func (c ChatConfig) MarshalYAML() (any, error) {
	var n yaml.Node
	if err := n.Encode(chatConfigPlain(c)); err != nil {
		return nil, err
	}
	return chatLayout.group(&n), nil
}

type resourcePlain Resource

// UnmarshalYAML reads the resource layout (workflow: requires, items, loop, onError).
func (r *Resource) UnmarshalYAML(node *yaml.Node) error {
	flat, err := resourceLayout.ungroup(node, "resource")
	if err != nil {
		return err
	}
	return flat.Decode((*resourcePlain)(r))
}

// MarshalYAML writes the resource layout.
func (r Resource) MarshalYAML() (any, error) {
	var n yaml.Node
	if err := n.Encode(resourcePlain(r)); err != nil {
		return nil, err
	}
	return resourceLayout.group(&n), nil
}

type workflowPlain Workflow

const settingsKey = "settings"

// UnmarshalYAML reads the workflow layout: deploy, memory and agent hold
// what Settings carries.
func (w *Workflow) UnmarshalYAML(node *yaml.Node) error {
	flat, err := workflowLayout.ungroup(node, "workflow")
	if err != nil {
		return err
	}
	if flat.Kind == yaml.MappingNode {
		settings := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		var rest []*yaml.Node
		for i := 0; i+1 < len(flat.Content); i += 2 {
			if _, ok := workflowLayout.paths[flat.Content[i].Value]; ok {
				settings.Content = append(settings.Content, flat.Content[i], flat.Content[i+1])
				continue
			}
			rest = append(rest, flat.Content[i], flat.Content[i+1])
		}
		rest = append(rest, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: settingsKey}, settings)
		flat.Content = rest
	}
	return flat.Decode((*workflowPlain)(w))
}

// MarshalYAML writes the workflow layout.
func (w Workflow) MarshalYAML() (any, error) {
	var n yaml.Node
	if err := n.Encode(workflowPlain(w)); err != nil {
		return nil, err
	}
	var flat []*yaml.Node
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == settingsKey {
			flat = append(flat, n.Content[i+1].Content...)
			continue
		}
		flat = append(flat, n.Content[i], n.Content[i+1])
	}
	n.Content = flat
	return workflowLayout.group(&n), nil
}
