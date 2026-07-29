# Chapter 21: Validate, Debug, and Develop

kdeps comes with a validation and diagnostics toolchain that catches problems before they hit runtime. This chapter covers `kdeps validate`, `kdeps doctor`, hot-reload development mode, and common debugging patterns.

## kdeps validate

`kdeps validate` checks your workflow for schema errors, dependency problems, and expression issues without starting the server:

```bash
$ kdeps validate workflow.yaml
$ kdeps validate ./my-agent/
$ kdeps validate myagent-1.0.0.kdeps    # also works on packages
```

### What It Checks

**Schema validation** — every field in `workflow.yaml` and resource files is checked against the schema. Typos in field names, wrong types, missing required fields, and invalid enums are caught.

```
Error: resources/llm.yaml:5: unknown field 'modle' (did you mean 'model'?)
```

**Dependency graph validation** — `requires:` references are checked against existing `actionId` values. Circular dependencies are detected.

```
Error: resources/a.yaml: requires 'b', but resources/b.yaml requires 'a' (cycle detected)
Error: resources/respond.yaml: requires 'missing-resource' (no resource with actionId 'missing-resource')
```

**Expression validation** — expressions in `before:`, `after:`, `check:`, and `skip:` are parsed and type-checked.

```
Error: resources/process.yaml: before[0]: 'get('q') + 5' — cannot add string and integer
Warning: resources/process.yaml: check[1]: expression always true
```

**targetActionId validation** — checks that `targetActionId` references an existing resource with an `apiResponse:` action.

```
Error: workflow.yaml: targetActionId 'final' does not exist (did you mean 'respond'?)
Error: workflow.yaml: targetActionId 'respond' does not have an apiResponse: action
```

### Validating Before Deploying

Add validate to your CI pipeline to catch problems before they reach production:

```yaml
# .github/workflows/ci.yaml
- name: Validate workflow
  run: kdeps validate workflow.yaml

- name: Package
  run: kdeps bundle package workflow.yaml
```

A workflow that fails `validate` will not produce a deployable package.

### Validation Exit Codes

- `0` — validation passed (may include warnings)
- `1` — validation errors found

Check for warnings in output even when exit code is 0:

```bash
$ kdeps validate workflow.yaml && echo "Validation passed"
```

## kdeps doctor

`kdeps doctor` diagnoses your runtime environment — it checks what is available and what is missing:

```bash
$ kdeps doctor
```

Sample output:

```
kdeps doctor — environment check
---------------------------------
[✓] kdeps binary:      /usr/local/bin/kdeps v2.1.0
[✓] Docker:            available (Docker 24.0.5)
[✓] kubectl:           available (v1.28.0)
[✓] Ollama:            skipped — default backend is file (llamafile, self-serving)
[✓] Python:            Python 3.11.5
[✓] pip:               pip 23.2.1
[✗] pandas:            not installed (run: pip install pandas)
[✓] PostgreSQL driver: available
[✗] Port 16395:        in use by PID 12345 (kdeps already running?)
[✓] Write permissions: /data ✓, /tmp ✓

Warnings: 3 issues found. Run with --fix to resolve where possible.
```

Run `kdeps doctor` when:
- You get unexpected errors and want to check the environment
- You are setting up a new deployment target and want to verify prerequisites
- You have a failing `exec:` resource and want to check if the required CLI tool is present

### Auto-Fix

```bash
$ kdeps doctor --fix
```

For some issues (missing Python packages declared in `agentSettings.pythonPackages`), kdeps can install them automatically. Others require manual intervention.

## Hot-Reload Development Mode

```bash
$ kdeps run workflow.yaml --dev
```

`--dev` watches your workflow files for changes and reloads without restarting the server:
- Changes to `resources/*.yaml` — reload the affected resource(s) and re-validate the DAG
- Changes to `workflow.yaml` — reload server configuration (may restart if port changes)
- Changes to `components/` — reload component definitions

The server stays running. In-flight requests complete before the reload takes effect.

### What Gets Reloaded

Hot reload updates resource definitions and DAG topology. It does not:
- Change the listening port without a server restart
- Re-install Python packages (restart needed for `agentSettings.pythonPackages` changes)
- Re-apply OS package changes

For changes to `agentSettings`, stop and restart:

```bash
$ kdeps run workflow.yaml    # no --dev; changes take effect at next start
```

### Watch Output

```bash
$ kdeps run workflow.yaml --dev
kdeps: started with hot reload
kdeps: watching: workflow.yaml, resources/, components/
kdeps: server on 127.0.0.1:16395

# (you save resources/llm.yaml)
kdeps: change detected: resources/llm.yaml
kdeps: reloading resources...
kdeps: DAG re-validated (5 resources, no cycles)
kdeps: reload complete
```

## Debugging Resource Execution

### Debug Mode

```bash
$ kdeps run workflow.yaml --debug
```

Debug mode logs expression evaluation details — each expression in `before:`, `after:`, and `validations` is logged with its evaluated value. Use this when an expression produces unexpected results and you cannot tell which value is wrong:

```
DEBUG [req-abc123] resource: validate, before[0]: lower(trim(get('q'))) → "what is entropy?"
DEBUG [req-abc123] resource: validate, check[0]: get('q') != '' → true (value: "what is entropy?")
DEBUG [req-abc123] resource: validate, check[1]: len(get('q')) < 500 → true (value: 18)
```

`--debug` is quieter than `--instrument` — it focuses on expression values rather than the full execution flow. Use it for expression debugging; use `--instrument` for DAG execution debugging.

`LOG_LEVEL=debug` (environment variable) does the same as `--debug`, and works when running inside Docker where you cannot change the CLI flags directly.

### Instrument Mode

```bash
$ kdeps run workflow.yaml --instrument
```

Instrument mode logs the execution of every resource: which ones are evaluated, which ones are skipped, which ones execute, their inputs, and their outputs.

```
TRACE [req-abc123] route matched: POST /api/v1/chat
TRACE [req-abc123] evaluating resource: validate
TRACE [req-abc123]   methods check: POST in [POST] → pass
TRACE [req-abc123]   routes check: /api/v1/chat in [/api/v1/chat] → pass
TRACE [req-abc123]   check[0]: get('q') != '' → true
TRACE [req-abc123]   executing action: exec ("echo validated")
TRACE [req-abc123]   output: "validated"
TRACE [req-abc123] evaluating resource: llm
TRACE [req-abc123]   waiting for: validate ✓
TRACE [req-abc123]   executing action: chat (model: llama3.2:1b)
TRACE [req-abc123]   prompt: "What is entropy?"
TRACE [req-abc123]   response: "Entropy is a measure of..."
TRACE [req-abc123] evaluating resource: respond
TRACE [req-abc123]   waiting for: llm ✓
TRACE [req-abc123]   apiResponse built
```

This is the fastest way to understand what is happening in a complex DAG.

### Request IDs

Every request gets a unique ID accessible via `info('ID')` and `request.ID`. Log it in your resources:

```yaml
after:
  - set('logged', json({
      "id": info('ID'),
      "action": "llm_call",
      "duration_ms": info('timestamp'),
      "model": "llama3.2:1b"
    }))
```

If a request fails in production, the request ID lets you trace it through all resource executions in the logs.

## FAQ and Common Problems

### "Resource not found" after adding a new resource file

kdeps discovers resources by scanning `resources/` at startup (or on reload in `--dev` mode). If you add a file and the server is not in `--dev` mode, restart it.

### Validation passes but the workflow hangs

Usually a `requires:` cycle or a resource waiting for another that is skipped. Run with `--instrument` to see which resource is waiting and for what.

### LLM response is empty

Check timeout settings. The default is 60s. A larger model or a long context might need 2-5 minutes. Set `timeout: 300s` on the `chat:` resource.

### SQL query returns empty when you expect data

Check the `connectionName:` matches a key in `sqlConnections:`. Check that the `DATABASE_URL` environment variable is set and pointing to the right database.

### Expression errors at runtime

Run `kdeps validate` first. If the expression is syntactically valid but produces wrong types at runtime, enable `--instrument` and look at the data store values at the point of failure.

### "Port already in use"

Another kdeps process is running on the same port. Check with `lsof -i :16395`. Kill the existing process or change the port in `workflow.yaml`.

### Python resource returns no output

The Python script must print exactly one JSON value to stdout. Any other stdout output (print statements for debugging, package import messages) will be parsed as the output and likely fail JSON parsing. Redirect debug output to stderr: `import sys; print("debug", file=sys.stderr)`.

### Component not found

Components must be installed with `kdeps registry install <name>` before use. Run `kdeps registry list` to see what is installed. For custom components, ensure the `components/` directory is in the workflow root.

## Logging and Observability

kdeps writes structured JSON logs to stdout by default:

```json
{"level":"info","time":"2024-01-15T10:23:45Z","request_id":"abc123","resource":"llm","action":"chat","model":"llama3.2:1b","duration_ms":1234}
```

Set the log level:

```bash
$ LOG_LEVEL=debug kdeps run workflow.yaml
```

Levels: `debug`, `info`, `warn`, `error`. `debug` includes expression evaluation details. `info` (default) includes resource execution summaries.

For integration with log aggregation systems (Datadog, Loki, CloudWatch), pipe stdout to your collector:

```bash
$ kdeps run workflow.yaml | tee -a /var/log/myagent.log | your-log-collector
```

Or in Docker/Kubernetes, simply redirect stdout to your logging driver — all kdeps output goes to stdout by design.

## The Management API

Every running kdeps server exposes a built-in management API at `/_kdeps/`. Use it to inspect or update a running workflow without rebuilding or redeploying the container.

Management routes use a separate token from workflow API auth (`KDEPS_API_AUTH_TOKEN`). The API auth middleware does not apply to `/_kdeps/*` — each management handler validates `KDEPS_MANAGEMENT_TOKEN` directly.

### Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/_kdeps/status` | required | Workflow name, version, description, resource count |
| `GET` | `/_kdeps/openapi` | required | OpenAPI spec for configured routes |
| `GET` | `/_kdeps/schema` | required | JSON Schema for the workflow |
| `PUT` | `/_kdeps/workflow` | required | Replace the running workflow with new YAML; hot-reload |
| `PUT` | `/_kdeps/package` | required | Extract a `.kdeps` archive and hot-reload |
| `POST` | `/_kdeps/reload` | required | Reload the workflow from the current on-disk file |

### Authentication

All management endpoints require a bearer token. Set it before starting kdeps:

```bash
$ export KDEPS_MANAGEMENT_TOKEN=my-mgmt-secret
$ kdeps run workflow.yaml
```

Clients send the token in every request:

```
Authorization: Bearer my-mgmt-secret
```

| Token state | Response |
|---|---|
| `KDEPS_MANAGEMENT_TOKEN` unset | `503 Service Unavailable` |
| Token wrong or header missing | `401 Unauthorized` |
| Token correct | Handler runs |

### Checking Status

```bash
$ curl -H "Authorization: Bearer $KDEPS_MANAGEMENT_TOKEN" \
    http://localhost:16395/_kdeps/status
{
  "status": "ok",
  "workflow": {
    "name": "my-agent",
    "version": "1.0.0",
    "description": "My AI agent",
    "targetActionId": "respond",
    "resources": 5
  }
}
```

### Hot-Updating a Running Container

Push a new workflow to a running container without stopping it:

```bash
# Update with a raw workflow YAML file
$ curl -X PUT http://prod-server:16395/_kdeps/workflow \
  -H "Authorization: Bearer my-mgmt-secret" \
  -H "Content-Type: application/yaml" \
  --data-binary @resources/llm.yaml

# Update with a full .kdeps package
$ curl -X PUT http://prod-server:16395/_kdeps/package \
  -H "Authorization: Bearer my-mgmt-secret" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @myagent-1.1.0.kdeps

# Force reload from disk (useful after a volume mount update)
$ curl -X POST http://prod-server:16395/_kdeps/reload \
  -H "Authorization: Bearer my-mgmt-secret"
```

The server hot-reloads the workflow. In-flight requests complete before the reload takes effect. New requests use the updated workflow.

### Size Limits

| Endpoint | Limit | Over-limit response |
|---|---|---|
| `PUT /_kdeps/workflow` | 5 MB | `413 Payload Too Large` |
| `PUT /_kdeps/package` | 200 MB | `413 Payload Too Large` |

### Security Notes

- Path-traversal entries in `.kdeps` archives are rejected with `422 Unprocessable Entity`
- Per-file decompression is capped at 500 MB to guard against zip-bomb payloads
- All management endpoints return `503` when `KDEPS_MANAGEMENT_TOKEN` is not set — set a strong random token in every production deployment
- Use a different token for `KDEPS_MANAGEMENT_TOKEN` and `KDEPS_API_AUTH_TOKEN`; management can hot-reload the workflow

### Restart Persistence

When a workflow is pushed via `PUT /_kdeps/workflow`, kdeps writes the new YAML to the path given at startup (or `/app/workflow.yaml` inside Docker). On the next container restart, the server reads the updated file automatically. Package pushes replace `resources/`, `data/`, and `scripts/` in-place.

This is the deployment path for lightweight updates — changing a prompt, adjusting a validation, tweaking a model — without the overhead of rebuilding and restarting a Docker container. For structural changes (new resources, changed dependencies), a full container restart is safer.

X> ## Exercise
X>
X> Diagnose a deliberately broken workflow using the toolchain from this chapter.
X>
X> Start with this broken `workflow.yaml` snippet (introduce these exact errors into a copy of your chatbot project):
X>
X> 1. Rename the `model` field in `resources/llm.yaml` to `modle`.
X> 2. Change `requires: [validate]` in `resources/llm.yaml` to `requires: [doesNotExist]`.
X> 3. Add `targetActionId: missingResource` to `workflow.yaml`.
X>
X> Then work through the toolchain:
X> - Run `kdeps validate workflow.yaml`. Record all three errors it reports. Fix them one at a time and rerun after each fix.
X> - Run `kdeps doctor`. Note which checks pass and which fail on your machine.
X> - After fixing all errors, start the server with `kdeps run workflow.yaml --dev`. Edit `resources/llm.yaml` while the server is running and confirm hot-reload triggers without restarting.
X> - Send a request and run it again with `--instrument`. Find the line in the trace output that shows `resource: llm` executing and confirm the request ID matches.
X> - Finally, use the management API to push a workflow update without restarting:
X>   ```bash
X>   curl -X PUT http://localhost:16395/_kdeps/workflow \
X>     -H "Authorization: Bearer $KDEPS_MANAGEMENT_TOKEN" \
X>     -H "Content-Type: application/yaml" \
X>     --data-binary @workflow.yaml
X>   ```
X>
X> **Stretch goal:** Introduce a type error in a `before:` expression (`get('q') + 5`) and use `--debug` (not `--instrument`) to find exactly which expression fails and what value it holds at the point of failure.
