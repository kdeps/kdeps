# Components reference

Full schema, lifecycle, and packaging reference for kdeps components. For an introduction, see [Components](/agencies/components).

Works in workflow and agent mode.

## component.yaml reference

```yaml
# component.yaml
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: greeter              # Required: component name
  description: "A greeting component"
  version: "1.0.0"
  targetActionId: greet      # Optional: default action to invoke
interface:
  inputs:
    - name: message            # Required: input parameter name
      type: string             # Required: string, integer, number, boolean, array, object
      required: true           # Optional (default: false)
      description: "Message to greet with"
      default: "Hello"         # Optional: used when the caller omits this input
resources:
  - actionId: greet
    name: Greet
    # ... resource definition
```

### Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `apiVersion` | string | yes | Must be `kdeps.io/v1` |
| `kind` | string | yes | Must be `Component` |
| `metadata.name` | string | yes | Unique component name within the workflow |
| `metadata.description` | string | no | Human-readable description |
| `metadata.version` | string | no | Component version for packaging |
| [`metadata.targetActionId`](/reference/glossary#targetactionid) | string | no | Default resource [`actionId`](/reference/glossary#actionid) when component is invoked |
| `setup.pythonPackages` | `[]string` | no | Python packages installed via `uv pip install` |
| `setup.osPackages` | `[]string` | no | OS packages installed via system package manager |
| `setup.commands` | `[]string` | no | Shell commands run after package installs |
| `teardown.commands` | `[]string` | no | Shell commands run after every component invocation |
| `interface.inputs[].name` | string | yes | Input parameter name |
| `interface.inputs[].type` | enum | yes | Data type: `string`, `integer`, `number`, `boolean`, `array`, `object`. A `with:` list or map reaches the component as a list or map (strings inside are interpolated); other values arrive as strings |
| `interface.inputs[].required` | bool | no | Whether input is required (default: false) |
| `interface.inputs[].description` | string | no | Human-readable description |
| `interface.inputs[].default` | any | no | Value used when the caller omits the input; a declared default satisfies `required: true` |
| `resources` | array | no | Inline resource definitions (same as workflow resources) |

## Dependency lifecycle: `setup` and `teardown`

### `setup` block

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
- `setup` runs one time for each component during the life of the engine. kdeps keeps the result in a cache, and later calls do nothing.
- kdeps installs `pythonPackages` with `uv pip install`. It skips packages that are already installed.
- kdeps installs `osPackages` with the system package manager that it finds (apk on Alpine, apt-get on Debian/Ubuntu, brew on macOS). If kdeps finds no supported package manager, it logs a warning and continues.
- `commands` run in order after package installs. A non-zero exit terminates setup with an error.

### `teardown` block

```yaml
# workflow.yaml
teardown:
  commands:           # Shell commands run after component resources finish
    - "rm -rf /tmp/mycomponent-*"
```

**Behaviour:**
- `teardown.commands` run after every invocation of the component (not cached).
- kdeps logs errors in teardown commands as warnings. They do not stop the run. Teardown is best-effort.

### Deprecated: top-level `pythonPackages`

Do not use the top-level `pythonPackages:` field on `Component`. It is deprecated. For new components, use `setup.pythonPackages:`.

## Interface and inputs

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

`type` is one of `string`, `integer`, `number`, `boolean`, `array`, `object`. Before the component runs, kdeps converts a scalar `with:` value to its declared type. `limit: 3` arrives as the number `3`, and `dry_run: true` as the boolean `true`, not as strings. If kdeps cannot parse a value, it gives the value with no change. The component or tool then rejects it.

Inside component resources, reference inputs with `input()`:

```yaml
# component.yaml
resources:
  - actionId: greet
    name: Greet
    chat:
      prompt: "{{ input('message') }}"
```

## Input validation

When a resource calls a component, kdeps validates `with:` against the interface:

| Condition | Behaviour |
|---|---|
| Required input omitted, no default | Error - validation fails |
| Input omitted, default declared | The declared default is applied |
| Unknown key in `with:` | Warning logged; key is ignored |
| Scalar declared `integer`, `number` or `boolean` | Converted to that type before the component runs |

## Input scoping

kdeps puts inputs from `with:` under two scoped keys. As a result, you can call the same component two times with different inputs:

| Key pattern | Example |
|---|---|
| `<callerActionId>.<inputName>` | `fetch-cv.url` |
| `<componentName>.<inputName>` | `web_scraper.url` |

## Accessing component output

After `component:` runs, kdeps stores the results under the `actionId` of the caller resource:

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch-article
name: Fetch Article
component:
  name: web_scraper
  with:
    url: "https://example.com/article"
```

```yaml
# resources/summarize.yaml
actionId: summarize
name: Summarize
flow:
  requires: [fetch-article]
chat:
  context:
    prompt: "Summarize: {{ output('fetch-article').content }}"
```

</div>

## Calling the same component twice

The scope of inputs is the `actionId` of the caller. As a result, the same component operates with different configurations:

<div v-pre>

```yaml
# resources/fetch-jd.yaml
actionId: fetch-jd
name: Fetch Job Description
component:
  name: web_scraper
  with:
    url: "{{ get('jd_url') }}"
```

```yaml
# resources/fetch-company.yaml
actionId: fetch-company
name: Fetch Company Page
component:
  name: web_scraper
  with:
    url: "{{ get('company_url') }}"
```

</div>

`output('fetch-jd')` and `output('fetch-company')` are independent.

## Resources

You can define component resources inline in `component.yaml` or in a `resources/` subdirectory. The `actionId` must be unique in the full workflow. Components cannot contain `settings`.

## Auto-discovery

At parse time, kdeps scans for `components/<name>/component.yaml` files in the same directory as the workflow. kdeps adds all component resources to the start of the resource list of the workflow.

- You do not declare them in `workflow.yaml`
- Local workflow resources override component resources on `actionId` collision
- Component loading is recursive (components can declare sub-components)

### Resolution order

1. Workflow's inline resources
2. Resources from auto-discovered components (alphabetical by component name)
3. Local workflow `resources/` directory

## Installing from the registry

```bash
kdeps install scraper-native
kdeps install chatbot
kdeps install embedding-rag
kdeps install ./my-component-1.0.0.komponent
# ... and more
```

kdeps puts installed components in `components/` and finds them automatically.

```bash
kdeps list                       # List installed components
kdeps uninstall scraper-native   # Uninstall a component
```

## Components as LLM tools

To give installed components to the LLM as tools, use `tools.components` on `chat:` resources:

```yaml
# resources/example.yaml
chat:
  context:
    prompt: "Research {{ get('q') }} and summarize."
  tools:
    components:
      - web_scraper
      - web_search
```

The component's `interface.inputs` become the tool's parameter schema. Rules:
- If `tools.components` is not set or is empty, kdeps registers no components (default).
- kdeps silently ignores names that are not installed.
- Explicit `tools.list` entries take precedence over `tools.components` entries.
- The schema does not mark an input with a declared `default` as required. The model can omit it, and the runtime applies the default.

| Priority | Source |
|---|---|
| 1 (highest) | Explicit `tools:` in the resource YAML |
| 2 | `tools.components` allowlist |

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

## Environment variable auto-derivation

When a component executes, kdeps checks for a component-scoped env var before falling back to the plain name:

```
{COMPONENT_NAME_UPPER}_{VAR_NAME}
```

| Component name | `env('OPENAI_API_KEY')` checks first | then falls back to |
|---|---|---|
| `web_scraper` | `WEB_SCRAPER_OPENAI_API_KEY` | `OPENAI_API_KEY` |
| `my-bot` | `MY_BOT_TELEGRAM_TOKEN` | `TELEGRAM_TOKEN` |

### `.env` file support

Components auto-load a `.env` file from their directory as a lowest-priority fallback. Resolution order:

1. `{COMPONENT_PREFIX}_{VAR}` in the process env (scoped override)
2. Plain `{VAR}` in the process env
3. Value from the component's `.env` file

```
components/
  web_scraper/
    component.yaml
    .env            <- auto-loaded when web_scraper runs
```

Example `.env`:
```bash
OPENAI_API_KEY=sk-my-key
WEB_SCRAPER_TIMEOUT=30
```

kdeps only reads a component's `.env`. It never creates or writes files in a component folder. The file holds keys, so keep it out of git (`echo '.env' >> .gitignore`).

## Example: greeter component

**`components/greeter/component.yaml`**

<div v-pre>

```yaml
# components/greeter/component.yaml
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
  - actionId: greet
    name: Greet
    exec:
      command: "echo '{{ input('message') }}, {{ input('recipient') }}!'"
```

</div>

**Workflow resource calling the component:**

```yaml
# resources/main.yaml
actionId: main
name: Call Greeter
component:
  name: greeter
  with:
    message: "Hello"
    recipient: "kdeps"
```

## Best practices

1. **Keep components focused**: single responsibility per component
2. **Version your components**: use semantic versioning in `version`
3. **Document inputs**: provide clear `description` fields
4. **Set targetActionId**: point to the primary action for simple `component:` invocation
5. **Avoid actionId collisions**: prefix with component name
6. **Test in isolation**: package and validate before using in workflows
7. **Minimize env vars**: declare all required environment dependencies. Let auto-derivation detect them

## See also

- [Components overview](/agencies/components) - what components are and when to use them
- [Agencies](/agencies/) - agent-to-agent call pattern
- [Expression functions reference](/reference/expression-functions) - `output()`, `get()`, `env()`
- [CLI: registry commands](/registry/cli) - install, list, uninstall components
