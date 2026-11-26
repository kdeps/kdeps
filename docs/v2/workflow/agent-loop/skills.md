# Skills and instructions

Skills and instructions are markdown files that tell the agent how to work. kdeps adds them to the system prompt.

A skill is one markdown file with a name and a description:

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
---

Always check for error handling. Prefer early returns over nested conditions.
```

## Read next

- [Skills and instructions reference](/reference/agent-loop/skills) - every key, option and detail
- [Agent loop](/workflow/agent-loop/) - overview
