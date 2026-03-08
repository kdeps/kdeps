# Jinja2 Template Integration

KDeps uses [Jinja2](https://jinja.palletsprojects.com/) compatible templates (via [gonja](https://github.com/nikolalohinski/gonja)) as the single unified template system for project scaffolding and resource generation. This replaces the previous dual-system approach of using both Mustache and Go's `text/template`.

## Why Jinja2?

- **Unified**: A single template engine for all project and resource generation
- **Expressive**: Full support for conditionals, loops, filters, and more
- **Familiar**: Jinja2 is widely known and used across many ecosystems
- **Readable**: Clean, intuitive syntax with `{{ }}` for variables and `{% %}` for control flow
- **Powerful**: Supports `in` operator, filters, macros, and raw blocks

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

Creates a new Jinja2 template renderer with support for the embedded filesystem.

### Render

```go
func (r *Jinja2Renderer) Render(templateContent string, data map[string]interface{}) (string, error)
```

Renders a Jinja2 template string with the provided data.

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

