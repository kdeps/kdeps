package cfg

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/kdeps/schema/gen/kdeps/path"
	"github.com/stretchr/testify/assert"
)

func TestGetKdepsPathComprehensive(t *testing.T) {
	// Test GetKdepsPath function to boost cfg package coverage

	ctx := context.Background()

	t.Run("User_path", func(t *testing.T) {
		// Test with User path type
		kdepsCfg := kdeps.Kdeps{
			KdepsDir:  ".kdeps",
			KdepsPath: path.User,
		}

		result, err := GetKdepsPath(ctx, kdepsCfg)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should contain home directory and .kdeps
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".kdeps")
		assert.Equal(t, expected, result)
	})

	t.Run("Project_path", func(t *testing.T) {
		// Test with Project path type
		kdepsCfg := kdeps.Kdeps{
			KdepsDir:  ".kdeps",
			KdepsPath: path.Project,
		}

		result, err := GetKdepsPath(ctx, kdepsCfg)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should contain current working directory and .kdeps
		cwd, _ := os.Getwd()
		expected := filepath.Join(cwd, ".kdeps")
		assert.Equal(t, expected, result)
	})

	t.Run("Xdg_path", func(t *testing.T) {
		// Test with Xdg path type
		kdepsCfg := kdeps.Kdeps{
			KdepsDir:  "kdeps",
			KdepsPath: path.Xdg,
		}

		result, err := GetKdepsPath(ctx, kdepsCfg)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should contain XDG config home and kdeps
		expected := filepath.Join(xdg.ConfigHome, "kdeps")
		assert.Equal(t, expected, result)
	})

	t.Run("Custom_KdepsDir", func(t *testing.T) {
		// Test with custom KdepsDir
		kdepsCfg := kdeps.Kdeps{
			KdepsDir:  "my-custom-kdeps",
			KdepsPath: path.User,
		}

		result, err := GetKdepsPath(ctx, kdepsCfg)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should contain home directory and custom kdeps dir
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, "my-custom-kdeps")
		assert.Equal(t, expected, result)
	})

	t.Run("Empty_KdepsDir", func(t *testing.T) {
		// Test with empty KdepsDir
		kdepsCfg := kdeps.Kdeps{
			KdepsDir:  "",
			KdepsPath: path.Project,
		}

		result, err := GetKdepsPath(ctx, kdepsCfg)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should just be the current working directory
		cwd, _ := os.Getwd()
		assert.Equal(t, cwd, result)
	})

	t.Run("Unknown_path_type", func(t *testing.T) {
		// Test with invalid path type (if possible)
		// Since path.Path is likely an enum, this might not be testable
		// But we can try with a zero value
		kdepsCfg := kdeps.Kdeps{
			KdepsDir:  ".kdeps",
			KdepsPath: "", // Empty string or zero value
		}

		result, err := GetKdepsPath(ctx, kdepsCfg)
		// Should either handle gracefully or return an error
		assert.True(t, err != nil || result != "")
	})
}
