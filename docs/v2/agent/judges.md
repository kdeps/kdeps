# Judge panel

A judge panel reviews a workflow chat answer before the resource finishes. Name the panel under `guardrails.judges`, or name a preset with `judgePreset`. `kdeps test` uses the same judges.

```text
chat -> judges -> all approve? -> output
           | any reject
           v
     chat again with the feedback -> judges again
```

Each judge runs its own tool loop. It finishes with one `judge_verdict` call. Prose alone is not a verdict.

The judges review the same answer at the same time. A rejection sends the feedback back. The chat writes a new answer. The panel reviews it again. The `judge-iterations` event sets the budget. The default is 2 rounds. When the budget is empty, kdeps returns the last answer.

A judge that errors, times out, or never calls `judge_verdict` counts as an approval. Judges do not fail the resource.

`guardrails.judges: auto` asks the model for up to three judges for that answer. An explicit list replaces auto.

A grader runs in process before the model. The kinds are `exact`, `contains`, `not_contains`, `regex`, `json`, and `expr`. A failed grader rejects that judge. A judge with graders and no criteria does not call the model.

The YAML is on [Chat judges](/workflow/resources/llm#judges).

## Preset panels

A preset can store a panel. A workflow chat names it with `judgePreset`. `/harness preset` applies events and harness text. It does not attach the panel to the REPL.

`design-review` stores four reviewers: `scope`, `clarity`, `failure-modes`, and `readiness`. Each may call `read_file`, `list_files`, and `search_local`.

```yaml
name: api-review
description: Two reviewers for an API change.
judges:
  - name: compatibility
    criteria: No existing route, field, or status code changes meaning.
    allowedTools: [read_file, search_local]
    maxRounds: 10
  - name: errors
    criteria: Every new failure returns a documented status and message.
    graders:
      - kind: not_contains
        pattern: TODO
```

## See also

- [Chat judges](/workflow/resources/llm#judges)
- [Workflow tests](/workflow/testing)
