package archiver

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// We reuse stubWf from resource_compiler_edge_test for Workflow implementation.

// TestPrepareRunDir ensures archive extraction happens into expected run path.
func TestPrepareRunDir(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	wf := stubWf{}

	tmp := t.TempDir()
	kdepsDir := filepath.Join(tmp, "kdepssys")
	if err := os.MkdirAll(kdepsDir, 0o755); err != nil {
		t.Fatalf("mkdir kdepsDir: %v", err)
	}

	// Build minimal tar.gz archive containing a dummy file.
	pkgPath := filepath.Join(tmp, "pkg.kdeps")
	pkgFile, err := os.Create(pkgPath)
	if err != nil {
		t.Fatalf("create pkg: %v", err)
	}
	gz := gzip.NewWriter(pkgFile)
	tw := tar.NewWriter(gz)
	// add dummy.txt
	hdr := &tar.Header{Name: "dummy.txt", Mode: 0o644, Size: int64(len("hi"))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("hdr: %v", err)
	}
	if _, err := io.WriteString(tw, "hi"); err != nil {
		t.Fatalf("write: %v", err)
	}
	tw.Close()
	gz.Close()
	pkgFile.Close()

	runDir, err := PrepareRunDir(fs, ctx, wf, kdepsDir, pkgPath, logging.NewTestLogger())
	if err != nil {
		t.Fatalf("PrepareRunDir error: %v", err)
	}

	// Expect dummy.txt extracted inside runDir
	if ok, _ := afero.Exists(fs, filepath.Join(runDir, "dummy.txt")); !ok {
		t.Fatalf("extracted file missing")
	}
}

// TestPackageProjectHappy creates minimal compiled project and ensures .kdeps created.
func TestPackageProjectHappy(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	wf := stubWf{}
	kdepsDir := "/kdeps"
	compiled := "/compiled"

	// build minimal structure
	_ = fs.MkdirAll(filepath.Join(compiled, "resources"), 0o755)
	_ = afero.WriteFile(fs, filepath.Join(compiled, "resources", "client.pkl"), []byte("run { }"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(compiled, "workflow.pkl"), []byte("amends \"package://schema.kdeps.com/core@0.0.1#/Workflow.pkl\"\n"), 0o644)
	_ = fs.MkdirAll(kdepsDir, 0o755)

	pkg, err := PackageProject(fs, ctx, wf, kdepsDir, compiled, logging.NewTestLogger())
	if err != nil {
		t.Fatalf("PackageProject error: %v", err)
	}

	if ok, _ := afero.Exists(fs, pkg); !ok {
		t.Fatalf("package file not written: %s", pkg)
	}

	// call again to ensure overwrite logic works (should not error)
	if _, err := PackageProject(fs, ctx, wf, kdepsDir, compiled, logging.NewTestLogger()); err != nil {
		t.Fatalf("second PackageProject error: %v", err)
	}
}
