package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// askTierPolicy puts bash_exec on the ask tier in a user override.
func askTierPolicy(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".kdeps", "policy")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	body := "default: allow\ntiers:\n  T1: workspace-write\n  T2: ask\ntools:\n  bash_exec: T2\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "permission.yaml"), []byte(body), 0o600))
	initPermissionPolicy()
	t.Cleanup(initPermissionPolicy)
}

func TestAskTier_ParsesAndRequiresAsk(t *testing.T) {
	p, err := parsePermissionPolicy([]byte("default: allow\ntiers:\n  T2: ask\ntools:\n  bash_exec: T2\n"))
	require.NoError(t, err)
	assert.Equal(t, PermissionAsk, requiredPermissionIn(p, "bash_exec"))
}

func TestAskTier_EnforcerNeverAllowsStatically(t *testing.T) {
	askTierPolicy(t)
	ok, reason := NewPermissionEnforcer(PermissionDangerFullAccess).Allow("bash_exec")
	assert.False(t, ok)
	assert.Contains(t, reason, "requires approval")
}

func TestAskTier_HeadlessIsRefusedInEveryMode(t *testing.T) {
	askTierPolicy(t)
	for _, mode := range []PermissionMode{PermissionDangerFullAccess, PermissionWorkspaceWrite, PermissionAsk} {
		l := &Loop{config: Config{PermissionMode: mode}}
		reason, blocked := l.checkToolPermission("bash_exec", "{}")
		assert.True(t, blocked, "mode %s", mode)
		assert.Contains(t, reason, "requires approval", "mode %s", mode)
	}
}

func TestAskTier_ApprovedCallRuns(t *testing.T) {
	askTierPolicy(t)
	l := &Loop{config: Config{
		PermissionMode: PermissionDangerFullAccess,
		Approver:       func(ApprovalRequest) ApprovalChoice { return ApproveOnce },
	}}
	reason, blocked := l.checkToolPermission("bash_exec", "{}")
	assert.False(t, blocked)
	assert.Empty(t, reason)
}

func TestAskTier_DeniedByApproverStaysBlocked(t *testing.T) {
	askTierPolicy(t)
	l := &Loop{config: Config{
		PermissionMode: PermissionDangerFullAccess,
		Approver:       func(ApprovalRequest) ApprovalChoice { return ApproveDeny },
	}}
	_, blocked := l.checkToolPermission("bash_exec", "{}")
	assert.True(t, blocked)
}
