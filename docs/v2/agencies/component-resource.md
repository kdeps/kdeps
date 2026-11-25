# Component resource

The `component:` resource calls a reusable component with inputs in `with:` and returns its result.

```yaml
# resources/scrape.yaml
actionId: scrape
name: Extract Page Content
component:
  name: web_scraper             # from the components asset set
  with:
    url: "https://example.com"
```

## Read next

- [Component resource reference](/reference/agencies/component-resource) - every key, option and detail
- [Components](/agencies/components) - registry vs. custom components, how to build and publish one
- [Agent resource](/agencies/agent-resource) - call a full sibling agent instead of a component
