package executor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestCheckStagePermission_DeniedKindBlocksTheResource(t *testing.T) {
	SetStagePermission(func(kind string) error {
		if kind == "exec" {
			return errors.New("denied by policy")
		}
		return nil
	})
	t.Cleanup(func() { SetStagePermission(nil) })

	denied := &domain.Resource{ActionID: "run", Exec: &domain.ExecConfig{}}
	err := checkStagePermission(denied)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stage run")
	assert.Contains(t, err.Error(), "denied by policy")

	allowed := &domain.Resource{ActionID: "chat", Python: &domain.PythonConfig{}}
	assert.NoError(t, checkStagePermission(allowed))
}

func TestCheckStagePermission_NoCheckAllowsEverything(t *testing.T) {
	SetStagePermission(nil)
	assert.NoError(t, checkStagePermission(&domain.Resource{ActionID: "run", Exec: &domain.ExecConfig{}}))
}
