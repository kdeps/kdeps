# Jinja2 templates

kdeps uses [Jinja2](https://jinja.palletsprojects.com/)-compatible templates (with [gonja](https://github.com/nikolalohinski/gonja)). This is the one template system for project scaffolding and for runtime YAML preprocessing. Preprocessing runs for every workflow and resource file in both modes.

## YAML preprocessing

Before kdeps parses the YAML, Jinja2 preprocesses each workflow and resource YAML file. With this, you can:

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
deploy:
  apiServer:
{% if env.PORT %}
    portNum: {{ env.PORT | int }}
{% else %}
    portNum: 8080
{% endif %}
```

### Auto-protection of runtime API calls

Before Jinja2 renders the file, kdeps puts each runtime API call in `{% raw %}...{% endraw %}`. You do not add raw blocks yourself. The runtime API calls are: <code v-pre>{{ get(...) }}</code>, <code v-pre>{{ set(...) }}</code>, <code v-pre>{{ info(...) }}</code>, <code v-pre>{{ input(...) }}</code>, <code v-pre>{{ output(...) }}</code>, <code v-pre>{{ file(...) }}</code>, <code v-pre>{{ item(...) }}</code>, <code v-pre>{{ loop(...) }}</code>, <code v-pre>{{ session(...) }}</code>, <code v-pre>{{ json(...) }}</code>, <code v-pre>{{ safe(...) }}</code>, <code v-pre>{{ debug(...) }}</code>, <code v-pre>{{ default(...) }}</code>.

Jinja2 evaluates static expressions like <code v-pre>{{ env.PORT }}</code> as usual, because they do not start with a kdeps API function name.

```yaml
# resource.yaml - no {% raw %} needed
{% if env.ENABLE_HTTP == 'true' %}
  component:
    name: http_request
    with:
      method: GET
      url: "{{ get('url') }}"
      headers:
        X-Request-ID: "{{ info('request_id') }}"
{% endif %}
```

### Available context variables

| Variable | Type | Description |
|----------|------|-------------|
| `env` | `map[string]string` | All current process environment variables |

## Template syntax

### Variables

```jinja
port: {{ env.PORT | int }}
name: {{ env.SERVICE_NAME | default('my-service') }}
```

### Conditionals

```jinja
{% if env.DEBUG == 'true' %}
  logLevel: debug
{% else %}
  logLevel: info
{% endif %}
```

### Comments

```jinja
{# This comment is stripped before parsing #}
```

### Whitespace control

Use `-` to trim surrounding whitespace:

```jinja
{%- if env.CERT_FILE %}
  certFile: {{ env.CERT_FILE }}
{%- endif %}
```

## Scaffolding templates

Project scaffolding templates (used by `kdeps new`) also use Jinja2 with `.j2` file extensions. Variables available in scaffolding templates:

| Variable | Type | Description |
|----------|------|-------------|
| `name` | string | Project name |
| `description` | string | Project description |
| `version` | string | Version string |
| `port` | int | API server port |
| `resources` | []string | Enabled resource types |

## See also

- [Expressions](/workflow/expressions)
- [Unified API](/workflow/data-access)
- [Jinja2 documentation](https://jinja.palletsprojects.com/en/stable/)
