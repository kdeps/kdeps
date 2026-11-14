# kdeps documentation style guide

This guide applies to all pages under `docs/v2/`. All prose follows
[ASD-STE100 Simplified Technical English](https://www.asd-ste100.org/) (STE).
The guide also uses the
[Good Docs Project](https://thegooddocsproject.dev/) style guidance and the
kdeps house rules. If this guide does not give a rule, use the
[Google developer documentation style guide](https://developers.google.com/style).

## Content types

Every page is exactly one content type. Pick the type first, then use its
skeleton from [Contributing to docs](./contributing-docs).

| Type | Answers | Lives in |
| :--- | :--- | :--- |
| Concept | "What is it and why does it matter?" | `start/` |
| Tutorial | "Teach me by building one example." | `examples/` |
| How-to | "How do I do one specific task?" | `agent/`, `workflow/`, `deploy/`, `agencies/`, `llm-server/`, `registry/` |
| Reference | "What are the exact fields, flags, and values?" | `reference/`, `workflow/resources/` |
| Quickstart | "Get me to a working result fast." | `workflow/quickstart`, `agent/quickstart` |
| Troubleshooting | "Something broke. Fix it." | `reference/troubleshooting` |
| Glossary | "Define this term." | `reference/glossary` |

## Two modes

The directory is the label. `agent/` is agent mode. `workflow/`, `deploy/`, and
`examples/` are workflow mode. On a shared page (`start/`, `reference/`), name
the mode in the first sentence. Do not add an "Applies to" banner. A CLI
reference and the glossary document the CLI and the terms, not a mode.

- **Workflow mode** - DAG-deterministic request/response pipelines.
- **Agent mode** - autonomous LLM agents with tool use and memory.

## Voice and tone

- Direct and serious. Not stiff.
- Second person ("you"), present tense, active voice.
- No opinions, no hype, no marketing adjectives.
- Answer first. Context second. Nice-to-know last (inverted pyramid).

## ASD-STE100 rules

All prose follows ASD-STE100. Code blocks, YAML, commands, and names of fields,
flags, and products are technical names. Do not change them.

- **Words**: use the approved meaning of a word. Use one word for one meaning.
  Use technical names and technical verbs from the product (`run`, `build`,
  `deploy`) as they are.
- **Sentence length**: a procedural sentence has a maximum of 20 words. A
  descriptive sentence has a maximum of 25 words.
- **Paragraphs**: one topic for each paragraph. A paragraph has a maximum of
  six sentences.
- **Instructions**: write one instruction in each sentence, in the imperative
  form. If two actions occur at the same time, put them in one sentence.
- **Conditions**: put the condition first: "If X, do Y."
- **Active voice**: use the active voice. Use the passive voice only in
  descriptive text, if you do not know the agent.
- **Verb tenses**: use the simple present, simple past, or simple future only.
- **-ing forms**: do not use the "-ing" form of a verb as a noun or as a
  modifier. Write "the process that runs", not "the running process".
- **Articles**: use "the", "a", or "this" before a noun when possible.
- **Warnings and cautions**: start with a short, clear command. Then give the
  risk.

Replace these words with their STE alternatives:

| Do not use | Use |
| :--- | :--- |
| allow, enable (to permit) | let, can |
| ensure | make sure |
| e.g., i.e., etc. | for example, that is, (list the items) |
| in order to | to |
| may, might, could | can |
| obtain | get |
| prior to | before |
| should | must, or the imperative |
| since (because) | because |
| utilize, leverage | use |
| via | through, with |
| whether | if |
| while (at the same time) | when |
| simply, just, easily | (delete) |

## Writing rules
- Sentence case for all titles, headings, list items, and table headers.
- Spell out an acronym on first use per page. Established acronyms (URL, HTTP,
  JSON, YAML, API, CLI, SQL) are exempt.
- Short sentences (see [ASD-STE100 rules](#asd-ste100-rules)). Remove words
  that do not give information.
- Do not cluster more than three nouns. Break with "of", "for", "with".
- No metaphors, idioms, or slang.
- Use full forms, not contractions: "do not", "it is", "cannot".
- State the point in the first sentence of a page or section. No warm-up.
- Do not hedge or apologize. State limits flatly: "X does not support Y."
- Bold only labels (start of a list item or table cell), a term's defining use,
  and warnings. No bold for emphasis in running prose.
- Refer to one thing with one term, always. Do not alternate synonyms.
- Cap each topic (each H2 section) at roughly 300 words. Split if longer.

## Typography - ASCII only

- Hyphens, not em dashes.
- Straight quotes, not curly.
- Three dots, not the ellipsis character.
- Hyphen or asterisk bullets, not Unicode bullets.
- No non-breaking spaces.

## Page structure

- Start with a one-sentence summary: "X does Y".
- Diagram before prose for any multi-step or multi-component concept: D2
  (` ```d2 ` block) for 4+ nodes or branching, ASCII (` ```text ` block) for
  linear flows of 3 nodes or fewer. Do not mix both on one page.
- Write for a working developer. Do not explain programming basics (imports,
  variables, functions, YAML, `curl`) or use analogies to them. State what the
  thing does.
- No audience lines ("this page is for..."), learning objectives, "you will be
  able to", or end-of-page summaries of what the reader did.
- State the mode only if the section does not make it clear. A page under
  `agent/` is agent mode. A page under `workflow/`, `deploy/`, or `examples/`
  is workflow mode. A shared page states the mode in its first sentence.
- Every non-obvious YAML field gets an inline comment explaining purpose and
  behavior: `timeout: 30s  # hard stop - returns error, does not retry`.
- No dot-notation for config (`deploy.apiServer.auth`). Show the real YAML block.
- Collect every inline link into a `See also` section at the end.

## Procedures

- End each procedure with the result that the reader sees: the real response, the
  changed row, or the file that now exists. Run it to get that output. Mark text
  produced by a model as varying.

- One sentence per step, starting with a verb.
- Chunk procedures longer than 10 steps into sub-sections of 5 to 10 steps.
- No more than four sub-steps per step.
- Mark optional steps with a leading `Optional:`.
- Start conditional steps with the condition: "If you use Docker, ...".

## Code examples

- Every example must stand alone. A developer copies it directly into a project
  without reading the surrounding prose.
- Show the mechanics: where the data goes, what runs next, what the output is.
- Build the kdeps binary with `make build`. The binary is `./kdeps`.

## Checklist before merging

- [ ] Prose follows the [ASD-STE100 rules](#asd-ste100-rules).
- [ ] Page is one content type and follows that skeleton.
- [ ] Mode(s) stated in the first paragraph.
- [ ] Summary sentence is the first line of body text.
- [ ] Diagram precedes prose for multi-step concepts.
- [ ] All acronyms expanded on first use.
- [ ] ASCII typography only.
- [ ] Every topic under ~300 words.
- [ ] `See also` section present with every inline link.
- [ ] `cd docs/v2 && npm run build` passes.
