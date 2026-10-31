# Schema: cards and notes

## Card (`projects/<name>/<name>.md`)

kdeps writes this the first time it runs in a place: in a git repository the
project is named after the repository's root folder, else after the folder;
a workflow's project is named after the workflow (`metadata.name`) and shared by every place it runs from.

```markdown
---
type: project          # project, client, area, research, bot - any word
status: active         # active | paused | done | archived (archived moves the folder to archive/)
next: the one next step
updated: 2026-10-15
---
| Source | Path |
|---|---|
| repo | ~/code/my-project |
| folder | ~/Documents/my-project-notes |
| workflow | ~/agents/invoices |
```

A card has no prose. A `repo`, `folder` or `workflow` row ties a place on
disk to the project: kdeps runs in that place (or below it) use this project.
Add rows to make several places one project. kdeps adds an `index` row for a place's search index.

## Note

```markdown
---
type: note             # session | decision | gotcha | concept | tool | note
status: draft          # the note's own state: draft | evergreen | outdated
tags: [kdeps, graph]
created: 2026-10-15
summary: one line saying what this note holds
canonical: ship-monthly    # optional stable id; fact_id is the same field
links: ["[[Other note]]"]
---
```

- One idea per note: `projects/<name>/notes/` for one project, `shared/` for
  knowledge across projects. Two notes with the same `canonical` or `fact_id`
  are one fact in two homes; `kdeps knowledge check` reports that.
- `summary` is what the knowledge recall shows before anyone opens the note.
  Write it so the reader can decide whether to open it.
- Link with `[[Note title]]`, or the full path `[[projects/app/notes/Title]]`.
  Links written in a note are facts; the graph marks them as extracted.
  kdeps writes every link with its full vault path: sessions and decisions
  link their card as `[[projects/<name>/<name>]]`.

## Written by kdeps

- Session note (`sessions/`): `## Focus`, `## What was built`, `## Mistakes`,
  `## Decisions`, `## Wins`. Updated after every turn; rewritten from the
  model's summary at compaction and session end.
- Decision note (`decisions/`): one per item of a summary's Key Decisions,
  titled by its bold lead.
- Memory note (`memory/`): one per `[MEMORY: key]`; the frontmatter keeps the
  key, type and dates.
- Workflow run note (`sessions/<date> <name> runs.md`): one line per chat
  call, when `knowledge.workflow_notes: true`.
