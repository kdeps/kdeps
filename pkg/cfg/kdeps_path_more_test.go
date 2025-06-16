package cfg

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/kdeps/schema/gen/kdeps"
	kpath "github.com/kdeps/schema/gen/kdeps/path"
	"github.com/stretchr/testify/assert"
)

func TestGetKdepsPathCases(t *testing.T) {
	tmpProject := t.TempDir()
	// Change working directory so path.Project branch produces deterministic path.
	oldWd, _ := os.Getwd()
	_ = os.Chdir(tmpProject)
	defer os.Chdir(oldWd)

	cases := []struct {
		name      string
		cfg       kdeps.Kdeps
		expectFn  func() string
		expectErr bool
	}{
		{
			"user path", kdeps.Kdeps{KdepsDir: "mykdeps", KdepsPath: kpath.User}, func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "mykdeps")
			}, false,
		},
		{
			"project path", kdeps.Kdeps{KdepsDir: "mykdeps", KdepsPath: kpath.Project}, func() string {
				cwd, _ := os.Getwd()
				return filepath.Join(cwd, "mykdeps")
			}, false,
		},
		{
			"xdg path", kdeps.Kdeps{KdepsDir: "mykdeps", KdepsPath: kpath.Xdg}, func() string {
				return filepath.Join(xdg.ConfigHome, "mykdeps")
			}, false,
		},
		{
			"unknown", kdeps.Kdeps{KdepsDir: "abc", KdepsPath: "bogus"}, nil, true,
		},
	}

	for _, tc := range cases {
		got, err := GetKdepsPath(context.Background(), tc.cfg)
		if tc.expectErr {
			assert.Error(t, err, tc.name)
			continue
		}
		assert.NoError(t, err, tc.name)
		assert.Equal(t, tc.expectFn(), got, tc.name)
	}
}
