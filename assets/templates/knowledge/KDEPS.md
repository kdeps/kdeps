# Vault routing

This vault is the long-term memory. Read this table first, then the
project card, then only the notes the knowledge recall names. Never read the
whole vault.

## Where everything lives

| What | Home | Written by |
|---|---|---|
| Project status and next step | `projects/<name>/<name>.md` (the card) | kdeps on a workflow's first logged run; you |
| A workflow's calls of one day | `projects/<name>/sessions/` | kdeps, with `knowledge.workflow_notes: true` |
| Saved facts (`[MEMORY: key] value`) | `projects/<name>/memory/` | kdeps, one note per key (`kdeps run --memory`) |
| Decisions and their rationale | `projects/<name>/decisions/` | you |
| Ideas, gotchas, concepts for one project | `projects/<name>/notes/` | you |
| Knowledge across projects | `shared/` | you |
| Unsorted notes | `inbox/` | you |
| Finished projects | `archive/<name>/` | you |
| Connections between notes | `~/.kdeps/knowledge/<vault-id>/` (outside this vault) | kdeps |
| How to write cards and notes | `schema.md` | |

## Rules

- One home per fact. Status lives in the card, the story in notes, the
  connections in the graph. A fact written in two places goes stale in one.
- Cards stay short: frontmatter and a links table. They point; they do not
  tell the story.
- Keep status in the card. Do not copy it into a note.
- `kdeps knowledge check` lists structural duplicates (the same summary, or the
  same `canonical` or `fact_id`), dangling links and notes without a summary.
