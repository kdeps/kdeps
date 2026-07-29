# Two modes

**Default mental model:** coding agent first; workflow when you need a fixed pipeline.

| | **Coding agent** | **Workflow mode** |
|---|---|---|
| Command | `kdeps` / `kdeps [path]` | `kdeps run workflow.yaml` |
| Shape | LLM loop + tools | Deterministic DAG |
| Entry | Your prompt | `metadata.targetActionId` |
| Best for | Coding, ops, multi-step goals | APIs, bots, file jobs |
| Resources | Built-in tools + workflows-as-tools | Steps via `requires:` |
| Ship | Optional | `kdeps bundle package` / `build` |

```text
agent:     prompt  -> plan tasks -> call tools (incl. workflows) -> answer
workflow:  request -> resolve graph -> run resources -> response
```

## How they connect

- Agent alone: no YAML required.
- `kdeps ./path` — workflows under path become tools; a tool call runs the **full** DAG.
- `kdeps run … --interactive` — serve the workflow and keep an agent REPL open.

Backends (`~/.kdeps/config.yaml`, env keys, `KDEPS_LLM_BASE_URL`) apply to both.

[Coding agent](/agent) · [Workflow mode](/workflow) · [CLI](/cli).
