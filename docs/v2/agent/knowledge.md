# Knowledge vault

The knowledge vault gives the agent loop a long-term knowledge base it can use without reading it. Each turn, kdeps walks the vault's link graph from the words of the request, with no model involved. It then hands the model the few notes worth reading (paths and one-line summaries), not the whole vault.

```d2
direction: down
q: "your request" {shape: oval}
route: "KDEPS.md routing table\n(where everything lives)"
card: "project card\nstatus, next step, places"
recall: "graph recall (no model)\nwords -> matching notes -> 2 hops"
notes: "3-8 notes named\npath + summary"
model: "model reads only\nthe notes it needs" {shape: oval}
q -> route -> card
q -> recall -> notes -> model
card -> model
```

Recall is the default step of every prompt kdeps sends a model: each agent-loop turn (terminal and desktop app) and each workflow [`chat`](/workflow/resources/llm#knowledge-recall) call. The agent loop's own internal calls (compaction, prompt refinement, goal planning) skip it.

A vault of 2,000 notes holds about a million words. That's more than any model can read, and partial reads cost accuracy, time and money. With recall, a question typically reads a routing table, one card and a handful of notes.

## One home per fact

| Fact | Its only home |
|---|---|
| Status, next step, places on disk | the project's card: `projects/<name>/<name>.md` |
| What happened: focus, what was built, mistakes, decisions, wins | the session note: `projects/<name>/sessions/` |
| Why something was decided | one decision note: `projects/<name>/decisions/` |
| What the agent remembers | memory notes: `projects/<name>/memory/` (see [Persistent memory](/agent/memory)) |
| An idea, gotcha or concept | one note: `projects/<name>/notes/`, or `shared/` when it spans projects |
| How notes connect | the link graph, in the vault's hidden `.index/` |

A fact written in two places goes stale in one of them. `kdeps knowledge check` finds those.

## Projects

Every place kdeps runs is a project, and each project has one folder that holds all of it. kdeps creates the folder and its card the first time it runs in a place, then fills it in as you work. Nothing is set up by hand.

```d2
direction: down
place: "kdeps runs in a folder" {shape: oval}
known: "a card's repo/folder/workflow\nrow covers it?" {shape: diamond}
use: "use that project"
name: "name it: git repository root,\nelse the folder; workflow: its name"
create: "create projects/<name>/\nand its card"
turn: "every turn fills\nsessions/ memory/ decisions/ notes/" {shape: oval}
place -> known
known -> use: yes
known -> name: no
name -> create -> turn
use -> turn
```

```text
~/.kdeps/notes/                       # the vault (knowledge.vault)
  KDEPS.md                            # routing table + rules; loaded into every session
  schema.md                           # how cards and notes are written
  projects/
    my-app/                           # created on the first prompt in ~/code/my-app
      my-app.md                       # the card: status, next step, places on disk
      sessions/                       # one note per session, updated every turn
      memory/                         # one note per [MEMORY: key]; .auto/ holds thinking and tool captures
      decisions/                      # one note per decision from a session summary
      notes/                          # notes the model saves, and your own
  shared/                             # knowledge across projects; shared/memory/ with KDEPS_MEMORY_GLOBAL=1
  inbox/                              # notes dropped by hand, not yet filed
  archive/                            # projects whose card says status: archived
  GRAPH_REPORT.md                     # god nodes, communities, surprising connections (rebuilt; not a note)
  graph.html                          # the graph explorer page: communities as clusters, a legend
  .index/                             # graph.json and caches (hidden; rebuilt)
```

The layout is the same for every use: a code repository, a folder of documents, a workflow, a bot, a research topic. What kind of project it is goes in the card's `type`, not in the folder names.

| Event | What kdeps writes |
|---|---|
| First prompt in a place no card covers | `projects/<name>/<name>.md` (`status: active`) |
| Every turn | `sessions/<date> <topic>.md`, rewritten in place |
| `[MEMORY: key] value` | `memory/<key>.md` |
| Compaction, or the end of a terminal session | the session note from the model's summary; one `decisions/` note per Key Decision; the first next step on the card |
| `knowledge_note_save` | `notes/` (`decisions/` for type decision), or `shared/`, `inbox/` |
| `knowledge_card_update` with `status: archived` | moves `projects/<name>/` to `archive/<name>/`; the next turn there moves it back |
| A workflow chat call, with `workflow_notes: true` | `sessions/<date> <workflow> runs.md`, one line per call |

## Set up

Nothing to set up: the vault is on by default, in `~/.kdeps/notes`. The first session creates it, and every session rebuilds its graph in the background.

| Turn it | How |
|---|---|
| off for good | `knowledge.enabled: false` in `~/.kdeps/config.yaml` |
| off or on, remembered across sessions | `/knowledge off`, `/knowledge on` |
| to another folder | `knowledge.vault` in config.yaml, or `kdeps knowledge init <folder>` |

```yaml
# ~/.kdeps/config.yaml
knowledge:
  enabled: true               # false: no recall, no notes, no knowledge tools
  vault: ~/.kdeps/notes       # the default
  session_notes: true         # agent sessions write a note per session (default true)
  workflow_notes: false       # workflow chat calls log to their project's sessions/ (default false)
  recall:
    hops: 2                   # how far from the matched notes to walk
    budget_tokens: 1500       # most tokens of recalled paths and summaries per turn
```

## Cards

A card is short on purpose. It points; it never tells the story. kdeps writes this one the first time it runs in `~/code/my-app`:

```markdown
---
type: project          # any word: project, client, area, research, bot
status: active         # the only place status lives; archived moves the folder to archive/
next: (none)           # the one next step; session summaries fill it in
updated: 2026-10-15
---
| Source | Path |
|---|---|
| repo | /home/me/code/my-app |   <- runs in this folder (or below it) use this project
```

Add rows to make several places one project: `repo` (a git repository; its Go packages join the graph), `folder` or `workflow`. The deepest matching row wins. kdeps adds an `index` row for each place that has a search index (built by `/search index`, kept under `~/.kdeps`), so the card says where everything about the project lives. Every turn carries a status overview of the active projects, read from the cards at that moment (`<knowledge-status>`); it is never stored. The model reads and changes cards with `knowledge_card_get` and `knowledge_card_update`.

## Notes

Every note is plain markdown with frontmatter. `summary` is what recall shows before anyone opens the note.

```markdown
---
type: decision          # session | decision | concept | gotcha | tool | note
status: evergreen
tags: [billing]
created: 2026-10-15
summary: Bill monthly, not per call - per-call billing doubled churn.
links: ["[[Stripe webhook retries]]"]
---
We moved to monthly billing because ...
```

- Link notes with `[[Note title]]` or `[[folder/Note title]]`. A link resolves by path, by title anywhere in the vault (the closest one wins when two share a title) or by `aliases:`.
- Every link kdeps writes uses the full vault path - `[[projects/app/app]]`, `[[projects/app/sessions/2026-10-15 0900 billing]]`, `[[shared/Stripe gotcha]]` - so it can never point at the wrong note. A link to a project that was archived later still resolves, by its title.
- Links inside code blocks are examples, not links.
- One idea per note. The model saves new ones with `knowledge_note_save` into the project's `notes/` (or `decisions/`, `shared/`, `inbox/`); it refuses a title that already has a note anywhere in the vault.

## Session notes

After every turn, kdeps writes the session's note into the project's `sessions/` folder and rewrites it in place as the conversation goes on. A note covers one thread: when a request is unrelated to the thread so far, the next note starts, so a session that moves from billing to Kubernetes leaves two notes. Each note links the project's card, so the card's backlinks list every session.

```text
switch billing to monthly invoices      -> 2026-10-16 0900 switch billing to monthly invoices.md
why are monthly invoices late           -> same note (shares the thread's words)
yes do it                               -> same note (a short follow-up)
configure kubernetes liveness probes    -> 2026-10-16 0912 configure kubernetes liveness probes.md
```

Telling them apart needs no model: a request continues the thread when it shares at least a fifth of its content words with the thread (stemmed, without stopwords or task verbs like fix, add, make), or has fewer than three content words. The model's summary at compaction and session end goes to the current thread's note, written from that thread's messages only.

```markdown
---
type: session
project: my-app
session: 7f3c...
summary: Ship knowledge recall for my-app
links: ["[[my-app]]"]
---
## Focus
## What was built
## Mistakes
## Decisions
## Wins
```

After each turn the note is updated without a model call: your prompts, the files written or edited, decisions saved to memory and tools that failed. At compaction the model's summary replaces it with a written one (no extra model call: compaction makes that summary anyway), and later turns are added on top. When an interactive terminal session ends, one summary call covers the turns since the last note; a piped or scripted session exits at once (its compactions still write notes). The summary's first next step goes to the card's `next`, not into the note, and each of its Key Decisions becomes a note in `decisions/`, titled by its bold lead (`- **Monthly billing**: per-call doubled churn` becomes `decisions/Monthly billing.md`). A decision that already has a note is left alone.

## Workflow mode

A workflow [`chat`](/workflow/resources/llm#knowledge-recall) call recalls from the vault by default. Writing is off: a workflow can run thousands of times. Turn it on to keep a log per workflow:

```yaml
# ~/.kdeps/config.yaml
knowledge:
  workflow_notes: true     # each chat call adds one line to its workflow's run note for the day
```

A workflow's project is keyed by its `metadata.name`: the first logged run creates `projects/<name>/`, with a card whose `workflow` row is the folder it ran from; the same workflow run from anywhere else logs to the same project. Each chat call adds `- 14:02:11 <prompt> -> <reply>` to `sessions/<date> <name> runs.md`. A resource with `knowledge: false` neither recalls nor writes. Run notes join the graph on its next build (any agent session, or `kdeps knowledge build`).

## The graph

`kdeps knowledge build` turns the vault into a graph:
- **Nodes:** each note, each tag, and each Go package of every card's `repo` row.
- **Edges:** wikilinks (`references`), frontmatter links (`relates_to`), tags (`tagged`) and Go imports (`depends_on`).
- **Provenance:** every edge records how it was found. Edges from links and imports are `extracted`; code goes through the Go parser, never a model. A local model can add `inferred` edges (see [Model extraction](#model-extraction)).

```text
$ kdeps knowledge build
2003 notes (2003 changed), 1986 code files -> 4210 nodes, 9873 edges in 1.2s
$ kdeps knowledge build
2003 notes (0 changed), 1986 code files -> 4210 nodes, 9873 edges in 270ms
```

Code files are the ones git tracks or would track, so `.gitignore`d folders stay out; without git, every `.go` file outside `vendor/`, `testdata/` and hidden folders.

The graph lives in the vault's hidden `.index/` folder (`graph.json` and the caches), so it travels with the vault and is never read as a note. `GRAPH_REPORT.md` and `graph.html` are written at the vault root, where you see them; the report is skipped when the vault is read, so it never becomes a note. A `GRAPH_REPORT.md` or `graph.html` of your own there is never replaced: kdeps only rewrites the ones it wrote, and yours stays a note. A rebuild re-reads only changed files.

## Model extraction

Links written in notes become `extracted` edges. A local model can add what the notes do not link: it reads each changed note and lists the projects, tools, decisions, rationales, gotchas and concepts it is about, and how they relate. Its findings join the graph as `inferred` edges with the model's confidence (a note `mentions` an entity; entities `depends_on`, `rationale_for`, ... each other), or `ambiguous` ones below 0.5 confidence, listed under "Edges to review" in the report. An entity named like a note is that note, so two notes about the same tool meet at two hops.

```d2
direction: right
changed: "changed notes" {shape: oval}
first: "first note alone\n(loads the model)"
rest: "the rest in parallel\n(workers)"
cache: "extract.json\nby note hash"
build: "graph build\ninferred / ambiguous edges"
changed -> first -> rest -> cache -> build
```

- **In agent sessions** it runs at session end, after the session note: up to `per_session` changed notes, with the session's model when it is a local one (llamafile, GGUF, Ollama). With a cloud model it is skipped unless `knowledge.extract.model` names a local model. It never runs during your turns. `/knowledge extract` runs it now, over every changed note.
- **From the shell:** `kdeps knowledge build --extract [--model <local model>] [--limit N]`.
- A note is read again only when it changes; a failed note is retried on the next run.

```yaml
# ~/.kdeps/config.yaml
knowledge:
  extract:
    enabled: true
    model: llama3.2:3b        # a 3B+ local model finds far more than a 1B one
    workers: 2                # parallel calls after the first
    per_session: 20           # changed notes read at each session end
```

## Communities and the report

There is one graph per vault, covering every project, `shared/` and the Go packages of each card's `repo` row. Each build groups it into communities with the Louvain method: notes and packages that link mostly among themselves, which usually follow projects and the areas inside them. Each node gets a `community` number in `graph.json`; 0 is the largest. At the vault root, the build writes `GRAPH_REPORT.md` and `graph.html`, a page to explore the graph: each community gathers in its own colored cluster, a legend lists every community with its size and best-connected members (click one to focus it), and you can search, zoom and hover for a note's summary. The page's template (`templates/knowledge`) is installed the first time it is needed, so it is written under any asset profile; `kdeps knowledge build` prints where it is.

```bash
kdeps knowledge report
```

```text
# Knowledge graph report
| Notes | Nodes | Edges | Extracted edges | Communities | Modularity |
| 2003  | 4210  | 9873  | 100%            | 37          | 0.61       |

## God nodes            the most connected notes and packages: changing them touches the most
## Communities          each group with its best-connected members and the summary of its top note
## Surprising connections
- [[projects/app/decisions/Billing decision]] (community 3, 40 nodes) -[references, extracted]-> [[shared/Kubernetes probes]] (community 9, 25 nodes)
## Links to notes that do not exist
```

A surprising connection is a single link between two communities that otherwise never touch, ranked by the size of the smaller one: a lone bridge between two large groups is the least expected.

Inside an agent session, kdeps rebuilds the graph in the background when the session starts and after it writes a note.

## Recall

Before each model call, kdeps:
1. Scores graph nodes on the request's words. Rarer words count more, a word that is part of a note's title counts double, and word forms meet (`billing` matches `bill`). Notes kdeps writes itself rank lower than notes you curate: session notes at 0.4x, memory notes at 0.6x. The session's own note is never recalled into that session.
2. Walks up to `hops` edges from the best matches. Extracted edges count fully, and each hop halves the score.
3. Hands the model the best notes within `budget_tokens`, as paths it can open with `read_file`.
4. When the project's places have a search index (`/search index`), adds the files that match most of the request's words, up to 3 per place, in a `<knowledge-files>` list headed by the folder and its index path. The model is told to open them with `read_file` or search the folder further with `search_local`. The index is only read, never built or changed, and a busy index is skipped.

```text
<knowledge-files>
Files in this project's search index that match the request (best first). Open the ones you need with read_file; search further with search_local, path set to the folder.
Folder /home/me/code/my-app (search index /home/me/.kdeps/projects/.../index.db):
- /home/me/code/my-app/billing/invoices.go - monthly invoices go out on the first day
</knowledge-files>
```

```text
<knowledge-recall>
Notes in the knowledge vault related to this request, from its link graph (best first). ...
- /home/me/notes/projects/my-app/decisions/Billing decision.md - Bill monthly, not per call. ...
- /home/me/notes/shared/Stripe webhook retries.md - Webhooks retry for 3 days. (linked from Billing decision)
</knowledge-recall>
```

Recall is added to the turn, not to the cached system prompt, so each request gets its own. The terminal shows what it cost:

```text
recall: 4 notes (~180 tok), 2 files
```

`/recall` shows the last turn's recall and why each note was chosen; `/recall <words>` recalls for other words; `/knowledge` shows whether the vault is on and where it is. From the shell:

```bash
kdeps knowledge recall "why monthly billing"
```

## Check

```bash
kdeps knowledge check
```

```text
  projects/my-app/notes/old status.md: has a next step; it belongs in the project card (one home per fact)
  shared/Graph recall.md: links to [[Louvain]], which has no note
  shared/Pricing.md: has the same summary as shared/Tiers.md; keep the fact in one note and link it
  projects/web/web.md: card holds 9 lines of prose; move the story to a vault note and link it
```

It exits non-zero when it finds something, so it can run in CI.

## Tools

| Tool | Does |
|---|---|
| `knowledge_recall` | Recall notes for other words than the request |
| `knowledge_card_get` | Read a card (default: the working directory's project, created on first use) |
| `knowledge_card_update` | Set `status` and/or `next`; stamps `updated`; `archived` moves the project to `archive/` |
| `knowledge_note_save` | Save a new one-idea note; `scope`: `project` (default), `shared` or `inbox` |

They are offered in agent sessions while the vault is on (the default), refuse while `/knowledge off`, and ship with the `agent` asset profile.
