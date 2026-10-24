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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPositiveNumber(t *testing.T) {
	for _, v := range []any{float64(2), 2, int64(2), json.Number("2"), " 2 "} {
		if n, ok := positiveNumber(v); !ok || n != 2 {
			t.Fatalf("%v -> %d %v", v, n, ok)
		}
	}
	for _, v := range []any{float64(0), float64(0.4), "0", "nope", "  ", json.Number("x"), json.Number("0"), true, nil, -1} {
		if _, ok := positiveNumber(v); ok {
			t.Fatalf("accepted %v", v)
		}
	}
	if stringArg(nil, argFilePath) != "" || stringArg(map[string]any{argFilePath: 3}, argFilePath) != "" {
		t.Fatal("stringArg")
	}
}

func TestValidateRootPath(t *testing.T) {
	if err := validateRootPath("/"); err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatal(err)
	}
	if err := validateRootPath(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "rootlink")
	if err := os.Symlink("/", link); err != nil {
		t.Fatal(err)
	}
	if err := validateRootPath(link); err == nil || !strings.Contains(err.Error(), "resolves to root") {
		t.Fatalf("symlink %v", err)
	}
	t.Setenv(envAllowRoot, "true")
	if err := validateRootPath("/"); err != nil {
		t.Fatalf("allowed root %v", err)
	}
}

func TestValidateWorkspace(t *testing.T) {
	if err := validateWorkspace("/anywhere"); err != nil {
		t.Fatal("no root set allows every path")
	}
	root := t.TempDir()
	t.Setenv(envWorkspaceRoot, root)
	if err := validateWorkspace(filepath.Join(root, "in.txt")); err != nil {
		t.Fatal(err)
	}
	err := validateWorkspace(filepath.Join(t.TempDir(), "out.txt"))
	if err == nil || !strings.Contains(err.Error(), "escapes workspace root") {
		t.Fatal(err)
	}
}

func TestPathWithinRoot(t *testing.T) {
	root := t.TempDir()
	if !PathWithinRoot(root, root) {
		t.Fatal("the root is inside itself")
	}
	// A path that does not exist yet is still inside: its longest existing
	// ancestor is the root, which resolves through a symlink on macOS where
	// the tempdir (and /tmp) live under /private.
	if !PathWithinRoot(filepath.Join(root, "new.txt"), root) {
		t.Fatal("a missing file directly under the root is inside it")
	}
	if !PathWithinRoot(filepath.Join(root, "a", "b", "new.txt"), root) {
		t.Fatal("a missing nested path under the root is inside it")
	}
	if PathWithinRoot(filepath.Join(root, "..", "sibling.txt"), root) {
		t.Fatal("a parent-relative escape is outside the root")
	}
	outside := t.TempDir()
	if PathWithinRoot(filepath.Join(outside, "new.txt"), root) {
		t.Fatal("a missing file in another tree is outside the root")
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if PathWithinRoot(filepath.Join(link, "new.txt"), root) {
		t.Fatal("a missing file behind an escaping symlink is outside the root")
	}
}
