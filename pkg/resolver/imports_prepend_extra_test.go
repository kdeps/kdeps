package resolver

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestPrependDynamicImportsExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}

	dr := &DependencyResolver{
		Fs:          fs,
		Context:     ctx,
		ActionDir:   "/tmp/action",
		RequestID:   "rid",
		Logger:      logging.NewTestLogger(),
		Environment: env,
	}

	// create directories and dummy files for Check=true imports
	folders := []string{"llm", "client", "exec", "python", "data"}
	for _, f := range folders {
		p := filepath.Join(dr.ActionDir, f, dr.RequestID+"__"+f+"_output.pkl")
		require.NoError(t, fs.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, afero.WriteFile(fs, p, []byte(""), 0644))
	}
	// Also the request pkl file itself counted with alias "request" (Check=true)
	dr.RequestPklFile = filepath.Join(dr.ActionDir, "req.pkl")
	require.NoError(t, fs.MkdirAll(filepath.Dir(dr.RequestPklFile), 0o755))
	require.NoError(t, afero.WriteFile(fs, dr.RequestPklFile, []byte(""), 0644))

	// Create test file with only amends line
	testPkl := filepath.Join(dr.ActionDir, "test.pkl")
	content := "amends \"something\"\n"
	require.NoError(t, afero.WriteFile(fs, testPkl, []byte(content), 0644))

	// Call function
	require.NoError(t, dr.PrependDynamicImports(testPkl))

	// Read back file and ensure dynamic import lines exist (e.g., import "pkl:json") and request alias line
	out, err := afero.ReadFile(fs, testPkl)
	require.NoError(t, err)
	s := string(out)
	require.True(t, strings.Contains(s, "import \"pkl:json\""))
	require.True(t, strings.Contains(s, "import \""+dr.RequestPklFile+"\" as request"))
}
