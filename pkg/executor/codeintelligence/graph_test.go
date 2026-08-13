//go:build !js

package codeintelligence

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func writeGraphFixture(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"a.md": "---\ntopics: [go]\n---\nSee [b](b.md).",
		"b.md": "---\ntopics: [go]\n---\nNo links.",
		"c.md": "---\ntopics: [rust]\n---\nUnrelated.",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func TestExecute_IndexFolder(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)

	exec := NewExecutor()
	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpIndexFolder,
		Path:      root,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected result type %T", res)
	}
	if m["success"] != true {
		t.Fatalf("expected success, got %v", m)
	}
	if m["filesIndexed"] != 3 {
		t.Fatalf("expected 3 files indexed, got %v", m["filesIndexed"])
	}
}

func TestExecute_GraphFile(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)
	dbPath := filepath.Join(root, ".kdeps", "graph.db")

	exec := NewExecutor()
	if _, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpIndexFolder,
		Path:        root,
		GraphDBPath: dbPath,
	}); err != nil {
		t.Fatalf("indexFolder: %v", err)
	}

	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpGraphFile,
		Path:        filepath.Join(root, "a.md"),
		GraphDBPath: dbPath,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected result type %T", res)
	}
	related, ok := m["relatedByTopic"].([]string)
	if !ok || len(related) != 1 || related[0] != filepath.Join(root, "b.md") {
		t.Fatalf("unexpected relatedByTopic: %v", m["relatedByTopic"])
	}
}

func TestExecute_GraphTopic(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)

	exec := NewExecutor()
	if _, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpIndexFolder,
		Path:      root,
	}); err != nil {
		t.Fatalf("indexFolder: %v", err)
	}

	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpGraphTopic,
		Path:      root,
		Topic:     "go",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected result type %T", res)
	}
	files, ok := m["files"].([]string)
	if !ok || len(files) != 2 {
		t.Fatalf("unexpected files for topic go: %v", m["files"])
	}
}

func TestExecute_GraphAll(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)

	exec := NewExecutor()
	if _, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpIndexFolder,
		Path:      root,
	}); err != nil {
		t.Fatalf("indexFolder: %v", err)
	}

	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpGraphAll,
		Path:      root,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected result type %T", res)
	}
	roots, ok := m["roots"].([]string)
	if !ok || len(roots) != 2 {
		t.Fatalf("unexpected roots: %v", m["roots"])
	}
}

func TestExecute_GraphFile_MissingPath(t *testing.T) {
	exec := NewExecutor()
	_, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpGraphFile,
		GraphDBPath: filepath.Join(t.TempDir(), "graph.db"),
	})
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestExecute_GraphTopic_MissingTopic(t *testing.T) {
	exec := NewExecutor()
	_, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpGraphTopic,
		GraphDBPath: filepath.Join(t.TempDir(), "graph.db"),
	})
	if err == nil {
		t.Fatal("expected error for missing topic")
	}
}

func TestGraphDBPath_MissingBoth(t *testing.T) {
	_, err := graphDBPath(&domain.CodeIntelligenceConfig{})
	if err == nil {
		t.Fatal("expected error when both graphDBPath and path are empty")
	}
}
