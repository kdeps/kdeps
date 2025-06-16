package cfg

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/schema/gen/kdeps"
	"github.com/kdeps/schema/gen/kdeps/path"
)

// helper to construct minimal config
func newKdepsCfg(dir string, p path.Path) kdeps.Kdeps {
	return kdeps.Kdeps{
		KdepsDir:  dir,
		KdepsPath: p,
	}
}

func TestGetKdepsPathUser(t *testing.T) {
	cfg := newKdepsCfg(".kdeps", path.User)
	got, err := GetKdepsPath(context.Background(), cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".kdeps")
	if got != want {
		t.Fatalf("want %s got %s", want, got)
	}
}

func TestGetKdepsPathProject(t *testing.T) {
	cfg := newKdepsCfg("kd", path.Project)
	cwd, _ := os.Getwd()
	got, err := GetKdepsPath(context.Background(), cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := filepath.Join(cwd, "kd")
	if got != want {
		t.Fatalf("want %s got %s", want, got)
	}
}

func TestGetKdepsPathXDG(t *testing.T) {
	cfg := newKdepsCfg("store", path.Xdg)
	got, err := GetKdepsPath(context.Background(), cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// do not assert exact path; just ensure ends with /store
	if filepath.Base(got) != "store" {
		t.Fatalf("unexpected path %s", got)
	}
}

func TestGetKdepsPathUnknown(t *testing.T) {
	// Provide invalid path using numeric constant outside defined ones.
	type customPath string
	bad := newKdepsCfg("dir", path.Path("bogus"))
	if _, err := GetKdepsPath(context.Background(), bad); err == nil {
		t.Fatalf("expected error for unknown path type")
	}
}
