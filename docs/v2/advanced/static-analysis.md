# Static Analysis

`kdeps validate` runs static analysis on your workflow beyond schema and syntax checks. This catches logical errors that only become visible at runtime without it.

## What It Checks

### Unreachable Resources

Resources not reachable from `metadata.targetActionId` through the dependency graph produce a warning. A resource is reachable if it is the target or is transitively required by the target via the `metadata.requires` field.

```yaml
# workflow.yaml
metadata:
  targetActionId: response

resources:
  - metadata:
      actionId: response
      requires: [fetch]   # reachable

  - metadata:
      actionId: fetch     # reachable (required by response)

  - metadata:
      actionId: unused    # warning: unreachable from targetActionId
```

Unreachable resources are **warnings** - validation still passes. Use them to find dead code.

### Expression References to Unknown actionIds

`get()`, `output()`, and template <code v-pre>{{ id.field }}</code> patterns that reference a non-existent `actionId` are **errors**.

<div v-pre>

```yaml
chat:
  prompt: "{{ fetchResult.body }}"  # error if 'fetchResult' is not a known actionId
```

```yaml
after:
  - set('val', get('missingStep.output'))  # error: 'missingStep' not found
```

Valid example:

```yaml
resources:
  - metadata:
      actionId: fetchResult
      requires: []
    httpClient:
      method: GET
      url: https://api.example.com/data

  - metadata:
      actionId: response
      requires: [fetchResult]
    chat:
      model: llama3.2:1b
      role: user
      prompt: "Summarize: {{ fetchResult.body }}"  # valid
```

</div>

### Missing Required Component Inputs

When calling a component via `run.component`, all `required: true` inputs declared in the component's `interface.inputs` must be provided in `with`.

```yaml
# component.yaml
interface:
  inputs:
    - name: url
      type: string
      required: true
    - name: selector
      type: string
      required: false
```

```yaml
# workflow.yaml - error: 'url' is required but not in 'with'
component:
  name: my-scraper
  with:
    selector: ".article"
```

```yaml
# correct
component:
  name: my-scraper
  with:
    url: "https://example.com"
    selector: ".article"
```

## Running Static Analysis

Static analysis runs automatically as part of `kdeps validate`:

```bash
kdeps validate workflow.yaml
```

```
Validating workflow: workflow.yaml

- YAML syntax valid
- Schema validation passed
- Business rules validated
- Dependencies resolved
- Expressions valid
- Static analysis passed
  warning: [warning] unused: resource is unreachable from targetActionId

Validation successful!
```

Errors cause validation to fail with a non-zero exit code. Warnings are printed but do not fail validation.

## Suppressing Unreachable Warnings

Remove or wire unused resources into the dependency graph. There is no suppress annotation - treat warnings as cleanup signals.

## See Also

- [Workflow Configuration](../configuration/workflow) - `metadata.requires` and `targetActionId`
- [Expression Functions](../reference/expression-functions-reference) - `get()` and `output()` syntax
- [Components](../concepts/components) - `interface.inputs` definition
