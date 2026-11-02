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

package knowledge

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kdeps/kartographer/codegraph"
)

// maxCodeFiles caps the Go files read per repo.
const maxCodeFiles = 20000

// addCode adds a node per Go package of each card's repo, with depends_on
// edges for imports inside the module, and links the card's note to the
// module root. Code goes through the Go parser only.
func (b *builder) addCode(ctx context.Context, v Vault) {
	cards, err := v.Cards()
	if err != nil {
		return
	}
	for _, c := range cards {
		repo := c.Link("repo")
		if repo == "" {
			continue
		}
		root := ExpandPath(repo, v.Root)
		module := goModule(root)
		if module == "" {
			continue
		}
		rootID := "code:" + module
		b.add(&Node{ID: rootID, Label: module, Kind: KindCode})
		if rel, relErr := filepath.Rel(v.Root, c.File); relErr == nil && !strings.HasPrefix(rel, "..") {
			b.link(
				noteID(filepath.ToSlash(rel)), rootID, RelReferences,
				filepath.ToSlash(rel), lineRef(lineInFile(c.File, repo)),
			)
		}
		b.addModule(ctx, root)
	}
}

// addModule adds a Go module's packages: the files git tracks (or would,
// honouring .gitignore), else every .go file outside vendor, testdata and
// hidden folders. Kartographer reads each file's in-module imports.
func (b *builder) addModule(ctx context.Context, root string) {
	files := gitGoFiles(ctx, root)
	if files == nil {
		files = walkGoFiles(root)
	}
	if len(files) > maxCodeFiles {
		files = files[:maxCodeFiles]
	}
	rels := make([]string, 0, len(files))
	for _, p := range files {
		if rel, err := filepath.Rel(root, p); err == nil {
			rels = append(rels, rel)
		}
	}
	b.st.CodeFiles += len(rels)
	pkgs, err := codegraph.ModuleImports(root, rels)
	if err != nil {
		return
	}
	for _, fi := range pkgs {
		b.addPackageFile(root, fi)
	}
}

// addPackageFile adds a Go file's package and its in-module imports. Only
// imports connect packages: an edge the code does not write is not added.
func (b *builder) addPackageFile(root string, fi codegraph.FileImports) {
	id := "code:" + fi.Package
	b.add(&Node{ID: id, Label: fi.Package, Kind: KindCode})
	file := filepath.Join(root, filepath.FromSlash(fi.File))
	for i, imp := range fi.Imports {
		b.add(&Node{ID: "code:" + imp, Label: imp, Kind: KindCode})
		b.link(id, "code:"+imp, RelDependsOn, file, lineRef(fi.Lines[i]))
	}
}

// gitGoFiles lists root's .go files that git tracks or would track, or nil
// when root is not in a git work tree, git is missing, or git lists none
// (root ignored by an enclosing repo): the caller then walks the folder.
func gitGoFiles(ctx context.Context, root string) []string {
	cmd := exec.CommandContext(
		ctx,
		"git",
		"-C",
		root,
		"ls-files",
		"--cached",
		"--others",
		"--exclude-standard",
		"-z",
		"--",
		"*.go",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f == "" || strings.Contains("/"+f, "/vendor/") || strings.Contains("/"+f, "/testdata/") {
			continue
		}
		files = append(files, filepath.Join(root, filepath.FromSlash(f)))
	}
	return files
}

// walkGoFiles lists root's .go files outside vendor, testdata and hidden
// folders.
func walkGoFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil //nolint:nilerr // unreadable paths are skipped
		case d.IsDir():
			if p != root && skipCodeDir(d.Name()) {
				return filepath.SkipDir
			}
		case strings.HasSuffix(d.Name(), ".go"):
			files = append(files, p)
		}
		return nil
	})
	return files
}

func skipCodeDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "testdata"
}

// goModule is the module path in root/go.mod, or "".
func goModule(root string) string {
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if m, ok := strings.CutPrefix(strings.TrimSpace(s.Text()), "module "); ok {
			return strings.Trim(strings.TrimSpace(m), `"`)
		}
	}
	return ""
}
