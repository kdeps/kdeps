# Jinja2 Templates

KDeps uses [Jinja2](https://jinja.palletsprojects.com/)-compatible templates (via [gonja](https://github.com/nikolalohinski/gonja)) as the unified template system for both project scaffolding and runtime YAML preprocessing.

## YAML Preprocessing

Every workflow and resource YAML file is preprocessed through Jinja2 **before** YAML parsing. This lets you:

- Conditionally include or exclude sections based on environment variables
- Inject environment values directly into YAML at parse time
- Strip comments before parsing

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-api
  version: "1.0.0"
  targetActionId: response
settings:
  apiServerMode: true
{% if env.PORT %}
  portNum: {{ env.PORT | int }}
{% else %}
  portNum: 8080
{% endif %}
```

### Auto-protection of Runtime API Calls

KDeps automatically wraps all runtime API function calls (<code v-pre>{{ get(...) }}</code>, <code v-pre>{{ set(...) }}</code>, <code v-pre>{{ info(...) }}</code>, <code v-pre>{{ input(...) }}</code>, <code v-pre>{{ output(...) }}</code>, <code v-pre>{{ file(...) }}</code>, <code v-pre>{{ item(...) }}</code>, <code v-pre>{{ loop(...) }}</code>, <code v-pre>{{ session(...) }}</code>, <code v-pre>{{ json(...) }}</code>, <code v-pre>{{ safe(...) }}</code>, <code v-pre>{{ debug(...) }}</code>, <code v-pre>{{ default(...) }}</code>) in `{% raw %}...{% endraw %}` before Jinja2 renders the file. You **do not** need to add raw blocks manually.

Static Jinja2 expressions like <code v-pre>{{ env.PORT }}</code> are evaluated normally because they do not start with a kdeps API function name.

```yaml
# resource.yaml — no {% raw %} needed
run:
{% if env.ENABLE_HTTP == 'true' %}
  httpClient:
    method: GET
    url: "{{ get('url') }}"
    headers:
      X-Request-ID: "{{ info('request_id') }}"
{% endif %}
```

### Available Context Variables

| Variable | Type | Description |
|----------|------|-------------|
| `env` | `map[string]string` | All current process environment variables |

## Template Syntax

### Variables

```jinja2
port: {{ env.PORT | int }}
name: {{ env.SERVICE_NAME | default('my-service') }}
```

### Conditionals

```jinja2
{% if env.DEBUG == 'true' %}
  logLevel: debug
{% else %}
  logLevel: info
{% endif %}
```

### Comments

```jinja2
{# This comment is stripped before parsing #}
```

### Whitespace Control

Use `-` to trim surrounding whitespace:

```jinja2
settings:
{%- if env.TLS_ENABLED == 'true' %}
  tls: true
{%- endif %}
```

## Scaffolding Templates

Project scaffolding templates (used by `kdeps new`) also use Jinja2 with `.j2` file extensions. Variables available in scaffolding templates:

| Variable | Type | Description |
|----------|------|-------------|
| `name` | string | Project name |
| `description` | string | Project description |
| `version` | string | Version string |
| `port` | int | API server port |
| `resources` | []string | Enabled resource types |

## See Also

- [Expressions](/concepts/expressions)
- [Unified API](/concepts/unified-api)
- [Jinja2 Documentation](https://jinja.palletsprojects.com/en/stable/)
