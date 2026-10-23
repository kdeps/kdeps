# component-input-source

A sub-workflow designed to be invoked from a parent workflow with a `component:` resource.

It declares `sources: [api]` and has no `settings.apiServer`, so kdeps starts no HTTP server, bot listener, or file reader. All input arrives through the calling resource's `with:` map.

## Structure

```
component-input-source/
├── workflow.yaml          # sources: [api], no apiServer
└── resources/
    └── transform.yaml     # transforms text via Python
```

## What It Does

Takes a `text` input and an optional `style` (`uppercase` | `lowercase` | `title`) and returns the transformed text as JSON.

## Using It From a Parent Workflow

```yaml
# In your parent workflow's resource:
component:
  name: component-input-source
  with:
    text: "hello world"
    style: title
```

Then access the result:
```yaml
"{{ output('myActionId') }}"  # {"original":"hello world","style":"title","result":"Hello World"}
```

## Input Settings

```yaml
settings:
  input:
    sources: [api]   # the only sources are api, bot and file
```

- `sources: [api]` names the input kind; the HTTP server starts only when `settings.apiServer` is set.
- With no `apiServer`, `webServer`, bot or file source, the workflow does not listen on anything; it runs when a parent calls it.
- There is no `component` input source and no `component:` block under `settings.input`; a sub-workflow is identified by the parent's `component:` resource.

## Validate

```bash
kdeps validate examples/component-input-source
```
