# Jinja2 Expressions Example

This example demonstrates **Jinja2 preprocessing** in kdeps workflow and resource YAML files.

## Overview

Every workflow and resource YAML file is preprocessed through Jinja2 before parsing. This lets you:

- Use environment variables to conditionally configure your workflow
- Inject environment values at parse time with `{{ env.VAR }}`
- Use Jinja2 conditionals (`{% if %}`) and comments (`{# ... #}`)

kdeps runtime API calls (`{{ get(...) }}`, `{{ info(...) }}`, `{{ set(...) }}`, etc.) are **automatically protected** from Jinja2 evaluation — you don't need `{% raw %}` blocks.

## Workflow Structure

```
jinja2-expressions/
├── workflow.yaml          # Workflow with Jinja2 conditionals
└── resources/
    └── response.yaml      # Resource using runtime API + env vars
```

## How It Works

### Environment-driven configuration

```yaml
# workflow.yaml
{% if env.PORT %}
  portNum: {{ env.PORT | int }}
{% else %}
  portNum: 16395
{% endif %}
```

Set `PORT=9000` at run time to override the default port.

### Auto-protected runtime API calls

In resource YAML files, kdeps API calls are automatically wrapped in `{% raw %}`:

```yaml
# resources/response.yaml
run:
  apiResponse:
    response:
      message: "{{ info('method') }} at {{ info('current_time') }}"
      query: "{{ get('q') }}"
```

`{{ get('q') }}` and `{{ info(...) }}` are **not** evaluated by Jinja2 — they pass through unchanged and are evaluated at runtime.

### Static Jinja2 expressions

Variables that don't match a kdeps API function name are evaluated by Jinja2:

```yaml
# Evaluated by Jinja2 at parse time
description: "{{ env.KDEPS_APP_DESC | default('My API') }}"
```

## Running This Example

```bash
cd examples/jinja2-expressions
kdeps run workflow.yaml --dev
```

Override the port via environment variable:

```bash
PORT=9000 kdeps run workflow.yaml --dev
```

Test with:

```bash
curl "http://localhost:16395/api/demo?q=hello"
```

## Key Concepts

| Syntax | Evaluated by | When |
|--------|-------------|------|
| `{{ env.VAR }}` | Jinja2 | Parse time |
| `{% if env.X %}...{% endif %}` | Jinja2 | Parse time |
| `{# comment #}` | Jinja2 | Parse time (stripped) |
| `{{ get('x') }}` | kdeps runtime | Request time |
| `{{ info('time') }}` | kdeps runtime | Request time |
| `{{ set('k', 'v') }}` | kdeps runtime | Request time |
