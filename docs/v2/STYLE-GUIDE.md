# kdeps documentation style guide

This guide governs every page under `docs/v2/`. It merges the
[Good Docs Project](https://thegooddocsproject.dev/) style guidance with the
kdeps house rules. When this guide is silent, fall back to the
[Google developer documentation style guide](https://developers.google.com/style).

## Content types

Every page is exactly one content type. Pick the type first, then use its
skeleton from [Contributing to docs](./CONTRIBUTING-DOCS.md).

| Type | Answers | Lives in |
| :--- | :--- | :--- |
| Concept | "What is it and why does it matter?" | `concepts/`, `modes/` |
| Tutorial | "Teach me by building one example." | `getting-started/`, `examples/` |
| How-to | "How do I do one specific task?" | `guides/`, `deployment/` |
| Reference | "What are the exact fields, flags, and values?" | `reference/`, `resources/`, `configuration/` |
| Quickstart | "Get me to a working result fast." | `getting-started/quickstart` |
| Troubleshooting | "Something broke. Fix it." | `guides/troubleshooting` |
| Glossary | "Define this term." | `reference/glossary` |
| Release notes | "What changed?" | `reference/release-notes` |

## Two modes - always label

kdeps has two modes. Every concept, feature, how-to, tutorial, and deployment
page states which mode(s) it applies to in the first paragraph or an info
callout. Pure CLI command references and the glossary are exempt - a command
reference documents the CLI itself, not a mode-specific feature.

- **Workflow mode** - DAG-deterministic request/response pipelines.
- **Agent mode** - autonomous LLM agents with tool use and memory.

## Voice and tone

- Mildly casual, serious but not stiff, warm and direct.
- Second person ("you"), present tense, active voice.
- No opinions, no hype, no marketing adjectives.
- Answer first. Context second. Nice-to-know last (inverted pyramid).

## Writing rules

- Sentence case for all titles, headings, list items, and table headers.
- Spell out an acronym on first use per page; established acronyms (URL, HTTP,
  JSON, YAML, API, CLI, SQL) are exempt.
- Short to medium sentences. Cut every word that has not earned its place.
- Do not cluster more than three nouns. Break with "of", "for", "with".
- No metaphors, idioms, or slang.
- Refer to one thing with one term, always. Do not alternate synonyms.
- Cap each topic (each H2 section) at roughly 300 words. Split if longer.

## Typography - ASCII only

- Hyphens, not em dashes.
- Straight quotes, not curly.
- Three dots, not the ellipsis character.
- Hyphen or asterisk bullets, not Unicode bullets.
- No non-breaking spaces.

## Page structure

- Start with a one-sentence summary: "X does Y" or "X is like Z, but W".
- Diagram before prose for any multi-step or multi-component concept: D2
  (` ```d2 ` block) for 4+ nodes or branching, ASCII (` ```text ` block) for
  linear flows of 3 nodes or fewer. Do not mix both on one page.
- Lead with plain English before any YAML or code.
- Every non-obvious YAML field gets an inline comment explaining purpose and
  behavior: `timeout: 30s  # hard stop - returns error, does not retry`.
- No dot-notation for config (`settings.apiServer.auth`). Show the real YAML block.
- Collect every inline link into a `See also` section at the end.

## Procedures

- One sentence per step, starting with a verb.
- Chunk procedures longer than 10 steps into sub-sections of 5 to 10 steps.
- No more than four sub-steps per step.
- Mark optional steps with a leading `Optional:`.
- Start conditional steps with the condition: "If you use Docker, ...".

## Code examples

- Every example must stand alone. A developer copies it directly into a project
  without reading the surrounding prose.
- Show the mechanics: where the data goes, what runs next, what the output is.
- Build the kdeps binary with `make build`; the binary is `./kdeps`.

## Checklist before merging

- [ ] Page is one content type and follows that skeleton.
- [ ] Mode(s) stated in the first paragraph.
- [ ] Summary sentence is the first line of body text.
- [ ] Diagram precedes prose for multi-step concepts.
- [ ] All acronyms expanded on first use.
- [ ] ASCII typography only.
- [ ] Every topic under ~300 words.
- [ ] `See also` section present with every inline link.
- [ ] `cd docs/v2 && npm run build` passes.
