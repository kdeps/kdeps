# Structured Execution Event Stream

The `--events` flag enables a machine-readable NDJSON event stream emitted to **stderr** on every
workflow lifecycle transition. Events let external systems monitor, react to, and audit kdeps
executions without screen-scraping logs.

## Quick Start

```bash
# Run a workflow and capture the event stream
kdeps run my-workflow/ --file input.txt --events 2>events.ndjson

# Stream events live while stdout stays clean
kdeps run my-workflow/ --file input.txt --events 2>&1 1>/dev/null | jq .
```

Each line of `events.ndjson` is a self-contained JSON object:

```json
{"event":"workflow.started","workflowId":"my-workflow","emittedAt":"2025-01-01T00:00:00Z"}
{"event":"resource.started","workflowId":"my-workflow","actionId":"fetch-data","resourceType":"httpClient","emittedAt":"2025-01-01T00:00:00.001Z"}
{"event":"resource.completed","workflowId":"my-workflow","actionId":"fetch-data","resourceType":"httpClient","emittedAt":"2025-01-01T00:00:00.500Z"}
{"event":"workflow.completed","workflowId":"my-workflow","emittedAt":"2025-01-01T00:00:00.501Z"}
```

## Event Taxonomy

### Workflow Events

| Event | When |
|-------|------|
| `workflow.started` | Execution graph has been built; first resource is about to run |
| `workflow.completed` | All resources completed without error |
| `workflow.failed` | At least one resource failed and the workflow was aborted |

### Resource Events

| Event | When |
|-------|------|
| `resource.started` | A resource is about to execute |
| `resource.completed` | A resource finished successfully |
| `resource.failed` | A resource encountered an error |
| `resource.skipped` | A resource was bypassed (dependency condition or skip rule) |
| `resource.retrying` | A resource is being retried after a transient error |

## Event Fields

| Field | Present On | Description |
|-------|-----------|-------------|
| `event` | All | Event name (see taxonomy above) |
| `workflowId` | All | `metadata.name` from `workflow.yaml` |
| `emittedAt` | All | RFC 3339 UTC timestamp |
| `actionId` | Resource events | `metadata.actionId` of the resource |
| `resourceType` | Resource events | Executor type: `exec`, `llm`, `httpClient`, `python`, `sql`, etc. |
| `message` | Failure events | Human-readable error description |
| `failureClass` | Failure events | Structured failure category (see below) |

## Failure Classes

When a resource or workflow fails, the `failureClass` field classifies the root cause:

| Class | Meaning |
|-------|---------|
| `validation` | Input validation or schema check failed |
| `provider` | External LLM or API provider returned an error |
| `timeout` | Operation exceeded its deadline |
| `compile` | Expression or template could not be compiled |
| `preflight` | Pre-execution check failed (missing dependency, bad config) |
| `infra` | Infrastructure error (network, filesystem, database) |
| `tool_runtime` | The executor itself failed at runtime (command not found, process crash) |

Example failure event:

```json
{
  "event": "resource.failed",
  "workflowId": "my-workflow",
  "actionId": "call-api",
  "resourceType": "httpClient",
  "message": "http client: connection refused",
  "failureClass": "infra",
  "emittedAt": "2025-01-01T00:00:01Z"
}
```

## NDJSON Format

Each event is a single line of JSON (newline-delimited). The stream is written to **stderr** so
stdout can carry the workflow's own output (e.g. `apiResponse` data, exec stdout) without
interference.

Rules:
- One JSON object per line, terminated by `\n`
- HTML special characters are **not** escaped (`<`, `>`, `&` appear literally)
- No trailing comma or array wrapper — each line is independently parseable

## Machine Integration Examples

### Filter only failures with jq

```bash
kdeps run . --file data.csv --events 2>&1 1>/dev/null \
  | jq 'select(.event | test("failed"))'
```

### Assert workflow completed in CI

```bash
kdeps run . --file payload.json --events 2>events.ndjson
grep -q '"workflow.completed"' events.ndjson || { echo "workflow did not complete"; exit 1; }
```

### Pipe to a monitoring system

```bash
kdeps run . --file payload.json --events 2> \
  >(while IFS= read -r line; do
      curl -s -X POST https://my-monitoring.example.com/events \
        -H 'Content-Type: application/json' \
        -d "$line"
    done)
```

### Validate NDJSON in Python

```python
import json

with open("events.ndjson") as f:
    events = [json.loads(line) for line in f if line.strip()]

completed = any(e["event"] == "workflow.completed" for e in events)
failures  = [e for e in events if e["event"] == "resource.failed"]
print(f"Completed: {completed}, Failures: {len(failures)}")
```

## Default Behaviour (No --events)

Without `--events`, a `NopEmitter` is used — zero overhead, no output. The flag is opt-in and
has no effect on execution semantics.

## When to Use Events

- **CI pipelines** — assert specific events fired (or did not fire)
- **Autonomous recovery** — drive retry or escalation logic from `resource.failed` + `failureClass`
- **Monitoring dashboards** — ingest the stream into Datadog, Grafana, Prometheus push gateway
- **Debugging** — correlate resource timing by diffing `emittedAt` between `started`/`completed`
- **Federation** — audit cross-agent calls by matching `workflowId` across agency members

## See Also

- [Input Sources](./input-sources.md) — including the `file` input source used with `--file`
- [Error Handling](./error-handling.md) — resource-level retry and skip configuration
- [Agency & Multi-Agent Orchestration](./agency.md) — how events propagate across agents
