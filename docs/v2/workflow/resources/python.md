# python component

The `python` component runs a Python script in a managed virtualenv and returns its stdout, parsed as JSON when possible and as raw text otherwise. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. Its call is the kdeps API's `python` tool, which runs the native Python executor. A workflow that uses `python:` directly - primary or in a `before:`/`after:` step - is rejected.

It works in both modes: as a DAG step in workflow mode, and inside any workflow the LLM calls as a tool in agent mode.

```d2
direction: right
call: "component: python" {shape: oval}
api: "POST /tools/python/call"
venv: "uv venv\nversion + packages"
run: "python -c script\nor python scriptFile"
out: "stdout parsed as JSON\nelse raw text" {shape: oval}
call -> api
api -> venv
venv -> run
run -> out
```

## Configuration

<div v-pre>

```yaml
# resources/analyze.yaml
actionId: analyze
name: Analyze Data
component:
  name: python                     # from the components asset set
  with:
    script: |                      # inline Python; print JSON for structured output
      import json
      print(json.dumps({"rows": 3}))
    timeout: 120s                  # hard stop; the call fails after it, no retry
```

</div>

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `script` | string | one of `script` / `scriptFile` | - | Inline Python source |
| `scriptFile` | string | one of `script` / `scriptFile` | - | Path to a `.py` file; relative paths resolve against the kdeps process working directory |
| `args` | array | no | - | Arguments passed to the script after the script path or `-c` |
| `env` | object | no | - | Extra environment variables; the process environment is inherited |
| `timeout` | string | no | `KDEPS_PYTHON_TIMEOUT`, else the kdeps default | Hard stop for the script (e.g. `30s`, `2m`) |
| `venvName` | string | no | package-hashed default venv | Dedicated virtualenv name; resources sharing a name share packages |
| `pythonVersion` | string | no | `KDEPS_PYTHON_VERSION`, else `3.12` | Interpreter version for the venv |
| `packages` | array | no | - | Python packages installed into the venv before the script runs |

## Script or script file

`script` runs inline through `python -c`. `scriptFile` runs a file on disk and receives `args` through `sys.argv`:

<div v-pre>

```yaml
# resources/process.yaml
actionId: process
component:
  name: python
  with:
    scriptFile: "./scripts/data_processor.py"
    args: ["--mode", "analyze", "--data", "{{ get('data') }}"]
```

</div>

```python
# scripts/data_processor.py
import argparse
import json

parser = argparse.ArgumentParser()
parser.add_argument("--mode", required=True)
parser.add_argument("--data", required=True)
args = parser.parse_args()

print(json.dumps({"status": "success", "mode": args.mode}))
```

## Packages and virtualenvs

kdeps manages Python with [uv](https://github.com/astral-sh/uv). Packages go in the call, not in workflow settings:

<div v-pre>

```yaml
# resources/report.yaml
actionId: report
component:
  name: python
  with:
    packages: ["pandas>=2.0", "requests"]   # installed into the venv on first run
    script: |
      import pandas as pd
      print(pd.__version__)
```

</div>

The venv is built once and reused. Without `venvName`, the name is derived from the package list, so two resources with the same packages share one venv. Give a `venvName` to isolate a resource from everything else:

<div v-pre>

```yaml
# resources/data-science.yaml
component:
  name: python
  with:
    venvName: "datascience-env"
    packages: ["pandas", "numpy"]

# resources/html-tools.yaml -- separate venv, separate package set
component:
  name: python
  with:
    venvName: "html-env"
    packages: ["requests", "beautifulsoup4"]
```

</div>

## Output

The script's stdout is the resource output. When stdout parses as a JSON object, the resource output is that object; otherwise it is the raw text.

<div v-pre>

```yaml
# resources/summarize.yaml
actionId: summarize
component:
  name: python
  with:
    script: |
      import json
      print(json.dumps({"status": "success", "data": [1, 2, 3]}))
```

</div>

```yaml
# resources/response.yaml
requires: [summarize]
apiResponse:
  success: true
  response:
    full: get('summarize')          # the parsed map
    status: get('summarize').status
    data: get('summarize').data
```

A non-zero exit does not return stdout: the call fails with `python execution failed: ..., stderr: ...` and `onError:` on the resource decides what happens next. stderr and the exit code stay in the logs.

## Environment variables

`env:` sets variables for that one call; values are expressions, evaluated per run. Pass secrets this way rather than in `args:`, which any user on the machine can read in the process list:

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch
component:
  name: python
  with:
    env:
      DB_PASSWORD: "{{ env('DB_PASSWORD') }}"
    script: |
      import os
      print(os.environ["DB_PASSWORD"] != "")
```

</div>

Workflow-wide values from `agentSettings.env` are inherited by the script:

```yaml
# In workflow.yaml
settings:
  agentSettings:
    env:
      API_KEY: "secret-key"
      DEBUG: "true"
```

```python
import os

api_key = os.environ.get('API_KEY')
debug = os.environ.get('DEBUG') == 'true'
```

| Variable | Effect |
|----------|--------|
| `KDEPS_PYTHON_VERSION` | Default `pythonVersion` when the call does not set one (then `3.12`) |
| `KDEPS_PYTHON_TIMEOUT` | Default `timeout` when the call does not set one |
| `KDEPS_PYTHON_MAX_OUTPUT_BYTES` | Cap on captured stdout; exceeding it fails the call |

## Errors and timeouts

| Error | Cause |
|-------|-------|
| `python: script or scriptFile is required` | Neither `script` nor `scriptFile` was given |
| `python execution failed: ..., stderr: ...` | The script exited non-zero; stderr is quoted in the message |
| `python stdout exceeds output limit of N bytes` | Output crossed `KDEPS_PYTHON_MAX_OUTPUT_BYTES` |
| `failed to ensure venv: ...` | uv could not build the virtualenv (bad version, network, package resolution) |

Handle a failing script without stopping the workflow with `onError:`:

<div v-pre>

```yaml
# resources/optional-step.yaml
actionId: optionalStep
component:
  name: python
  with:
    script: |
      raise SystemExit(3)
onError:
  action: continue
```

</div>

## See also

- [Python examples](/reference/python-examples) - data transformation, ML inference, image processing, error handling, debugging
- [exec component](/workflow/resources/exec) - shell command execution
- [Inline resources](/workflow/inline-resources) - `component:` in `before:` / `after:`
- [LLM resource](/workflow/resources/llm) - combine with AI
