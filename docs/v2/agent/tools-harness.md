# Harness reminders

`remind: true` on a harness section puts its text onto every prompt and every tool result. Permission modes stay on [Built-in tools](/agent/tools).

## Every prompt and every tool result

Each harness section has a normal place. A preamble section goes into the system prompt. A standalone entry operates only at its call site. Examples are a nudge, a praise line, and the compaction, judge, and refine prompts. `remind: true` overrides this.

The raw text of the section then goes onto each prompt, from the first turn. It also goes onto each tool result, also when its trigger does not occur.

```yaml
# ~/.kdeps/harness/tools-reminder.yaml (or harness/ in the workflow project)
name: tools-reminder
kind: standalone
remind: true        # add this text to every prompt and every tool result
body: >-
  Every capability you have is a kdeps tool. Call it.
```

Use it for a model that needs guidance more often than its normal place gives: a sandbox habit, a recurring unwanted pattern, a house style. Each reminder adds its tokens to every turn and every tool result, so keep the text short.

- Reminders are independent of `disabled:`. A disabled section can still be a reminder.
- An override file replaces the shipped entry of the same name, so copy the shipped `body:` when you only want to add `remind: true`.
