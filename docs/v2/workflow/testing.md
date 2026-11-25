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
      path: /health            # method defaults to GET; path is required
    assert:
      status: 200
      body:
        jsonPath:
          - path: $.status
            equals: ok
```

Run `kdeps test` in the workflow folder. It exits non-zero when a case fails.

## Read next

- [Evals: test a workflow reference](/reference/workflow/testing) - declare the cases, assertions, score with judges, auth, what runs before the cases, scope
- [`kdeps test`](/reference/cli-dev#kdeps-test) - flags and exit behavior
- [Validation and control](/workflow/validation) - per-resource request gates
- [Expressions](/workflow/expressions) - expression syntax used inside resources
