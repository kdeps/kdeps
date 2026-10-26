package main

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionContextHasSingleRoot(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate context guard test")
	}
	root := filepath.Dir(filename)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch rel {
			case ".git", "private", "vendor", filepath.FromSlash("kdeps-io/node_modules"):
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" ||
			strings.HasSuffix(path, "_test.go") ||
			rel == filepath.FromSlash("cmd/serve.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for line := 1; scanner.Scan(); line++ {
			text := scanner.Text()
			if strings.Contains(text, "context.Background()") ||
				strings.Contains(text, "context.TODO()") {
				t.Errorf("%s:%d creates a context outside cmd/serve.go", rel, line)
			}
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
}
