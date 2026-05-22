# Components Reference

Full schema, lifecycle, and packaging reference for kdeps components. For an introduction, see [Components](/concepts/components).

## component.yaml Reference

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Component
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
    actionId: greet
    # ... resource definition
```

### Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `apiVersion` | string | yes | Must be `kdeps.io/v1` |
| `kind` | string | yes | Must be `Component` |
| `name` | string | yes | Unique component name within the workflow |
| `version` | string | no | Component version for packaging |
| [`targetActionId`](/reference/glossary#targetactionid) | string | no | Default resource [`actionId`](/reference/glossary#actionid) when component is invoked |
| `setup.pythonPackages` | `[]string` | no | Python packages installed via `uv pip install` |
| `setup.osPackages` | `[]string` | no | OS packages installed via system package manager |
| `setup.commands` | `[]string` | no | Shell commands run after package installs |
| `teardown.commands` | `[]string` | no | Shell commands run after every component invocation |
| `interface.inputs[].name` | string | yes | Input parameter name |
| `interface.inputs[].type` | enum | yes | Data type: `string`, `integer`, `number`, `boolean` |
| `interface.inputs[].required` | bool | no | Whether input is required (default: false) |
| `interface.inputs[].description` | string | no | Human-readable description |
| `interface.inputs[].default` | any | no | Default value (only meaningful when required=false) |
| `resources` | array | no | Inline resource definitions (same as workflow resources) |

## Dependency Lifecycle: `setup` and `teardown`

### `setup` Block

```yaml
# workflow.yaml
setup:
  pythonPackages:     # Python packages installed into the workflow venv via uv
    - requests
    - beautifulsoup4
  osPackages:         # OS packages installed via apt-get / apk / brew
    - wkhtmltopdf
  commands:           # Shell commands run after package installation
    - "playwright install chromium"
```

**Behaviour:**
- `setup` runs **once per component per engine lifetime** (cached -- subsequent calls are no-ops).
- `pythonPackages` are installed via `uv pip install`. Already-present packages are skipped.
- `osPackages` are installed via the detected system package manager (apk on Alpine, apt-get on Debian/Ubuntu, brew on macOS). If no supported package manager is found, a warning is logged and execution continues.
- `commands` run in order after package installs. A non-zero exit terminates setup with an error.

### `teardown` Block

```yaml
# workflow.yaml
teardown:
  commands:           # Shell commands run after component resources finish
    - "rm -rf /tmp/mycomponent-*"
```

**Behaviour:**
- `teardown.commands` run after **every invocation** of the component (not cached).
- Errors in teardown commands are logged as warnings but do not propagate -- teardown is best-effort.

### Deprecated: Top-level `pythonPackages`

The top-level `pythonPackages:` field on `Component` is deprecated. Prefer `setup.pythonPackages:` for new components.

## Interface and Inputs

The `interface` section defines the component's public contract. Inputs behave like function arguments:

```yaml
# resources/example.yaml
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

Inside component resources, reference inputs:

```yaml
# resources/example.yaml
resources:
  - apiVersion: kdeps.io/v1
    actionId: greet
    chat:
      prompt: "{{ inputs.message }}"
```

## Input Validation

When a resource calls a component, kdeps validates `with:` against the interface:

| Condition | Behaviour |
|---|---|
| Required input missing | Error -- execution stops |
| Unknown key in `with:` | Warning logged; key is ignored |
| Optional input omitted | Component default value is applied |

## Input Scoping

Inputs from `with:` are injected under **two** scoped keys, so the same component can be called twice with different inputs:

| Key pattern | Example |
|---|---|
| `<callerActionId>.<inputName>` | `fetch-cv.url` |
| `<componentName>.<inputName>` | `scraper.url` |

## Accessing Component Output

After `component:` executes, results are stored under the caller resource's `actionId`:

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch-article
component:
  name: scraper
  with:
    url: "https://example.com/article"
    selector: ".content"

---
actionId: summarize
requires: [fetch-article]
chat:
  prompt: "Summarize: {{ output('fetch-article').content }}"
```

</div>

## Calling the Same Component Twice

Inputs are scoped to the caller's `actionId`, so the same component works with different configurations:

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch-jd
component:
  name: scraper
  with:
    url: "{{ get('jd_url') }}"

actionId: fetch-company
component:
  name: scraper
  with:
    url: "{{ get('company_url') }}"
    timeout: 60
```

</div>

`output('fetch-jd')` and `output('fetch-company')` are independent.

## Resources

Component resources can be defined inline in `component.yaml` or in a `resources/` subdirectory. The `actionId` must be unique across the entire workflow. Components cannot contain `settings`.

## Auto-Discovery

At parse time, kdeps scans for `components/<name>/component.yaml` files in the same directory as the workflow. All component resources are prepended to the workflow's resource list.

- No explicit declaration in `workflow.yaml` is needed
- Local workflow resources override component resources on `actionId` collision
- Component loading is recursive (components can declare sub-components)

### Resolution Order

1. Workflow's inline resources
2. Resources from auto-discovered components (alphabetical by component name)
3. Local workflow `resources/` directory

## Installing from the Registry

```bash
kdeps registry install scraper
kdeps registry install search
kdeps registry install embedding
kdeps registry install browser
# ... and more
```

Installed components are placed in `components/` and auto-discovered.

```bash
kdeps registry list              # List installed components
kdeps registry uninstall scraper # Uninstall a component
```

## Components as LLM Tools

Installed components can be exposed as LLM tools via `componentTools:` on `chat:` resources:

```yaml
# resources/example.yaml
chat:
  prompt: "Research {{ get('q') }} and summarize."
  componentTools:
    - scraper
    - search
```

The component's `interface.inputs` become the tool's parameter schema. Rules:
- `componentTools:` absent or empty -- no components are registered (default).
- Names not installed are silently ignored.
- Explicit `tools:` entries take precedence over `componentTools:` entries.

| Priority | Source |
|---|---|
| 1 (highest) | Explicit `tools:` in the resource YAML |
| 2 | `componentTools:` allowlist |

## Packaging (.komponent)

Create a portable `.komponent` archive:

```bash
kdeps bundle package ./my-component
# Output: my-component-1.0.0.komponent
```

Archive contents:
```
my-component-1.0.0.komponent
├── component.yaml
├── resources/
├── template.html
└── (other data files, scripts, etc.)
```

## Environment Variable Auto-Derivation

When a component executes, kdeps checks for a component-scoped env var before falling back to the plain name:

```
{COMPONENT_NAME_UPPER}_{VAR_NAME}
```

| Component name | `env('OPENAI_API_KEY')` checks first | then falls back to |
|---|---|---|
| `scraper` | `SCRAPER_OPENAI_API_KEY` | `OPENAI_API_KEY` |
| `my-bot` | `MY_BOT_TELEGRAM_TOKEN` | `TELEGRAM_TOKEN` |

### `.env` File Support

Components auto-load a `.env` file from their directory as a lowest-priority fallback. Resolution order:

1. `{COMPONENT_PREFIX}_{VAR}` in the process env (scoped override)
2. Plain `{VAR}` in the process env
3. Value from the component's `.env` file

```
components/
  scraper/
    component.yaml
    .env            ← auto-loaded when scraper runs
```

Example `.env`:
```bash
OPENAI_API_KEY=sk-my-key
SCRAPER_TIMEOUT=30
```

### Auto-Scaffolded Files

When a component runs for the first time, kdeps auto-creates these files if absent:
- **`.env`** -- template listing all `env()` variables found in resources, with empty values
- **`README.md`** -- generated from `component.yaml` metadata

Existing files are never overwritten.

### `kdeps registry update`

Scaffold or merge `.env` and `README.md` without running the component:

```bash
kdeps registry update ./components/scraper
```

- If `.env` does not exist: full template created with all detected `env()` vars.
- If `.env` already exists: only missing vars appended. Existing values preserved.
- `README.md` is created from metadata only when absent.

## Complete Example: Scraper Component

<div v-pre>

```yaml
# resources/scrape-page.yaml
actionId: scrape-page
name: Scrape Article
component:
  name: scraper
  with:
    url: "https://news.example.com/article"
    selector: ".article-body"
    timeout: 30

---
# resources/summarize.yaml
actionId: summarize
name: Summarize Article
requires:
  - scrape-page
chat:
  prompt: "Summarize the following article in 3 bullet points:\n\n{{ output('scrape-page').content }}"

---
# resources/respond.yaml
actionId: respond
name: Return Summary
requires:
  - summarize
apiResponse:
  success: true
  response:
    summary: "{{ output('summarize') }}"
```

</div>

## Example: Greeter Component

**`components/greeter/component.yaml`**

<div v-pre>

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Component
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
    actionId: greet
    exec:
      command: "echo '{{ inputs.message }}, {{ inputs.recipient }}!'"
```

</div>

**Workflow resource calling the component:**

```yaml
# resources/main.yaml
actionId: main
component:
  name: greeter
  with:
    message: "Hello"
    recipient: "KDeps"
```

## Best Practices

1. **Keep components focused**: single responsibility per component
2. **Version your components**: use semantic versioning in `version`
3. **Document inputs**: provide clear `description` fields
4. **Set targetActionId**: point to the primary action for simple `component:` invocation
5. **Avoid actionId collisions**: prefix with component name
6. **Test in isolation**: package and validate before using in workflows
7. **Minimize env vars**: declare all required environment dependencies; let auto-derivation detect them

## See Also

- [Components Overview](/concepts/components) -- what components are and when to use them
- [Agencies](/concepts/agency) -- agent-to-agent call pattern
- [Expression Functions Reference](/reference/expression-functions-reference) -- `output()`, `get()`, `env()`
- [CLI: Registry Commands](/reference/cli/registry) -- install, list, uninstall components
