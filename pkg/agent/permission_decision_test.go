package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/instrument"
)

func TestRecordPermissionDecision_WritesAnAuditEvent(t *testing.T) {
	if instrument.Active() != nil {
		t.Skip("a session is already recording")
	}
	t.Setenv(instrument.RootEnv, t.TempDir())
	s, err := instrument.Start("permission-audit", []string{"test"}, time.Now())
	require.NoError(t, err)

	recordPermissionDecision("bash_exec", PermissionWorkspaceWrite, PermissionReadOnly,
		`permission denied: tool "bash_exec" requires permission`, true)
	recordPermissionDecision("read_file", PermissionReadOnly, PermissionReadOnly, "", false)
	require.NoError(t, s.Close(nil))

	data, err := os.ReadFile(filepath.Join(s.Dir, instrument.EventsFile))
	require.NoError(t, err)
	var decisions []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.Contains(line, `"kind":"permission_decision"`) {
			decisions = append(decisions, line)
		}
	}
	require.Len(t, decisions, 2)
	require.Contains(t, decisions[0], `"name":"bash_exec"`)
	require.Contains(t, decisions[0], `"required":"workspace-write"`)
	require.Contains(t, decisions[0], `"mode":"read-only"`)
	require.Contains(t, decisions[0], `"decision":"deny"`)
	require.Contains(t, decisions[1], `"decision":"allow"`)
	require.NotContains(t, decisions[1], `"reason"`)
}

func TestRecordPermissionDecision_NoSessionIsANoOp(t *testing.T) {
	if instrument.Active() != nil {
		t.Skip("a session is already recording")
	}
	require.NotPanics(t, func() {
		recordPermissionDecision("read_file", PermissionReadOnly, PermissionDangerFullAccess, "", false)
	})
}
