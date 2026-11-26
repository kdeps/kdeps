# Expression helper functions

Expression helpers add safe nested-property access, null coalescing, JSON serialization, URL encoding and conditionals beyond `get()` and `set()`. They are available wherever an expression runs.

```yaml
# resources/example.yaml
after:
  - set('errorMessage', safe(get('apiResponse'), 'error.details.message'))
  - set('items', safe(get('apiResponse'), 'data.items'))
```

## Read next

- [Expression helper functions reference](/reference/workflow/expression-helpers) - every key, option and detail
- [Unified API](/workflow/data-access) - Core get() and set() functions
- [Expressions](/workflow/expressions) - Expression syntax and operators
