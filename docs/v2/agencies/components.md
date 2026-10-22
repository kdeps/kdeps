# Components

A component is a reusable, shareable resource bundle. kdeps has two kinds: registry components you install with `kdeps install`, and custom components you build yourself in a `components/` directory.

Components work in both modes. In workflow mode, a resource invokes a component with `component:`. In agent mode, a component is registered as an LLM tool and runs only when the model calls it.

## Overview

Components encapsulate resources, configuration, and dependencies into a single package. They are callable sub-workflows: you invoke them with `component:` from any resource, pass typed inputs via `with:`, and get structured output back.

## Types of components

### Registry components (installable)

Pre-built capability extensions distributed as `.komponent` archives. Install once, available to any workflow:

```bash
kdeps install search      # web search
kdeps install embedding   # vector embeddings
```

Invoke with `component:`:

```yaml
# resources/component-resource.yaml
component:
  name: scraper
  with:
    url: "https://example.com"
```

### Custom components (user-defined)

Components you build: a `component.yaml` manifest plus resources in a `components/<name>/` directory. Auto-discovered at run time - no changes to `workflow.yaml` needed.

```
my-workflow/
├── workflow.yaml
└── components/                  ← auto-discovered
    └── greeter/
        ├── component.yaml       ← component manifest
        └── resources/           ← component-specific resources
```

## How components work

1. Place a `component.yaml` in `components/<name>/` or install via `kdeps install`
2. At parse time, kdeps scans `components/` and loads all component manifests
3. Component resources are merged with the workflow's own resources
4. Resources invoke components via `component:` with typed inputs in `with:`
5. Inputs are validated against the component's `interface.inputs` declaration
6. Component output is accessed via `output('<callerActionId>')`
7. kdeps ships its own components in the `components` asset set ([ocr](/workflow/resources/ocr), [transcribe](/workflow/resources/transcribe), [git](/workflow/resources/git), [file](/workflow/resources/file), [sql](/workflow/resources/sql), web_search and web_scraper are tools, not components); one is loaded only when a workflow calls it and does not provide a component of that name itself

A component's resources run in order when it is called; a resource with `validations.skip` is skipped when a condition is true, so one component can choose between resources from its inputs, and `validations.check` fails the call with its `error.message` when a condition is false:

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

An input whose value names a connection in `~/.kdeps/config.yaml` declares its kind with `connection:` (`search`, `sql`, `smtp`, `imap`, `http`). `kdeps run` then asks for a missing connection before the workflow starts, as it does for a resource's `connectionName`:

```yaml
# components/web_search/component.yaml (interface, abridged)
interface:
  inputs:
    - name: connectionName
      type: string
      default: ""
      connection: search   # with: {connectionName: brave} -> search_connections.brave must exist
```

The component's output is the last resource that ran, unless `interface.output` shapes it. `output` is evaluated after the resources run: strings are expressions, maps and lists are walked, so the caller gets fixed field names whatever the steps returned:

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

`setup.osPackages` are installed by `kdeps build` in the image and, at run time, by the machine's package manager (apk, apt-get, brew). List the programs they provide under `setup.provides`: when every one is already on `PATH`, no package manager runs - a run on a machine that has the tool costs nothing extra.

```yaml
# components/ocr/component.yaml (as shipped)
setup:
  osPackages: [tesseract-ocr]   # apt/apk package name, installed in images
  provides: [tesseract]         # on PATH already: skip the install
```

A component shipped in the `components` asset set lists what it needs from other asset sets under `requires:` in its asset header (`template.yaml`, next to `component.yaml`), so `kdeps update components/<name>` installs those first ([how requirements install](/agent/repl-updating)). Every shipped component calls kdeps API tools, so it requires each tool asset it calls:

```yaml
# components/web_search/template.yaml (as shipped)
version: 1.0.3
requires:
  schemas/v1: ">=1.0.3"   # the schema that accepts connection: on an input
  tools/http_request: ""  # the kdeps API tool this component calls
```

Components cannot contain `settings` (no server modes, port bindings) - they are purely resource bundles.

## Calling a component

```yaml
# resources/fetch.yaml
actionId: fetch-article
name: Fetch Article
component:
  name: scraper
  with:
    url: "https://example.com/article"
    selector: ".content"
```

After execution, access results via `output('fetch-article')`.

## Components as LLM tools

Installed components can be exposed as LLM function-calling tools via `componentTools:` on a `chat:` resource. By default, no components are registered - you opt in explicitly:

```yaml
# resources/example.yaml
chat:
  prompt: "Research {{ get('q') }} and summarize."
  componentTools:
    - scraper
    - search
```

The component's `interface.inputs` become the tool's parameter schema. The LLM uses this to decide when and how to call the tool.

## Use cases

- Pull in a pre-built capability (web scraping, search, embeddings, browser
  automation) with one `kdeps install` instead of writing it.
- Package logic you reuse across projects - a `component.yaml` plus a
  `components/<name>/` directory - and share it through the registry.
- Give an LLM a bounded, typed capability by exposing a component as a tool
  through `componentTools:`.

## See also

- [Components reference](/agencies/component-reference) - full schema, input validation, env var auto-derivation, packaging
- [Agencies](/agencies/) - agent-to-agent call pattern
- [CLI: registry commands](/registry/cli) - install, list, uninstall components
- [Glossary: component](/reference/glossary#component) - definition
