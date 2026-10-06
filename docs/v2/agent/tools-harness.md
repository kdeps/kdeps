# Harness reminders

`/harness reminders <name> on` forces one section of model guidance onto every prompt and every tool result. Permission modes stay on [Built-in tools](/agent/tools).

## Every prompt and every tool result

Every harness section (the pieces of text kdeps sends the LLM to shape its behavior - see [Custom harness](/agent/repl#custom-harness)) has its own normal home: a preamble section goes into the once-per-session system prompt in order, a standalone entry fires only at its one specific call site (a nudge, a praise line, the m365 sandbox warning, the compaction/goal/judge/refine system prompts). `/harness reminders <name> on` overrides that: it forces the named section's raw text onto every LLM prompt (from the very first turn, not just after turn one like the tools reminder) and onto every tool call result, regardless of whether that section's own trigger condition would ever fire on its own.

```
/harness reminders m365-sandbox on
```

This is a blunt, session-wide instrument for a specific situation: a model that keeps needing a particular piece of guidance repeated more often than its normal placement provides - a stubborn sandbox-hallucination habit, a persistent unwanted pattern, or house-style text you want reinforced constantly rather than once. Turning it on for several sections at once compounds their token cost every single turn and every single tool result, so treat it as a short-lived escalation, not a permanent setting.

## List, off, and the status line

`/harness reminders list` shows every section with its current on/off state; `/harness reminders <name> off` turns one off again. The setting persists to `~/.kdeps/harness/<name>.yaml` (the same override file `/harness enable|disable` uses - the two flags are independent of each other and of `disabled`: a reminder can be forced on for a section that is itself disabled everywhere else, and vice versa) and is included in [konfig](/agent/konfig) export/import. While any reminder is active, a `reminder:<names>` segment appears in the REPL's status line next to `turo:`, comma-joining every active name; it disappears entirely the moment the last one is turned off.

## Reminders for /instruct topics

The same switch works for the [`/instruct`](/agent/commands) topics (`overview`, `modes`, `tools`, `available`, `memory`, `goals`, `feedback`, `files`). Turning one on forces that topic's briefing text onto every LLM prompt and every tool call result - handy for a model that keeps forgetting how to call a tool (`instruct:tools`) or what tools exist (`instruct:available`, rendered live from the session's tool catalog each time).

```
/harness reminders instruct:tools on
/harness reminders goals on        # bare name works when no harness section shares it
/harness reminders instruct:tools off
```

`tools` and `memory` are also harness section names, and a bare name always means the harness section when both exist - write `instruct:tools` / `instruct:memory` for the topic. `/harness reminders list` shows every topic as `instruct:<topic>`; the status line shows active ones as `instruct:<topic>` inside the `reminder:` segment. Instruct-topic reminders persist in `~/.kdeps/agent-loop-settings.yaml` (with the other tool settings, included in [konfig](/agent/konfig) export/import), not in `~/.kdeps/harness/`.
