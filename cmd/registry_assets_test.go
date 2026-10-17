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

package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/assets"
)

func TestIsAssetRefAndSplitRefs(t *testing.T) {
	assert.True(t, isAssetRef("tools"))
	assert.True(t, isAssetRef("tools/bash_exec"))
	assert.True(t, isAssetRef("harness/safety@1.2.0"))
	assert.False(t, isAssetRef("invoice-extractor"))
	assert.False(t, isAssetRef("owner/repo"))
	assert.False(t, isAssetRef("./tools/x.kdeps"))
	assert.False(t, isAssetRef("github:tools/repo"), "github: forces a GitHub ref")

	a, p := splitRefs([]string{"themes/vim,themes/linux", "agent-x", "owner/repo:sub", "tools"})
	assert.Equal(t, []string{"themes/vim", "themes/linux", "tools"}, a)
	assert.Equal(t, []string{"agent-x", "owner/repo:sub"}, p)
}

func TestRegistryAssetCommands(t *testing.T) {
	// Before the Setenv calls: reload once they are undone.
	t.Cleanup(agent.ReloadAssets)
	dir := t.TempDir()
	body := []byte("version: 9.1.0\ndescription: Newer tail text.\n")
	p := filepath.Join(dir, filepath.FromSlash(assets.PublishedPath("tools/tail_file", "9.1.0")))
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o750))
	require.NoError(t, os.WriteFile(p, body, 0o600))
	require.NoError(t, assets.WriteIndex(dir, assets.Index{Items: map[string]assets.IndexItem{
		"tools/tail_file": {
			Latest:   "9.1.0",
			Versions: []assets.IndexVersion{{Version: "9.1.0", SHA256: assets.Sum(body)}},
		},
	}}))
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	t.Cleanup(srv.Close)
	t.Setenv("KDEPS_ASSETS_URL", srv.URL)
	t.Setenv("KDEPS_ASSETS_DIR", t.TempDir())

	run := func(args ...string) (string, error) {
		c := &cobra.Command{Use: "kdeps"}
		c.AddCommand(registryCommands()...)
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&out)
		c.SetIn(bytes.NewReader(nil))
		c.SetArgs(args)
		err := c.Execute()
		return out.String(), err
	}

	out, err := run("update", "--check")
	require.NoError(t, err)
	assert.Contains(t, out, "tools/tail_file")
	assert.Contains(t, out, "available")

	_, err = run("update", "--check", "some-package")
	require.ErrorIs(t, err, errCheckPackages)

	out, err = run("install", "tools/tail_file@9.1.0")
	require.NoError(t, err)
	assert.Contains(t, out, "pin")

	out, err = run("list", "--assets")
	require.NoError(t, err)
	assert.Regexp(t, `tools/tail_file\s+9\.1\.0\s+pinned`, out)

	out, err = run("uninstall", "themes/vim,themes/linux")
	require.NoError(t, err)
	assert.Contains(t, out, "themes/linux")

	out, err = run("update", "tools/tail_file")
	require.NoError(t, err)
	assert.NotContains(t, out, "failed")

	_, err = run("uninstall", "themes/normal")
	require.ErrorContains(t, err, "required")

	_, err = run("profile", "nope")
	require.ErrorContains(t, err, "unknown profile")
	_, err = run("profile")
	require.ErrorContains(t, err, "needs a terminal")
	_, err = run("profile", "themes")
	require.NoError(t, err)
}
