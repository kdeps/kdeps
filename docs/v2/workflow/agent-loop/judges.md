# Judge panel

The judge panel is the eval. A workflow chat sets it in `guardrails` (a [`judges:` list or `judges: auto`](/reference/resources/llm#judges), see [Answer quality](/workflow/quality)). It reviews an answer on its own. Roster generation and each judge use the [subagent channel](/workflow/agent-loop/subagent). That is the chat's model unless `subagent:` under `llm:` names another one.

```text
answer -> [judge 1] -> [judge 2] -> [judge 3] -> all approved? -> answer
              |             |             |
              v (tools)     v (tools)     v (tools)
          read files    run tests    search the web
                    any rejected -> revise -> re-judge
```

```yaml
name: api-review
description: Two reviewers for an API change
judges:
  - name: compatibility
    criteria: >-                  # the rubric folded into the judge's system prompt
      No existing route, field or status code changes meaning.
    allowedTools: [read_file, search_local]   # tools it may call to check the answer; empty = judge_verdict only
    maxRounds: 10                 # its tool rounds; 0 = the judge-max-rounds event
  - name: errors
    criteria: Every new failure returns a documented status and message.
    graders:                      # checks without a model call; a failure rejects
      - kind: not_contains
        pattern: TODO
```

## Read next

- [Judge panel reference](/reference/agent-loop/judges) - judge-panel presets
- [Agent loop](/workflow/agent-loop/) - overview
