# Scope environment variables per component

*Applies to workflow mode.*

## Overview

In this tutorial you build two components that both read the same environment
variable name but can be given different values - one per component - without
touching the global variable. You will also see the `.env` file kdeps
scaffolds for each component on first run.

This tutorial is for developers who have completed the
[Reusable component](/examples/custom-component) tutorial. It assumes you know:

- Basic YAML
- How environment variables work

By the end you will be able to:

- Read an environment variable in a component with `env()`
- Override it for one component with a `{COMPONENT}_{VAR}` prefix
- Fall back to a component's `.env` file
- Understand the auto-scaffolded `.env` and `README.md`

## Background

When a component runs, `env('API_KEY')` is resolved in this order:

1. `{COMPONENT_NAME_UPPER}_API_KEY` in the process environment (scoped override)
2. Plain `API_KEY` in the process environment
3. `API_KEY` in the component's own `.env` file (lowest priority)

So a `translator` component and a `summarizer` component can each get their own
key while sharing one variable name in the YAML.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

## Step 1: create the structure

```bash
mkdir -p envdemo/components/translator
mkdir -p envdemo/components/summarizer
mkdir envdemo/resources
cd envdemo
```

## Step 2: two components that read the same variable

Create `components/translator/component.yaml`:

<div v-pre>

```yaml
# components/translator/component.yaml
apiVersion: kdeps.io/v1
kind: Component

metadata:
  name: translator
  version: "1.0.0"

interface:
  inputs:
    - name: text
      type: string
      required: true

resources:
  - actionId: translate
    name: Translate
    exec:
      command: "echo"
      args:
        - "translator using key: {{ env('API_KEY', 'none') }} -- text: {{ input('text') }}"
    apiResponse:
      success: true
      response:
        out: "{{ get('translate') }}"
```

</div>

Create `components/summarizer/component.yaml` - identical but with
`name: summarizer` and `"summarizer using key: ..."` in the echo.

## Step 3: call both components

Create `resources/01-translate.yaml`:

<div v-pre>

```yaml
# resources/01-translate.yaml
actionId: runTranslate
name: Run translator
component:
  name: translator
  with:
    text: "hello"
```

</div>

Create `resources/02-summarize.yaml`:

<div v-pre>

```yaml
# resources/02-summarize.yaml
actionId: runSummarize
name: Run summarizer
component:
  name: summarizer
  with:
    text: "a long paragraph"
```

</div>

Create `resources/03-response.yaml`:

<div v-pre>

```yaml
# resources/03-response.yaml
actionId: response
name: Response
requires: [runTranslate, runSummarize]
apiResponse:
  success: true
  response:
    translator: "{{ output('runTranslate').out }}"
    summarizer: "{{ output('runSummarize').out }}"
```

</div>

## Step 4: add the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: envdemo
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /run
        methods: [GET]
```

## Step 5: run with scoped keys

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token

# Shared default for both components
export API_KEY=global-key
# Override just the translator
export TRANSLATOR_API_KEY=translate-key

kdeps run .
```

```bash
curl "http://localhost:16395/run" -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
```

Response:

```json
{
  "success": true,
  "data": {
    "translator": "translator using key: translate-key -- text: hello",
    "summarizer": "summarizer using key: global-key -- text: a long paragraph"
  }
}
```

The translator saw `TRANSLATOR_API_KEY`; the summarizer fell back to `API_KEY`.

## Step 6: inspect the scaffolded files

On the first run kdeps created these (it never overwrites existing ones):

```bash
cat components/translator/.env        # template listing every env() var, blank
cat components/translator/README.md   # generated from component.yaml metadata
```

Fill in `.env` to provide a lowest-priority fallback value when no process
environment variable is set.

## Summary

You built two components that:

- Read the same variable name with `env('API_KEY')`
- Resolve it independently: `{COMPONENT}_API_KEY` -> `API_KEY` -> `.env`
- Got an auto-scaffolded `.env` and `README.md` on first run

## Next steps

- [Components reference](/reference/components) - env derivation, `.env` details
- [Reusable component tutorial](/examples/custom-component) - building a component
- [Global config](/configuration/advanced) - named connections with shared credentials
- [Jinja2 templates](/concepts/jinja2-templates) - `env()` in YAML preprocessing
