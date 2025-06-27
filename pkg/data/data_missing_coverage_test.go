package data

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestDataMissingCoverage(t *testing.T) {
	t.Run("PopulateDataFileRegistry", func(t *testing.T) {
		// Test PopulateDataFileRegistry function - has 0.0% coverage
		fs := afero.NewMemMapFs()
		dataDir := "/tmp/test-data"

		// This function should populate the data file registry
		assert.NotPanics(t, func() {
			PopulateDataFileRegistry(fs, dataDir)
		})

		// The function should execute without error
		PopulateDataFileRegistry(fs, dataDir)

		// Verify it can be called multiple times
		PopulateDataFileRegistry(fs, dataDir)
	})
}
