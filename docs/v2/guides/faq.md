# Frequently Asked Questions

Common questions about kdeps installation, usage, and architecture.

## Is kdeps free?

Yes. kdeps is open source under the Apache 2.0 license. The CLI, engine, and all resources are free to use.

## What's the difference between workflow mode and agent mode?

[Workflow mode](/modes/workflow-mode) (`kdeps run`) runs resources in a deterministic DAG order defined by [`requires`](/reference/glossary#requires) dependencies. You control exactly what runs and when.

[Agent mode](/modes/agent-mode) (`kdeps serve`) registers whole workflows and components as tools and lets an LLM decide which to invoke in response to user prompts. Workflow tools execute as a complete pipeline so all `requires:` dependencies resolve. Component tools run a single reusable component in isolation. Point at a single file or a folder -- folder mode exposes every workflow and agency found recursively, plus all their components.

Use workflow mode when you know the pipeline upfront. Use agent mode when you want an interactive, conversational interface.

## Do I need to know Go to use kdeps?

No. Workflows are written in YAML. The only code you might write is inline Python scripts or shell commands in `python`/`exec` resources.

## What LLM providers are supported?

Any provider compatible with the OpenAI API format. This includes:
- llamafile (local, the default - models run as self-contained binaries, no server install)
- Ollama (local, opt-in)
- OpenAI
- Anthropic
- Groq
- Any custom endpoint that speaks the OpenAI chat completions protocol

Set the backend, base URL, and model via flags or environment variables.

## Can I run kdeps without an LLM?

Yes. Resources like `httpClient`, `sql`, `python`, `exec`, `email`, `scraper`, `browser`, `file`, and `apiResponse` don't require an LLM. You can build pure data pipelines with no AI at all.

## How is this different from writing a Python script?

kdeps separates orchestration (the DAG, dependencies, error handling) from implementation (the actual LLM calls, HTTP requests, SQL queries). The orchestration layer handles:
- Dependency resolution and execution order
- Concurrent execution of independent resources
- Error propagation and retry
- Session and memory management
- Input validation

You'd have to write all of that yourself in a script. kdeps gives it to you from a YAML file.

## How is this different from LangChain?

LangChain is a Python/JS library. kdeps is a standalone binary configured in YAML.

- LangChain: you write code that calls library functions
- kdeps: you write YAML that the engine executes

kdeps has no code dependency -- install the binary, write a YAML file, and run.

## Can I call one workflow from another?

Yes, via [agencies](/reference/glossary#agency). Use the `agent:` action type to call another agent or component as a sub-agent. The caller passes a prompt, the callee runs autonomously, and the result is returned.

## Can I deploy kdeps as an API server?

Yes. Run `kdeps serve ./my-agent/` for agent mode (registers the workflow as a tool by its `metadata.name`), or use `kdeps run` behind the built-in web server for workflow mode. See [Web Server Mode](/deployment/webserver).

For production, use the [Docker](/deployment/docker) or [Kubernetes](/deployment/kubernetes) deployment options.

## How do I handle secrets and API keys?

Use environment variables. Reference them in your workflow with `get('SECRET_NAME', 'env')`:

<div v-pre>

```yaml
# resources/example.yaml
httpClient:
  url: https://api.example.com
  headers:
    Authorization: "Bearer {{ get('API_KEY', 'env') }}"
```

</div>

Never hardcode secrets in workflow YAML files.

## What's the maximum workflow size?

There's no hard limit. The engine builds an in-memory dependency graph -- workflows with thousands of resources will use more memory but should work. The practical limit is readability of the YAML.

## Can I use kdeps in CI/CD pipelines?

Yes. The `kdeps run` command is designed for one-shot execution. Pipe input via stdin, pass data via environment variables, and capture stdout. See the [Stateless Bot](/examples/stateless-bot/) example.

## Does kdeps support streaming?

Yes. Set `streaming: true` on a `chat:` resource to stream LLM responses token-by-token. This works in both workflow mode and agent mode.

## Where does kdeps store session data?

Session data is stored in a WAL-based embedded database. The location is configurable in `settings.session`. See [Session & Persistence](/configuration/session).

## See Also

- [Quickstart](/getting-started/quickstart) -- build your first workflow
- [Execution Flow](/guides/execution-flow) -- how the engine runs resources
- [Troubleshooting](/guides/troubleshooting) -- common errors and fixes
- [Glossary](/reference/glossary) -- all kdeps terms defined
