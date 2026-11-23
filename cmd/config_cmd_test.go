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

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCmd_PrintsSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cfgPath := filepath.Join(home, ".kdeps", "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", cfgPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o755))
	require.NoError(t, os.WriteFile(cfgPath,
		[]byte("llm:\n  backend: ollama\n  models:\n    - llama3.2\nregistries:\n  - acme/tools\n"), 0o600))
	stamp := filepath.Join(home, ".kdeps", "registries", "acme", "tools", ".fetched")
	require.NoError(t, os.MkdirAll(filepath.Dir(stamp), 0o755))
	require.NoError(t, os.WriteFile(stamp, []byte("2026-11-23T10:00:00Z\n"), 0o600))

	c := newConfigCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetArgs(nil)
	require.NoError(t, c.Execute())
	s := out.String()
	assert.Contains(t, s, "kdeps:")
	assert.Contains(t, s, "Config:     "+cfgPath)
	assert.Contains(t, s, "Backend:    ollama")
	assert.Contains(t, s, "Models:     llama3.2")
	assert.Regexp(t, `kdeps/registry\s+not downloaded yet`, s)
	assert.Regexp(t, `acme/tools\s+downloaded 2026-11-23T10:00:00Z`, s)
}

func TestConfigCmd_NoConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("KDEPS_CONFIG_PATH", filepath.Join(home, "missing.yaml"))
	var out bytes.Buffer
	require.NoError(t, printConfig(&out))
	assert.Contains(t, out.String(), "(not created yet)")
	assert.Contains(t, out.String(), "Backend:    file")
	assert.Contains(t, out.String(), "pybot")
}

func TestConfigCmd_Edit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("KDEPS_CONFIG_PATH", filepath.Join(home, ".kdeps", "config.yaml"))
	var opened string
	orig := launchEditorFunc
	launchEditorFunc = func(p string) error { opened = p; return nil }
	t.Cleanup(func() { launchEditorFunc = orig })
	c := newConfigCmd()
	c.SetArgs([]string{"--edit"})
	require.NoError(t, c.Execute())
	assert.NotEmpty(t, opened)
}
