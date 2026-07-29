# Validate, doctor, debug

## kdeps validate

Catches schema typos, bad `requires:`, expression issues, and bad `targetActionId` before runtime.

```bash
kdeps validate workflow.yaml
kdeps validate ./my-agent/
kdeps validate ./my-agency/
kdeps validate myagent-1.0.0.kdeps
```

Exit `0` = ok (warnings possible). Exit `1` = errors.

CI:

```yaml
- run: kdeps validate .
```

## kdeps doctor

Environment health: binary, Docker, Python, ports, backend hints.

```bash
kdeps doctor
# some builds: kdeps doctor --fix
```

## Run-time debugging

```bash
kdeps run workflow.yaml --dev          # reload on change
kdeps run workflow.yaml --debug
kdeps run workflow.yaml --verbose
kdeps run workflow.yaml --events       # NDJSON lifecycle on stderr
kdeps run workflow.yaml --instrument   # call-chain tracing
```

Agent loop: `/model tool`, `/context`, tool status lines, stall timeout (`/model tool set stall-timeout …`).

## Common failures

| Symptom | Check |
|---------|--------|
| `get('x')` null | Is `x` produced? Is it in `requires:`? |
| Resource never runs | Path to `targetActionId`? `validations` / `skip`? |
| Cycle error | Circular `requires:` |
| API 401 | `KDEPS_API_AUTH_TOKEN` / Bearer header |
| Port in use | `kdeps doctor`, `--port` |
| Expression error | `kdeps validate`, types in `check:` |
| Bot silent | `bot_connections`, `sources: [bot]`, `botReply:` |

## Management API

When enabled, `/_kdeps/*` uses a **management** token (not the API token). Treat as admin. See [Security](/security).

[CLI](/cli) · [Errors](/errors) · [Workflow](/workflow).
