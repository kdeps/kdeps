# Knowledge vault

The knowledge vault gives the agent loop a long-term knowledge base it can use without reading it. Each turn, kdeps walks the vault's link graph from the words of the request, with no model involved. It then hands the model the few notes worth reading (paths and one-line summaries), not the whole vault.

```d2
direction: down
q: "your request" {shape: oval}
route: "KDEPS.md routing table\n(where everything lives)"
card: "project card\nstatus, next step, links"
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
| Status, next step, where things live | the project's card: `records/projects/<name>.md` |
| What happened: focus, what was built, mistakes, decisions, wins | a note in the vault |
| How notes connect | the link graph, in the vault's hidden `.index/` |
| What the agent remembers | memory notes in `Memory/<project>/` (see [Persistent memory](/agent/memory)) |
| The current session's scratch facts | [persistent memory](/agent/memory) |

A fact written in two places goes stale in one of them. `kdeps knowledge check` finds those.

## Set up

Nothing to set up: the vault is on by default for agent sessions, in `~/.kdeps/notes`. The first session creates it, every session rebuilds its graph in the background and writes its own note.

| Turn it | How |
|---|---|
| off for good | `knowledge.enabled: false` in `~/.kdeps/config.yaml` |
| off or on, remembered across sessions | `/knowledge off`, `/knowledge on` |
| to another folder | `knowledge.vault` in config.yaml, or `kdeps knowledge init <folder>` (an existing notes folder works as is) |

`init` creates only what the folder lacks (and records a non-default folder in `~/.kdeps/config.yaml`):

```text
~/.kdeps/notes/
  KDEPS.md              # routing table + rules; loaded into every session
  schema.md             # how to write cards and notes
  records/
    projects/           # one card per project
    clients/            # one card per client
    sessions/  decisions/
  references/
  Memory/<project>/    # agent memory, one note per entry
  .index/              # graph, caches, GRAPH_REPORT.md, graph.html
  Inbox/  Projects/  Areas/  Resources/  Archives/   # reused if the folder has them (any number prefix: "1-Projects")
```

```yaml
# ~/.kdeps/config.yaml
knowledge:
  enabled: true               # false: no recall, no session notes, no knowledge tools
  vault: ~/.kdeps/notes       # the default
  workspace: ~/.kdeps/notes   # where KDEPS.md + records/ live; default: the vault
  session_notes: true         # write a note per session (default true)
  recall:
    hops: 2                   # how far from the matched notes to walk
    budget_tokens: 1500       # most tokens of recalled paths and summaries per turn
```

## Cards

A card is short on purpose. It points; it never tells the story.

```markdown
---
type: project          # project | client
status: active         # the only place status lives
next: wire recall into the desktop app
updated: 2026-10-15
---
| Source | Path |
|---|---|
| repo | ~/code/my-app            |   <- kdeps finds this card when it runs in this folder
| vault | Projects/my-app         |   <- where the project's notes and sessions go
| memory | kdeps memory (repo folder) |
| skills | skills/my-app          |
```

Every turn carries a status overview read from the cards at that moment (`<knowledge-status>`); the overview is never stored. The model reads and changes cards with `knowledge_card_get` and `knowledge_card_update`.

## Notes

Every note is plain markdown with frontmatter. `summary` is what recall shows before anyone opens the note.

```markdown
---
type: decision          # session | decision | hub | concept | gotcha | tool | note
status: evergreen
tags: [billing]
created: 2026-10-15
summary: Bill monthly, not per call - per-call billing doubled churn.
links: ["[[Stripe webhook retries]]"]
---
We moved to monthly billing because ...
```

- Link notes with `[[Note title]]`. A link resolves by path, by title anywhere in the vault (the closest one wins when two share a title) or by `aliases:`.
- Links inside code blocks are examples, not links.
- One idea per note in `Resources/`. The model saves new ones with `knowledge_note_save`, which refuses a title that already has a note.

## Session notes

After every turn, kdeps writes (and rewrites in place) one note per session into the project's `sessions/` folder (the inbox when no card covers the folder). It rewrites that note in place as the session goes on, and links it from the project's hub note:

```markdown
---
type: session
project: my-app
session: 7f3c...
summary: Ship knowledge recall for my-app
---
## Focus
## What was built
## Mistakes
## Decisions
## Wins
```

After each turn the note is updated without a model call: your prompts, the files written or edited, decisions saved to memory and tools that failed. At compaction the model's summary replaces it with a written one (no extra model call: compaction makes that summary anyway), and later turns are added on top. When an interactive terminal session ends, one summary call covers the turns since the last note; a piped or scripted session exits at once (its compactions still write notes). The summary's first next step goes to the card's `next`, not into the note.

## The graph

`kdeps knowledge build` turns the vault into a graph:
- **Nodes:** each note, each tag, and each Go package of every card's `repo`.
- **Edges:** wikilinks (`references`), frontmatter links (`relates_to`), tags (`tagged`) and Go imports (`depends_on`).
- **Provenance:** every edge records how it was found. Edges from links and imports are `extracted`; code goes through the Go parser, never a model. A local model can add `inferred` edges (see [Model extraction](#model-extraction)).

```text
$ kdeps knowledge build
2003 notes (2003 changed), 1986 code files -> 4210 nodes, 9873 edges in 1.2s
$ kdeps knowledge build
2003 notes (0 changed), 1986 code files -> 4210 nodes, 9873 edges in 270ms
```

Code files are the ones git tracks or would track, so `.gitignore`d folders stay out; without git, every `.go` file outside `vendor/`, `testdata/` and hidden folders.

The graph lives in the vault's hidden `.index/` folder (`graph.json`, caches, `GRAPH_REPORT.md`, `graph.html`), so it travels with the vault and is never read as a note. A rebuild re-reads only changed files.

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

Each build groups the graph into communities with the Louvain method: notes and packages that link mostly among themselves. Each node gets a `community` number in `graph.json`; 0 is the largest. Next to the graph in `.index/`, the build writes `GRAPH_REPORT.md` and `graph.html`, a page to explore the graph (search, zoom, hover for a note's summary; colors are communities).

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
- [[Projects/app/Billing decision]] (community 3, 40 nodes) -[references, extracted]-> [[Resources/Kubernetes probes]] (community 9, 25 nodes)
## Links to notes that do not exist
```

A surprising connection is a single link between two communities that otherwise never touch, ranked by the size of the smaller one: a lone bridge between two large groups is the least expected.

Inside an agent session, kdeps rebuilds the graph in the background when the session starts and after it writes a note.

## Recall

Before each model call, kdeps:
1. Scores graph nodes on the request's words. Rarer words count more, a word that is part of a note's title counts double, and word forms meet (`billing` matches `bill`). Notes kdeps writes itself rank lower than notes you curate: session notes at 0.4x, memory notes at 0.6x. The session's own note is never recalled into that session.
2. Walks up to `hops` edges from the best matches. Extracted edges count fully, and each hop halves the score.
3. Hands the model the best notes within `budget_tokens`, as paths it can open with `read_file`.

```text
<knowledge-recall>
Notes in the knowledge vault related to this request, from its link graph (best first). ...
- /home/me/notes/Projects/my-app/Billing decision.md - Bill monthly, not per call. ...
- /home/me/notes/Resources/Stripe webhook retries.md - Webhooks retry for 3 days. (linked from Billing decision)
</knowledge-recall>
```

Recall is added to the turn, not to the cached system prompt, so each request gets its own. The terminal shows what it cost:

```text
recall: 4 notes (~180 tok)
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
  Projects/my-app/old status.md: has a next step; it belongs in the project card (one home per fact)
  Resources/Graph recall.md: links to [[Louvain]], which has no note
  Resources/Pricing.md: has the same summary as Resources/Tiers.md; keep the fact in one note and link it
  records/projects/web.md: card holds 9 lines of prose; move the story to a vault note and link it
```

It exits non-zero when it finds something, so it can run in CI.

## Tools

| Tool | Does |
|---|---|
| `knowledge_recall` | Recall notes for other words than the request |
| `knowledge_card_get` | Read a card (default: the working directory's) |
| `knowledge_card_update` | Set `status` and/or `next`; stamps `updated` |
| `knowledge_note_save` | Save a new one-idea note with frontmatter |

They are offered in agent sessions while the vault is on (the default), refuse while `/knowledge off`, and ship with the `agent` asset profile.
