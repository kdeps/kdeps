# Subagents

A subagent is a fresh agent loop. It starts without the conversation and sees only its task. Knowledge-graph extraction runs as a subagent. The model can start one with the `subagent` tool. Both use the subagent channel's model. The turn, compaction, and branch summaries stay on the session model.

```text
turn -> subagent {task, instructions, tools}
          |
          v
        fresh loop: system prompt = instructions, user = task,
        tools = read-only only, no subagent tool
          |
          v
        report -> tool result of the turn
```

Each runs beside the turn where it can:

| Subagent | When it runs |
|---|---|
| Knowledge-graph extraction | in the background. Recall never waits for it |
| `subagent` calls a reply opens with | in parallel, in one round. Each call gets its own report |

Parallel work runs on the same scheduler and cap as workflow resources: the [`parallel` event](/workflow/execution-flow#independent-resources) (4 at the same time).

## The subagent tool

The model calls `subagent` when a task needs a second opinion with no shared assumptions, such as reviewing a design or a change.

```json
{
  "task": "Review docs/designs/webhooks.md. List missing failure cases.",
  "instructions": "You are a strict reviewer. Lead with what is wrong.",
  "tools": "read_file, list_files, search_local"
}
```

- `task` is the whole job: the subagent knows nothing else, so it names files by path.
- `instructions` becomes the subagent's system prompt, for example a reviewer persona.
- `tools` lists read-only tools (tier `T0` in `policy/permission.yaml`). The default is every read-only tool of the session. kdeps refuses a write tool or `bash_exec` before the subagent starts.
- A subagent cannot start another subagent. The `subagent-max-rounds` event (20) sets the maximum number of tool rounds.

## The subagent channel

```text
/subagent                         # session model (the default)
/subagent current                 # back to the session model
/subagent free                    # a large OpenRouter free model, chosen per call
/subagent google/gemma-4-31b-it:free   # pin one model, every subagent call
/subagent gpt-4o                        # any model, not only OpenRouter
```

`free` picks from the larger half of the OpenRouter free catalog (`models/openrouter`), one id per call. Any other id is a pin. A catalog id uses that provider. A slash id that is not in the catalog goes to OpenRouter. Any other name uses the session backend. The choice persists in `~/.kdeps/agent-loop-settings.yaml` and has priority over the configuration file after you set it.

```yaml
# ~/.kdeps/config.yaml
llm:
  subagent: current    # current | free | a model id
```

