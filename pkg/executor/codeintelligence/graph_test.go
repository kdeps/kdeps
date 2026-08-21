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

// withCwd overrides the package-level getCwd for the duration of the test,
// so indexFolder (which always indexes "the current working directory") can
// be pointed at a fixture directory without touching the real process cwd.
func withCwd(t *testing.T, dir string) {
	t.Helper()
	orig := getCwd
	getCwd = func() (string, error) { return dir, nil }
	t.Cleanup(func() { getCwd = orig })
}

func withCwdErr(t *testing.T, msg string) {
	t.Helper()
	orig := getCwd
	getCwd = func() (string, error) { return "", errors.New(msg) }
	t.Cleanup(func() { getCwd = orig })
}

func TestExecute_IndexFolder(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)
	withCwd(t, root)

	exec := NewExecutor()
	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpIndexFolder,
		GraphDBPath: filepath.Join(t.TempDir(), "graph.db"),
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
	if m["path"] != root {
		t.Fatalf("expected indexed path to be the CWD override %q, got %v", root, m["path"])
	}
}

func TestExecute_IndexFolder_IgnoresPath(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)
	withCwd(t, root)

	// A Path pointing somewhere else must be ignored: indexFolder only ever
	// indexes the CWD.
	elsewhere := t.TempDir()
	if err := os.WriteFile(filepath.Join(elsewhere, "unrelated.md"), []byte("nope"), 0o600); err != nil {
		t.Fatalf("write unrelated.md: %v", err)
	}

	exec := NewExecutor()
	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpIndexFolder,
		Path:        elsewhere,
		GraphDBPath: filepath.Join(t.TempDir(), "graph.db"),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m := res.(map[string]interface{})
	if m["filesIndexed"] != 3 {
		t.Fatalf("expected 3 files indexed from the CWD override, got %v", m["filesIndexed"])
	}
}

func TestExecute_GraphFile(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)
	withCwd(t, root)
	dbPath := filepath.Join(t.TempDir(), "graph.db")

	exec := NewExecutor()
	if _, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpIndexFolder,
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
	withCwd(t, root)
	dbPath := filepath.Join(t.TempDir(), "graph.db")

	exec := NewExecutor()
	if _, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpIndexFolder,
		GraphDBPath: dbPath,
	}); err != nil {
		t.Fatalf("indexFolder: %v", err)
	}

	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpGraphTopic,
		Topic:       "go",
		GraphDBPath: dbPath,
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
	withCwd(t, root)
	dbPath := filepath.Join(t.TempDir(), "graph.db")

	exec := NewExecutor()
	if _, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpIndexFolder,
		GraphDBPath: dbPath,
	}); err != nil {
		t.Fatalf("indexFolder: %v", err)
	}

	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpGraphAll,
		GraphDBPath: dbPath,
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

func TestExecute_GraphAll_DefaultsDBPathToCWD(t *testing.T) {
	root := t.TempDir()
	writeGraphFixture(t, root)
	withCwd(t, root)

	exec := NewExecutor()
	if _, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpIndexFolder,
	}); err != nil {
		t.Fatalf("indexFolder: %v", err)
	}

	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpGraphAll,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m := res.(map[string]interface{})
	roots, ok := m["roots"].([]string)
	if !ok || len(roots) != 2 {
		t.Fatalf("unexpected roots: %v", m["roots"])
	}
	if _, statErr := os.Stat(filepath.Join(root, ".kdeps", "graph.db")); statErr != nil {
		t.Fatalf("expected graph.db under the CWD override: %v", statErr)
	}
}

// TestExecute_IndexFolder_SourceCodeExtensions verifies that passing
// extensions: [".go"] to indexFolder opts into kartographer's source-code
// import extraction (added in kartographer v0.2.0), on top of the default
// markdown/docs behavior exercised by the other tests in this file.
func TestExecute_IndexFolder_SourceCodeExtensions(t *testing.T) {
	root := t.TempDir()
	// Named differently from its own directory so the resolution isn't
	// accidentally ambiguous between the file and its containing dir.
	pkgDir := filepath.Join(root, "pkgname")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "impl.go"), []byte("package pkgname\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(`package main

import "example.com/root/pkgname"

func main() {}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	withCwd(t, root)
	dbPath := filepath.Join(t.TempDir(), "graph.db")

	exec := NewExecutor()
	if _, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpIndexFolder,
		Extensions:  []string{".go"},
		GraphDBPath: dbPath,
	}); err != nil {
		t.Fatalf("indexFolder: %v", err)
	}

	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpGraphFile,
		Path:        filepath.Join(root, "main.go"),
		GraphDBPath: dbPath,
	})
	if err != nil {
		t.Fatalf("graphFile: %v", err)
	}
	m := res.(map[string]interface{})
	references, ok := m["references"].(map[string][]string)
	if !ok {
		t.Fatalf("unexpected references type %T", m["references"])
	}
	got := references[filepath.Join(root, "main.go")]
	if len(got) != 1 || got[0] != pkgDir {
		t.Fatalf("expected main.go's import resolved to %q, got %v", pkgDir, got)
	}
}

// HTML/PDF indexing for code_index_folder is opt-in via an explicit
// extensions list (["...", ".html"]), not the tool's default -- the default
// stays narrow (kartographer's own .md/.markdown/.txt/.yaml/.yml) so that
// pointing indexFolder at a project root never silently walks an entire
// source/build tree by surprise, per its documented design.
func TestExecute_IndexFolder_HTMLRequiresExplicitOptIn(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "page.html"), []byte("<html>hi</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	withCwd(t, root)
	dbPath := filepath.Join(t.TempDir(), "graph.db")

	exec := NewExecutor()
	res, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpIndexFolder,
		GraphDBPath: dbPath,
	})
	if err != nil {
		t.Fatalf("indexFolder: %v", err)
	}
	m := res.(map[string]interface{})
	filesIndexed, _ := m["filesIndexed"].(int)
	if filesIndexed != 0 {
		t.Fatalf("filesIndexed = %v, want 0 (.html not in the default extension set)", filesIndexed)
	}

	res, err = exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation:   domain.CodeIntOpIndexFolder,
		Extensions:  []string{".html"},
		GraphDBPath: dbPath,
	})
	if err != nil {
		t.Fatalf("indexFolder with explicit extensions: %v", err)
	}
	m = res.(map[string]interface{})
	filesIndexed, _ = m["filesIndexed"].(int)
	if filesIndexed != 1 {
		t.Fatalf("filesIndexed = %v, want 1 once .html is explicitly opted in", filesIndexed)
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

func TestGraphDBPath_ExplicitGraphDBPath(t *testing.T) {
	want := filepath.Join(t.TempDir(), "graph.db")
	got, err := graphDBPath(&domain.CodeIntelligenceConfig{GraphDBPath: want})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGraphDBPath_DefaultsToCWD(t *testing.T) {
	dir := t.TempDir()
	withCwd(t, dir)

	for _, op := range []domain.CodeIntelligenceOperation{
		domain.CodeIntOpIndexFolder, domain.CodeIntOpGraphFile,
		domain.CodeIntOpGraphTopic, domain.CodeIntOpGraphAll,
	} {
		got, err := graphDBPath(&domain.CodeIntelligenceConfig{Operation: op})
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", op, err)
		}
		want := filepath.Join(dir, ".kdeps", "graph.db")
		if got != want {
			t.Fatalf("%s: got %q, want %q", op, got, want)
		}
	}
}

func TestGraphDBPath_IndexFolderAndGraphFileIgnorePath(t *testing.T) {
	dir := t.TempDir()
	withCwd(t, dir)

	for _, op := range []domain.CodeIntelligenceOperation{domain.CodeIntOpIndexFolder, domain.CodeIntOpGraphFile} {
		got, err := graphDBPath(&domain.CodeIntelligenceConfig{Operation: op, Path: "/some/other/place"})
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", op, err)
		}
		want := filepath.Join(dir, ".kdeps", "graph.db")
		if got != want {
			t.Fatalf("%s: Path was not ignored: got %q, want %q", op, got, want)
		}
	}
}

func TestGraphDBPath_GraphTopicUsesPath(t *testing.T) {
	root := t.TempDir()
	got, err := graphDBPath(&domain.CodeIntelligenceConfig{Operation: domain.CodeIntOpGraphTopic, Path: root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, ".kdeps", "graph.db")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGraphDBPath_GetCwdError(t *testing.T) {
	withCwdErr(t, "no cwd")
	_, err := graphDBPath(&domain.CodeIntelligenceConfig{Operation: domain.CodeIntOpGraphAll})
	if err == nil {
		t.Fatal("expected error when getCwd fails")
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
	withCwdErr(t, "no cwd")
	exec := NewExecutor()
	_, err := exec.executeGraph(&domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpGraphAll,
	})
	if err == nil {
		t.Fatal("expected error when the CWD default can't be resolved")
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

func TestGraphIndexFolder_GetCwdError(t *testing.T) {
	withCwdErr(t, "no cwd")
	_, err := graphIndexFolder(nil, &domain.CodeIntelligenceConfig{})
	if err == nil {
		t.Fatal("expected error when getCwd fails")
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
	withCwd(t, root)
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

	_, err = graphIndexFolder(ig, &domain.CodeIntelligenceConfig{})
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
