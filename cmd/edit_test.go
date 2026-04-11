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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditCmd_CreatesConfigIfMissing(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", cfgPath)
	// Use 'true' as a no-op editor so we don't open a real editor.
	t.Setenv("KDEPS_EDITOR", "true")

	rootCmd := createRootCommand()
	rootCmd.SetArgs([]string{"edit"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Scaffold must have written the config.
	_, statErr := os.Stat(cfgPath)
	assert.NoError(t, statErr)
}

func TestEditCmd_OpensExistingConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("llm:\n"), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", cfgPath)
	t.Setenv("KDEPS_EDITOR", "true") // no-op editor

	rootCmd := createRootCommand()
	rootCmd.SetArgs([]string{"edit"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestEditCmd_InvalidEditor(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", cfgPath)
	t.Setenv("KDEPS_EDITOR", "/nonexistent-binary-kdeps-xyz")

	rootCmd := createRootCommand()
	rootCmd.SetArgs([]string{"edit"})
	err := rootCmd.Execute()
	assert.Error(t, err)
}
