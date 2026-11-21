# Answer quality

A workflow `chat` turns on kdeps's quality features with keys in its `harness` and `guardrails` groups. They help a small local model answer better, with no cloud model. Each one is off until the chat sets it.

```text
prompt -> model call -> tool loop (promptMode) -> judges (review, revise) -> output
```

```yaml
# resources/answer.yaml
chat:
  model:
    name: llama3.2:1b
  context:
    prompt: "{{ get('q') }}"
  harness:
    sections: [accuracy, honesty]   # the agent loop's system-prompt sections, in this order
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
| `harness.sections` | Adds the agent loop's system-prompt sections | tokens | [Harness sections](/workflow/resources/llm#harness-sections) |
| `harness.promptMode` | `full` (default), `summary` or `auto` harness text after the first request | saves tokens | [Prompt mode](/workflow/resources/llm#prompt-mode) |
| `guardrails.judges` | Judges review the answer. A rejected answer is revised | 1 call per judge, plus revisions | [Judge panel](/workflow/agent-loop/judges) |

A failed tool call always gets a failure brief: the error and the context to repair it. The `failure-briefs` event caps the briefs.

## The workflow ships its own limits

The limits are [events](/workflow/agent-loop/events): `judge-*`, `chat-max-rounds`, `failure-briefs`. The prompts are harness entries: `judge-system`, `accuracy`, and the other sections. A workflow ships its own copies in folders next to `resources/`, in the same format as `~/.kdeps/`:

```text
my-agent/
  workflow.yaml
  resources/
  events/       # chat-max-rounds.yaml, judge-iterations.yaml, ...
  harness/      # judge-system.yaml, accuracy.yaml, ...
  actions/
  presets/
```

```yaml
# my-agent/events/chat-max-rounds.yaml
name: chat-max-rounds
on:
  rounds: 10       # this workflow allows 10 tool rounds, on every machine
run: force_answer
```

| Layer | Wins over |
|---|---|
| the workflow's `events/`, `harness/`, `actions/`, `presets/` | everything |
| `~/.kdeps/<set>/` on the machine | the installed assets |
| the installed assets | - |

The folders go everywhere the workflow goes:

- `kdeps run`, `kdeps test` and `kdeps exec` load them.
- `kdeps validate` parses them. A file that does not parse fails `validate` and `run`.
- `kdeps bundle package` puts them in the `.kdeps` archive. A prepackaged binary embeds that archive.
- `kdeps build` copies them into the Docker image next to `resources/`. `kdeps export iso` and `kdeps export k8s` use that image.

`kdeps run` also installs the `harness`, `events` and `actions` asset sets. A workflow-only install (the `minimum`, `workflows` or `deploy` profile) has every feature.

## In a chat with `loop:`

A chat with `loop:` runs as the agent loop and reads the same keys.

## See also

- [LLM resource](/workflow/resources/llm) - every chat key
- [Reactive LLM events](/workflow/agent-loop/events) - the limits
- [Test a workflow](/workflow/testing) - judges as evals with `kdeps test`
