package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/executor"
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

func TestToolGate_ReadOnlyRunRefusesWriteTools(t *testing.T) {
	t.Setenv("KDEPS_PERMISSION_MODE", "read-only")
	err := toolGate("write_file")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `tool "write_file" requires permission "workspace-write"`)
	assert.NoError(t, toolGate("read_file"))
	t.Setenv("KDEPS_PERMISSION_MODE", "danger-full-access")
	assert.NoError(t, toolGate("write_file"))
}

func TestToolGate_ApprovalAndTokenTiers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	policy := "default: allow\ntiers:\n  T2: ask\n  T3: token\ntools:\n  write_file: T2\n  bash_exec: T3\n"
	writeUserPolicy(t, home, policy)
	initPermissionPolicy()
	t.Cleanup(initPermissionPolicy)
	t.Setenv("KDEPS_PERMISSION_MODE", "danger-full-access")
	require.ErrorContains(t, toolGate("write_file"), "requires approval, and a workflow run has no one to ask")
	require.ErrorContains(t, toolGate("bash_exec"), "requires a capability token: kdeps token grant bash_exec")
}

func TestInstallStagePermission_GatesWorkflowChatTools(t *testing.T) {
	InstallStagePermission()
	t.Setenv("KDEPS_PERMISSION_MODE", "read-only")
	require.Error(t, executor.ToolPermission("bash_exec"))
	require.NoError(t, executor.ToolPermission("read_file"))
}
