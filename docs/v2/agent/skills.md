# Skills and prompt templates

Markdown files that shape how the agent behaves: skills (reusable behavior guidelines), prompt templates (reusable named prompts), and instructions (project rules discovered automatically). This is agent mode only. These are files the kdeps REPL loads. They are not the [kdeps skill for coding agents](/agent/ai-assisted-authoring) that scaffolds YAML in Claude Code or Cursor.

## Skills

Skills are markdown files with optional YAML frontmatter that teach the agent how to behave in specific contexts. Place them in `~/.kdeps/skills/` or pass `--skill <path>` at startup.

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
---

Always check for error handling. Prefer early returns over nested conditions.
```

kdeps finds skills in these locations:

- `~/.kdeps/skills/` (global)
- `~/.agents/skills/` (where `npx skills` installs)
- `./.kdeps/skills/` (the project's own, see [The .kdeps folder](#the-kdeps-folder))
- paths passed with `--skill` (explicit, repeatable)

Invoke a skill from the REPL with `/<skill-name>` or `/<skill-name> extra context here`. The skill's text reaches the model with its folder named first, so a relative link in it (`agents/reviewer.md`, `scripts/run.sh`) resolves.

### Progressive disclosure

The system prompt lists only the name and description of each skill. It never has the full body. kdeps sends the system prompt again on each LLM call. With the full bodies of many skills, each turn uses many tokens. For this reason, the agent calls the built-in `load_skill` tool with a skill name. This tool gets the full instructions of that skill only when a task needs them.

### Related skills

When it loads or reloads, kdeps records the related skills, without a model call:

- **A skill in the [knowledge vault](/agent/knowledge):** from the vault graph, refreshed first. A `references` edge either way (markdown or wiki link) or a shared tag relates two skills.
- **Any other skill:** from a reference/topic graph over each skill-library root, the graph `search_local` builds outside the vault. Two skills are related if the `SKILL.md` of one skill links to the other (`[other](../other/SKILL.md)`). They are also related if both have the same `topics:` or `tags:` in frontmatter:

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
topics: [go, quality]
---

Always check for error handling. See [testing](../testing/SKILL.md) for coverage expectations.
```

If the graph of a skill has related skills, `load_skill` adds a hint with their names. The model can then decide if it loads them too. This only adds data. A skill with no links or topics has the same behavior as before.

## Prompt templates

Prompt templates are reusable named prompts loaded from `.md` files. They work exactly like skills: invoke them by name from the REPL.

```markdown
---
name: review-pr
description: Review a GitHub pull request
argument-hint: <PR number or URL>
---

Review the pull request at $1. Check for: correctness, test coverage, and breaking changes.
```

Place templates in `~/.kdeps/prompts/` or the project's `./.kdeps/prompts/`. Templates use the same placeholder syntax as skills: `$1`, `$2`, `$@`, `${1:-default}`.

```bash
/review-pr 1234
/summarize this document for a technical audience
```

## Instructions

The agent automatically discovers instruction files by walking up the directory tree from the current directory:

- `KDEPS.md`, `KDEPS.local.md` at any ancestor directory
- `CLAUDE.md`, `CLAUDE.local.md`, `AGENTS.md`, `GEMINI.md` at any ancestor directory
- every `*.md` file directly in a `.kdeps/` folder at any ancestor directory (`.kdeps/KDEPS.md` first, then the rest by name)

`KDEPS.md` is kdeps' own instruction file and has the highest priority. kdeps finds it first, and it keeps its full character budget. The agent gets an instruction that `KDEPS.md` has priority over `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, and the other files.

kdeps removes duplicate content (by hash). The maximum added context is about 12 KB. kdeps adds the instructions to the system prompt at startup.

## The .kdeps folder

A project's `.kdeps/` folder holds Markdown you write for the agent. kdeps only reads it, never writes to it, so it is safe to commit. Everything kdeps generates (memory, chat history, indexes) lives in `~/.kdeps`.

```text
my-project/
  KDEPS.md              # project rules, read first
  .kdeps/
    style.md            # read as instructions, like KDEPS.md
    architecture.md     # every *.md here is read
    skills/
      review.md         # /review
    prompts/
      release-notes.md  # /release-notes
```

## See also

- [Skill for coding agents](/agent/ai-assisted-authoring) - YAML scaffolding for Claude Code, Cursor, Grok
- [Agent mode](/agent/) - starting the loop
- [REPL slash commands](/agent/commands) - invoking skills and templates
