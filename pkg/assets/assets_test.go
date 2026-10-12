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

package assets

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func sourceFiles() fstest.MapFS {
	return fstest.MapFS{
		"harness/safety.yaml":                 {Data: []byte("version: 1.0.0\nbody: source safety\n")},
		"harness/tools.yaml":                  {Data: []byte("version: 1.0.0\nbody: source tools\n")},
		"templates/api/template.yaml":         {Data: []byte("version: 1.0.0\n")},
		"templates/api/workflow.yaml.j2":      {Data: []byte("source workflow")},
		"templates/api/resources/llm.yaml.j2": {Data: []byte("source llm")},
	}
}

// writeSource writes files into a temp directory and makes it Source().
func writeSource(t *testing.T, files fstest.MapFS) {
	t.Helper()
	dir := t.TempDir()
	for name, f := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o750))
		require.NoError(t, os.WriteFile(p, f.Data, 0o600))
	}
	t.Setenv(SourceEnv, dir)
}

func registerTestSets(t *testing.T) {
	t.Helper()
	writeSource(t, sourceFiles())
	Register("harness", Spec{
		Required: func(name string) bool { return name == "core" },
		Validate: func(data []byte, _ string) error {
			h, err := ParseHeader(data)
			if err == nil && h.Version == "9.9.9" {
				return errors.New("rejected by the set parser")
			}
			return err
		},
	})
	Register(BundleSet, Spec{})
}

func read(t *testing.T, fsys ReadFS, p string) string {
	t.Helper()
	data, err := fsys.ReadFile(p)
	require.NoError(t, err)
	return string(data)
}

func TestRoot(t *testing.T) {
	t.Setenv("KDEPS_ASSETS_DIR", "/tmp/x")
	assert.Equal(t, "/tmp/x", Root())
}

func TestSplitID(t *testing.T) {
	set, name, err := SplitID("harness/safety")
	require.NoError(t, err)
	assert.Equal(t, []string{"harness", "safety"}, []string{set, name})

	set, name, err = SplitID("themes")
	require.NoError(t, err)
	assert.Equal(t, "themes", set)
	assert.Empty(t, name)

	for _, bad := range []string{"nope/x", "harness/..", "harness/a/b", ""} {
		_, _, err = SplitID(bad)
		assert.Error(t, err, bad)
	}
}

func TestLock_RoundTripAndErrors(t *testing.T) {
	root := t.TempDir()
	lock, err := ReadLock(root)
	require.NoError(t, err)
	assert.Empty(t, lock.Items, "missing lock is empty")

	lock.Items["harness/safety"] = LockEntry{Version: "1.1.0", Pinned: true}
	require.NoError(t, WriteLock(root, lock))
	got, err := ReadLock(root)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", got.Items["harness/safety"].Version)
	assert.True(t, got.Items["harness/safety"].Pinned)

	require.NoError(t, os.WriteFile(filepath.Join(root, lockFileName), []byte("{"), 0o600))
	_, err = ReadLock(root)
	require.Error(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(root, lockFileName), []byte(`{"items":null}`), 0o600))
	got, err = ReadLock(root)
	require.NoError(t, err)
	assert.NotNil(t, got.Items)
}

func TestCompatibleAndNewer(t *testing.T) {
	cases := []struct {
		rng, cur string
		want     bool
	}{
		{"", "2.58.0", true},
		{">=2.58.0", "2.58.0", true},
		{">=2.59.0", "2.58.2", false},
		{">= 2.1.0", "v2.58.2", true},
		{"<3.0.0", "2.58.0", false},
		{">=bad", "2.58.0", false},
		{">=9.0.0", "2.0.0-dev", true},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, compatibleWith(c.rng, c.cur), "%q vs %s", c.rng, c.cur)
	}
	assert.True(t, Newer("1.2.0", "1.1.9"))
	assert.False(t, Newer("1.1.0", "1.1.0"))
	assert.False(t, Newer("junk", "1.0.0"))
	assert.True(t, Newer("1.0.0", ""))
}

func TestInstallOverlayRemoveReset(t *testing.T) {
	registerTestSets(t)
	root := t.TempDir()

	// Source only.
	o := overlay("harness", sourceFiles(), root)
	assert.Contains(t, read(t, o, "harness/safety.yaml"), "source safety")

	// Downloaded version replaces the source file; new items appear.
	require.NoError(t, Install(root, "harness/safety", []byte("version: 1.1.0\nbody: new safety\n"), false))
	require.NoError(t, Install(root, "harness/extra", []byte("version: 1.0.0\nbody: extra\n"), true))
	o = overlay("harness", sourceFiles(), root)
	assert.Contains(t, read(t, o, "harness/safety.yaml"), "new safety")
	assert.Contains(t, read(t, o, "harness/extra.yaml"), "extra")
	entries, err := o.ReadDir("harness")
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	items, err := Items("harness", root)
	require.NoError(t, err)
	byName := map[string]Item{}
	for _, it := range items {
		byName[it.Name] = it
	}
	assert.Equal(
		t,
		Item{
			ID:          "harness/safety",
			Set:         "harness",
			Name:        "safety",
			BaseVersion: "1.0.0",
			Version:     "1.1.0",
			Downloaded:  true,
		},
		byName["safety"],
	)
	assert.True(t, byName["extra"].Pinned)
	assert.Empty(t, byName["extra"].BaseVersion)
	assert.Equal(t, "1.0.0", byName["tools"].Version)

	// A hand-edited download fails its checksum and the source comes back.
	require.NoError(
		t,
		os.WriteFile(FilePath(root, "harness", "safety"), []byte("version: 1.1.0\nbody: tampered\n"), 0o600),
	)
	o = overlay("harness", sourceFiles(), root)
	assert.Contains(t, read(t, o, "harness/safety.yaml"), "source safety")

	// Removal hides the item (source included) and deletes the download.
	require.NoError(t, Remove(root, "harness/tools"))
	require.NoError(t, Remove(root, "harness/extra"))
	o = overlay("harness", sourceFiles(), root)
	_, err = o.ReadFile("harness/tools.yaml")
	require.ErrorIs(t, err, fs.ErrNotExist)
	_, err = os.Stat(FilePath(root, "harness", "extra"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	items, _ = Items("harness", root)
	for _, it := range items {
		if it.Name == "tools" {
			assert.True(t, it.Removed)
		}
	}

	// Reset brings the source back.
	require.NoError(t, Reset(root, "harness/tools"))
	o = overlay("harness", sourceFiles(), root)
	assert.Contains(t, read(t, o, "harness/tools.yaml"), "source tools")
}

func TestOverlay_IncompatibleDownloadIgnored(t *testing.T) {
	root := t.TempDir()
	data := []byte("version: 2.0.0\nkdeps: \">=999.0.0\"\nbody: future\n")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "harness"), 0o750))
	require.NoError(t, os.WriteFile(FilePath(root, "harness", "safety"), data, 0o600))
	require.NoError(t, WriteLock(root, Lock{Items: map[string]LockEntry{
		"harness/safety": {Version: "2.0.0", SHA256: Sum(data)},
		"bogus":          {Version: "1.0.0"},
	}}))
	if compatibleWith(">=999.0.0", "2.58.0") {
		t.Fatal("test precondition")
	}
	o := overlay("harness", sourceFiles(), root)
	got := read(t, o, "harness/safety.yaml")
	if Compatible(">=999.0.0") {
		assert.Contains(t, got, "future", "dev build accepts any range")
	} else {
		assert.Contains(t, got, "source safety")
	}
}

func TestInstall_Rejections(t *testing.T) {
	registerTestSets(t)
	root := t.TempDir()
	assert.Error(t, Install(root, "nope/x", []byte("version: 1.0.0"), false))
	assert.Error(t, Install(root, "harness", []byte("version: 1.0.0"), false), "set, not item")
	assert.Error(t, Install(root, "harness/x", []byte("body: no version"), false))
	assert.Error(t, Install(root, "harness/x", []byte("version: ["), false))
	assert.ErrorContains(t, Install(root, "harness/x", []byte("version: 9.9.9"), false), "rejected by the set parser")
	if !Compatible(">=999.0.0") {
		assert.ErrorContains(
			t,
			Install(root, "harness/x", []byte("version: 1.0.0\nkdeps: \">=999.0.0\""), false),
			"needs kdeps",
		)
	}
	assert.Error(t, Remove(root, "harness"), "set, not item")
	assert.ErrorContains(t, Remove(root, "harness/core"), "required")
	// A hand-written removal of a required item is ignored by the overlay.
	require.NoError(t, WriteLock(root, Lock{Items: map[string]LockEntry{"harness/core": {Removed: true}}}))
	o := overlay("harness", fstest.MapFS{"harness/core.yaml": {Data: []byte("version: 1.0.0\n")}}, root)
	_, err := o.ReadFile("harness/core.yaml")
	require.NoError(t, err)
	assert.Error(t, Remove(root, "nope/x"))
	assert.Error(t, Reset(root, "nope/x"))
}

func TestOverlay_Bundle(t *testing.T) {
	registerTestSets(t)
	root := t.TempDir()
	bundle := []byte(`version: 1.2.0
files:
  workflow.yaml.j2: new workflow
  agents/a.yaml.j2: agent a
  ../escape.txt: nope
`)
	require.NoError(t, Install(root, "templates/api", bundle, false))
	o := overlay(BundleSet, sourceFiles(), root)
	assert.Equal(t, "new workflow", read(t, o, "templates/api/workflow.yaml.j2"))
	assert.Equal(t, "agent a", read(t, o, "templates/api/agents/a.yaml.j2"))
	assert.Contains(t, read(t, o, "templates/api/"+ManifestFile), "1.2.0")
	_, err := o.ReadFile("templates/api/resources/llm.yaml.j2")
	require.ErrorIs(t, err, fs.ErrNotExist, "bundle replaces the whole directory")
	_, err = o.ReadFile("templates/escape.txt")
	require.ErrorIs(t, err, fs.ErrNotExist, "paths outside the item are dropped")

	items, err := Items(BundleSet, root)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "1.2.0", items[0].Version)
	assert.Equal(t, "1.0.0", items[0].BaseVersion)

	require.NoError(t, Remove(root, "templates/api"))
	o = overlay(BundleSet, sourceFiles(), root)
	entries, err := o.ReadDir(BundleSet)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestOpenAndItems_UnknownSet(t *testing.T) {
	_, err := Items("never-registered", t.TempDir())
	require.Error(t, err)

	registerTestSets(t)
	t.Setenv("KDEPS_ASSETS_DIR", t.TempDir())
	assert.Contains(t, read(t, Open("harness"), "harness/safety.yaml"), "source safety")
}

func TestSourceAndRoot_UnderTest(t *testing.T) {
	t.Setenv(SourceEnv, "")
	t.Setenv("KDEPS_ASSETS_DIR", "")
	src := Source()
	require.NotEmpty(t, src, "go test loads the repo's assets/")
	_, err := os.Stat(filepath.Join(src, "harness"))
	require.NoError(t, err)
	assert.Contains(t, Root(), "kdeps-test-assets-", "tests never touch ~/.kdeps")

	t.Setenv(SourceEnv, "/x")
	assert.Equal(t, "/x", Source())
}

func TestMissingAndEnsure(t *testing.T) {
	registerTestSets(t)
	t.Setenv("KDEPS_ASSETS_DIR", t.TempDir())
	assert.NotContains(t, Missing(), "harness")

	// An empty source and store: the set is missing, and with downloads off
	// Ensure says how to install it.
	writeSource(t, fstest.MapFS{})
	assert.Contains(t, Missing(), "harness")
	t.Setenv("KDEPS_ASSETS_URL", "off")
	downloaded, err := Ensure(t.Context(), Root(), NewClient())
	assert.False(t, downloaded)
	require.ErrorContains(t, err, "run kdeps update")
}

func TestExport_StoreOfWhatLoadersSee(t *testing.T) {
	registerTestSets(t)
	root := t.TempDir()
	t.Setenv("KDEPS_ASSETS_DIR", root)
	require.NoError(t, Install(root, "harness/safety", []byte("version: 1.1.0\nbody: new safety\n"), false))
	require.NoError(t, Remove(root, "harness/tools"))

	files, err := Export()
	require.NoError(t, err)
	assert.Equal(t, "version: 1.1.0\nbody: new safety\n", string(files["harness/safety.yaml"]))
	assert.NotContains(t, files, "harness/tools.yaml", "removed items are not exported")
	var b Bundle
	require.NoError(t, yaml.Unmarshal(files["templates/api.yaml"], &b))
	assert.Equal(t, "source workflow", b.Files["workflow.yaml.j2"])

	// The export is a working store: loaded with no source, it shows the same.
	store := t.TempDir()
	for name, data := range files {
		p := filepath.Join(store, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o750))
		require.NoError(t, os.WriteFile(p, data, 0o600))
	}
	o := overlay("harness", fstest.MapFS{}, store)
	assert.Contains(t, read(t, o, "harness/safety.yaml"), "new safety")
	o = overlay(BundleSet, fstest.MapFS{}, store)
	assert.Equal(t, "source llm", read(t, o, "templates/api/resources/llm.yaml.j2"))
}
