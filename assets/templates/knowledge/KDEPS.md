# Vault routing

This vault is the long-term memory. Read this table first, then the
project card, then only the notes the knowledge recall names. Never read the
whole vault.

## Where everything lives

Every place kdeps runs is a project with one folder, created and filled in by
kdeps. Nothing here needs to be set up by hand.

| What | Home | Written by |
|---|---|---|
| Project status and next step | `projects/<name>/<name>.md` (the card) | kdeps on first run; `knowledge_card_update` |
| Session history (focus, built, mistakes, decisions, wins) | `projects/<name>/sessions/` | kdeps, every turn |
| Saved facts (`[MEMORY: key] value`) | `projects/<name>/memory/` | kdeps, one note per key |
| Decisions and their rationale | `projects/<name>/decisions/` | kdeps from session summaries; `knowledge_note_save` with type decision |
| Ideas, gotchas, concepts for one project | `projects/<name>/notes/` | `knowledge_note_save`, or by hand |
| Knowledge across projects | `shared/` | `knowledge_note_save` with scope shared, or by hand |
| Unsorted notes | `inbox/` | by hand |
| Finished projects | `archive/<name>/` | kdeps moves the folder when the card says `status: archived` |
| Connections between notes | `~/.kdeps/knowledge/<vault-id>/` (outside this vault) | kdeps |
| How to write cards and notes | `schema.md` | |

## Rules

- One home per fact. Status lives in the card, the story in notes, the
  connections in the graph. A fact written in two places goes stale in one.
- Cards stay short: frontmatter and a links table. They point; they do not
  tell the story.
- Update a card with `knowledge_card_update`, never by copying status into a
  note.
- `kdeps knowledge check` lists structural duplicates (the same summary, or the
  same `canonical` or `fact_id`), dangling links and notes without a summary.

## Status overview

kdeps lists every active project's status and next step on each turn, read
from the cards themselves. Do not copy status into this file.
