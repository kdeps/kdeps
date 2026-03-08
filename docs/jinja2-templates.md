# Jinja2 Template Integration

KDeps uses [Jinja2](https://jinja.palletsprojects.com/) compatible templates (via [gonja](https://github.com/nikolalohinski/gonja)) as the single unified template system for **both** project scaffolding and runtime workflow/resource YAML files. This replaces the previous dual-system approach of using both Mustache and Go's `text/template`.

## Why Jinja2?

- **Unified**: A single template engine for scaffolding templates, workflow YAML, and resource YAML
- **Expressive**: Full support for conditionals, loops, filters, and more
- **Familiar**: Jinja2 is widely known and used across many ecosystems
- **Readable**: Clean, intuitive syntax with `{{ }}` for variables and `{% %}` for control flow
- **Powerful**: Supports `in` operator, filters, macros, and raw blocks

## Scope of Jinja2 Rendering

| Use case | When applied | Available variables |
|----------|-------------|---------------------|
| Project scaffolding (`.j2` files) | During `kdeps new` / project generation | `name`, `description`, `version`, `port`, `resources`, feature flags |
| Workflow YAML files | At parse time (before YAML is processed) | `env` (process environment variables) |
| Resource YAML files | At parse time (before YAML is processed) | `env` (process environment variables) |

### Workflow and Resource YAML Preprocessing

When a workflow or resource YAML file contains Jinja2 control tags (`{%`) or comment tags (`{#`), KDeps preprocesses the file with Jinja2 **before** YAML parsing. This allows you to:

- Conditionally include or exclude sections based on environment variables
- Set values from environment variables with defaults
- Add comments that are stripped before parsing

Files that contain **only** `{{ expr }}` runtime expressions (without any `{%` or `{#` tags) are **not** preprocessed by Jinja2, ensuring full backward-compatibility with existing workflow and resource files.

#### Auto-protection of kdeps runtime API calls

KDeps **automatically** protects all runtime API function calls (`{{ get(...) }}`, `{{ set(...) }}`, `{{ info(...) }}`, `{{ input(...) }}`, `{{ output(...) }}`, `{{ file(...) }}`, `{{ item(...) }}`, `{{ loop(...) }}`, `{{ session(...) }}`, `{{ json(...) }}`, `{{ safe(...) }}`, `{{ debug(...) }}`, `{{ default(...) }}`) from Jinja2 evaluation. You do **not** need to wrap them in `{% raw %}...{% endraw %}`.

Static Jinja2 variable expressions such as `{{ env.PORT }}` are evaluated normally because they do not start with a kdeps API function name.

#### Example: Conditional block based on environment

```jinja2
# workflow.yaml
apiVersion: v2
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

#### Example: Mixing Jinja2 control flow with runtime expressions (no `{% raw %}` needed)

```jinja2
# resources/fetch.yaml
apiVersion: v2
kind: Resource
metadata:
  actionId: fetchData
  name: Fetch Data
run:
{% if env.ENABLE_HTTP == 'true' %}
  httpClient:
    method: GET
    url: "{{ get('url') }}"
    headers:
      X-Request-ID: "{{ info('request_id') }}"
{% endif %}
```

`{{ get('url') }}` and `{{ info('request_id') }}` are automatically preserved unchanged by the Jinja2 preprocessor and evaluated later by the kdeps runtime expression evaluator during request handling.

#### Available context variables

| Variable | Type | Description |
|----------|------|-------------|
| `env` | `map[string]string` | All current process environment variables |


## Template Syntax

### Variables

```jinja2
Hello {{ name }}!
Port: {{ port }}
```

### Conditionals

```jinja2
{% if "http-client" in resources %}
  - HTTP Client enabled
{% endif %}

{% if "llm" in resources %}
  - LLM enabled
{% else %}
  - No LLM
{% endif %}
```

### Loops

```jinja2
{% for resource in resources %}
  - {{ resource }}
{% endfor %}
```

### Comments

```jinja2
{# This is a comment and won't be rendered #}
```

### Raw Blocks (for runtime expressions)

Use `{% raw %}...{% endraw %}` to preserve `{{ }}` syntax in generated files for use as runtime expressions:

```jinja2
url: "{% raw %}{{ get('url', 'https://api.example.com') }}{% endraw %}"
```

This outputs: `url: "{{ get('url', 'https://api.example.com') }}"` in the generated file.

### Whitespace Control

Use `-` to trim whitespace around tags:

```jinja2
resources:
{%- if "http-client" in resources %}
  - apiVersion: v2
    kind: Resource
{%- endif %}
```

## Using Jinja2 Templates

### Creating a Project Template

1. Create a directory in `pkg/templates/templates/`
2. Add files with `.j2` extension
3. Use Jinja2 syntax in your templates

Example: `pkg/templates/templates/my-service/workflow.yaml.j2`

```jinja2
apiVersion: v2
kind: Workflow
metadata:
  name: {{ name }}
  description: {{ description }}
  version: {{ version }}

settings:
  apiServerMode: true
  apiServer:
    portNum: {{ port }}

resources:
{%- if "http-client" in resources %}
  - apiVersion: v2
    kind: Resource
    metadata:
      actionId: fetchData
      name: Fetch Data
    run:
      httpClient:
        method: GET
        url: "{% raw %}{{ get('url') }}{% endraw %}"
{%- endif %}
```

### Template Data

The `TemplateData` struct provides these variables to Jinja2 templates:

| Variable | Type | Description |
|----------|------|-------------|
| `name` | string | Agent/project name |
| `description` | string | Description |
| `version` | string | Version string |
| `port` | int | API server port |
| `resources` | []string | List of enabled resources |

Plus any keys from `Features map[string]bool`.

### Generating Projects

```go
generator, err := templates.NewGenerator()
if err != nil {
    // handle error
}

data := templates.TemplateData{
    Name:        "my-api",
    Description: "API service using Jinja2 templates",
    Version:     "1.0.0",
    Port:        9000,
    Resources:   []string{"http-client", "llm"},
}

err = generator.GenerateProject("api-service", "./output", data)
```

### Special Filename Handling

| Template file | Generated file |
|--------------|----------------|
| `workflow.yaml.j2` | `workflow.yaml` |
| `README.md.j2` | `README.md` |
| `env.example.j2` | `.env.example` |

## Comparing Template Syntax

The migration from the old dual systems to Jinja2:

| Feature | Old Go Templates | Old Mustache | Jinja2 (new) |
|---------|-----------------|--------------|--------------|
| Variables | `{{ .Name }}` | `{{name}}` | `{{ name }}` |
| Conditionals | `{{ if has .Resources "http" }}` | `{{#hasHttp}}` | `{% if "http" in resources %}` |
| Loops | `{{ range .Items }}` | `{{#items}}` | `{% for item in items %}` |
| Nesting | `{{ .User.Name }}` | `{{user.name}}` | `{{ user.name }}` |
| Comments | `{{/* comment */}}` | `{{! comment }}` | `{# comment #}` |
| Raw output | `{{ "{{" }} expr {{ "}}" }}` | N/A | `{% raw %}{{ expr }}{% endraw %}` |

## API Reference

### NewJinja2Renderer

```go
func NewJinja2Renderer(fs embed.FS) *Jinja2Renderer
```

Creates a new Jinja2 template renderer with support for the embedded filesystem. Parsed templates are cached to avoid repeated parsing of the same content.

### Render

```go
func (r *Jinja2Renderer) Render(templateContent string, data map[string]interface{}) (string, error)
```

Renders a Jinja2 template string with the provided data. The parsed template is cached by content.

### RenderFile

```go
func (r *Jinja2Renderer) RenderFile(templatePath string, data map[string]interface{}) (string, error)
```

Renders a Jinja2 template file from the embedded filesystem.

### ToJinja2Data

```go
func (t TemplateData) ToJinja2Data() map[string]interface{}
```

Converts `TemplateData` to a format suitable for Jinja2 templates.

## Further Reading

- [Jinja2 Documentation](https://jinja.palletsprojects.com/en/stable/)
- [gonja (Go Jinja2 library)](https://github.com/nikolalohinski/gonja)
