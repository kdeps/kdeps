# Components

A component is a reusable, shareable resource bundle. kdeps has two kinds: registry components you install with `kdeps registry install`, and custom components you build yourself in a `components/` directory.

## Overview

Components encapsulate resources, configuration, and dependencies into a single package. Think of them as callable sub-workflows -- you invoke them with `component:` from any resource, pass typed inputs via `with:`, and get structured output back.

## Types of Components

### Registry components (installable)

Pre-built capability extensions distributed as `.komponent` archives. Install once, available to any workflow:

```bash
kdeps registry install scraper     # web/doc text extraction
kdeps registry install search      # web search
kdeps registry install embedding   # vector embeddings
kdeps registry install browser     # browser automation
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

Components you build: a `component.yaml` manifest plus resources in a `components/<name>/` directory. Auto-discovered at run time -- no changes to `workflow.yaml` needed.

```
my-workflow/
├── workflow.yaml
└── components/                  ← auto-discovered
    └── greeter/
        ├── component.yaml       ← component manifest
        └── resources/           ← component-specific resources
```

## How Components Work

1. Place a `component.yaml` in `components/<name>/` or install via `kdeps registry install`
2. At parse time, kdeps scans `components/` and loads all component manifests
3. Component resources are merged with the workflow's own resources
4. Resources invoke components via `component:` with typed inputs in `with:`
5. Inputs are validated against the component's `interface.inputs` declaration
6. Component output is accessed via `output('<callerActionId>')`

Components cannot contain `settings` (no server modes, port bindings) -- they are purely resource bundles.

## Calling a Component

```yaml
# resources/fetch.yaml
actionId: fetch-article
component:
  name: scraper
  with:
    url: "https://example.com/article"
    selector: ".content"
```

After execution, access results via `output('fetch-article')`.

## Components as LLM Tools

Installed components can be exposed as LLM function-calling tools via `componentTools:` on a `chat:` resource. By default, no components are registered -- you opt in explicitly:

```yaml
# resources/example.yaml
chat:
  prompt: "Research {{ get('q') }} and summarize."
  componentTools:
    - scraper
    - search
```

The component's `interface.inputs` become the tool's parameter schema. The LLM uses this to decide when and how to call the tool.

## See Also

- [Components Reference](/reference/components) -- full schema, input validation, env var auto-derivation, packaging
- [Agencies](/concepts/agency) -- agent-to-agent call pattern
- [CLI: Registry Commands](/reference/cli/registry) -- install, list, uninstall components
- [Glossary: component](/reference/glossary#component) -- definition
