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

package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

//nolint:gochecknoglobals // test-replaceable
var jsonMarshalIndent = json.MarshalIndent

//nolint:gochecknoglobals // test-replaceable
var AppFS = afero.NewOsFs()

//nolint:gochecknoglobals // test-replaceable
var osUserHomeDir = os.UserHomeDir

// Turn represents one exchange in the conversation history.
type Turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GeneratedWorkflow holds the files produced by the generator.
type GeneratedWorkflow struct {
	// Files maps relative path (e.g. "workflow.yaml", "resources/main.yaml") to content.
	Files map[string]string
}

// Session manages per-conversation state: temp directory, history, current workflow.
type Session struct {
	ID       string
	Dir      string
	History  []Turn
	Workflow *GeneratedWorkflow
}

// NewSession creates a new session with a unique temp directory.
func NewSession() (*Session, error) {
	home, err := osUserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home directory: %w", err)
	}

	sessionsRoot := filepath.Join(home, ".kdeps", "chat-sessions")
	if mkErr := AppFS.MkdirAll(sessionsRoot, 0o700); mkErr != nil {
		return nil, fmt.Errorf("could not create sessions directory: %w", mkErr)
	}

	id := fmt.Sprintf("session-%d", time.Now().UnixNano())
	dir := filepath.Join(sessionsRoot, id)
	if mkErr := AppFS.MkdirAll(dir, 0o700); mkErr != nil {
		return nil, fmt.Errorf("could not create session directory: %w", mkErr)
	}

	return &Session{
		ID:      id,
		Dir:     dir,
		History: []Turn{},
	}, nil
}

// AddTurn appends a conversation turn.
func (s *Session) AddTurn(role, content string) {
	s.History = append(s.History, Turn{Role: role, Content: content})
}

// WriteWorkflow writes the current workflow files to the session directory.
func (s *Session) WriteWorkflow() error {
	if s.Workflow == nil {
		return errors.New("no workflow to write")
	}
	for path, content := range s.Workflow.Files {
		full := filepath.Join(s.Dir, path)
		if mkErr := AppFS.MkdirAll(filepath.Dir(full), 0o700); mkErr != nil {
			return fmt.Errorf("could not create directory for %s: %w", path, mkErr)
		}
		if writeErr := afero.WriteFile(AppFS, full, []byte(content), 0o600); writeErr != nil {
			return fmt.Errorf("could not write %s: %w", path, writeErr)
		}
	}
	return nil
}

// SaveTo copies the current workflow to the given destination directory.
func (s *Session) SaveTo(dest string) error {
	if s.Workflow == nil {
		return errors.New("no workflow to save")
	}
	if mkErr := AppFS.MkdirAll(dest, 0o750); mkErr != nil {
		return fmt.Errorf("could not create destination directory: %w", mkErr)
	}
	for path, content := range s.Workflow.Files {
		full := filepath.Join(dest, path)
		if mkErr := AppFS.MkdirAll(filepath.Dir(full), 0o750); mkErr != nil {
			return fmt.Errorf("could not create directory for %s: %w", path, mkErr)
		}
		if writeErr := afero.WriteFile(AppFS, full, []byte(content), 0o600); writeErr != nil {
			return fmt.Errorf("could not write %s: %w", path, writeErr)
		}
	}
	return nil
}

// SaveHistory writes conversation history to disk for later resumption.
func (s *Session) SaveHistory() error {
	data, err := jsonMarshalIndent(s.History, "", "  ")
	if err != nil {
		return err
	}
	return afero.WriteFile(AppFS, filepath.Join(s.Dir, "history.json"), data, 0o600)
}

// LoadSession loads a previously saved session by ID.
func LoadSession(id string) (*Session, error) {
	home, err := osUserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home directory: %w", err)
	}

	dir := filepath.Join(home, ".kdeps", "chat-sessions", id)
	if _, statErr := AppFS.Stat(dir); statErr != nil {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	s := &Session{ID: id, Dir: dir}

	histFile := filepath.Join(dir, "history.json")
	data, readErr := afero.ReadFile(AppFS, histFile)
	if readErr == nil {
		if jsonErr := json.Unmarshal(data, &s.History); jsonErr != nil {
			return nil, fmt.Errorf("could not load history: %w", jsonErr)
		}
	}

	return s, nil
}

// Cleanup removes the session directory.
func (s *Session) Cleanup() {
	_ = AppFS.RemoveAll(s.Dir)
}

// Reset clears history and workflow, keeping the directory.
func (s *Session) Reset() {
	s.History = []Turn{}
	s.Workflow = nil
	// Clear files in dir except history.json
	entries, _ := afero.ReadDir(AppFS, s.Dir)
	for _, e := range entries {
		if e.Name() == "history.json" {
			continue
		}
		_ = AppFS.RemoveAll(filepath.Join(s.Dir, e.Name()))
	}
}
