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

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

func TestBootstrap_ExistingConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	existing := "llm:\n  openai_api_key: keep-me\n"
	require.NoError(t, os.WriteFile(path, []byte(existing), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)

	// Bootstrap should be a no-op when file already exists.
	require.NoError(t, config.Bootstrap(os.Stdout))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, existing, string(data))
}

func TestBootstrap_NonInteractive_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	// stdin is not a terminal in tests → falls back to Scaffold.
	require.NoError(t, config.Bootstrap(os.Stdout))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "openai_api_key")
}

func TestWriteConfigAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	// Bootstrap creates the file (non-interactive path → scaffold).
	require.NoError(t, config.Bootstrap(os.Stdout))

	// The file should be readable by Load.
	_, err := config.Load()
	require.NoError(t, err)
}
