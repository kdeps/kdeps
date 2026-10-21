# exec component

The `exec` component runs a shell command and returns a structured result: `{success, exitCode, stdout, stderr, command, result, timedOut}`. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. Its call is the kdeps API's `exec` tool, which runs the native exec executor. A workflow that uses `exec:` directly - primary or in a `before:`/`after:` step - is rejected.

It works in both modes: as a DAG step in workflow mode, and inside any workflow the LLM calls as a tool in agent mode.

```d2
direction: right
call: "component: exec" {shape: oval}
api: "POST /tools/exec/call"
shell: "sh -c command\n(cmd /C on Windows)"
direct: "command + args\nexecuted directly"
out: "{success, exitCode,\nstdout, stderr, timedOut}" {shape: oval}
call -> api
api -> shell: "no args"
api -> direct: "args given"
shell -> out
direct -> out
```

## Configuration

<div v-pre>

```yaml
# resources/system-info.yaml
actionId: systemInfo
name: System Information
component:
  name: exec                       # from the components asset set; a components/exec/ of your own wins
  with:
    command: "uname -a && date"    # required; multiline scripts allowed
    timeout: 10s                   # hard stop; the call fails after it, no retry
```

</div>

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `command` | string | yes | - | The shell command to run. Supports multiline scripts |
| `args` | array | no | - | Arguments for `command`. When set, `command` runs directly instead of through a shell |
| `workingDir` | string | no | process working directory | The directory the command runs in |
| `env` | object | no | - | Extra environment variables for this execution; the process environment is inherited |
| `timeout` | string | no | `KDEPS_EXEC_TIMEOUT`, else the kdeps default | Maximum time allowed (e.g. `30s`, `1m`) |

## Shell or direct execution

Without `args`, the command string runs through `sh -c` on Unix and `cmd /C` on Windows, so pipes, `&&`, globs, and redirections work.

With `args`, the command is executed directly with an argv array - no shell - which avoids quoting problems and shell injection:

<div v-pre>

```yaml
# resources/grep-logs.yaml
actionId: grepLogs
component:
  name: exec
  with:
    command: grep                       # executed directly, not through sh
    args: ["-c", "ERROR", "/var/log/app.log"]
```

</div>

## Output

```json
{
  "success": true,
  "exitCode": 0,
  "stdout": "Linux kdeps 6.8.0\nWed Oct 21 09:00:00 UTC 2026\n",
  "stderr": "",
  "command": "uname -a && date",
  "result": "Linux kdeps 6.8.0\nWed Oct 21 09:00:00 UTC 2026\n",
  "timedOut": false
}
```

Read fields with `output('systemInfo').stdout`, `output('systemInfo').exitCode`, and so on.

| Key | Type | Description |
|-----|------|-------------|
| `success` | bool | `true` when the command exited 0 |
| `exitCode` | integer | Process exit code; `-1` on timeout |
| `stdout` | string | Captured standard output |
| `stderr` | string | Captured standard error |
| `command` | string | The full command line that ran, arguments included |
| `result` | string | Same as `stdout`; kept for compatibility with simple consumers |
| `timedOut` | bool | `true` when the timeout killed the command |

A non-zero exit does not return this map: the call fails with `command failed: ...` and `onError:` on the resource decides what happens next.

## Examples

**Multi-line script:**

<div v-pre>

```yaml
# resources/report.yaml
actionId: report
component:
  name: exec
  with:
    command: |
      echo "Starting process..."
      date
      uname -a
      echo "Done!"
    timeout: 30s
```

</div>

**Interpolate request data:**

<div v-pre>

```yaml
# resources/lookup.yaml
actionId: lookup
component:
  name: exec
  with:
    command: "curl -s https://api.example.com/users/{{ get('user_id') }}"
    timeout: 30s
```

</div>

**Structured output - echo JSON, then parse it:**

<div v-pre>

```yaml
# resources/git-info.yaml
actionId: gitInfo
component:
  name: exec
  with:
    command: |
      echo "{\"branch\": \"$(git rev-parse --abbrev-ref HEAD)\", \"commit\": \"$(git rev-parse --short HEAD)\"}"
    workingDir: /app
    timeout: 30s

---
actionId: response
requires: [gitInfo]
apiResponse:
  success: true
  response:
    branch: "{{ fromJSON(output('gitInfo').stdout).branch }}"
    commit: "{{ fromJSON(output('gitInfo').stdout).commit }}"
```

</div>

**Run with extra environment variables:**

<div v-pre>

```yaml
# resources/whoami.yaml
actionId: whoami
component:
  name: exec
  with:
    command: "echo \"$DEPLOY_ENV\""
    env:
      DEPLOY_ENV: "staging"
```

</div>

## Environment variables

The command inherits the kdeps process environment, so `agentSettings.env` values are visible:

```yaml
# In workflow.yaml
settings:
  agentSettings:
    env:
      API_TOKEN: "secret-token"
      DEBUG: "true"
```

```yaml
# In a resource
component:
  name: exec
  with:
    command: |
      echo "Token: $API_TOKEN"
      if [ "$DEBUG" = "true" ]; then
        echo "Debug mode enabled"
      fi
```

`env:` adds or overrides variables for that one execution.

## Errors and timeouts

| Error | Cause |
|-------|-------|
| `exec: command is required` | `command` is missing or whitespace |
| `command failed: ...` | The command exited non-zero; the call fails, `exitCode` and `stderr` stay in the logs |
| `command timed out after 30s` | `timeout` expired; the process was killed |
| `exec stdout exceeds output limit of N bytes` | Output crossed `KDEPS_EXEC_MAX_OUTPUT_BYTES` |

Handle a failing command without stopping the workflow with `onError:`:

<div v-pre>

```yaml
# resources/optional-step.yaml
actionId: optionalStep
component:
  name: exec
  with:
    command: "exit 3"
onError:
  action: continue
```

</div>

## Security notes

When user input reaches a command, prefer `args:` over string interpolation - it never goes through a shell:

<div v-pre>

```yaml
# Dangerous -- command injection possible
command: "echo {{ get('user_input') }}"

# Safer -- no shell, each value is one argument
command: echo
args: ["{{ get('user_input') }}"]
```

</div>

Validate input as well:

```yaml
validations:
  check:
    - get('user_input') matches '^[a-zA-Z0-9_-]+$'
```

The workflow-side `exec` tool has no agent guardrails: unlike `bash_exec` in agent mode, it does not check `KDEPS_ALLOW_BASH` or the bash command blocklist. A workflow YAML is trusted configuration; treat `command:` as code you ship.

## See also

- [Python resource](/workflow/resources/python) - complex data processing
- [HTTP client](/workflow/resources/http-client) - API calls (preferred over curl)
- [Workflow configuration](/workflow/configuration) - OS package settings
