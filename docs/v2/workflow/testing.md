# Evals: test a workflow

`kdeps test` runs the `evals:` cases declared in `workflow.yaml` against the workflow's own router, in-process, with no port bound.

```text
workflow.yaml (evals:) -> kdeps test -> in-process router -> engine -> assertions + judges
```

```yaml
# workflow.yaml
evals:
  - name: health
    request:
      path: /health
    assert:
      status: 200
```

## Read next

- [Evals: test a workflow reference](/reference/workflow/testing) - every key, option and detail
- [Validation and control](/workflow/validation) - per-resource request gates
- [Expressions](/workflow/expressions) - expression syntax used inside resources
