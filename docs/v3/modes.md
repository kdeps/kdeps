# Two modes

kdeps has one product surface and two ways to run it.

| | **Workflow mode** | **Agent mode** |
|---|---|---|
| Command | `kdeps run workflow.yaml` | `kdeps` or `kdeps [path]` |
| Shape | Deterministic DAG | LLM loop with tools |
| Entry | `metadata.targetActionId` | Your prompt |
| Best for | APIs, bots, fixed pipelines | Interactive work, multi-step tasks |
| Resources | Steps in order via `requires:` | Whole workflows registered as tools |
| Ship | `kdeps bundle package` / `build` | same packages; path also loads as tools |

```text
workflow:  request -> resolve graph -> run resources -> response
agent:     prompt  -> plan tasks    -> call tools     -> answer
```

## When to use which

- **Workflow** — you want the same path every time: validate input, call a model, hit SQL, return JSON.
- **Agent** — you want the model to decide next steps, use tools, and finish a goal across turns.

They share the same YAML, resources, and backends. An agent that loads `./my-agent/` can call that workflow as one tool; the engine still runs the full DAG inside.

## Backend config is shared

Local or cloud models apply to both modes. Set them once in `~/.kdeps/config.yaml` or env vars (`KDEPS_DEFAULT_BACKEND`, `KDEPS_LLM_BASE_URL`, provider API keys).

Details: [Workflow mode](/workflow) · [Agent mode](/agent).
