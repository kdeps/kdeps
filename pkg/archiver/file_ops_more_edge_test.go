package archiver

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestMoveFolderSuccessEdge(t *testing.T) {
	fs := afero.NewMemMapFs()

	// create src dir with subfile
	src := "/srcdir"
	dst := "/dstdir"
	if err := fs.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := afero.WriteFile(fs, filepath.Join(src, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := MoveFolder(fs, src, dst); err != nil {
		t.Fatalf("MoveFolder error: %v", err)
	}

	// src should be removed, dst should contain file
	if exists, _ := afero.DirExists(fs, src); exists {
		t.Fatalf("expected src dir removed")
	}
	if ok, _ := afero.Exists(fs, filepath.Join(dst, "file.txt")); !ok {
		t.Fatalf("destination file missing")
	}
}

func TestGetFileMD5Truncate(t *testing.T) {
	fs := afero.NewMemMapFs()
	file := "/file.bin"
	data := []byte("1234567890abcdef")
	_ = afero.WriteFile(fs, file, data, 0o644)

	md5Full, err := GetFileMD5(fs, file, 32)
	if err != nil {
		t.Fatalf("md5 error: %v", err)
	}
	if len(md5Full) != 32 {
		t.Fatalf("expected full md5 length got %d", len(md5Full))
	}

	md5Short, _ := GetFileMD5(fs, file, 8)
	if len(md5Short) != 8 {
		t.Fatalf("expected truncated md5 len 8 got %d", len(md5Short))
	}
	if md5Short != md5Full[:8] {
		t.Fatalf("truncated md5 mismatch")
	}
}

func TestParseActionIDEdgeCases(t *testing.T) {
	name, ver := parseActionID("@other/action:2.1.0", "agent", "1.0.0")
	if name != "other" || ver != "2.1.0" {
		t.Fatalf("unexpected parse result %s %s", name, ver)
	}

	// Missing explicit name
	name2, ver2 := parseActionID("myAction:0.3.0", "agent", "1.0.0")
	if name2 != "agent" || ver2 != "0.3.0" {
		t.Fatalf("unexpected default name parse")
	}

	// No version specified
	name3, ver3 := parseActionID("@foo/bar", "agent", "1.2.3")
	if name3 != "foo" || ver3 != "1.2.3" {
		t.Fatalf("default version fallback failed")
	}
}
