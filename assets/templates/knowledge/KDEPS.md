# Workspace routing

This folder is the instructions. Read this table first, then the one card it
points to, then only the notes the knowledge recall names. Never read the
whole vault.

## Where everything lives

| What | Home | Notes |
|---|---|---|
| Project status, next step, links | `records/projects/<name>.md` (one card per project) | the only place status lives |
| Client status, contacts, links | `records/clients/<name>.md` (one card per client) | |
| Session history (focus, built, mistakes, decisions, wins) | the project's vault folder, `sessions/` | written by kdeps after each session |
| Decisions and their rationale | session notes, and one note per decision in the vault | the card links, never repeats |
| One idea per note (zettelkasten) | `Resources/` in the vault | |
| Connections between notes | the knowledge graph (`kdeps knowledge build`) | kept outside the vault |
| How to write cards and notes | `schema.md` | |

## Rules

- One home per fact. Status lives in a card, the story in a note, connections
  in the graph. A fact written in two places goes stale in one of them.
- Cards stay short: frontmatter and a links table. They point; they do not
  tell the story.
- Update a card with `knowledge_card_update`, never by copying status into a
  note.
- `kdeps knowledge check` lists facts with more than one home, dangling links
  and notes without a summary.

## Status overview

kdeps lists every card's status and next step here when it loads this file,
read from the cards themselves. Do not copy status into this file.
