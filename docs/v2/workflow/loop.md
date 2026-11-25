# While-loop iteration

The [`loop`](/reference/glossary#loop) block gives conditional iteration with no fixed limit. This makes kdeps workflows Turing complete.

<div v-pre>

```yaml
# resources/count.yaml

actionId: countToFive
name: Count to Five
flow:
  loop:
    while: "loop.index() < 5"
    maxIterations: 1000   # safety cap (default: 1000)
after:
  - "{{ set('result', loop.count()) }}"
apiResponse:
  success: true
  response:
    count: "{{ get('result') }}"
```

</div>

## Read next

- [While-loop iteration reference](/reference/workflow/loop) - basic usage, loop context, loop-scoped storage, `loop.results()` - self-referential termination, streaming response, `maxIterations` safety cap, `every:` - repeated scheduled tasks, `at:` - specific dates and times
- [Items iteration](/workflow/items) - Fixed-list iteration
- [Expressions](/workflow/expressions) - Expression syntax
- [Expression functions reference](/reference/expression-functions) - Complete function list
- [Validation and control flow](/workflow/validation) - Skip conditions, preflight checks
