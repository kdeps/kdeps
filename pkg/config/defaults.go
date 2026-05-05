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
	_ "embed"
	"fmt"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed defaults.yml
var defaultsYAML []byte

// ExecutorDefaults holds all embedded default values for executors.
type ExecutorDefaults struct {
	Chat      ChatExecutorDefaults   `yaml:"chat"`
	HTTP      HTTPExecutorDefaults   `yaml:"http"`
	Python    PythonExecutorDefaults `yaml:"python"`
	Exec      ExecExecutorDefaults   `yaml:"exec"`
	SQL       SQLExecutorDefaults    `yaml:"sql"`
	Scraper   ScraperDefaults        `yaml:"scraper"`
	SearchWeb SearchWebDefaults      `yaml:"search_web"`
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

// SQLExecutorDefaults holds default values for SQL query execution.
type SQLExecutorDefaults struct {
	Timeout         string `yaml:"timeout"`
	MaxRows         int    `yaml:"max_rows"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxIdleTime string `yaml:"conn_max_idle_time"`
}

// ScraperDefaults holds default values for web scraper execution.
type ScraperDefaults struct {
	Timeout int `yaml:"timeout"`
}

// SearchWebDefaults holds default values for web search execution.
type SearchWebDefaults struct {
	Timeout    int `yaml:"timeout"`
	MaxResults int `yaml:"max_results"`
}

// EmbeddingDefaults holds default values for embedding execution.
type EmbeddingDefaults struct {
	DBPath     string `yaml:"db_path"`
	Collection string `yaml:"collection"`
	Limit      int    `yaml:"limit"`
}

var (
	parseDefaultsOnce sync.Once         //nolint:gochecknoglobals // package-level cache
	parsedDefaults    *ExecutorDefaults //nolint:gochecknoglobals // package-level cache
	errDefaultsParse  error
)

// GetDefaults returns the parsed embedded executor defaults.
func GetDefaults() (*ExecutorDefaults, error) {
	parseDefaultsOnce.Do(func() {
		var d ExecutorDefaults
		if err := yaml.Unmarshal(defaultsYAML, &d); err != nil {
			errDefaultsParse = fmt.Errorf("parse embedded defaults.yml: %w", err)
			return
		}
		parsedDefaults = &d
	})
	return parsedDefaults, errDefaultsParse
}

// TimeoutDuration parses the chat timeout string into a time.Duration.
func (c *ChatExecutorDefaults) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 60 * time.Second //nolint:mnd // fallback if embedded YAML is unparseable
	}
	return d
}

// TimeoutDuration parses the HTTP timeout string into a time.Duration.
func (h *HTTPExecutorDefaults) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(h.Timeout)
	if err != nil {
		return 30 * time.Second //nolint:mnd // fallback if embedded YAML is unparseable
	}
	return d
}

// TimeoutDuration parses the Python timeout string into a time.Duration.
func (p *PythonExecutorDefaults) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(p.Timeout)
	if err != nil {
		return 60 * time.Second //nolint:mnd // fallback if embedded YAML is unparseable
	}
	return d
}

// TimeoutDuration parses the Exec timeout string into a time.Duration.
func (e *ExecExecutorDefaults) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(e.Timeout)
	if err != nil {
		return 30 * time.Second //nolint:mnd // fallback if embedded YAML is unparseable
	}
	return d
}

// TimeoutDuration parses the SQL timeout string into a time.Duration.
func (s *SQLExecutorDefaults) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return 30 * time.Second //nolint:mnd // fallback if embedded YAML is unparseable
	}
	return d
}

// ConnMaxIdleTimeDuration parses the SQL connection max idle time.
func (s *SQLExecutorDefaults) ConnMaxIdleTimeDuration() time.Duration {
	d, err := time.ParseDuration(s.ConnMaxIdleTime)
	if err != nil {
		return 5 * time.Minute //nolint:mnd // fallback if embedded YAML is unparseable
	}
	return d
}
