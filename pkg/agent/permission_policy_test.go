package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionPolicy_EmbeddedKeepsTheOldModes(t *testing.T) {
	p := loadPermissionPolicy()
	assert.Equal(t, "allow", p.Default)
	assert.Equal(t, PermissionReadOnly, requiredPermissionIn(p, "read_file"))
	assert.Equal(t, PermissionReadOnly, requiredPermissionIn(p, "web_search"))
	assert.Equal(t, PermissionWorkspaceWrite, requiredPermissionIn(p, "bash_exec"))
	assert.Equal(t, PermissionWorkspaceWrite, requiredPermissionIn(p, "http_request"))
	assert.Equal(t, PermissionWorkspaceWrite, requiredPermissionIn(p, "some_workflow_tool"))
}

func TestPermissionPolicy_DefaultDenyBlocksUnlistedTools(t *testing.T) {
	p, err := parsePermissionPolicy([]byte("default: deny\ntiers:\n  T0: read-only\ntools:\n  read_file: T0\n"))
	require.NoError(t, err)
	assert.Equal(t, PermissionReadOnly, requiredPermissionIn(p, "read_file"))
	assert.Equal(t, PermissionDangerFullAccess, requiredPermissionIn(p, "bash_exec"))
}

func TestPermissionPolicy_UnknownTierFailsClosed(t *testing.T) {
	p, err := parsePermissionPolicy([]byte("default: allow\ntiers:\n  T0: read-only\ntools:\n  mystery: T9\n"))
	require.NoError(t, err)
	assert.Equal(t, PermissionDangerFullAccess, requiredPermissionIn(p, "mystery"))
}

func TestPermissionPolicy_RejectsBadDefaultAndMode(t *testing.T) {
	_, err := parsePermissionPolicy([]byte("default: maybe\n"))
	require.Error(t, err)
	_, err = parsePermissionPolicy([]byte("default: allow\ntiers:\n  T0: bogus\n"))
	require.Error(t, err)
	_, err = parsePermissionPolicy([]byte("default: [\n"))
	require.Error(t, err)
}

func TestPermissionPolicy_CachedUntilReload(t *testing.T) {
	initPermissionPolicy()
	first := loadPermissionPolicy()
	second := loadPermissionPolicy()
	assert.Equal(t, first, second)

	initPermissionPolicy()
	permissionPolicyCache.Lock()
	cleared := permissionPolicyCache.policy == nil
	permissionPolicyCache.Unlock()
	assert.True(t, cleared, "reload drops the cached policy")
	assert.Equal(t, first, loadPermissionPolicy(), "the reread policy matches")
}

func TestPermissionPolicy_UserOverrideSwitchesToDeny(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".kdeps", "policy")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "permission.yaml"),
		[]byte("default: deny\ntiers:\n  T0: read-only\ntools:\n  read_file: T0\n"), 0o600))
	initPermissionPolicy()
	t.Cleanup(initPermissionPolicy)

	p := loadPermissionPolicy()
	assert.Equal(t, "deny", p.Default)
	assert.Equal(t, PermissionDangerFullAccess, requiredPermissionIn(p, "bash_exec"))
	assert.Equal(t, PermissionReadOnly, requiredPermissionIn(p, "read_file"))
}

func TestPermissionPolicy_BrokenUserOverrideFailsClosed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".kdeps", "policy")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "permission.yaml"), []byte("default: maybe\n"), 0o600))
	initPermissionPolicy()
	t.Cleanup(initPermissionPolicy)

	assert.Equal(t, "deny", loadPermissionPolicy().Default)
}
