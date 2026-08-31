# Wrap a shell command in an API

*Applies to workflow mode.*

## Overview

In this tutorial you build an API endpoint that runs a shell command with the
`exec:` resource and returns its output as JSON. This is the pattern for
exposing a script, a CLI tool, or a system check as an HTTP service.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Basic shell commands

By the end you will be able to:

- Run a command with `exec:` and a timeout
- Scope a resource to a method and route
- Read request metadata with `info()`

## Background

The `exec:` resource runs a command and stores its stdout as the resource's
output. It is the escape hatch for anything kdeps has no native resource for.
The command runs with the privileges of the kdeps process, so only expose
commands you trust.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

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
exec:
  command: "echo 'System Info:' && uname -a && echo 'Date:' && date"
  timeout: 10s                # kill the command and fail after 10s
```

The resource's output is the command's stdout.

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
    system_info: "{{ get('systemInfo') }}"       # command stdout
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
    "system_info": "System Info:\nDarwin host 24.6.0 ...\nDate:\nMon Sep  1 12:00:00 UTC 2026\n",
    "timestamp": "2026-09-01T12:00:00Z",
    "workflow": "sysinfo-api"
  }
}
```

## Summary

You built an API that:

- Runs a shell command with `exec:` and a 10-second timeout
- Restricts the endpoint to `GET` with `validations.methods`
- Returns the command output plus request metadata from `info()`

## Next steps

- [Exec resource](/resources/scripting/exec) - stdin, environment, exit-code access
- [Shell execution](/modes/agent-loop-shell) - the agent-mode `bash_exec` tool
- [Error handling (onError)](/concepts/error-handling) - retry and fallback on a failed command
- [CORS configuration](/configuration/cors) - the `cors:` block

::: warning
The command runs with the kdeps process's privileges. Never interpolate
untrusted request input directly into `command:`.
:::
