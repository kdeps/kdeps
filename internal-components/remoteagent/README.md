# remoteagent

> Call a remote kdeps agent via HTTP POST.

The remoteagent component sends a query to another running kdeps agent's HTTP API and returns the response. Use it to chain agents together, delegate subtasks to specialised agents, or integrate with kdeps agents deployed on other hosts.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `url` | string | yes | — | Base URL of the remote kdeps agent (e.g. `http://host:3000`) |
| `query` | string | yes | — | Query or request body to send to the agent |

## Outputs

Accessed via `output('<your-action-id>')`:

The output is the raw parsed JSON response body from the remote agent. The exact shape depends on the remote agent's `apiResponse` configuration.

Typical kdeps agent response envelope:

```json
{
  "success": true,
  "data": { ... },
  "meta": { "requestID": "...", "timestamp": "...", "path": "...", "method": "POST" }
}
```

Access nested fields with dot notation:

```yaml
"{{ output('callAgent').data.answer }}"
```

## Usage

Call a remote summariser agent:

```yaml
run:
  component:
    name: remoteagent
    with:
      url: "http://summariser-agent:4000"
      query: "{{ input('document') }}"
  apiResponse:
    success: true
    response:
      summary: "{{ output('callSummariser').data.summary }}"
```

Fan out to multiple agents:

```yaml
- actionId: callClassifier
  run:
    component:
      name: remoteagent
      with:
        url: "http://classifier:5001"
        query: "{{ input('text') }}"

- actionId: callTranslator
  run:
    component:
      name: remoteagent
      with:
        url: "http://translator:5002"
        query: "{{ input('text') }}"
```

Using environment variable for the URL:

```yaml
run:
  component:
    name: remoteagent
    with:
      url: "{{ get('env.AGENT_URL') }}"
      query: "{{ input('q') }}"
```

## Requirements

- The remote kdeps agent must be running and reachable at the specified URL.
- The remote agent must have `apiServerMode: true` and an accessible POST route.

## Notes

- The component sends `Content-Type: application/json` with body `{"query": "<value>"}`.
- If the remote agent expects a different request shape, use `run.httpClient:` directly for full control over headers and body.
- For agents on the same machine, use `http://127.0.0.1:<port>` or the `run.agent:` resource type (in-process, no network required).
- Timeouts are governed by the workflow's default HTTP client timeout.
