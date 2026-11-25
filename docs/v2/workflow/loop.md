# While-loop iteration

The [`loop`](/reference/glossary#loop) block gives conditional iteration with no fixed limit.

```yaml
# resources/count.yaml
flow:
  loop:
    while: "loop.index() < 5"
    maxIterations: 1000   # safety cap
```

## Read next

- [While-loop iteration reference](/reference/workflow/loop) - every key, option and detail
- [Items iteration](/workflow/items) - Fixed-list iteration
- [Expressions](/workflow/expressions) - Expression syntax
