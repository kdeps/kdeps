package knowledge_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/knowledge"
)

// A project's whole life in the vault: created for a new place, filled by a
// session (note, decisions, memory), recalled, checked, archived with its
// memory following, and reopened.
func TestProjectLifecycle(t *testing.T) {
	vault := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfg, []byte("knowledge:\n  vault: "+vault+"\n"), 0o600))
	t.Setenv("KDEPS_CONFIG_PATH", cfg)
	v := knowledge.Vault{Root: vault}
	_, err := knowledge.Init(v)
	require.NoError(t, err)

	repo := filepath.Join(t.TempDir(), "billing-svc")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o750))
	cwd := filepath.Join(repo, "internal")
	require.NoError(t, os.MkdirAll(cwd, 0o750))
	now := time.Now()

	card, created, err := v.EnsureProject(cwd, now)
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, "billing-svc", card.Name, "named after the git repository root")

	ms := agent.NewMemoryStore("")
	ms.SetCwd(cwd)
	require.NoError(t, ms.Load())
	require.NoError(t, ms.Set("fact:currency", "Invoices are in EUR"))

	summary := "## Goal\nMove billing to monthly invoices\n\n## Key Decisions\n" +
		"- **Monthly invoices**: per-call billing doubled churn\n\n## Next Steps\n1. write the invoice job\n"
	sn, err := knowledge.WriteSession(card, "s1", summary, now)
	require.NoError(t, err)
	written, err := knowledge.WriteDecisions(card, sn.File, summary, now)
	require.NoError(t, err)
	require.Len(t, written, 1)

	for _, p := range []string{
		"projects/billing-svc/billing-svc.md",
		"projects/billing-svc/memory/fact-currency.md",
		"projects/billing-svc/decisions/Monthly invoices.md",
	} {
		assert.FileExists(t, filepath.Join(vault, filepath.FromSlash(p)))
	}
	sessions, _ := filepath.Glob(filepath.Join(vault, "projects", "billing-svc", "sessions", "*.md"))
	assert.Len(t, sessions, 1)

	_, _, err = knowledge.Build(t.Context(), v, knowledge.GraphDir(vault), now)
	require.NoError(t, err)
	r := knowledge.SharedIndex(knowledge.GraphDir(vault)).Recall("why monthly invoices", 1, 0)
	require.NotEmpty(t, r.Hits)
	assert.Equal(t, "projects/billing-svc/decisions/Monthly invoices.md", r.Hits[0].Note,
		"the decision note outranks the session and memory notes")
	findings, err := knowledge.Check(v)
	require.NoError(t, err)
	assert.Empty(t, findings, "everything kdeps wrote has one home")

	require.NoError(t, v.SetStatus(card, knowledge.StatusArchived, "", now))
	ms.Rebase()
	require.NoError(t, ms.Set("fact:late", "written after archiving"))
	assert.FileExists(t, filepath.Join(vault, "archive", "billing-svc", "memory", "fact-late.md"))
	assert.NoDirExists(t, filepath.Join(vault, "projects", "billing-svc"))

	back, created, err := v.EnsureProject(cwd, now)
	require.NoError(t, err)
	assert.False(t, created)
	require.NoError(t, v.Reopen(back, now))
	assert.FileExists(t, filepath.Join(vault, "projects", "billing-svc", "memory", "fact-late.md"),
		"reopening brings the whole folder back")
}
