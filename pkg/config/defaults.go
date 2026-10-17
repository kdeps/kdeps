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

package config

import (
	"fmt"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/assets"
)

// The executor defaults are the "executors" item of the "defaults" asset set.
const (
	defaultsSet  = "defaults"
	defaultsItem = "executors"
)

//nolint:gochecknoinits // registers the executor defaults as a versioned asset set
func init() {
	assets.Register(defaultsSet, assets.Spec{
		Validate: func(data []byte, _ string) error { return yaml.Unmarshal(data, &ExecutorDefaults{}) },
		Required: []string{defaultsItem},
	})
}

// readDefaults returns the executor defaults file (test-replaceable).
//
//nolint:gochecknoglobals // test-replaceable
var readDefaults = func() ([]byte, error) {
	return assets.Open(defaultsSet).ReadFile(defaultsSet + "/" + defaultsItem + ".yaml")
}

// ExecutorDefaults holds the default values for executors.
type ExecutorDefaults struct {
	Chat      ChatExecutorDefaults   `yaml:"chat"`
	HTTP      HTTPExecutorDefaults   `yaml:"http"`
	Python    PythonExecutorDefaults `yaml:"python"`
	Exec      ExecExecutorDefaults   `yaml:"exec"`
	Embedding EmbeddingDefaults      `yaml:"embedding"`
}

// ChatExecutorDefaults holds default values for LLM chat execution.
type ChatExecutorDefaults struct {
	Timeout       string `yaml:"timeout"`
	ContextLength int    `yaml:"context_length"`
	Streaming     bool   `yaml:"streaming"`
}

// HTTPExecutorDefaults holds default values for HTTP client execution.
type HTTPExecutorDefaults struct {
	Timeout         string `yaml:"timeout"`
	FollowRedirects bool   `yaml:"follow_redirects"`
}

// PythonExecutorDefaults holds default values for Python script execution.
type PythonExecutorDefaults struct {
	Timeout string `yaml:"timeout"`
}

// ExecExecutorDefaults holds default values for shell command execution.
type ExecExecutorDefaults struct {
	Timeout string `yaml:"timeout"`
}

// EmbeddingDefaults holds default values for embedding execution.
type EmbeddingDefaults struct {
	DBPath     string `yaml:"db_path"`
	Collection string `yaml:"collection"`
	Limit      int    `yaml:"limit"`
}

//nolint:gochecknoglobals // package-level cache of the first successful load
var (
	defaultsMu     sync.Mutex
	parsedDefaults *ExecutorDefaults
)

func parseDurationOrDefault(raw string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

// GetDefaults returns the executor defaults from the defaults asset set. It
// never returns nil: when the set is missing or broken it returns zero
// values (each accessor then uses its own fallback) with the error. Only a
// successful load is cached, so a later call sees assets installed since.
//
//kdeps:flow defaults: a resource without a timeout (or another default) runs > [the executor defaults fill it in] > the executor runs the resource
func GetDefaults() (*ExecutorDefaults, error) {
	defaultsMu.Lock()
	defer defaultsMu.Unlock()
	if parsedDefaults != nil {
		return parsedDefaults, nil
	}
	data, err := readDefaults()
	if err != nil {
		return &ExecutorDefaults{}, fmt.Errorf("read executor defaults (run kdeps update): %w", err)
	}
	var d ExecutorDefaults
	if err = yaml.Unmarshal(data, &d); err != nil {
		return &ExecutorDefaults{}, fmt.Errorf("parse executor defaults: %w", err)
	}
	parsedDefaults = &d
	return parsedDefaults, nil
}

// TimeoutDuration parses the chat timeout string into a time.Duration.
func (c *ChatExecutorDefaults) TimeoutDuration() time.Duration {
	return parseDurationOrDefault(c.Timeout, 60*time.Second) //nolint:mnd // fallback
}

// TimeoutDuration parses the HTTP timeout string into a time.Duration.
func (h *HTTPExecutorDefaults) TimeoutDuration() time.Duration {
	return parseDurationOrDefault(h.Timeout, 30*time.Second) //nolint:mnd // fallback
}

// TimeoutDuration parses the Python timeout string into a time.Duration.
func (p *PythonExecutorDefaults) TimeoutDuration() time.Duration {
	return parseDurationOrDefault(p.Timeout, 60*time.Second) //nolint:mnd // fallback
}

// TimeoutDuration parses the Exec timeout string into a time.Duration.
func (e *ExecExecutorDefaults) TimeoutDuration() time.Duration {
	return parseDurationOrDefault(e.Timeout, 30*time.Second) //nolint:mnd // fallback
}
