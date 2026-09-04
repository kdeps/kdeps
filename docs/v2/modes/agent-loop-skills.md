# Skills and prompt templates

Markdown files that shape how the agent behaves: **skills** (reusable behavior guidelines), **prompt templates** (reusable named prompts), and **instructions** (project rules discovered automatically). This is **agent mode** only. To install pre-built skill sets, see [Agent skills](/getting-started/agent-skills).

## Skills

Skills are markdown files with optional YAML frontmatter that teach the agent how to behave in specific contexts. Place them in `~/.kdeps/skills/` or pass `--skill <path>` at startup.

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
---

Always check for error handling. Prefer early returns over nested conditions.
```

Skills are discovered from:

- `~/.kdeps/skills/` (global)
- `./.kdeps/skills/` (project-local)
- paths passed with `--skill` (explicit, repeatable)

Invoke a skill from the REPL with `/<skill-name>` or `/<skill-name> extra context here`.

### Progressive disclosure

The system prompt lists only each skill's name and description - never the full body. Skill instructions are re-sent on every LLM call as part of the system prompt, so embedding full bodies for a large skill set would burn tokens every turn. Instead, the agent calls the built-in `load_skill` tool with a skill name to pull that skill's full instructions on demand, only when a task actually needs it.

### Related skills

When skills are (re)loaded, kdeps builds a small [kartographer](https://github.com/kdeps/kartographer) reference/topic graph over each skill-library root - the same mechanism `codeIntelligence`'s [`indexFolder` / `graphFile`](/resources/code-intelligence/graph) uses on any folder. A skill is related to another if its `SKILL.md` links to it (`[other](../other/SKILL.md)`), or if both declare the same `topics:` / `tags:` in frontmatter:

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
topics: [go, quality]
---

Always check for error handling. See [testing](../testing/SKILL.md) for coverage expectations.
```

When `load_skill` returns a skill whose graph has related skills, it appends a hint listing their names so the model can decide whether to load them too. This is purely additive - a skill with no links or topics behaves exactly as before.

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

Place templates in `~/.kdeps/prompts/` or `./.kdeps/prompts/`, or pass `--prompt <dir>`. Templates use the same placeholder syntax as skills: `$1`, `$2`, `$@`, `${1:-default}`.

```bash
/review-pr 1234
/summarize this document for a technical audience
```

## Instructions

The agent automatically discovers instruction files by walking up the directory tree from the current directory:

- `KDEPS.md`, `KDEPS.local.md` at any ancestor directory
- `CLAUDE.md`, `CLAUDE.local.md`, `AGENTS.md`, `GEMINI.md` at any ancestor directory
- `.kdeps/KDEPS.md`, `.kdeps/CLAUDE.md`, `.kdeps/instructions.md` at any ancestor directory

`KDEPS.md` is kdeps' own instruction file and takes the highest priority: it is discovered first, keeps its full character budget, and the agent is told that `KDEPS.md` wins any conflict with `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, or the rest.

Duplicate content (by hash) is deduplicated. Total injected context is capped at ~12 KB. Instructions are injected into the system prompt at startup.

## See also

- [Agent skills](/getting-started/agent-skills) - installing pre-built skill sets
- [Agent loop mode](/modes/agent-loop-mode) - starting the loop
- [REPL slash commands](/modes/agent-loop-commands) - invoking skills and templates
