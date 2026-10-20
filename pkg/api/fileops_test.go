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

package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fileResult(t *testing.T, args map[string]any) map[string]any {
	t.Helper()
	out, err := FileOps(context.Background(), args)
	if err != nil {
		t.Fatalf("file_ops %+v: %v", args, err)
	}
	var result map[string]any
	if err = json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json: %v %s", err, out)
	}
	return result
}

func fileError(t *testing.T, args map[string]any) string {
	t.Helper()
	_, err := FileOps(context.Background(), args)
	if err == nil {
		t.Fatalf("expected an error for %+v", args)
	}
	return err.Error()
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFileOps_NilContextAndUnsupported(t *testing.T) {
	var nilCtx context.Context
	if _, err := FileOps(nilCtx, nil); err == nil || err.Error() != "file_ops: nil context" {
		t.Fatalf("nil context: %v", err)
	}
	if got := fileError(t, map[string]any{argOperation: "bogus"}); got != "file: unsupported operation 'bogus'" {
		t.Fatalf("unsupported: %s", got)
	}
	if got := fileError(t, map[string]any{}); got != "file: unsupported operation ''" {
		t.Fatalf("empty op: %s", got)
	}
}

func TestFileOps_Read(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")
	writeTestFile(t, p, "alpha\nbeta\n")

	result := fileResult(t, map[string]any{argOperation: "read", argPath: p})
	if result["success"] != true || result["content"] != "alpha\nbeta\n" || result["encoding"] != "text" ||
		result["size"] != float64(11) || result["exists"] != true || result["path"] != p {
		t.Fatalf("read: %+v", result)
	}
	lines, _ := result["lines"].([]any)
	if len(lines) != 3 || lines[0] != "alpha" || lines[1] != "beta" || lines[2] != "" {
		t.Fatalf("lines: %+v", result["lines"])
	}

	raw := []byte{0xff, 0xfe, 'o', 'k'}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := fileResult(t, map[string]any{argOperation: "read", argPath: p})["content"]; got != "�ok" {
		t.Fatalf("invalid utf8: %q", got)
	}

	data := []byte{0x00, 0x01, 0x02}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	encoded := fileResult(t, map[string]any{argOperation: "read", argPath: p, argEncoding: "base64"})
	if encoded["encoding"] != "base64" || encoded["content"] != base64.StdEncoding.EncodeToString(data) {
		t.Fatalf("base64: %+v", encoded)
	}

	if got := fileError(t, map[string]any{argOperation: "read"}); got != "file: path is required for read operation" {
		t.Fatalf("no path: %s", got)
	}
	missingRead := map[string]any{argOperation: "read", argPath: filepath.Join(dir, "missing")}
	if got := fileError(t, missingRead); !strings.HasPrefix(got, "file: ") {
		t.Fatalf("missing: %s", got)
	}
}

func TestFileOps_Write(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "notes.txt")

	result := fileResult(t, map[string]any{
		argOperation: "write", argPath: p, argContent: "alpha", argAppendNewline: "true",
	})
	if result["written"] != true || result["exists"] != false || result["size"] != float64(6) ||
		result["backup"] != false || result["backupPath"] != nil || result["mode"] != "-rw-r--r--" {
		t.Fatalf("write: %+v", result)
	}
	if got, _ := os.ReadFile(p); string(got) != "alpha\n" {
		t.Fatalf("content: %q", got)
	}

	second := fileResult(t, map[string]any{
		argOperation: "write", argPath: p, argContent: "beta", argBackup: true, argMode: "0600",
	})
	if second["exists"] != true || second["backup"] != true ||
		second["backupPath"] != p+".bak" || second["mode"] != "-rw-------" {
		t.Fatalf("second write: %+v", second)
	}
	if got, _ := os.ReadFile(p + ".bak"); string(got) != "alpha\n" {
		t.Fatalf("backup: %q", got)
	}

	dry := fileResult(t, map[string]any{
		argOperation: "write", argPath: p, argContent: "gamma", argDryRun: true,
	})
	if dry["dryRun"] != true || dry["written"] != false || dry["exists"] != true {
		t.Fatalf("dry: %+v", dry)
	}
	if got, _ := os.ReadFile(p); string(got) != "beta" {
		t.Fatalf("dry changed the file: %q", got)
	}

	if got := fileError(t, map[string]any{argOperation: "write", argContent: "x"}); got !=
		"file: path is required for write operation" {
		t.Fatalf("no path: %s", got)
	}
}

const testPatch = "@@ -1,2 +1,2 @@\n alpha\n-beta\n+gamma\n"

func TestFileOps_Patch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")
	writeTestFile(t, p, "alpha\nbeta\n")

	result := fileResult(t, map[string]any{argOperation: "patch", argPath: p, argPatch: testPatch})
	if result["patched"] != true || result["patchLines"] != float64(5) {
		t.Fatalf("patch: %+v", result)
	}
	if got, _ := os.ReadFile(p); string(got) != "alpha\ngamma\n" {
		t.Fatalf("patched content: %q", got)
	}

	if got := fileError(t, map[string]any{
		argOperation: "patch", argPath: p, argPatch: "@@ -1 +1 @@\n-nope\n+x\n",
	}); !strings.Contains(got, "file: patch failed: patch removal mismatch") {
		t.Fatalf("mismatch: %s", got)
	}
	if got, _ := os.ReadFile(p); string(got) != "alpha\ngamma\n" {
		t.Fatalf("failed patch changed the file: %q", got)
	}
	if got := fileError(t, map[string]any{argOperation: "patch", argPath: p}); got !=
		"file: patch content is required for patch operation" {
		t.Fatalf("no patch: %s", got)
	}
	if got := fileError(t, map[string]any{
		argOperation: "patch", argPath: p, argPatch: "not a diff\n",
	}); got != "file: patch failed: no valid hunks found in patch" {
		t.Fatalf("no hunks: %s", got)
	}
	if got := fileError(t, map[string]any{
		argOperation: "patch", argPath: p, argPatch: "@@ -x +1 @@\n",
	}); got != "file: patch failed: invalid hunk header: @@ -x +1 @@" {
		t.Fatalf("bad header: %s", got)
	}
	if got := fileError(t, map[string]any{
		argOperation: "patch", argPath: p, argPatch: "@@ -1,1 +1,1 @@\n too far\n",
	}); !strings.Contains(got, "patch context mismatch") {
		t.Fatalf("context mismatch: %s", got)
	}
	if got := fileError(t, map[string]any{
		argOperation: "patch", argPath: p, argPatch: "@@ -9,1 +9,1 @@\n far\n",
	}); !strings.Contains(got, "patch context line out of range") {
		t.Fatalf("context out of range: %s", got)
	}

	dry := fileResult(t, map[string]any{
		argOperation: "patch", argPath: p, argPatch: "@@ -1 +1 @@\n-alpha\n+delta\n", argDryRun: true,
	})
	if dry["dryRun"] != true || dry["patched"] != false {
		t.Fatalf("dry: %+v", dry)
	}
	if got, _ := os.ReadFile(p); string(got) != "alpha\ngamma\n" {
		t.Fatalf("dry changed the file: %q", got)
	}

	backed := fileResult(t, map[string]any{
		argOperation: "patch", argPath: p, argPatch: "@@ -1 +1 @@\n-alpha\n+delta\n", argBackup: true,
	})
	if backed["patched"] != true {
		t.Fatalf("backup patch: %+v", backed)
	}
	if got, _ := os.ReadFile(p + ".bak"); string(got) != "alpha\ngamma\n" {
		t.Fatalf("patch backup: %q", got)
	}
	if got, _ := os.ReadFile(p); string(got) != "delta\ngamma\n" {
		t.Fatalf("backup patch content: %q", got)
	}

	unterminated := filepath.Join(dir, "no-newline.txt")
	writeTestFile(t, unterminated, "alpha")
	fileResult(t, map[string]any{
		argOperation: "patch", argPath: unterminated, argPatch: "@@ -1 +1 @@\n-alpha\n+beta\n",
	})
	if got, _ := os.ReadFile(unterminated); string(got) != "beta" {
		t.Fatalf("trailing newline changed: %q", got)
	}
}

func TestFileOps_List(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.txt"), "a")
	writeTestFile(t, filepath.Join(dir, "b.go"), "b")
	writeTestFile(t, filepath.Join(dir, "sub", "c.txt"), "c")

	all := fileResult(t, map[string]any{argOperation: "list", argPath: dir})
	if all["count"] != float64(3) || all["path"] != dir {
		t.Fatalf("list: %+v", all)
	}
	entries, _ := all["entries"].([]any)
	if len(entries) != 3 {
		t.Fatalf("entries: %+v", all["entries"])
	}
	first, _ := entries[0].(map[string]any)
	if first["name"] != "a.txt" || first["path"] != filepath.Join(dir, "a.txt") ||
		first["isDir"] != false || first["size"] != float64(1) ||
		first["mode"] != "-rw-r--r--" || first["modTime"] == "" {
		t.Fatalf("entry: %+v", first)
	}

	filtered := fileResult(t, map[string]any{argOperation: "list", argPath: dir, argPattern: "*.txt"})
	if filtered["count"] != float64(1) {
		t.Fatalf("pattern: %+v", filtered)
	}
	recursive := fileResult(t, map[string]any{argOperation: "list", argPath: dir, argRecursive: true})
	if recursive["count"] != float64(4) {
		t.Fatalf("recursive: %+v", recursive)
	}
	recursive = fileResult(t, map[string]any{
		argOperation: "list", argPath: dir, argRecursive: true, argPattern: "*.txt",
	})
	if recursive["count"] != float64(2) {
		t.Fatalf("recursive pattern: %+v", recursive)
	}
	single := fileResult(t, map[string]any{
		argOperation: "list", argPath: filepath.Join(dir, "a.txt"),
	})
	if single["count"] != float64(1) {
		t.Fatalf("file: %+v", single)
	}
	if got := fileError(t, map[string]any{
		argOperation: "list", argPath: filepath.Join(dir, "missing"),
	}); got != "file: cannot access path: "+filepath.Join(dir, "missing") {
		t.Fatalf("missing: %s", got)
	}
}

func TestFileOps_DeleteExistsMkdir(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "gone.txt")
	writeTestFile(t, p, "x")

	dry := fileResult(t, map[string]any{argOperation: "delete", argPath: p, argDryRun: true})
	if dry["deleted"] != false || dry["dryRun"] != true {
		t.Fatalf("dry delete: %+v", dry)
	}
	deleted := fileResult(t, map[string]any{argOperation: "delete", argPath: p})
	if deleted["deleted"] != true {
		t.Fatalf("delete: %+v", deleted)
	}
	missing := fileResult(t, map[string]any{argOperation: "delete", argPath: p})
	if missing["deleted"] != false || missing["reason"] != "not_found" || missing["success"] != true {
		t.Fatalf("missing delete: %+v", missing)
	}
	tree := filepath.Join(dir, "tree")
	writeTestFile(t, filepath.Join(tree, "x.txt"), "x")
	fileResult(t, map[string]any{argOperation: "delete", argPath: tree})
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Fatalf("tree still exists: %v", err)
	}

	if got := fileResult(t, map[string]any{argOperation: "exists", argPath: dir}); got["exists"] != true ||
		got["isDir"] != true || got["success"] != true {
		t.Fatalf("exists: %+v", got)
	}
	if got := fileResult(t, map[string]any{
		argOperation: "exists", argPath: filepath.Join(dir, "nope"),
	}); got["exists"] != false || got["success"] != false {
		t.Fatalf("not exists: %+v", got)
	}

	mkdir := fileResult(t, map[string]any{
		argOperation: "mkdir", argPath: filepath.Join(dir, "a", "b"), argMode: "0700",
	})
	if mkdir["created"] != true || mkdir["mode"] != "-rwx------" {
		t.Fatalf("mkdir: %+v", mkdir)
	}
	if info, err := os.Stat(filepath.Join(dir, "a", "b")); err != nil || !info.IsDir() {
		t.Fatalf("mkdir missing: %v", err)
	}
	if got := fileResult(t, map[string]any{
		argOperation: "mkdir", argPath: filepath.Join(dir, "dry"), argDryRun: true,
	}); got["created"] != false || got["dryRun"] != true {
		t.Fatalf("dry mkdir: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "dry")); !os.IsNotExist(err) {
		t.Fatalf("dry mkdir created it: %v", err)
	}
}

func TestFileOps_CopyMoveAppend(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	writeTestFile(t, src, "data")

	copied := fileResult(t, map[string]any{
		argOperation: "copy", argSource: src, argPath: filepath.Join(dir, "out", "dst.txt"),
	})
	if copied["copied"] != true || copied["dest"] != filepath.Join(dir, "out", "dst.txt") {
		t.Fatalf("copy: %+v", copied)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "out", "dst.txt")); string(got) != "data" {
		t.Fatalf("copied content: %q", got)
	}
	tree := filepath.Join(dir, "tree")
	writeTestFile(t, filepath.Join(tree, "one.txt"), "1")
	fileResult(t, map[string]any{
		argOperation: "copy", argSource: tree, argPath: filepath.Join(dir, "tree2"),
	})
	if got, _ := os.ReadFile(filepath.Join(dir, "tree2", "one.txt")); string(got) != "1" {
		t.Fatalf("copied tree: %q", got)
	}
	if got := fileError(t, map[string]any{
		argOperation: "copy", argSource: filepath.Join(dir, "nope"), argPath: filepath.Join(dir, "x"),
	}); got != "file: source not accessible: "+filepath.Join(dir, "nope") {
		t.Fatalf("copy missing: %s", got)
	}
	if got := fileResult(t, map[string]any{
		argOperation: "copy", argSource: src, argPath: filepath.Join(dir, "dry.txt"), argDryRun: true,
	}); got["copied"] != false || got["dryRun"] != true {
		t.Fatalf("dry copy: %+v", got)
	}

	moved := fileResult(t, map[string]any{
		argOperation: "move", argSource: src, argPath: filepath.Join(dir, "moved.txt"),
	})
	if moved["moved"] != true {
		t.Fatalf("move: %+v", moved)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("src still exists: %v", err)
	}
	into := filepath.Join(dir, "into")
	if err := os.MkdirAll(into, 0o755); err != nil {
		t.Fatal(err)
	}
	fileResult(t, map[string]any{
		argOperation: "move", argSource: filepath.Join(dir, "moved.txt"), argPath: into,
	})
	if got, _ := os.ReadFile(filepath.Join(into, "moved.txt")); string(got) != "data" {
		t.Fatalf("move into dir: %q", got)
	}
	if got := fileError(t, map[string]any{
		argOperation: "move", argSource: filepath.Join(dir, "nope"), argPath: filepath.Join(dir, "y"),
	}); got != "file: source does not exist: "+filepath.Join(dir, "nope") {
		t.Fatalf("move missing: %s", got)
	}

	appended := fileResult(t, map[string]any{
		argOperation: "append", argPath: filepath.Join(dir, "log.txt"), argContent: "one",
		argAppendNewline: true,
	})
	if appended["appended"] != true || appended["size"] != float64(4) {
		t.Fatalf("append: %+v", appended)
	}
	fileResult(t, map[string]any{
		argOperation: "append", argPath: filepath.Join(dir, "log.txt"), argContent: "two",
		argAppendNewline: "false",
	})
	if got, _ := os.ReadFile(filepath.Join(dir, "log.txt")); string(got) != "one\ntwo" {
		t.Fatalf("appended content: %q", got)
	}
	if got := fileResult(t, map[string]any{
		argOperation: "append", argPath: filepath.Join(dir, "log.txt"), argContent: "x", argDryRun: true,
	}); got["appended"] != false || got["dryRun"] != true {
		t.Fatalf("dry append: %+v", got)
	}
}
