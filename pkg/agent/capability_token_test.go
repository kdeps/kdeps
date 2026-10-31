package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tokenHome points HOME at a fresh directory and maps a tool and a stage to T3.
func tokenHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeUserPolicy(t, home, "default: allow\ntiers:\n  T3: token\ntools:\n  wire_money: T3\nresources:\n  exec: T3\n")
	initPermissionPolicy()
	t.Cleanup(initPermissionPolicy)
}

func TestCapabilityToken_GrantListRevoke(t *testing.T) {
	tokenHome(t)
	now := time.Now()
	tok, err := GrantCapabilityToken("wire_money", "ops", time.Hour, now)
	require.NoError(t, err)
	require.NotEmpty(t, tok.ID)

	tokens, err := ListCapabilityTokens()
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.True(t, tokens[0].Valid(now))
	assert.False(t, tokens[0].Valid(now.Add(2*time.Hour)), "an expired token is not valid")

	require.NoError(t, RevokeCapabilityToken(tok.ID))
	tokens, err = ListCapabilityTokens()
	require.NoError(t, err)
	assert.Empty(t, tokens)
	assert.Error(t, RevokeCapabilityToken(tok.ID))
}

func TestCapabilityToken_RejectsBadGrants(t *testing.T) {
	tokenHome(t)
	_, err := GrantCapabilityToken("", "ops", time.Hour, time.Now())
	assert.Error(t, err)
	_, err = GrantCapabilityToken("wire_money", "ops", 0, time.Now())
	assert.Error(t, err)
	assert.Error(t, RevokeCapabilityToken("../escape"))
}

func TestCapabilityToken_GatesAToolTier(t *testing.T) {
	tokenHome(t)
	t.Setenv("KDEPS_PERMISSION_MODE", "danger-full-access")
	ok, reason := NewPermissionEnforcer(PermissionDangerFullAccess).Allow("wire_money")
	assert.False(t, ok, "no token: refused even in danger mode")
	assert.Contains(t, reason, "kdeps token grant wire_money")

	_, err := GrantCapabilityToken("wire_money", "ops", time.Hour, time.Now())
	require.NoError(t, err)
	ok, _ = NewPermissionEnforcer(PermissionDangerFullAccess).Allow("wire_money")
	assert.True(t, ok, "a valid token allows the tool")
}

func TestCapabilityToken_GatesAStage(t *testing.T) {
	tokenHome(t)
	t.Setenv("KDEPS_PERMISSION_MODE", "danger-full-access")
	err := stageGate("exec")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kdeps token grant stage:exec")

	_, err = GrantCapabilityToken("stage:exec", "ops", time.Hour, time.Now())
	require.NoError(t, err)
	assert.NoError(t, stageGate("exec"))
}
