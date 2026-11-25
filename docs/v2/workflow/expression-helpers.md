# Expression helper functions

Expression helpers add safe nested-property access, null coalescing, JSON serialization, URL encoding and conditionals beyond `get()` and `set()`. They are available wherever an expression runs.

| Function | Purpose |
|----------|---------|
| `json(data)` | Convert data to JSON string |
| `toJSON(data)` | Alias for `json()` |
| `safe(obj, path)` | Safely access nested properties |
| `debug(obj)` | Format data for debugging |
| `default(value, fallback)` | Null coalescing |
| `urlencode(str)` | URL-encode a string |
| `ternary(cond, trueVal, falseVal)` | Conditional expression |
| `input(key)` | Access request input |
| `output(resourceId)` | Access resource output |

## Read next

- [Expression helper functions reference](/reference/workflow/expression-helpers) - overview, json(), safe(), debug(), default(), toJSON(), urlencode(), ternary()
- [Unified API](/workflow/data-access) - Core get() and set() functions
- [Expressions](/workflow/expressions) - Expression syntax and operators
- [Expression functions reference](/reference/expression-functions) - Complete function reference with array, string, and type operations
- [Items iteration](/workflow/items) - Processing multiple items
