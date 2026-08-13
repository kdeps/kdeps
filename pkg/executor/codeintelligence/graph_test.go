//go:build !js

package codeintelligence

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"

	"github.com/kdeps/kartographer/graph"

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

func TestExecuteGraph_UnsupportedOperation(t *testing.T) {
	exec := NewExecutor()
	dbPath := filepath.Join(t.TempDir(), "graph.db")
	_, err := exec.executeGraph(&domain.CodeIntelligenceConfig{
		Operation:   "bogus",
		GraphDBPath: dbPath,
	})
	if err == nil {
		t.Fatal("expected error for unsupported graph operation")
	}
}

func TestExecuteGraph_DBPathError(t *testing.T) {
	exec := NewExecutor()
	_, err := exec.executeGraph(&domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpGraphAll,
	})
	if err == nil {
		t.Fatal("expected error when graphDBPath and path are both empty")
	}
}

func TestExecuteGraph_MkdirError(t *testing.T) {
	root := t.TempDir()
	// A regular file where a directory component is expected: MkdirAll
	// must fail trying to create a child under it.
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	exec := NewExecutor()
	_, err := exec.executeGraph(&domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpGraphAll,
		GraphDBPath: filepath.Join(blocker, "sub", "graph.db"),
	})
	if err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestExecuteGraph_OpenDBError(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "graph.db")
	// A non-bbolt file at dbPath: bolt.Open must fail to read its header.
	if err := os.WriteFile(dbPath, []byte("not a bbolt database"), 0o600); err != nil {
		t.Fatalf("write bogus db: %v", err)
	}

	exec := NewExecutor()
	_, err := exec.executeGraph(&domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpGraphAll,
		GraphDBPath: dbPath,
	})
	if err == nil {
		t.Fatal("expected bbolt open error")
	}
}

func TestGraphIndexFolder_MissingPath(t *testing.T) {
	_, err := graphIndexFolder(nil, &domain.CodeIntelligenceConfig{})
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

// failOpenFs wraps an afero.Fs and fails Open for one specific path, letting
// Walk (which uses Stat/ReadDir, not Open) succeed while the subsequent
// ReadFile fails -- deterministically exercising IndexFolder's error path.
type failOpenFs struct {
	afero.Fs
	failPath string
}

func (f *failOpenFs) Open(name string) (afero.File, error) {
	if name == f.failPath {
		return nil, errors.New("simulated open failure")
	}
	return f.Fs.Open(name)
}

func TestGraphIndexFolder_ReadError(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)
	dbPath := filepath.Join(t.TempDir(), "graph.db")

	failPath := filepath.Join(root, "a.md")
	origFS := AppFS
	AppFS = &failOpenFs{Fs: origFS, failPath: failPath}
	defer func() { AppFS = origFS }()

	ig, err := graph.NewIndexedGraph(AppFS, nil, dbPath)
	if err != nil {
		t.Fatalf("NewIndexedGraph: %v", err)
	}
	defer ig.Close()

	_, err = graphIndexFolder(ig, &domain.CodeIntelligenceConfig{Path: root})
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestGraphFile_StoreError(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)
	dbPath := filepath.Join(t.TempDir(), "graph.db")

	ig, err := graph.NewIndexedGraph(afero.NewOsFs(), nil, dbPath)
	if err != nil {
		t.Fatalf("NewIndexedGraph: %v", err)
	}
	if _, indexErr := ig.IndexFolder(root, nil); indexErr != nil {
		t.Fatalf("IndexFolder: %v", indexErr)
	}
	if closeErr := ig.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	_, err = graphFile(ig, &domain.CodeIntelligenceConfig{Path: filepath.Join(root, "a.md")})
	if err == nil {
		t.Fatal("expected error querying a closed graph store")
	}
}

func TestGraphTopic_StoreError(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)
	dbPath := filepath.Join(t.TempDir(), "graph.db")

	ig, err := graph.NewIndexedGraph(afero.NewOsFs(), nil, dbPath)
	if err != nil {
		t.Fatalf("NewIndexedGraph: %v", err)
	}
	if _, indexErr := ig.IndexFolder(root, nil); indexErr != nil {
		t.Fatalf("IndexFolder: %v", indexErr)
	}
	if closeErr := ig.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	_, err = graphTopic(ig, &domain.CodeIntelligenceConfig{Topic: "go"})
	if err == nil {
		t.Fatal("expected error querying a closed graph store")
	}
}

func TestGraphAll_StoreError(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)
	dbPath := filepath.Join(t.TempDir(), "graph.db")

	ig, err := graph.NewIndexedGraph(afero.NewOsFs(), nil, dbPath)
	if err != nil {
		t.Fatalf("NewIndexedGraph: %v", err)
	}
	if _, indexErr := ig.IndexFolder(root, nil); indexErr != nil {
		t.Fatalf("IndexFolder: %v", indexErr)
	}
	if closeErr := ig.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	_, err = graphAll(ig)
	if err == nil {
		t.Fatal("expected error querying a closed graph store")
	}
}
