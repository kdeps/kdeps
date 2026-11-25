# Components

A component is a reusable, shareable resource bundle. kdeps has two kinds: registry components you install with `kdeps install`, and custom components you build yourself in a `components/` directory.

```yaml
# resources/component-resource.yaml
component:
  name: web_scraper
  with:
    url: "https://example.com"
```

## Read next

- [Components reference](/reference/agencies/components) - overview, types of components, how components work, calling a component, components as LLM tools, use cases
- [Component schema reference](/reference/agencies/component-reference) - full schema, input validation, env var auto-derivation, packaging
- [Agencies](/agencies/) - agent-to-agent call pattern
- [CLI: registry commands](/reference/registry/cli) - install, list, uninstall components
- [Glossary: component](/reference/glossary#component) - definition
