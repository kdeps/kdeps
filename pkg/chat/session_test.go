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

package chat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	s, err := NewSession()
	require.NoError(t, err)
	assert.NotEmpty(t, s.ID)
	assert.DirExists(t, s.Dir)
	assert.Empty(t, s.History)
	assert.Nil(t, s.Workflow)
}

func TestSession_AddTurn(t *testing.T) {
	s := &Session{}
	s.AddTurn("user", "hello")
	s.AddTurn("assistant", "world")
	assert.Len(t, s.History, 2)
	assert.Equal(t, "user", s.History[0].Role)
	assert.Equal(t, "world", s.History[1].Content)
}

func TestSession_WriteWorkflow(t *testing.T) {
	dir := t.TempDir()
	s := &Session{
		Dir: dir,
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{
				"workflow.yaml":       "apiVersion: kdeps.io/v1\n",
				"resources/main.yaml": "id: main\n",
			},
		},
	}

	require.NoError(t, s.WriteWorkflow())
	assert.FileExists(t, filepath.Join(dir, "workflow.yaml"))
	assert.FileExists(t, filepath.Join(dir, "resources", "main.yaml"))
}

func TestSession_WriteWorkflow_NoWorkflow(t *testing.T) {
	s := &Session{}
	err := s.WriteWorkflow()
	assert.Error(t, err)
}

func TestSession_SaveTo(t *testing.T) {
	sessionDir := t.TempDir()
	destDir := filepath.Join(t.TempDir(), "output")

	s := &Session{
		Dir: sessionDir,
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{
				"workflow.yaml": "name: test\n",
			},
		},
	}

	require.NoError(t, s.SaveTo(destDir))
	assert.FileExists(t, filepath.Join(destDir, "workflow.yaml"))

	data, err := os.ReadFile(filepath.Join(destDir, "workflow.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "name: test\n", string(data))
}

func TestSession_SaveTo_NoWorkflow(t *testing.T) {
	s := &Session{}
	err := s.SaveTo(t.TempDir())
	assert.Error(t, err)
}

func TestSession_SaveHistory_And_Load(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	s, err := NewSession()
	require.NoError(t, err)
	s.AddTurn("user", "do something")
	s.AddTurn("assistant", "done")

	require.NoError(t, s.SaveHistory())

	loaded, err := LoadSession(s.ID)
	require.NoError(t, err)
	assert.Len(t, loaded.History, 2)
	assert.Equal(t, "do something", loaded.History[0].Content)
}

func TestLoadSession_NotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	_, err := LoadSession("nonexistent-session")
	assert.Error(t, err)
}

func TestSession_Reset(t *testing.T) {
	dir := t.TempDir()

	// Create a file that should be removed
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte("x"), 0o600))

	s := &Session{
		Dir: dir,
		History: []Turn{
			{Role: "user", Content: "hello"},
		},
		Workflow: &GeneratedWorkflow{Files: map[string]string{"workflow.yaml": "x"}},
	}

	s.Reset()

	assert.Empty(t, s.History)
	assert.Nil(t, s.Workflow)
	assert.NoFileExists(t, filepath.Join(dir, "workflow.yaml"))
}

func TestSession_Cleanup(t *testing.T) {
	dir := t.TempDir()
	inner := filepath.Join(dir, "subdir")
	require.NoError(t, os.MkdirAll(inner, 0o700))

	s := &Session{Dir: inner}
	s.Cleanup()
	assert.NoDirExists(t, inner)
}
