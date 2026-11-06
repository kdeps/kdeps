# Subagents

A subagent is a fresh agent loop: it starts without the conversation and sees only the task it was given. Judges, knowledge-graph extraction and prompt refinement run as subagents, and the model can start one itself with the `subagent` tool. All of them use the subagent channel's model. The turn itself, compaction, the handshake, and branch summaries stay on the session model.

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
| Judge-roster generation | starts with the turn, collected after the answer |
| Judges | in parallel, after the answer |
| Knowledge-graph extraction | in the background; recall never waits for it |
| `subagent` calls a reply opens with | in parallel, in one round; each call gets its own report |
| Prompt refinement | before the turn, beside roster generation: the turn runs on its rewrite |

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
- `tools` lists read-only tools (tier `T0` in `policy/permission.yaml`). The default is every read-only tool of the session. A write tool or `bash_exec` is refused before the subagent starts.
- A subagent cannot start another subagent. Its tool rounds are capped by the `subagent-max-rounds` event (20).

## The subagent channel

```text
/subagent                         # session model (the default)
/subagent current                 # back to the session model
/subagent free                    # a large OpenRouter free model, chosen per call
/subagent google/gemma-4-31b-it:free   # pin one model, every subagent call
/subagent gpt-4o                        # any model, not only OpenRouter
```

`free` picks from the larger half of the OpenRouter free catalog (`models/openrouter`), one id per call. Any other id is a pin. A catalog id uses that provider. A slash id that is not in the catalog goes to OpenRouter. Any other name uses the session backend. The choice persists in `~/.kdeps/agent-loop-settings.yaml` and wins over the config file once you have set it.

```yaml
# ~/.kdeps/config.yaml
llm:
  subagent: current    # current | free | a model id
```

`kdeps knowledge build --extract` uses the same order: `--model`, then `subagent:` under `llm:` when it names a model, then `model:` under `knowledge:` `extract:`, then the current model. `/knowledge build` chooses when that pass runs: `manual` (when you ask), `auto` (also at session end and before a background build), or `off`. In an agent session the automatic pass does not hold the prompt. The link graph still rebuilds without a model.

```bash
kdeps knowledge build --extract
kdeps knowledge build --extract --model google/gemma-4-31b-it:free
```

An unserved local session model still skips extraction, refinement, and roster generation. Pointing the channel at another provider runs those calls without that local server.
