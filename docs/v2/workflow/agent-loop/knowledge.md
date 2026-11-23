# Knowledge vault

The knowledge vault is a long-term knowledge base of markdown notes. A workflow [`chat`](/workflow/resources/llm#knowledge-recall) uses it without a full read. On each call, kdeps follows the link graph of the vault from the words of the prompt, with no model. Then it gives the model some note paths with one-line summaries.

```d2
direction: down
q: "chat prompt" {shape: oval}
recall: "graph recall (no model)\nwords -> matching notes -> 2 hops"
notes: "3-8 notes named\npath + summary"
model: "model reads only\nthe notes it needs" {shape: oval}
q -> recall -> notes -> model
```

Recall runs on each workflow `chat` call. A chat with `loop:` gets no recall; its model can call [`knowledge_recall`](#tools) when `loop.tools` lists it.

A 2,000-note vault holds about a million words, more than any model can read. With recall, a question typically reads a handful of notes.

## One home per fact

| Fact | Its only home |
|---|---|
| Status, next step, places on disk | the project's card: `projects/<name>/<name>.md` |
| A workflow's calls of one day | the run log: `projects/<name>/sessions/` (with `workflow_notes: true`) |
| Why something was decided | one decision note: `projects/<name>/decisions/` |
| What the agent remembers | memory notes: `projects/<name>/memory/` (see [Persistent memory](/workflow/agent-loop/memory)) |
| An idea, gotcha or concept | one note: `projects/<name>/notes/`, or `shared/` when it spans projects |
| How notes connect | the link graph, under `~/.kdeps/knowledge/<vault-id>/` |

These folder names and the card's front matter keys are defaults in the `knowledge` asset set's `vault.yaml`. Override the file under `~/.kdeps/` to rename them:

```yaml
folders:
  projects: projects  # one folder per project
  shared: shared  # knowledge that spans projects
project:
  sessions: sessions  # a workflow's daily run log
  decisions: decisions  # one note per decision; you write them
card:
  next: next  # front matter key of the one next step; it lives only in the card
  links_header: [Source, Path]  # the card's places table: a link kind, then a path
note:
  summary: summary  # the line recall shows before anyone opens a note
  canonical: [canonical, fact_id]  # a stable fact id; the first key that is set wins
```

You must set each name. Install rejects a `vault.yaml` with an empty name.

Every note kdeps writes comes from a Go template in the set's `notes.yaml`: the card, a workflow's daily run log, `GRAPH_REPORT.md` and `kdeps knowledge explain` lines. Install rejects a `notes.yaml` with a missing or broken template.

The `extract.yaml` of the set defines what model extraction can name and how its answer joins the graph:

```yaml
entity_kinds: [project, tool, decision, rationale, gotcha, concept]  # other kinds become default_kind
relations: [depends_on, rationale_for, relates_to, supersedes, references]  # kept as the model named them
fallback_relation: relates_to  # any other relation becomes this...
fallback_max_confidence: 0.49  # ...at most this sure, so it is marked ambiguous
```

`kdeps knowledge check` reports structural duplicates: the same summary twice, a next step outside a card, or two notes with the same `canonical` or `fact_id`. It does not detect the same fact in different wording.

## Projects

A project is one folder of the vault with a card. kdeps creates it the first time something writes for a place no card covers:

- a workflow run log (`workflow_notes: true`), named after the workflow;
- [persistent memory](/workflow/agent-loop/memory) (`kdeps run --memory`), named after the git repository root or the folder;
- `kdeps knowledge folders add`, for the working folder.

```text
~/.kdeps/notes/                       # the vault (knowledge.vault)
  KDEPS.md                            # routing table + rules
  schema.md                           # how cards and notes are written
  projects/
    my-app/
      my-app.md                       # the card: status, next step, places on disk
      sessions/                       # a workflow's daily run log
      memory/                         # one note per memory entry; .auto/ holds thinking and tool captures
      decisions/                      # one note per decision; you write them
      notes/                          # your notes
  shared/                             # knowledge across projects; shared/memory/ with KDEPS_MEMORY_GLOBAL=1
  inbox/                              # notes dropped by hand, not yet filed
  archive/                            # projects whose card says status: archived
```

The link graph, its caches, `GRAPH_REPORT.md` and `graph.html` live outside the vault, under `~/.kdeps/knowledge/<vault-id>/` (a hash of the vault path).

## Set up

The vault is on by default, in `~/.kdeps/notes`. The first chat call creates it and refreshes its graph before recall.

| Turn it | How |
|---|---|
| off for good | `knowledge.enabled: false` in `~/.kdeps/config.yaml` |
| to another folder | `knowledge.vault` in config.yaml, or `kdeps knowledge init <folder>` |

```yaml
# ~/.kdeps/config.yaml
knowledge:
  enabled: true               # false: no recall and no notes
  vault: ~/.kdeps/notes       # the default
  workflow_notes: false       # workflow chat calls log to their project's sessions/ (default false)
  recall:
    hops: 2                   # 1 or 2; a higher value is clamped to 2
    budget_tokens: 1500       # most tokens of recalled paths and summaries per call
```

## Cards

A card points. It never tells the story. kdeps writes this one for `~/code/my-app`:

```markdown
---
type: project          # any word: project, client, area, research, bot
status: active         # the only place status lives; archived moves the folder to archive/
next: (none)           # the one next step
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
| `index` | a place's search index (built by `kdeps knowledge folders index`, kept under `~/.kdeps`); added by kdeps |

You edit cards by hand. `kdeps knowledge folders` changes the `search` rows.

## Notes

Every note is markdown with frontmatter. `summary` is what recall shows.

```markdown
---
type: decision          # decision | concept | gotcha | tool | note
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
- One idea per note. `kdeps knowledge check` reports two notes with the same summary or `canonical` id.

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
- Run logs join the graph on the next refresh (a chat call or a vault search).

## The graph

The structural graph refreshes automatically at these times:

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

Extraction runs from the shell: `kdeps knowledge build --extract [--model <model>] [--limit N]`. The model is `--model`, then `llm.subagent`, then `knowledge.extract.model`, then the current model. Recall reads the graph and never waits for a model.

kdeps reads a note again only when it changes. It tries a failed note again on the next run. A build without `--extract` prints how many notes are waiting. The graph records the model and time of the latest successful extraction.

```yaml
# ~/.kdeps/config.yaml
knowledge:
  extract:
    model: llama3.2:3b        # used when --model and llm.subagent are not set
    workers: 2                # parallel calls after the first
```

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

Workflow chat calls, vault searches and skill linking refresh the link graph before they read it. Refreshes never call a model.

## Recall

Before each chat call, kdeps:

1. **Scores nodes on the request's words.** Rarer words count more, title words count double, and word forms meet (`billing` matches `bill`). Run logs score 0.4x and memory notes 0.6x, below curated notes.
2. **Walks at most 2 edges** from the best matches, strongest first. Extracted edges count fully, inferred edges by their confidence, ambiguous edges at 0.3 times their confidence. Tag edges are then multiplied by 0.3. Each hop divides the score by 2. If `knowledge.recall.hops` is more than 2, kdeps uses 2.
3. **Lists the best notes** within `budget_tokens`, as paths for `read_file`. A note reached through edges shows the chain, e.g. `projects/app/decisions/Billing decision.md --relates_to--> shared/Stripe gotcha.md`.

```text
<knowledge-recall>
Notes in the knowledge vault related to this request, from its link graph (best first). ...
- /home/me/notes/projects/my-app/decisions/Billing decision.md - Bill monthly, not per call. ...
- /home/me/notes/shared/Stripe webhook retries.md - Webhooks retry for 3 days. (projects/app/decisions/Billing decision.md --relates_to--> shared/Stripe webhook retries.md)
</knowledge-recall>
```

kdeps adds recall as a system message after the chat's system prompt. To test recall from the shell, run `kdeps knowledge recall "why monthly billing"`.

## Search folders

A search folder is a folder outside the project (docs, specs, a shared library) that the project's card records. kdeps indexes it under `~/.kdeps`. It writes nothing into the folder. `search_local` with `path` set to the folder reads that index.

```bash
kdeps knowledge folders                           # list the project's folders and their indexes
kdeps knowledge folders add ~/docs/billing-specs  # add and index now
kdeps knowledge folders index                     # update every folder's index
kdeps knowledge folders remove ~/docs/billing-specs
```

The card stores it as a `search` row, and the index as an `index` row:

```markdown
| Source | Path |
|---|---|
| repo | /home/me/code/my-app |
| search | /home/me/docs/billing-specs |
| index | /home/me/.kdeps/projects/billing-specs-3f2a91c0/index.db |
```

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

`knowledge_recall` recalls notes for other words than the prompt. A `chat` lists it in `tools:`, or a chat with `loop:` in `loop.tools`. It is also a kdeps API tool.
