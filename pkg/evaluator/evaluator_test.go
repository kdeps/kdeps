package evaluator_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/evaluator"
)

func TestCreateAndProcessPklFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := log.New(os.Stdout)

	sections := []string{"section1", "section2"}
	finalFileName := "/tmp/final.pkl"
	pklTemplate := "Kdeps.pkl"

	processFunc := func(fs afero.Fs, tmpFile string, headerSection string, logger *log.Logger) (string, error) {
		content, err := afero.ReadFile(fs, tmpFile)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s\n%s", headerSection, string(content)), nil
	}

	t.Run("CreateAndProcessAmends", func(t *testing.T) {
		err := evaluator.CreateAndProcessPklFile(fs, sections, finalFileName, pklTemplate, logger, processFunc, false)
		assert.NoError(t, err, "CreateAndProcessPklFile should not return an error")
		content, err := afero.ReadFile(fs, finalFileName)
		require.NoError(t, err, "Final file should be created successfully")
		assert.Contains(t, string(content), "amends", "Final file content should include 'amends'")
		assert.Contains(t, string(content), sections[0], "Final file content should include section1")
	})

	t.Run("CreateAndProcessExtends", func(t *testing.T) {
		err := evaluator.CreateAndProcessPklFile(fs, sections, finalFileName, pklTemplate, logger, processFunc, true)
		assert.NoError(t, err, "CreateAndProcessPklFile should not return an error")
		content, err := afero.ReadFile(fs, finalFileName)
		require.NoError(t, err, "Final file should be created successfully")
		assert.Contains(t, string(content), "extends", "Final file content should include 'extends'")
		assert.Contains(t, string(content), sections[1], "Final file content should include section2")
	})
}
