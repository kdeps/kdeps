# Process data with a Python script

*Applies to workflow mode.*

## Overview

In this tutorial you build an API that runs a Python script to validate and
convert data formats (JSON, YAML). It shows the `python:` resource: how it
receives request data, and how its printed output becomes the resource result.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Basic Python

By the end you will be able to:

- Read request fields in a script with `input()` / `get()`
- Return structured data by printing JSON
- Install a Python package for the script

## Background

The `python:` resource runs a script and captures its stdout as the resource's
output. If the script prints a JSON object, downstream resources can read its
fields. Expression placeholders are substituted into the script text before it
runs, so wrap them in triple quotes to keep the script valid Python.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

## Step 1: create the project

```bash
mkdir data-tools
cd data-tools
mkdir resources
```

## Step 2: define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: data-tools
  version: "1.0.0"
  targetActionId: process

settings:
  apiServer:
    portNum: 16396
    routes:
      - path: /format
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
```

## Step 3: the Python resource

Create `resources/process.yaml`:

<div v-pre>

```yaml
# resources/process.yaml
actionId: process
name: Format operations
validations:
  methods: [POST]
  routes: [/format]
python:
  packages:
    - pyyaml                      # installed into the script's environment
  script: |
    import json, sys

    data      = """{{ input('data') }}"""
    fmt       = "{{ input('format', 'json') }}".lower()
    operation = "{{ input('operation', 'validate') }}".lower()

    def run():
        if operation == "validate":
            if fmt == "json":
                json.loads(data)
            elif fmt == "yaml":
                import yaml; yaml.safe_load(data)
            else:
                return {"error": f"unsupported format: {fmt}"}
            return {"valid": True}

        if operation == "convert":   # json -> yaml
            import yaml
            return {"output": yaml.dump(json.loads(data), default_flow_style=False).rstrip()}

        return {"error": f"unknown operation: {operation}"}

    try:
        result = run()
    except Exception as exc:
        result = {"valid": False, "error": str(exc)}

    print(json.dumps(result))       # stdout becomes the resource output
apiResponse:
  success: true
  response:
    result: "{{ output('process') }}"
```

</div>

## Step 4: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

Validate a JSON string:

```bash
curl -X POST http://localhost:16396/format \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"data": "{\"a\": 1}", "format": "json", "operation": "validate"}'
```

Convert JSON to YAML:

```bash
curl -X POST http://localhost:16396/format \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"data": "{\"a\": 1, \"b\": [2, 3]}", "operation": "convert"}'
```

Response:

```json
{ "success": true, "data": { "result": { "output": "a: 1\nb:\n- 2\n- 3" } } }
```

## Summary

You built an API where a Python script:

- Reads request fields with `input()` (templated into the script)
- Uses a third-party package declared under `packages:`
- Returns structured data by printing a JSON object
- Is exposed through `output('process')`

## Next steps

- [Python resource](/resources/scripting/python) - files, virtual environments, stdin
- [Inline resources tutorial](/examples/inline-resources) - Python in `after:`
- [Function calling tutorial](/examples/function-calling) - a Python script as an LLM tool
- [Exec resource](/resources/scripting/exec) - plain shell commands
