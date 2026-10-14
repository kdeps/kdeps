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

package executor_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The file component returns the structured results the file: resource did,
// for every operation.
func TestFileComponent_Operations(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not installed (python resources)")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("KDEPS_COMPONENT_DIR", t.TempDir())
	root := t.TempDir()
	at := func(rel string) string { return filepath.ToSlash(filepath.Join(root, rel)) }
	run := func(with string) map[string]interface{} {
		t.Helper()
		out, err := runComponentCall(t, "file", with)
		require.NoError(t, err)
		return out
	}

	w := run(
		"    operation: write\n    path: \"" + at(
			"d/a.txt",
		) + "\"\n    content: \"one\\ntwo\"\n    appendNewline: true\n",
	)
	assert.Equal(t, true, w["written"])
	assert.Equal(t, false, w["exists"])
	assert.EqualValues(t, 8, w["size"])
	assert.Equal(t, "-rw-r--r--", w["mode"])

	r := run("    operation: read\n    path: \"" + at("d/a.txt") + "\"\n")
	assert.Equal(t, "one\ntwo\n", r["content"])
	assert.Equal(t, []interface{}{"one", "two", ""}, r["lines"])
	run("    operation: write\n    path: \"" + at("b.bin") + "\"\n    content: this\n")
	assert.Equal(
		t,
		"dGhpcw==",
		run("    operation: read\n    encoding: base64\n    path: \"" + at("b.bin") + "\"\n")["content"],
	)

	p := run(
		"    operation: patch\n    backup: true\n    path: \"" + at(
			"d/a.txt",
		) + "\"\n    patch: \"@@ -1,2 +1,2 @@\\n one\\n-two\\n+TWO\\n\"\n",
	)
	assert.Equal(t, true, p["patched"])
	got, err := os.ReadFile(filepath.Join(root, "d", "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "one\nTWO\n", string(got))
	bak, err := os.ReadFile(filepath.Join(root, "d", "a.txt.bak"))
	require.NoError(t, err)
	assert.Equal(t, "one\ntwo\n", string(bak))

	_, err = runComponentCall(t, "file",
		"    operation: patch\n    path: \""+at("d/a.txt")+"\"\n    patch: \"@@ -1 +1 @@\\n-nope\\n+x\\n\"\n")
	require.Error(t, err, "a patch that does not match fails the resource")

	run("    operation: append\n    path: \"" + at("d/a.txt") + "\"\n    content: three\n    appendNewline: true\n")
	assert.Equal(t, "one\nTWO\nthree\n", run("    operation: read\n    path: \"" + at("d/a.txt") + "\"\n")["content"])

	run("    operation: mkdir\n    path: \"" + at("d/sub") + "\"\n")
	assert.Equal(
		t,
		true,
		run("    operation: copy\n    source: \"" + at("d/a.txt") + "\"\n    path: \"" + at("d/sub/c.txt") + "\"\n")["copied"],
	)
	l := run("    operation: list\n    recursive: true\n    pattern: \"*.txt\"\n    path: \"" + at("d") + "\"\n")
	assert.EqualValues(t, 2, l["count"], "a.txt and sub/c.txt; the .bak and the directory do not match")

	assert.Equal(
		t,
		true,
		run("    operation: move\n    source: \"" + at("d/sub/c.txt") + "\"\n    path: \"" + at("e/c.txt") + "\"\n")["moved"],
	)
	assert.Equal(t, true, run("    operation: exists\n    path: \"" + at("e/c.txt") + "\"\n")["exists"])
	missing := run("    operation: exists\n    path: \"" + at("d/sub/c.txt") + "\"\n")
	assert.Equal(t, false, missing["success"])

	dry := run("    operation: delete\n    dryRun: true\n    path: \"" + at("e") + "\"\n")
	assert.Equal(t, true, dry["dryRun"])
	assert.DirExists(t, filepath.Join(root, "e"))
	assert.Equal(t, true, run("    operation: delete\n    path: \"" + at("e") + "\"\n")["deleted"])
	assert.Equal(t, "not_found", run("    operation: delete\n    path: \"" + at("e") + "\"\n")["reason"])
}
