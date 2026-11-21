# Subagents

A subagent is a fresh agent loop. It starts without the conversation and sees only its task. The model starts one with the `subagent` tool. Judges and knowledge extraction use the same channel. All of them use the subagent channel's model. The turn, compaction, and branch summaries stay on the chat's model.

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
| `subagent` calls a reply opens with | in parallel, in one round; each call gets its own report |

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
- `tools` lists read-only tools (tier `T0` in `policy/permission.yaml`). The default is every read-only tool of the chat. kdeps refuses a write tool or `bash_exec` before the subagent starts.
- A subagent cannot start another subagent. The `subagent-max-rounds` event (20) sets the maximum number of tool rounds.

## The subagent channel

`subagent:` under `llm:` sets the channel's model:

```yaml
# ~/.kdeps/config.yaml
llm:
  subagent: current    # current | free | a model id
```

- `current` is the chat's model (the default).
- `free` picks from the larger half of the OpenRouter free catalog (`models/openrouter`), one id per call.
- Any other id is a pin. A catalog id uses that provider. A slash id that is not in the catalog goes to OpenRouter. Any other name uses the chat's backend.

`kdeps knowledge build --extract` uses the same order: `--model`, then `subagent:` under `llm:` when it names a model, then `model:` under `knowledge:` `extract:`, then the current model. The link graph still rebuilds without a model.

```bash
kdeps knowledge build --extract
kdeps knowledge build --extract --model google/gemma-4-31b-it:free
```

An unserved local chat model still skips extraction and roster generation. Pointing the channel at another provider runs those calls without that local server.
