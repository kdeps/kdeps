# Harness reminders

`/harness reminders <name> on` forces one section of model guidance onto every prompt and every tool result. Permission modes stay on [Built-in tools](/agent/tools).

## Every prompt and every tool result

Each harness section (see [Custom harness](/agent/repl#custom-harness)) has a normal place. A preamble section goes into the system prompt; a standalone entry fires only at its call site (a nudge, a praise line, the m365 sandbox warning, the compaction/judge/refine prompts). `/harness reminders <name> on` overrides that. The section's raw text goes onto every prompt, from the first turn, and onto every tool result, whether or not its trigger fires.

```
/harness reminders m365-sandbox on
```

Use it for a model that needs guidance repeated more often than its normal placement: a stubborn sandbox habit, a recurring unwanted pattern, house style. Each reminder adds its tokens to every turn and every tool result, so keep it short-lived.

## List, off, and the status line

`/harness reminders list` shows each section's state; `/harness reminders <name> off` turns one off.

- The setting persists in `~/.kdeps/harness/<name>.yaml`, the file `/harness enable|disable` uses, and is in [konfig](/agent/konfig) export/import.
- Reminders are independent of enable/disable: a disabled section can still be forced on as a reminder.
- While any reminder is on, the status line shows `reminder:<names>` next to `turo:`.

## Reminders for /instruct topics

The same switch works for [`/instruct`](/agent/commands) topics (`overview`, `modes`, `tools`, `available`, `memory`, `feedback`, `files`). Turning one on forces that briefing onto every prompt and tool result. Use `instruct:tools` for a model that forgets how to call a tool, or `instruct:available` (rendered live from the tool catalog) for one that forgets what exists.

```
/harness reminders instruct:tools on
/harness reminders feedback on     # bare name works when no harness section shares it
/harness reminders instruct:tools off
```

`tools` and `memory` are also harness section names, and a bare name always means the harness section when both exist - write `instruct:tools` / `instruct:memory` for the topic. `/harness reminders list` shows every topic as `instruct:<topic>`; the status line shows active ones as `instruct:<topic>` inside the `reminder:` segment. Instruct-topic reminders persist in `~/.kdeps/agent-loop-settings.yaml` (with the other tool settings, included in [konfig](/agent/konfig) export/import), not in `~/.kdeps/harness/`.
