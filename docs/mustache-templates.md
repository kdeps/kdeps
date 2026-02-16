# Mustache Template Integration

KDeps now supports [Mustache](https://mustache.github.io/) templates in addition to Go's `text/template` package. Mustache is a logic-less template syntax that is simpler and more portable across different programming languages.

## Why Mustache?

- **Logic-less**: Encourages separation of logic and presentation
- **Cross-platform**: Same syntax works across many programming languages
- **Simple**: Easy to learn with minimal syntax
- **Safe**: No code execution in templates

## Template Detection

KDeps automatically detects whether to use Go templates or Mustache templates based on:

1. **File extension**: Files ending with `.mustache` are always processed as Mustache templates
2. **Content analysis**: Files with `.tmpl` extension are analyzed to detect which type they are:
   - Mustache syntax: `{{name}}`, `{{#section}}`, `{{^inverted}}`, `{{! comment}}`
   - Go template syntax: `{{ .Name }}`, `{{- trim }}`, `{{ "escaped" }}`

## Mustache Syntax

### Variables

```mustache
Hello {{name}}!
Port: {{port}}
```

### Sections (Conditionals/Loops)

```mustache
{{#hasHttpClient}}
  - HTTP Client enabled
{{/hasHttpClient}}

{{#items}}
  * {{name}}
{{/items}}
```

### Inverted Sections

```mustache
{{^items}}
  No items available
{{/items}}
```

### Comments

```mustache
{{! This is a comment and won't be rendered }}
```

### HTML Escaping

```mustache
{{name}}        <!-- HTML escaped -->
{{{rawHtml}}}   <!-- Not escaped -->
```

## Using Mustache Templates

### Creating a Mustache Template

1. Create a directory in `pkg/templates/templates/`
2. Add files with `.mustache` extension
3. Use mustache syntax in your templates

Example: `pkg/templates/templates/my-service/workflow.yaml.mustache`

```mustache
apiVersion: v2
kind: Workflow
metadata:
  name: {{name}}
  description: {{description}}
  version: {{version}}

settings:
  apiServerMode: true
  apiServer:
    portNum: {{port}}

{{#hasHttpClient}}
resources:
  - apiVersion: v2
    kind: Resource
    metadata:
      actionId: httpClient
      name: HTTP Client
{{/hasHttpClient}}
```

### Template Data Conversion

The `TemplateData` struct has a `ToMustacheData()` method that converts resource names into boolean flags for easier conditional rendering:

```go
data := TemplateData{
    Name:        "my-api",
    Version:     "1.0.0",
    Port:        8080,
    Resources:   []string{"http-client", "llm", "response"},
}

// Automatically creates flags:
// hasHttpClient: true
// hasLlm: true
// hasResponse: true
```

### Generating Projects

Use the same API as before - KDeps will automatically detect and use the appropriate template engine:

```go
generator, err := templates.NewGenerator()
if err != nil {
    // handle error
}

data := templates.TemplateData{
    Name:        "my-mustache-api",
    Description: "API service using mustache templates",
    Version:     "1.0.0",
    Port:        9000,
    Resources:   []string{"http-client", "llm"},
}

err = generator.GenerateProject("mustache-api-service", "./output", data)
```

## Comparing Go Templates vs Mustache

| Feature | Go Templates | Mustache |
|---------|-------------|----------|
| Variables | `{{ .Name }}` | `{{name}}` |
| Conditionals | `{{ if .Flag }}` | `{{#flag}}` |
| Loops | `{{ range .Items }}` | `{{#items}}` |
| Functions | `{{ has .Resources "http" }}` | Use boolean flags |
| Nesting | `{{ .User.Name }}` | `{{user.name}}` |
| Comments | `{{/* comment */}}` | `{{! comment }}` |

## Example: Mustache vs Go Template

### Go Template Style
```go
{{ .Name }} - {{ .Version }}
{{- if has .Resources "http-client" }}
HTTP Client: enabled
{{- end }}
```

### Mustache Style
```mustache
{{name}} - {{version}}
{{#hasHttpClient}}
HTTP Client: enabled
{{/hasHttpClient}}
```

## Best Practices

1. **Use .mustache extension**: Makes it explicit which template engine to use
2. **Convert complex logic to data**: Instead of complex conditionals, prepare data with boolean flags
3. **Keep templates simple**: Mustache's logic-less nature encourages cleaner separation
4. **Test both engines**: Ensure backward compatibility if migrating from Go templates

## Migration Guide

If you have existing Go templates and want to use Mustache:

1. **Rename files**: Change `.tmpl` to `.mustache`
2. **Update variable syntax**: `{{ .Name }}` → `{{name}}`
3. **Convert conditions**: `{{ if .Flag }}` → `{{#flag}}`
4. **Replace custom functions**: Use `ToMustacheData()` to create boolean flags
5. **Update loops**: `{{ range .Items }}` → `{{#items}}`

## API Reference

### NewMustacheRenderer

```go
func NewMustacheRenderer(fs embed.FS) *MustacheRenderer
```

Creates a new Mustache template renderer with support for the embedded filesystem.

### Render

```go
func (r *MustacheRenderer) Render(templateContent string, data interface{}) (string, error)
```

Renders a mustache template string with the provided data.

### RenderFile

```go
func (r *MustacheRenderer) RenderFile(templatePath string, data interface{}) (string, error)
```

Renders a mustache template file from the embedded filesystem.

### ToMustacheData

```go
func (t TemplateData) ToMustacheData() map[string]interface{}
```

Converts TemplateData to a format suitable for mustache templates, adding boolean flags for each resource type.

## Supported Resource Flags

When using `ToMustacheData()`, the following boolean flags are automatically created:

- `hasHttpClient` - for "http-client" resource
- `hasLlm` - for "llm" resource
- `hasSql` - for "sql" resource
- `hasPython` - for "python" resource
- `hasExec` - for "exec" resource
- `hasResponse` - for "response" resource

## Further Reading

- [Mustache Manual](https://mustache.github.io/mustache.5.html)
- [Mustache Specification](https://github.com/mustache/spec)
- [cbroglie/mustache Library](https://github.com/cbroglie/mustache)
