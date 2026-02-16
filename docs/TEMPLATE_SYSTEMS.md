# Template Systems in kdeps

kdeps uses THREE different template/expression systems for different purposes. Understanding the distinction is crucial:

## 1. Runtime Expression System (PRIMARY) - UNIFIED!

**Location**: `pkg/parser/expression/`  
**Purpose**: Dynamic value evaluation in workflow YAML files at runtime  
**Syntax**: Unified - use any style, mix freely!
- **Simple vars**: `{{var}}` or `{{ var }}` (both work!)
- **Functions**: `{{ get('var') }}`, `{{ info('field') }}`
- **Mixed**: `"Hello {{name}}, time is {{ info('time') }}"`
**Engine**: [expr-lang/expr](https://github.com/expr-lang/expr) + smart fallback

### How It Works - Unified Evaluation

Each `{{ }}` block is evaluated independently:
1. **Try mustache first**: For simple variables (no operators, no functions)
2. **Fall back to expr-lang**: For functions, calculations, conditionals

### Examples from `examples/**/*.yaml`:

```yaml
# Simple variables - all work the same:
name: "{{name}}"              # No spaces
name: "{{ name }}"            # With spaces - both work!
email: "{{user.email}}"       # Dot notation
  
# Functions and complex expressions:
timestamp: "{{ info('current_time') }}"
result: "{{ get('count') + 10 }}"
status: "{{ score > 80 ? 'Pass' : 'Fail' }}"

# Mix them naturally!
message: "Hello {{name}}, you have {{ get('items') }} items at {{ info('time') }}"
```

### Key Points:
- ✅ Used in ALL workflow YAML files
- ✅ Evaluated at runtime when workflow executes
- ✅ **No whitespace rules** - `{{var}}` and `{{ var }}` identical
- ✅ **Mix freely** - simple vars + functions in same template
- ✅ **Smart detection** - tries mustache first, falls back to expr-lang
- ✅ Access to unified API: get(), set(), info(), env(), safe()
- ✅ Supports conditionals: `{{ condition ? valueIfTrue : valueIfFalse }}`
- ✅ This is the MAIN template system users interact with

### When Each Engine is Used

**Mustache lookup** (fast, simple):
- ✅ Simple variables: `{{name}}`
- ✅ Nested objects: `{{user.email}}`
- ✅ No operators, no functions

**expr-lang** (powerful):
- ✅ Function calls: `{{ get() }}`, `{{ info() }}`
- ✅ Calculations: `{{ count + 10 }}`
- ✅ Conditionals: `{{ x > 80 ? 'Pass' : 'Fail' }}`
- ✅ Any operators: `+`, `-`, `==`, etc.

## 2. Go Templates (text/template)

**Location**: `pkg/templates/`, `pkg/infra/docker/`, `pkg/infra/wasm/`  
**Purpose**: Code/file generation at build/scaffold time  
**Syntax**: `{{ .Variable }}`, `{{- if .Flag }}`, `{{ range .Items }}`  
**Engine**: Go's `text/template` package

### Examples:

```go
// pkg/infra/docker/builder_templates.go - Generates Dockerfiles
tmpl, err := template.New("dockerfile").Parse(templateStr)

// pkg/templates/generator.go - Scaffolds new projects
tmpl.Execute(out, data)
```

### Key Points:
- ✅ Used for generating files/code
- ✅ Used at development/build time (not runtime)
- ✅ Three use cases:
  1. Project scaffolding (pkg/templates)
  2. Dockerfile generation (pkg/infra/docker)
  3. WASM bundle generation (pkg/infra/wasm)

## 3. Mustache Templates

**Location**: `pkg/templates/` (ONLY)  
**Purpose**: Alternative syntax for project scaffolding  
**Syntax**: `{{name}}`, `{{#section}}`, `{{^inverted}}`  
**Engine**: [cbroglie/mustache](https://github.com/cbroglie/mustache)

### Current Status:
- ✅ Added as alternative to Go templates for scaffolding
- ⚠️ Only in pkg/templates, not in docker/wasm builders
- ⚠️ May cause confusion with runtime expression syntax
- ⚠️ Limited integration

### Key Points:
- ❌ NOT used in workflow YAML files
- ❌ NOT evaluated at runtime
- ✅ Only for generating new projects via `kdeps new`

## The Confusion

The mustache PR added a THIRD template system, but:

1. **Users already have `{{ }}` syntax** for runtime expressions (the main use case)
2. **Mustache uses similar `{{ }}` syntax** but for different purpose (scaffolding)
3. **Limited scope** - only scaffolding, not docker/wasm generation
4. **Documentation unclear** about the three systems

## Recommendations

### Option A: Remove Mustache (Simplify)
- Keep only 2 systems: Runtime expressions + Go templates
- Clearer separation: `{{ }}` = runtime, Go template = generation
- Less confusion for users

### Option B: Fully Integrate Mustache
- Add mustache support to docker/wasm builders
- Make it the primary generation template system
- Update all generation code to use mustache
- Clear docs: Runtime expressions vs Mustache generation

### Option C: Keep but Document
- Keep current limited mustache support
- Add clear documentation explaining all three systems
- Warn users about syntax similarity

## Examples Breakdown

| File | System Used | When Evaluated | Purpose |
|------|-------------|----------------|---------|
| `examples/chatbot/workflow.yaml` | Runtime Expressions | At workflow runtime | Dynamic values in running workflow |
| `pkg/templates/templates/api-service/workflow.yaml.tmpl` | Go Templates | At scaffold time | Generate new project |
| `pkg/templates/templates/mustache-api-service/workflow.yaml.mustache` | Mustache | At scaffold time | Generate new project (alternative syntax) |
| `pkg/infra/docker/builder_templates.go` | Go Templates | At docker build time | Generate Dockerfile |

## See Also

- Runtime expressions: `pkg/parser/expression/evaluator.go`
- Go template scaffolding: `pkg/templates/generator.go`
- Mustache scaffolding: `pkg/templates/mustache_renderer.go`
- Docker generation: `pkg/infra/docker/builder_templates.go`
- WASM generation: `pkg/infra/wasm/bundler.go`
