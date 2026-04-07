# Components

KDeps has three categories of components. Understanding the difference is important before reading anything else here.

## Types of Components

### 1. Built-in components (internal)

The five core executors are always available in every workflow — no installation required. The CLI surfaces them as "Internal components (built-in)" when you run `kdeps component list`:

| Component | YAML key | Description |
|-----------|----------|-------------|
| LLM | `chat:` | LLM interaction (Ollama, OpenAI, Anthropic, etc.) |
| HTTP | `httpClient:` | REST API calls |
| SQL | `sql:` | Database queries (Postgres, MySQL, SQLite) |
| Python | `python:` | Python scripts via isolated `uv` environments |
| Exec | `exec:` | Shell commands |

These are compiled into the `kdeps` binary and require no `kdeps component install`.

### 2. Registry components (installable)

Pre-built capability extensions distributed as `.komponent` archives. Install once, available to any workflow on the machine:

```bash
kdeps component install scraper     # web/doc text extraction
kdeps component install search      # web search (Tavily)
kdeps component install tts         # text-to-speech
kdeps component install email       # send email via SMTP
kdeps component install pdf         # generate PDFs
kdeps component install calendar    # generate .ics event files
kdeps component install embedding   # vector embeddings (OpenAI)
kdeps component install memory      # key-value store (SQLite)
kdeps component install browser     # browser automation (Playwright)
kdeps component install botreply    # chat bot replies
kdeps component install remoteagent # call a remote kdeps agent
kdeps component install autopilot   # LLM-directed task execution
kdeps component install federation  # UAF node management
```

Invoked with `run.component:` in any resource:

```yaml
run:
  component:
    name: scraper
    with:
      url: "https://example.com"
```

### 3. Custom components (user-defined)

Components you build yourself: a `component.yaml` manifest plus resource files in a `components/<name>/` directory. Auto-discovered at run time — no changes to `workflow.yaml` needed.

---

## Custom Component Structure

A custom component is defined by a `component.yaml` manifest and lives in a `components/<name>/` directory alongside your workflow. When the workflow runs, all component resources are automatically loaded and merged with the workflow's own resources.

### Key Benefits

- **Reusability**: Use the same component across multiple workflows
- **Encapsulation**: Hide implementation details; expose only inputs
- **Shareable**: Package as `.komponent` archives for distribution
- **Auto-discovery**: No need to declare components in `workflow.yaml`; just place them in `components/`

## Directory Layout

```
my-workflow/
├── workflow.yaml
├── resources/
└── components/                  ← auto-discovered
    └── greeter/
        ├── component.yaml       ← component manifest
        ├── resources/           ← component-specific resources
        │   └── greet.yaml
        └── template.html        ← optional UI template
```

## component.yaml Reference

```yaml
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: greeter                # Required: component name
  description: "A greeting component"
  version: "1.0.0"
  targetActionId: greet        # Optional: default action to invoke
interface:
  inputs:
    - name: message            # Required: input parameter name
      type: string             # Required: string, integer, number, boolean
      required: true           # Optional (default: false)
      description: "Message to greet with"
      default: "Hello"         # Optional: default value if not required
resources:
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: greet
    # ... resource definition
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | string | yes | Must be `kdeps.io/v1` |
| `kind` | string | yes | Must be `Component` |
| `metadata.name` | string | yes | Unique component name within the workflow |
| `metadata.version` | string | no | Component version (used for packaging) |
| `metadata.targetActionId` | string | no | Default resource `actionId` when component is invoked |
| `interface.inputs[].name` | string | yes | Input parameter name |
| `interface.inputs[].type` | enum | yes | Data type: `string`, `integer`, `number`, `boolean` |
| `interface.inputs[].required` | bool | no | Whether input is required (default: false) |
| `interface.inputs[].description` | string | no | Human-readable description |
| `interface.inputs[].default` | any | no | Default value (only meaningful when required=false) |
| `resources` | array | no | Inline resource definitions (same as workflow resources) |

## Interface and Inputs

The `interface` section defines the component's public contract — the named parameters that parent workflows can provide. Inputs behave like function arguments:

```yaml
interface:
  inputs:
    - name: user_query
      type: string
      required: true
    - name: temperature
      type: number
      required: false
      default: 0.7
```

When the parent workflow calls the component's target action, it supplies these inputs. The component's resources can reference them via expressions:

```yaml
resources:
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: greet
    run:
      chat:
        model: "llama3.2:latest"
        prompt: "{{ inputs.message }}"
```

## Resources

Component resources can be defined inline in `component.yaml` or placed in the `resources/` subdirectory. The `actionId` **must be unique** across the entire workflow (component resources take precedence on conflict, but avoid collisions).

> **Note**: Components cannot contain `settings` (no server modes, port bindings, etc.). They are purely resource bundles.

## Auto-Discovery

When a workflow is parsed, KDeps automatically scans for `components/<name>/component.yaml` files in the same directory as the workflow. All resources from each component are prepended to the workflow's resource list, making them available as if they were defined locally.

- No explicit declaration in `workflow.yaml` is needed
- Local workflow resources override component resources on `actionId` collision
- Component loading happens recursively (imported workflows can also have components)

### Resolution Order

1. Workflow's inline resources
2. Resources from imported workflows (`metadata.workflows`)
3. Resources from auto-discovered components (alphabetical by component name)
4. Local workflow `resources/` directory

## Installing Components from the Registry

The 12 standard capability extensions are distributed as pre-built `.komponent` archives and can be installed with a single command:

```bash
kdeps component install scraper
kdeps component install search
kdeps component install embedding
kdeps component install botreply
kdeps component install remoteagent
kdeps component install tts
kdeps component install email
kdeps component install calendar
kdeps component install pdf
kdeps component install memory
kdeps component install browser
kdeps component install autopilot
```

Installed components are placed in the `components/` directory of your workflow and are auto-discovered at run time — no changes to `workflow.yaml` are needed.

```bash
kdeps component list           # List installed components
kdeps component remove scraper # Uninstall a component
```

## Components as LLM Tools (Opt-In)

Installed components can be exposed as LLM function-calling tools via the `componentTools:` allowlist on the `chat:` resource. **By default no components are registered** — you must explicitly name which ones the LLM may call.

```yaml
# kdeps component install scraper
# kdeps component install search

run:
  chat:
    model: gpt-4o
    prompt: "Research {{ get('q') }} and summarize the findings."
    componentTools:
      - scraper
      - search
```

The component's `interface.inputs` become the tool's parameter schema. The LLM uses this schema to decide when and how to call the tool.

**Rules:**

- `componentTools:` absent or empty — no components are registered (default).
- Names in `componentTools:` that are not installed are silently ignored.
- Explicit `tools:` entries always take precedence over `componentTools:` entries with the same name — no duplication.

| Priority | Source |
|----------|--------|
| 1 (highest) | Explicit `tools:` in the resource YAML |
| 2 | `componentTools:` allowlist |

---

## Calling a Component: `run.component:` Syntax

Once a component is installed, resources invoke it using the `run.component:` block instead of a raw executor key. The `with:` map passes typed inputs that are validated against the component's `interface.inputs` declaration.

```yaml
run:
  component:
    name: scraper
    with:
      url: "https://example.com"
      selector: ".article"
      timeout: 15
```

### Input Validation

When the resource executes, kdeps validates the `with:` map against the component manifest:

| Condition | Behaviour |
|-----------|-----------|
| Required input missing | Error — execution stops |
| Unknown key in `with:` | Warning logged; key is ignored |
| Optional input omitted | Component default value is applied |

### Input Scoping

Inputs supplied via `with:` are injected into the shared context under **two** scoped keys, so the same component can be called twice in a single workflow with different inputs:

| Key pattern | Example |
|-------------|---------|
| `<callerActionId>.<inputName>` | `fetch-cv.url` |
| `<componentName>.<inputName>` | `scraper.url` |

Inside component resources, reference inputs with either key or via `inputs.<inputName>` expressions.

### Accessing Component Output

After `run.component:` executes, results are stored under the caller resource's `actionId` and retrieved with `output('<callerActionId>')`:

<div v-pre>

```yaml
metadata:
  actionId: fetch-article
run:
  component:
    name: scraper
    with:
      url: "https://example.com/article"
      selector: ".content"

---

metadata:
  actionId: summarize
  requires: [fetch-article]
run:
  chat:
    model: llama3.2:1b
    prompt: "Summarize: {{ output('fetch-article').content }}"
```

</div>

### Calling the Same Component Twice

Because inputs are scoped to the caller's `actionId`, you can use the same component twice with different configurations:

<div v-pre>

```yaml
# First call — fetch the job description
metadata:
  actionId: fetch-jd
run:
  component:
    name: scraper
    with:
      url: "{{ get('jd_url') }}"

# Second call — fetch the company page
metadata:
  actionId: fetch-company
run:
  component:
    name: scraper
    with:
      url: "{{ get('company_url') }}"
      timeout: 60
```

</div>

`output('fetch-jd')` and `output('fetch-company')` are independent result maps.

### Complete Example: Scraper Component

Install, call, and consume results from the `scraper` component in one workflow:

```bash
kdeps component install scraper
```

<div v-pre>

```yaml
# resources/scrape-page.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: scrape-page
  name: Scrape Article

run:
  component:
    name: scraper
    with:
      url: "https://news.example.com/article"
      selector: ".article-body"
      timeout: 30

---

# resources/summarize.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: summarize
  name: Summarize Article
  requires:
    - scrape-page

run:
  chat:
    model: llama3.2:1b
    prompt: "Summarize the following article in 3 bullet points:\n\n{{ output('scrape-page').content }}"

---

# resources/respond.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: respond
  name: Return Summary
  requires:
    - summarize

run:
  apiResponse:
    success: true
    response:
      summary: "{{ output('summarize') }}"
```

</div>



Use `kdeps package` to create a portable `.komponent` archive:

```bash
# From a component directory (containing component.yaml)
kdeps package ./my-component

# Output: my-component-1.0.0.komponent
```

The resulting archive can be:
- Shared with other developers
- Stored in a component registry
- Used as a dependency in agency workflows (extracted at runtime)

### Archive Contents

```
my-component-1.0.0.komponent
├── component.yaml
├── resources/
│   └── greet.yaml
├── template.html
└── (other data files, scripts, etc.)
```

Hidden files and `.kdepsignore` patterns are respected.

## Environment Variable Auto-Derivation

After component resources are loaded, KDeps automatically scans for environment variable references:

- String fields containing <span v-pre>`{{ env('VAR_NAME') }}`</span>
- `ChatConfig.APIKey` with `env:` prefix
- `MCPConfig.Env` and `PythonConfig.Env` maps
- Any `env('VAR')` expression in resource fields

Detected variables are tracked as required by the component. The parent workflow must provide them via `agentSettings.env` or the runtime environment. This ensures all external dependencies are explicit.

## Example: Greeter Component

**`components/greeter/component.yaml`**
```yaml
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: greeter
  version: "1.0.0"
  targetActionId: greet
interface:
  inputs:
    - name: message
      type: string
      required: true
      description: Greeting message
    - name: recipient
      type: string
      required: false
      default: "World"
resources:
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: greet
    run:
      exec:
        command: "echo '{{ inputs.message }}, {{ inputs.recipient }}!'"
```

**`my-workflow/resources/main.yaml`**

Call the `greeter` component from a workflow resource using `run.component:`:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: main

run:
  component:
    name: greeter
    with:
      message: "Hello"
      recipient: "KDeps"
```

After execution, access the result with `output('main')`.

Running `kdeps run my-workflow/` will automatically load the `greeter` component from `components/greeter/` and make its `greet` action available.

## Best Practices

1. **Keep components focused**: Each component should have a single responsibility
2. **Version your components**: Use semantic versioning in `metadata.version`
3. **Document inputs**: Provide clear `description` fields for each input
4. **Use targetActionId**: Set it to the primary action for easy invocation
5. **Avoid actionId collisions**: Prefix actionIds with the component name (e.g., `greeter-greet`)
6. **Test in isolation**: Package and validate your component before using it in workflows
7. **Minimize env vars**: Declare all required environment dependencies; let auto-derivation detect them
