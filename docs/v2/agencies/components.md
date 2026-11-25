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

- [Components reference](/reference/agencies/components) - every key, option and detail
- [Agencies](/agencies/) - agent-to-agent call pattern
