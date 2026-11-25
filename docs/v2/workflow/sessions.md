# Session configuration

A session keeps values across requests from the same caller: `set('key', val, 'session')`.

```yaml
# workflow.yaml
session:
  type: sqlite    # or memory
  ttl: 30m
```

## Read next

- [Session configuration reference](/reference/workflow/sessions) - every key, option and detail
- [Workflow configuration](/workflow/configuration) - Full settings reference
- [Data access](/workflow/data-access) - get() and set() usage
