package resolver

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfPKLError replicates helper from exec tests so we can ignore environments
// where the PKL binary / registry is not available.
func skipIfPKLErrorPy(t *testing.T, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "Cannot find module") ||
		strings.Contains(msg, "unexpected status code") ||
		strings.Contains(msg, "apple PKL not found") {
		t.Skipf("Skipping due to missing PKL: %v", err)
	}
}

func setupTestPyResolver(t *testing.T) *DependencyResolver {
	dr := setupTestResolver(t)
	// override dirs for python
	_ = dr.Fs.MkdirAll(filepath.Join(dr.ActionDir, "python"), 0o755)
	return dr
}

func TestAppendPythonEntryExtra(t *testing.T) {
	t.Parallel()

	newResolver := func(t *testing.T) (*DependencyResolver, string) {
		dr := setupTestPyResolver(t)
		pklPath := filepath.Join(dr.ActionDir, "python/"+dr.RequestID+"__python_output.pkl")
		return dr, pklPath
	}

	t.Run("NewEntry", func(t *testing.T) {
		dr, pklPath := newResolver(t)

		initial := fmt.Sprintf(`extends "package://schema.kdeps.com/core@%s#/Python.pkl"

resources {
}`,
			schema.SchemaVersion(dr.Context))
		require.NoError(t, afero.WriteFile(dr.Fs, pklPath, []byte(initial), 0o644))

		py := &pklPython.ResourcePython{
			Script:    "print('hello')",
			Stdout:    utils.StringPtr("output"),
			Timestamp: &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond},
		}

		err := dr.AppendPythonEntry("res", py)
		skipIfPKLErrorPy(t, err)
		assert.NoError(t, err)

		content, err := afero.ReadFile(dr.Fs, pklPath)
		skipIfPKLErrorPy(t, err)
		require.NoError(t, err)
		assert.Contains(t, string(content), "res")
		// encoded script should appear
		assert.Contains(t, string(content), utils.EncodeValue("print('hello')"))
	})

	t.Run("ExistingEntry", func(t *testing.T) {
		dr, pklPath := newResolver(t)

		initial := fmt.Sprintf(`extends "package://schema.kdeps.com/core@%s#/Python.pkl"

resources {
  ["res"] {
    script = "cHJpbnQoJ29sZCc pyk="
    timestamp = 1.ns
  }
}`,
			schema.SchemaVersion(dr.Context))
		require.NoError(t, afero.WriteFile(dr.Fs, pklPath, []byte(initial), 0o644))

		py := &pklPython.ResourcePython{
			Script:    "print('new')",
			Stdout:    utils.StringPtr("new out"),
			Timestamp: &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond},
		}

		err := dr.AppendPythonEntry("res", py)
		skipIfPKLErrorPy(t, err)
		assert.NoError(t, err)

		content, err := afero.ReadFile(dr.Fs, pklPath)
		skipIfPKLErrorPy(t, err)
		require.NoError(t, err)
		assert.Contains(t, string(content), utils.EncodeValue("print('new')"))
		assert.NotContains(t, string(content), "cHJpbnQoJ29sZCc pyk=")
	})
}
