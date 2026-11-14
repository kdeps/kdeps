# Knowledge vault

The knowledge vault is a long-term knowledge base. The agent loop uses it without a full read. On each turn, kdeps follows the link graph of the vault from the words of the request, with no model. Then it gives the model some note paths with one-line summaries.

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

Recall runs on every prompt kdeps sends a model: each agent-loop turn (terminal and desktop app) and each workflow [`chat`](/workflow/resources/llm#knowledge-recall) call. Internal loop calls (compaction, prompt refinement) skip it.

A 2,000-note vault holds about a million words, more than any model can read. With recall, a question typically reads a routing table, one card and a handful of notes.

## One home per fact

| Fact | Its only home |
|---|---|
| Status, next step, places on disk | the project's card: `projects/<name>/<name>.md` |
| What happened: focus, what was built, mistakes, decisions, wins | the session note: `projects/<name>/sessions/` |
| Why something was decided | one decision note: `projects/<name>/decisions/` |
| What the agent remembers | memory notes: `projects/<name>/memory/` (see [Persistent memory](/agent/memory)) |
| An idea, gotcha or concept | one note: `projects/<name>/notes/`, or `shared/` when it spans projects |
| How notes connect | the link graph, under `~/.kdeps/knowledge/<vault-id>/` |

These folder names and the card's front matter keys are defaults in the `knowledge` asset set's `vault.yaml`. Override the file under `~/.kdeps/` to rename them:

```yaml
folders:
  projects: projects  # one folder per project
  shared: shared  # knowledge that spans projects
project:
  sessions: sessions  # one note per agent session
  decisions: decisions  # one note per decision
card:
  next: next  # front matter key of the one next step; it lives only in the card
  links_header: [Source, Path]  # the card's places table: a link kind, then a path
note:
  summary: summary  # the line recall shows before anyone opens a note
  canonical: [canonical, fact_id]  # a stable fact id; the first key that is set wins
```

You must set each name. Install rejects a `vault.yaml` with an empty name.

Every note kdeps writes comes from a Go template in the set's `notes.yaml`: cards, session notes, decision notes, a workflow's daily run log, `knowledge_note_save` notes, `GRAPH_REPORT.md` and `kdeps knowledge explain` lines. The file also maps compaction summary sections to session headings:

```yaml
session:
  file_time: "2006-01-02 1504"  # a new session note's file name starts with its start time, as a Go time layout
session_sections:  # session note headings, each filled by the first summary section that is set
  - heading: What was built
    from: [done, progress]
templates:
  run_entry: |  # one workflow chat call in the day's run log
    - {{.Time}} {{.Prompt}} -> {{.Reply}}
```

Install rejects a `notes.yaml` with a missing or broken template.

A link passed to `knowledge_note_save` resolves like a graph link: by path, then by title or alias, nearest note first. kdeps writes it with its full vault path. If no note matches, kdeps writes it as you gave it.

The `extract.yaml` of the set defines what model extraction can name and how its answer joins the graph:

```yaml
entity_kinds: [project, tool, decision, rationale, gotcha, concept]  # other kinds become default_kind
relations: [depends_on, rationale_for, relates_to, supersedes, references]  # kept as the model named them
fallback_relation: relates_to  # any other relation becomes this...
fallback_max_confidence: 0.49  # ...at most this sure, so it is marked ambiguous
```

`kdeps knowledge check` reports structural duplicates: the same summary twice, a next step outside a card, or two notes with the same `canonical` or `fact_id`. It does not detect the same fact in different wording.

## Projects

Each place where kdeps runs is a project with one folder. kdeps creates the folder and its card on the first run in a place. Then it adds data as you work. You do not set up anything manually.

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
```

The link graph, its caches, `GRAPH_REPORT.md` and `graph.html` live outside the vault, under `~/.kdeps/knowledge/<vault-id>/` (a hash of the vault path).

The layout is the same for a code repository, a document folder, a workflow, a bot or a research topic. The project kind goes in the card's `type`.

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

The vault is on by default, in `~/.kdeps/notes`. The first session creates it. Every session rebuilds its graph in the background.

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
    hops: 2                   # 1 or 2; a higher value is clamped to 2
    budget_tokens: 1500       # most tokens of recalled paths and summaries per turn
```

## Cards

A card points. It never tells the story. kdeps writes this one on the first run in `~/code/my-app`:

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

Card rows:

| Row | Meaning |
|---|---|
| `repo` | a git repository; its Go packages join the graph |
| `folder`, `workflow` | another place that belongs to this project; the deepest matching row wins |
| `search` | a [search folder](#search-folders): searched with the project, not one of its places |
| `branch` | the project's git branch, updated at the start of a turn when it changes (kdeps keeps one) |
| `index` | a place's search index (built by `/search index`, kept under `~/.kdeps`); added by kdeps |

Every turn carries a `<knowledge-status>` overview of active projects, read from the cards at that moment and never stored. The model reads and changes cards with `knowledge_card_get` and `knowledge_card_update`.

## Notes

Every note is markdown with frontmatter. `summary` is what recall shows.

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

- Link with `[[Note title]]` or `[[folder/Note title]]`. A link resolves by path, by title anywhere in the vault (closest wins on a tie), or by `aliases:`.
- Links kdeps writes use the full vault path (`[[projects/app/app]]`, `[[shared/Stripe gotcha]]`), so they never point at the wrong note. A link to a later-archived project still resolves by title.
- Links inside code blocks are examples, not links.
- One idea per note. `knowledge_note_save` writes into `notes/` (or `decisions/`, `shared/`, `inbox/`) and refuses a title that already exists anywhere in the vault.

## Session notes

After every turn, kdeps rewrites the session's note in the project's `sessions/` folder. A note covers one thread: an unrelated request starts a new note, so a session moving from billing to Kubernetes leaves two. Each note links the project's card, so the card's backlinks list every session.

```text
switch billing to monthly invoices      -> 2026-10-16 0900 switch billing to monthly invoices.md
why are monthly invoices late           -> same note (shares the thread's words)
yes do it                               -> same note (a short follow-up)
configure kubernetes liveness probes    -> 2026-10-16 0912 configure kubernetes liveness probes.md
```

No model decides this. A request continues the thread in two conditions. The first condition is that it has fewer than three content words. The second condition is that one fifth or more of its content words are also in the thread. kdeps uses word stems and ignores stopwords and task verbs (for example fix, add, make). Summaries at compaction and session end go to the current thread's note, written from that thread's messages only.

```markdown
---
type: session
project: my-app
session: 7f3c...
summary: Ship knowledge recall for my-app
links: ["[[my-app]]"]
branch: feat/recall              # in a git repository: the branch the thread ran on
commits: 3f2a91c0b7de..9c41e2d07a55  # HEAD when the thread began .. HEAD now
---
## Focus
## What was built
## Mistakes
## Decisions
## Wins
## Git
On branch `feat/recall`, commits `3f2a91c0b7de..9c41e2d07a55`: `git log 3f2a91c0b7de..9c41e2d07a55` shows them.
```

In a git repository the note points into git instead of copying history. The note has `branch`, `commits`, and a `## Git` line. `commits` goes from `HEAD` at the start of the thread to `HEAD` now. kdeps reads it with `git rev-parse` on each turn. With these lines, the model can run `git log` or `git diff`.

A rebase or squash never leaves the vault stale. A rewritten note keeps its original start, even in a resumed session. Outside git, a note has no git lines.

How kdeps adds data to the note:

- **Each turn, no model call:** your prompts, files written or edited, decisions saved to memory, failed tools.
- **At compaction:** the summary of the model replaces it (compaction always makes that summary). kdeps adds later turns on top.
- **When an interactive terminal session ends:** one summary call includes the turns after the last note. A piped or scripted session stops immediately. Its compactions write notes.
- The summary's first next step goes to the card's `next`, not the note.
- Each Key Decision becomes a `decisions/` note titled by its bold lead: `- **Monthly billing**: per-call doubled churn` becomes `decisions/Monthly billing.md`. kdeps does not change a decision note that exists.

## Workflow mode

A workflow [`chat`](/workflow/resources/llm#knowledge-recall) call recalls from the vault by default. Write is off, because a workflow can run thousands of times. To get a log for each workflow, turn it on:

```yaml
# ~/.kdeps/config.yaml
knowledge:
  workflow_notes: true     # each chat call adds one line to its workflow's run note for the day
```

- The key of the project is the workflow name in `metadata`. The first logged run creates `projects/<name>/` with a `workflow` row for the folder it ran from. The same workflow run elsewhere logs to the same project.
- Each chat call adds `- 14:02:11 <prompt> -> <reply>` to `sessions/<date> <name> runs.md`.
- A resource with `knowledge: false` neither recalls nor writes.
- Run notes join the graph on the next refresh (a chat call, vault search or session).

## The graph

The structural graph refreshes automatically at these times:

- At agent session start and before each recall.
- Before the recall of a workflow chat call.
- On a vault search with `graphBoost`.
- On skill linking, when a skill file is a vault note. `kdeps knowledge build` runs the same build by hand.

- **Nodes:** each note, each tag, and each Go package of every card's `repo` row.
- **Edges:** wikilinks (`references`), frontmatter links (`relates_to`), tags (`tagged`) and Go imports (`depends_on`).
- **Provenance:** each edge records how kdeps found it. Link and import edges are `extracted`. Code goes through the Go parser, never a model. A local model can add `inferred` edges (see [Model extraction](#model-extraction)).

```text
$ kdeps knowledge build
2003 notes (2003 changed), 1986 code files -> 4210 nodes, 9873 edges in 1.2s
$ kdeps knowledge build
2003 notes (0 changed), 1986 code files -> 4210 nodes, 9873 edges in 270ms
```

Code files are the files that git tracks or can track. For this reason, kdeps does not include `.gitignore`d folders. Without git: every `.go` file outside `vendor/`, `testdata/` and hidden folders. A rebuild re-reads only changed files.

`graph.json`, caches, `GRAPH_REPORT.md` and `graph.html` go under `~/.kdeps/knowledge/<vault-id>/`, never into the vault. `kdeps knowledge build --export` copies the report and page into the vault. It does this only if the copy is not there or kdeps wrote it. A file that you wrote there stays a note. If there is a graph in the old `.index/` of the vault, the next build reads it one time. Then kdeps removes it, if kdeps wrote it.

## Model extraction

A local model can add relations the notes do not link. It reads each changed note and lists the projects, tools, decisions, rationales, gotchas and concepts it covers, and how they relate. Findings join the graph as `inferred` edges with the model's confidence (a note `mentions` an entity; entities `depends_on`, `rationale_for`, ... each other). Below 0.5 confidence they are `ambiguous` and listed under "Edges to review" in the report. An entity named like a note is that note, so two notes about the same tool meet at two hops.

```d2
direction: right
changed: "changed notes" {shape: oval}
first: "first note alone\n(loads the model)"
rest: "the rest in parallel\n(workers)"
cache: "extract.json\nby note hash"
build: "graph build\ninferred / ambiguous edges"
changed -> first -> rest -> cache -> build
```

**In agent sessions**, `knowledge.build` or `/knowledge build` sets the mode:

| Mode | Runs |
|---|---|
| `manual` (default) | only on `/knowledge extract` or `kdeps knowledge build --extract` |
| `auto` | also at session end (after the session note, up to `per_session` notes) and before a background build; keeps running after the prompt returns |
| `off` | never |

Recall waits for the link graph, not for the model. The call uses the [subagent channel](/agent/subagent): the session model unless `/subagent` or `llm.subagent` names another. `free` picks a large OpenRouter free model. `knowledge.extract.model` applies only when the channel is `current`. kdeps skips a local model that is not available.

**From the shell:** `kdeps knowledge build --extract [--model <model>] [--limit N]`. The model is `--model`, then `llm.subagent`, then `knowledge.extract.model`, then the current model.

kdeps reads a note again only when it changes. It tries a failed note again on the next run. A build without `--extract` prints how many notes are waiting. The graph records the model and time of the latest successful extraction.

```text
/knowledge build            # manual, the default
/knowledge build auto       # session end and before a background build
/knowledge build off
```

```yaml
# ~/.kdeps/config.yaml
knowledge:
  build: manual               # manual | auto | off
  extract:
    model: llama3.2:3b        # used when llm.subagent is current; empty: the session model
    workers: 2                # parallel calls after the first
    per_session: 20           # changed notes read on an automatic run
```

### Media into notes

`/ingest` reads a file with a kdeps tool and saves its text as a note titled by the second argument, so extraction can read it:

```text
/ingest image  ~/scans/receipt.png  receipt-march
/ingest audio  ~/calls/standup.mp3  standup-2026-11-02
/ingest document ~/docs/spec.pdf    spec-v2
```

| Subcommand | Reader tool | Accepts |
|---|---|---|
| `image` | `ocr_image` (tesseract, local) | png, jpg, tiff, bmp, gif |
| `audio` | `transcribe_audio` (Whisper) | mp3, mp4, m4a, wav and other audio or video |
| `document` | `load_document` | PDF, DOCX, HTML, CSV, text |

Run `/knowledge extract` afterwards to pull facts from the new notes.

## Communities and the report

One graph covers the whole vault: every project, `shared/`, and the Go packages of each card's `repo` row.

- Each build groups it with the knowledge policy's `community_method`: `louvain` (default) or `leiden` (always-connected communities). Both are deterministic.
- Each node gets a `community` number in `graph.json`. 0 is the largest.
- The build writes `GRAPH_REPORT.md` and `graph.html` next to the graph. The page draws each community as a colored cluster with a legend (click to focus), plus search, zoom and hover summaries. Its template (`templates/knowledge`) installs on first need.
- `kdeps knowledge build` prints the paths. `kdeps knowledge explain "<query>"` lists matching edges with the file and line each came from.

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

A surprising connection is a single link between two otherwise disconnected communities, ranked by the smaller one's size.

In an agent session, kdeps rebuilds the link graph in the background at session start and after writing a note. Workflow chat calls, vault searches and skill linking refresh it before reading. Refreshes never call a model.

## Recall

Before each model call, kdeps:

1. **Scores nodes on the request's words.** Rarer words count more, title words count double, and word forms meet (`billing` matches `bill`). Session notes score 0.4x and memory notes 0.6x, below curated notes. A session's own note is never recalled into it.
2. **Walks at most 2 edges** from the best matches, strongest first. Extracted edges count fully, inferred edges by their confidence, ambiguous edges at 0.3 times their confidence. Tag edges are then multiplied by 0.3. Each hop divides the score by 2. If `knowledge.recall.hops` is more than 2, kdeps uses 2.
3. **Lists the best notes** within `budget_tokens`, as paths for `read_file`. A note reached through edges shows the chain, e.g. `projects/app/decisions/Billing decision.md --relates_to--> shared/Stripe gotcha.md`.
4. **Adds indexed files.** The places of the project or the [search folders](#search-folders) can have a search index (`/search index`). In this condition, the 3 best-matching files for each folder go in a `<knowledge-files>` list. The list has the folder and index path at the top. A file shows one time, also when folders overlap. If recall already found a vault note, kdeps repeats its chain. The model gets an instruction to open the files with `read_file` or to search with `search_local` (path = one folder, or none for all). Here the index is read-only. kdeps skips a busy index.

```text
<knowledge-files>
Files in this project's search index that match the request (best first). Open the ones you need with read_file; search further with search_local, path set to the folder (or no path to search all of the project's folders).
Folder /home/me/code/my-app (search index /home/me/.kdeps/projects/.../index.db):
- /home/me/code/my-app/billing/invoices.go - monthly invoices go out on the first day
Folder /home/me/docs/billing-specs (search index /home/me/.kdeps/projects/.../index.db):
- /home/me/docs/billing-specs/proration.md - prorate upgrades to the day
</knowledge-files>
```

```text
<knowledge-recall>
Notes in the knowledge vault related to this request, from its link graph (best first). ...
- /home/me/notes/projects/my-app/decisions/Billing decision.md - Bill monthly, not per call. ...
- /home/me/notes/shared/Stripe webhook retries.md - Webhooks retry for 3 days. (projects/app/decisions/Billing decision.md --relates_to--> shared/Stripe webhook retries.md)
</knowledge-recall>
```

kdeps adds recall to the turn, not to the cached system prompt. The terminal shows its cost:

```text
recall: 4 notes (~180 tok), 2 files
```

| Command | Shows |
|---|---|
| `/recall` | the last turn's recall and why each note was chosen |
| `/recall <words>` | recall for other words |
| `/knowledge` | whether the vault is on, and where |
| `kdeps knowledge recall "why monthly billing"` | recall from the shell |

## Search folders

A search folder is a folder outside the project (docs, specs, a shared library) that recall and `search_local` cover as part of the project. kdeps keeps its index under `~/.kdeps`. It writes nothing into the folder.

```d2
direction: right
add: "kdeps --index ~/docs\n/search add ~/docs\nknowledge_card_update" {shape: oval}
card: "project card\n| search | /home/me/docs |"
index: "~/.kdeps/projects/docs-<hash>/index.db"
recall: "recall: <knowledge-files>\nper folder"
search: "search_local\n(no path: every folder)"
add -> card: "records the full path"
add -> index: "indexes now"
index -> recall
index -> search
card -> search: "which folders"
```

The card stores it as a `search` row. Unlike `repo` or `folder`, a session started inside a search folder gets its own project:

```markdown
| Source | Path |
|---|---|
| repo | /home/me/code/my-app |
| search | /home/me/docs/billing-specs |
| index | /home/me/.kdeps/projects/billing-specs-3f2a91c0/index.db |
```

Add one at session start, from the shell, in a session, or let the model add it:

```bash
kdeps --index ~/docs/billing-specs                # add and index before the session starts; repeatable, fine to pass every time
kdeps knowledge folders                           # list the project's folders and their indexes
kdeps knowledge folders add ~/docs/billing-specs  # add and index now
kdeps knowledge folders index                     # update every folder's index
kdeps knowledge folders remove ~/docs/billing-specs
```

```text
/search folders                   list them
/search add ~/docs/billing-specs  add and index now
/search remove ~/docs/billing-specs
/search index                     update the index of every folder the project searches
```

The model uses `knowledge_card_update` with `add_search_folder` or `remove_search_folder`.

kdeps refuses: your home folder, a filesystem root, the vault, the project's own folder, a folder inside one already searched, and a non-folder path.

Key and credential files (SSH keys, `.env`, `*.pem`, `credentials`, anything in `.ssh`, `.aws` or `.gnupg`) are never indexed or returned, in any folder. The lists are in the set's `search.yaml` (`secret_names`, `secret_prefixes`, `secret_extensions`, `secret_dirs`). Without that file, kdeps indexes nothing. A deleted search folder shows up in `kdeps knowledge check`.

## Check

```bash
kdeps knowledge check
```

```text
  projects/my-app/notes/old status.md: has a next step; it belongs in the project card (one home per fact)
  shared/Graph recall.md: links to [[Louvain]], which has no note
  shared/Pricing.md: has the same summary as shared/Tiers.md; keep the fact in one note and link it
  projects/app/notes/Alpha.md: claims canonical id ship-monthly, also claimed by /home/me/notes/projects/app/notes/Beta.md; one fact has one home
  projects/web/web.md: card holds 9 lines of prose; move the story to a vault note and link it
  projects/my-app/my-app.md: search folder /home/me/old-specs is gone; remove it with kdeps knowledge folders remove /home/me/old-specs
```

It exits non-zero on any finding, so it can run in CI.

The set's `check.yaml` sets which notes need a summary and each finding's wording. Override it under `~/.kdeps/`:

```yaml
summaries:
  shared: true  # notes in the shared folder need a summary
  project_parts: [notes, decisions]  # project parts (vault.yaml keys) whose notes need one
  exempt_parts: [memory]  # parts whose notes may share a summary and need none
messages:
  dangling_link: "links to [[{{.Target}}]], which has no note"  # .Target: the link target
```

Install rejects a `check.yaml` with an empty message, an invalid template, or an unknown part.

## Tools

| Tool | Does |
|---|---|
| `knowledge_recall` | Recall notes for other words than the request |
| `knowledge_card_get` | Read a card (default: the working directory's project, created on first use) |
| `knowledge_card_update` | Set `status` and/or `next`; stamps `updated`; `archived` moves the project to `archive/`; `add_search_folder` / `remove_search_folder` change the [search folders](#search-folders) |
| `knowledge_note_save` | Save a new one-idea note; `scope`: `project` (default), `shared` or `inbox` |

They are available in agent sessions when the vault is on. They refuse when `/knowledge off` is set. They are part of the `agent` asset profile.
