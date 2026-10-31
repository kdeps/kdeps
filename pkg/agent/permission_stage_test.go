package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageGate_DangerModeAllowsEveryKind(t *testing.T) {
	t.Setenv("KDEPS_PERMISSION_MODE", "danger-full-access")
	assert.NoError(t, stageGate("exec"))
	assert.NoError(t, stageGate("httpClient"))
}

func TestStageGate_ReadOnlyRunRefusesWriteKinds(t *testing.T) {
	t.Setenv("KDEPS_PERMISSION_MODE", "read-only")
	require.Error(t, stageGate("exec"))
	require.Error(t, stageGate("httpClient"))
	assert.NoError(t, stageGate("chat"))
	assert.NoError(t, stageGate("searchLocal"))
}

func TestStageGate_UnlistedKindFollowsDefault(t *testing.T) {
	t.Setenv("KDEPS_PERMISSION_MODE", "workspace-write")
	assert.NoError(t, stageGate("someFutureKind"), "default allow needs workspace-write")
	t.Setenv("KDEPS_PERMISSION_MODE", "read-only")
	assert.Error(t, stageGate("someFutureKind"), "default allow needs workspace-write, read-only has less")
}

func TestStageGate_ApprovalKindRefusedHeadless(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeUserPolicy(t, home, "default: allow\ntiers:\n  T2: ask\nresources:\n  exec: T2\n")
	initPermissionPolicy()
	t.Cleanup(initPermissionPolicy)
	t.Setenv("KDEPS_PERMISSION_MODE", "danger-full-access")
	err := stageGate("exec")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires approval")
}
