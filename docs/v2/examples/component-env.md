# Scope environment variables per component

Build two components that read the same environment variable name. Each
component can have a different value, and the global variable does not change.
You also see the `.env` file that kdeps makes for each component on the first
run.

Each resource calls one component:

```yaml
# resources/01-translate.yaml
actionId: runTranslate
name: Run translator
component:
  name: translator
  with:
    text: "hello"
```

## Read next

- [Full walkthrough](/reference/examples/component-env) - every step, file and command
- [Reusable component tutorial](/examples/custom-component) - building a component
- [Jinja2 templates](/workflow/templates) - `env()` in YAML preprocessing
