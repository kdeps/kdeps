# Frequently asked questions

Common questions about kdeps installation, usage, and architecture.

## Is kdeps free?

Yes. kdeps is open source under the Apache 2.0 license. The CLI, engine, and all resources are free to use.

## What runs a workflow?

`kdeps` runs `./workflow.yaml`. Requests come from an HTTP API, a bot, a file, or an interactive prompt. Resources run in the order that [`requires`](/reference/glossary#requires) sets.

An agent is a `chat` with `loop:` in that file. The model calls tools over many rounds inside that one step.

## Do I need to know go to use kdeps?

No. You write workflows in YAML. The only code is optional inline Python or shell in `python`/`exec` resources.

## What LLM providers are supported?

Any provider compatible with the OpenAI API format. This includes:
- PyBot (local, the model of a chat with no `model:` - a rule-based chatbot in plain Python, no weights)
- llamafile (local - models run as self-contained binaries, no server install)
- Ollama (local, opt-in)
- OpenAI
- Anthropic
- Groq
- Any custom endpoint that speaks the OpenAI chat completions protocol

Set the backend, base URL, and model with flags or environment variables.

## Can I run kdeps without an LLM?

Yes. Resources like `agent` and `apiResponse` (and the `http_request`, `python`, `exec`, `sql`, `git`, and `file` components) do not require an LLM. You can build pure data pipelines with no AI at all.

## How is this different from writing a Python script?

kdeps separates orchestration (the DAG, dependencies, error handling) from implementation (the actual LLM calls, HTTP requests, SQL queries). The orchestration layer handles:
- Dependency resolution and execution order
- Concurrent execution of independent resources
- Error propagation and retry
- Session and memory management
- Input validation

Without kdeps, you must write all of that yourself in a script. kdeps gives it to you from a YAML file.

## How is this different from LangChain?

LangChain is a Python/JS library. kdeps is a standalone binary configured in YAML.

- LangChain: you write code that calls library functions
- kdeps: you write YAML that the engine executes

kdeps has no code dependency - install the binary, write a YAML file, and run.

## Can I call one workflow from another?

Yes, with [agencies](/reference/glossary#agency). Use the `agent:` action type to call another agent's full pipeline. The caller passes `params:`. The callee reads them with `get('key')` and returns its `apiResponse.response`.

## Can I deploy kdeps as an API server?

Yes. `kdeps run workflow.yaml` starts the HTTP API. See [Web server](/deploy/webserver).

For production, use the [Docker](/deploy/docker) or [Kubernetes](/deploy/kubernetes) deployment options.

## How do I handle secrets and API keys?

Use environment variables. Reference them in your workflow with `get('SECRET_NAME', 'env')`:

<div v-pre>

```yaml
# resources/example.yaml
component:
  name: http_request
  with:
    url: https://api.example.com
    headers:
      Authorization: "Bearer {{ get('API_KEY', 'env') }}"
```

</div>

Never hardcode secrets in workflow YAML files.

## What is the maximum workflow size?

There is no hard limit. The engine builds an in-memory dependency graph - workflows with thousands of resources use more memory, but they operate correctly. The practical limit is readability of the YAML.

## Can I use kdeps in CI/CD pipelines?

Yes. The `kdeps run` command is for one-shot runs. Send input through stdin, give data with environment variables, and capture stdout. See the [Stateless bot](/examples/stateless-bot/) example.

## Does kdeps support streaming?

Yes. Set `streaming: true` on a `chat:` resource to stream LLM responses token-by-token.

## Where does kdeps store session data?

HTTP session data (`set(..., 'session')`) lives in SQLite or in-memory storage, configured under `settings:` `session:`. See [Session configuration](/workflow/sessions).

The conversation of a chat with `loop:` is a different store, set by `loop.session`. See [Agent loop](/workflow/agent-loop/).

## Does kdeps write into my project folder?

No. Everything kdeps generates lives under `~/.kdeps`, so nothing it writes can end up in a project's git history. A project folder only holds files you write yourself.

| Data | Where |
|------|-------|
| Agent chat history | `~/.kdeps/sessions/<folder>/` |
| Agent memory | `<vault>/projects/<project>/memory/` (markdown notes; default vault `~/.kdeps/notes`) |
| Knowledge graph, caches, report, explorer | `~/.kdeps/knowledge/<vault-id>/` |
| `search_local` index and graph | `~/.kdeps/projects/<folder name>-<hash>/index.db`, `graph.db` |
| HTTP sessions (`session:` under `settings:`) | `~/.kdeps/sessions.db` unless you set `path` |
| Models, config, API keys, connections | `~/.kdeps/models/`, `~/.kdeps/config.yaml` |

kdeps only reads files you write: `KDEPS.md` (or `AGENTS.md`, `CLAUDE.md`, ...), a `.kdeps/` folder of instructions, skills and prompt templates, workflow files, and a component `.env` file. It never writes into a project's `.kdeps/`, so that folder is safe to commit.

```bash
export KDEPS_DATA_DIR=/fast-disk/kdeps-projects   # moves ~/.kdeps/projects (index and graph dbs) elsewhere
```

## See also

- [Quickstart](/workflow/quickstart) - build your first workflow API
- [Execution flow](/workflow/execution-flow) - how the engine runs resources
- [Troubleshooting](/reference/troubleshooting) - common errors and fixes
- [Glossary](/reference/glossary) - all kdeps terms defined
