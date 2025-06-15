package cfg

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kdeps/schema/gen/kdeps"
	"github.com/kdeps/schema/gen/kdeps/path"
)

func TestGetKdepsPathVariants(t *testing.T) {
	ctx := context.Background()

	tmpHome := t.TempDir()
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("setenv: %v", err)
	}

	tmpProject := t.TempDir()
	if err := os.Chdir(tmpProject); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	dirName := "kdeps-system"
	build := func(p path.Path) kdeps.Kdeps {
		return kdeps.Kdeps{KdepsDir: dirName, KdepsPath: p}
	}

	cases := []struct {
		name    string
		cfg     kdeps.Kdeps
		want    string
		wantErr bool
	}{
		{"user", build(path.User), filepath.Join(tmpHome, dirName), false},
		{"project", build(path.Project), filepath.Join(tmpProject, dirName), false},
		{"xdg", build(path.Xdg), filepath.Join(os.Getenv("XDG_CONFIG_HOME"), dirName), false},
		{"unknown", build("weird"), "", true},
	}

	for _, c := range cases {
		got, err := GetKdepsPath(ctx, c.cfg)
		if c.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error", c.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if filepath.Base(got) != dirName {
			t.Fatalf("%s: expected path ending with %s, got %s", c.name, dirName, got)
		}
	}

	// Restore cwd for other tests on Windows.
	if runtime.GOOS == "windows" {
		_ = os.Chdir("\\")
	}
}
