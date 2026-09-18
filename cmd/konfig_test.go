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

	"github.com/kdeps/kdeps/v2/pkg/agent"
)

// isolateKonfigHome points os.UserHomeDir (via HOME/USERPROFILE) at a fresh
// temp dir, so this never touches the real ~/.kdeps.
func isolateKonfigHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestKonfigExportCmd_FreshHomeExportsCompleteDefaultConfig(t *testing.T) {
	isolateKonfigHome(t)
	path := filepath.Join(t.TempDir(), "konfig.yaml")

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"konfig", "export", path})
	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "Exported")

	k, err := agent.ReadKonfig(path)
	require.NoError(t, err)
	assert.NotEmpty(t, k.Harness, "even a bare ~/.kdeps must export every built-in harness section")
	assert.NotEmpty(t, k.Themes, "even a bare ~/.kdeps must export every built-in theme")
}

func TestKonfigExportCmd_DefaultPath(t *testing.T) {
	isolateKonfigHome(t)
	dir := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"konfig", "export"})
	require.NoError(t, root.Execute())

	_, err := agent.ReadKonfig(filepath.Join(dir, "konfig.yaml"))
	require.NoError(t, err)
}
