# Answer quality

A workflow `chat` turns on kdeps's quality features with keys in its `harness` and `guardrails` groups. They help a small local model answer better, with no cloud model. Each one is off until the chat sets it.

```d2
direction: right
prompt: "prompt" {shape: oval}
refine: "refine\nclearer prompt"
plan: "plan\ncouncil job orders"
turo: "turo\nfewer tokens"
call: "model call"
loop: "tool loop\nefficiency + promptMode"
judges: "judges\nreview, revise"
out: "output" {shape: oval}
prompt -> refine -> plan -> turo -> call -> loop -> judges -> out
```

```yaml
# resources/answer.yaml
chat:
  model:
    name: llama3.2:1b
  context:
    prompt: "{{ get('q') }}"
  harness:
    sections: [accuracy, honesty]   # agent mode's system-prompt sections, in this order
    refine: true                    # rewrite the prompt into a clear request first
    plan: true                      # a council agrees on job orders; the plan goes in the system prompt
    turo: true                      # reduce the prompt with the turo binary; no turo, no change
    efficiency: true                # soft-stop runs of reads, repeated calls and failed model calls
    promptMode: summary             # after the first request, one summary line per section
  tools:
    list:
      - name: read_file
      - name: edit_file
  guardrails:
    judges: auto                    # one call picks up to 3 judges for each request
```

| Key | What it does | Cost | Details |
|---|---|---|---|
| `harness.refine` | Rewrites the prompt with the `refine-system` prompt and the chat history | 1 model call | [Prompt refine](/agent/refine) |
| `harness.plan` | Members draft, revise and rank job orders. The sorted plan goes in the system prompt | several calls: draft, revise, rank | [Planning council](/agent/plan) |
| `harness.turo` | Reduces the system prompt and the prompt to fewer tokens | none (local binary) | [turo](/agent/turo) |
| `harness.efficiency` | Drops reads past the limit and sends the model a hidden note | none | [Efficiency](/agent/efficiency) |
| `harness.promptMode` | `full` (default), `summary` or `auto` harness text after the first request | saves tokens | [Efficiency](/agent/efficiency#prompt-mode) |
| `guardrails.judges` | Judges review the answer. A rejected answer is revised | 1 call per judge, plus revisions | [Judge panel](/agent/judges) |

A failed tool call always gets a failure brief: the error and the context to repair it. The `efficiency-failures` event caps the briefs.

## Limits come from events

The limits are [events](/agent/events): `efficiency-*`, `plan-council-*`, `judge-*`, `chat-max-rounds`. An override in `~/.kdeps/events/` changes them for every chat on that machine. A Docker image or a prepackaged binary carries the build machine's events.

`kdeps run` installs the harness, events and actions asset sets. A workflow-only install (the `minimum`, `workflows` or `deploy` profile) has every feature.

## In a chat with `loop:`

A chat with `loop:` runs as the agent loop and reads the same keys. The agent REPL turns the same features on with slash commands (`/refine`, `/plan`, `/turo`, `/efficiency`, `/judges`).

## See also

- [LLM resource](/workflow/resources/llm) - every chat key
- [Reactive LLM events](/agent/events) - the limits
- [Test a workflow](/workflow/testing) - judges as evals with `kdeps test`
