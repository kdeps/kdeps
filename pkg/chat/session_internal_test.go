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
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveHistory_MarshalError(t *testing.T) {
	orig := jsonMarshalIndent
	t.Cleanup(func() { jsonMarshalIndent = orig })
	jsonMarshalIndent = func(_ any, _, _ string) ([]byte, error) {
		return nil, errors.New("injected marshal error")
	}
	s := &Session{History: []Turn{{Role: "user", Content: "hello"}}}
	err := s.SaveHistory()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected marshal error")
}

func TestNewSession_HomeDirError(t *testing.T) {
	orig := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = orig })
	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home")
	}

	_, err := NewSession()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not determine home directory")
}

func TestSaveTo_MkdirAllError(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	AppFS = afero.NewReadOnlyFs(afero.NewMemMapFs())

	s := &Session{Workflow: &GeneratedWorkflow{Files: map[string]string{"test.txt": "content"}}}
	err := s.SaveTo("/readonly/dir")
	require.Error(t, err)
}

func TestLoadSession_DINotFound(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	AppFS = afero.NewMemMapFs()

	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) { return "/fakehome", nil }

	_, err := LoadSession("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}
