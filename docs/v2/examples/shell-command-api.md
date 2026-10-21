# Wrap a shell command in an API

Build an API endpoint that runs a shell command with the `exec` component
and returns its output as JSON. This is the pattern for exposing a script, a
CLI tool, or a system check as an HTTP service.

The `exec` component runs the command through the kdeps API's `exec` tool and
returns a structured result - `stdout`, `stderr`, `exitCode`, and more. It is
the escape hatch for anything kdeps has no native resource for. The command
runs with the privileges of the kdeps process, so only expose commands you
trust.

## Step 1: create the project

```bash
mkdir sysinfo-api
cd sysinfo-api
mkdir resources
```

## Step 2: define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: sysinfo-api
  version: "1.0.0"
  targetActionId: finalResult

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/exec
        methods: [GET, POST]
    cors:
      allowOrigins:
        - "*"
```

## Step 3: run the command

Create `resources/system-info.yaml`:

```yaml
# resources/system-info.yaml
actionId: systemInfo
name: System information
validations:
  methods: [GET]              # only GET reaches this resource
  routes: [/api/v1/exec]
component:                    # the exec component calls the kdeps API's exec tool
  name: exec
  with:
    command: "echo 'System Info:' && uname -a && echo 'Date:' && date"
    timeout: 10s              # kill the command and fail after 10s
```

The resource's output is the command's structured result:
`{success, exitCode, stdout, stderr, command, result, timedOut}`.

## Step 4: format the response

Create `resources/final-result.yaml`:

<div v-pre>

```yaml
# resources/final-result.yaml
actionId: finalResult
name: Final result
requires: [systemInfo]
apiResponse:
  success: true
  response:
    system_info: "{{ get('systemInfo') }}"       # the exec result map
    timestamp: "{{ info('current_time') }}"
    workflow: "{{ info('name') }}"
```

</div>

`info('current_time')` and `info('name')` read request and workflow metadata.

## Step 5: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

```bash
curl http://localhost:16395/api/v1/exec \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
```

Response:

```json
{
  "success": true,
  "data": {
    "system_info": {
      "success": true,
      "exitCode": 0,
      "stdout": "System Info:\nDarwin host 24.6.0 ...\nDate:\nMon Sep  1 12:00:00 UTC 2026\n",
      "stderr": "",
      "command": "echo 'System Info:' && uname -a && echo 'Date:' && date",
      "result": "System Info:\nDarwin host 24.6.0 ...\nDate:\nMon Sep  1 12:00:00 UTC 2026\n",
      "timedOut": false
    },
    "timestamp": "2026-09-01T12:00:00Z",
    "workflow": "sysinfo-api"
  }
}
```

## Next steps

- [exec component](/workflow/resources/exec) - stdin, environment, exit-code access
- [Shell execution](/agent/shell) - the agent mode `bash_exec` tool
- [Error handling (onError)](/workflow/error-handling) - retry and fallback on a failed command
- [CORS configuration](/workflow/cors) - the `cors:` block

::: warning
The command runs with the kdeps process's privileges. Never interpolate
untrusted request input directly into `command:`.
:::
