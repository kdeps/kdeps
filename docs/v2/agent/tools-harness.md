# Harness reminders

`/harness reminders <name> on` forces one section of model guidance onto every prompt and every tool result. Permission modes stay on [Built-in tools](/agent/tools).

## Every prompt and every tool result

Each harness section (see [Custom harness](/agent/repl#custom-harness)) has a normal place. A preamble section goes into the system prompt. A standalone entry operates only at its call site. Examples are a nudge, a praise line, the m365 sandbox warning, and the compaction, judge, and refine prompts. `/harness reminders <name> on` overrides this.

The raw text of the section then goes onto each prompt, from the first turn. It also goes onto each tool result, also when its trigger does not occur.

```
/harness reminders m365-sandbox on
```

Use it for a model that needs guidance repeated more often than its normal placement: a stubborn sandbox habit, a recurring unwanted pattern, house style. Each reminder adds its tokens to every turn and every tool result, so keep it short-lived.

## List, off, and the status line

`/harness reminders list` shows each section's state; `/harness reminders <name> off` turns one off.

- The setting persists in `~/.kdeps/harness/<name>.yaml`, the file `/harness enable|disable` uses, and is in [konfig](/agent/konfig) export/import.
- Reminders are independent of enable and disable. You can turn on a disabled section as a reminder.
- When a reminder is on, the status line shows `reminder:<names>` next to `turo:`.

## Reminders for /instruct topics

The same switch works for [`/instruct`](/agent/commands) topics (`overview`, `modes`, `tools`, `available`, `memory`, `feedback`, `files`). When you turn on a topic, its briefing goes onto each prompt and tool result. If a model forgets how to call a tool, use `instruct:tools`. If a model forgets which tools exist, use `instruct:available` (kdeps makes it live from the tool catalog).

```
/harness reminders instruct:tools on
/harness reminders feedback on     # bare name works when no harness section shares it
/harness reminders instruct:tools off
```

`tools` and `memory` are also harness section names. If both exist, a bare name always means the harness section. For the topic, write `instruct:tools` or `instruct:memory`. `/harness reminders list` shows every topic as `instruct:<topic>`; the status line shows active ones as `instruct:<topic>` inside the `reminder:` segment. Instruct-topic reminders persist in `~/.kdeps/agent-loop-settings.yaml` (with the other tool settings, included in [konfig](/agent/konfig) export/import), not in `~/.kdeps/harness/`.
