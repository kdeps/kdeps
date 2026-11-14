# Components

A component is a reusable, shareable resource bundle. kdeps has two kinds: registry components you install with `kdeps install`, and custom components you build yourself in a `components/` directory.

Components work in both modes. In workflow mode, a resource invokes a component with `component:`. In agent mode, kdeps registers a component as an LLM tool. It runs only when the model calls it.

## Overview

A component puts resources, configuration, and dependencies into one package. Components are sub-workflows that you can call. Call them with `component:` from a resource, and give typed inputs with `with:`. You get structured output.

## Types of components

### Registry components (installable)

Pre-built extensions in `.komponent` archives. Install them one time, and all workflows can use them:

```bash
kdeps install search      # web search
kdeps install embedding   # vector embeddings
```

Invoke with `component:`:

```yaml
# resources/component-resource.yaml
component:
  name: web_scraper
  with:
    url: "https://example.com"
```

### Custom components (user-defined)

Components you build: a `component.yaml` manifest plus resources in a `components/<name>/` directory. Auto-discovered at run time - no changes to `workflow.yaml` needed.

```
my-workflow/
├── workflow.yaml
└── components/                  <- auto-discovered
    └── greeter/
        ├── component.yaml       <- component manifest
        └── resources/           <- component-specific resources
```

## How components work

1. Put a `component.yaml` in `components/<name>/`, or install a component with `kdeps install`
2. At parse time, kdeps scans `components/` and loads all component manifests
3. kdeps merges component resources with the resources of the workflow
4. Resources call components with `component:` and typed inputs in `with:`
5. kdeps validates inputs against the `interface.inputs` declaration of the component
6. Get the component output with `output('<callerActionId>')`
7. kdeps ships its own components in the `components` asset set: [ocr](/workflow/resources/ocr), [transcribe](/workflow/resources/transcribe), [git](/workflow/resources/git), [file](/workflow/resources/file), [sql](/workflow/resources/sql) (web_search and web_scraper are tools). One loads only when a workflow calls it and has no component of that name itself

A component's resources run in order. kdeps skips a resource if its `skip:` condition (under `validations:`) is true. As a result, a component can select resources from its inputs. A false `check:` condition fails the call with its `error.message`:

```yaml
# components/fetch/component.yaml (resources, abridged)
resources:
  - actionId: viaAPI
    validations:
      skip:
        - "get('fetch.mode') != 'api'"     # inputs are get('<component>.<input>')
    httpClient: {method: GET, url: "{{ get('fetch.url') }}"}
  - actionId: viaCLI
    validations:
      skip:
        - "get('fetch.mode') != 'cli'"
    exec: {command: curl, args: ["-s", "{{ get('fetch.url') }}"]}
```

An input whose value names a connection in `~/.kdeps/config.yaml` declares its kind with `connection:` (`search`, `sql`, `http`). `kdeps run` then asks for a missing connection before the workflow starts, as it does for a resource's `connectionName`:

```yaml
# components/web_search/component.yaml (interface, abridged)
interface:
  inputs:
    - name: connectionName
      type: string
      default: ""
      connection: search   # with: {connectionName: brave} -> search_connections.brave must exist
```

The component's output is the last resource that ran, unless `interface.output` shapes it. kdeps evaluates `output` after the resources run. Strings are expressions. kdeps goes through maps and lists. As a result, the caller always gets fixed field names, for all step results:

<div v-pre>

```yaml
# components/fetch/component.yaml (interface, abridged)
interface:
  inputs:
    - {name: url, type: string, required: true}
  output:                                   # what output('<caller>') returns
    text: "{{ htmlText(output('viaAPI').body, 'article') }}"
    status: "{{ output('viaAPI').statusCode }}"
```

</div>

`kdeps build` installs the `osPackages` listed under `setup:` into the image; at run time the machine's package manager (apk, apt-get, brew) installs them. List the programs they provide under `provides:`. When all are already on `PATH`, no package manager runs.

```yaml
# components/my-tool/component.yaml
setup:
  osPackages: [tesseract-ocr]   # apt/apk package name, installed in images
  provides: [tesseract]         # on PATH already: skip the install
```

A shipped component lists what it needs from other asset sets under `requires:` in its asset header (`template.yaml`, next to `component.yaml`). As a result, `kdeps update components/<name>` installs those first ([how requirements install](/agent/repl-updating)). Every shipped component calls kdeps API tools, so it requires each tool asset it calls:

```yaml
# components/web_search/template.yaml (as shipped)
version: 1.0.3
requires:
  schemas/v1: ">=1.0.3"   # the schema that accepts connection: on an input
  tools/http_request: ""  # the kdeps API tool this component calls
```

Components cannot contain `deploy` (no server modes, port bindings) - they are purely resource bundles.

## Calling a component

```yaml
# resources/fetch.yaml
actionId: fetch-article
name: Fetch Article
component:
  name: web_scraper
  with:
    url: "https://example.com/article"
```

After the run, get the results with `output('fetch-article')`.

## Components as LLM tools

To give installed components to the LLM as function-calling tools, use `tools.components` on a `chat:` resource. By default, kdeps registers no components. You must enable them explicitly:

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

The component's `interface.inputs` become the tool's parameter schema. The LLM uses this to decide when and how to call the tool.

## Use cases

- Pull in a pre-built capability (web scraping, search, embeddings)
  with one `kdeps install` instead of writing it.
- Package logic you reuse across projects - a `component.yaml` plus a
  `components/<name>/` directory - and share it through the registry.
- Give an LLM a bounded, typed capability by exposing a component as a tool
  through `tools.components`.

## See also

- [Components reference](/agencies/component-reference) - full schema, input validation, env var auto-derivation, packaging
- [Agencies](/agencies/) - agent-to-agent call pattern
- [CLI: registry commands](/registry/cli) - install, list, uninstall components
- [Glossary: component](/reference/glossary#component) - definition
