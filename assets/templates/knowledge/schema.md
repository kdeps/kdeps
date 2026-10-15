# Schema: cards and notes

## Card (`records/projects/<name>.md`, `records/clients/<name>.md`)

```markdown
---
type: project          # project | client
status: active         # active | paused | done | archived
next: the one next step
updated: 2026-10-15
---
| Source | Path |
|---|---|
| repo | ~/code/my-project |
| vault | Projects/my-project |
| memory | kdeps memory for the repo folder |
| skills | skills/my-project |
```

A card has no prose. `repo` links the card to a code folder (kdeps finds the
card for the folder it runs in); `vault` names the project's note folder.

## Note (everything in the vault)

```markdown
---
type: session          # session | decision | hub | concept | gotcha | tool | note
status: draft          # the note's own state: draft | evergreen | outdated
tags: [kdeps, graph]
created: 2026-10-15
summary: one line saying what this note holds
links: ["[[Other note]]"]
---
```

- One idea per note in `Resources/`.
- `summary` is what the knowledge recall shows before anyone opens the note.
  Write it so the reader can decide whether to open it.
- Link with `[[Note title]]`. Links written in a note are facts; the graph
  marks them as extracted.
- A project folder has one hub note (`type: hub`) that links its sessions.

## Session note (written by kdeps)

`## Focus`, `## What was built`, `## Mistakes`, `## Decisions`, `## Wins`.
