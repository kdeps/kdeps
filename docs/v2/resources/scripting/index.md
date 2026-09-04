# Scripting resources

Two resources for running code kdeps has no native resource for. Each has its
own reference page.

*Applies to both workflow mode and agent mode.*

| Resource | Runs | Output | Reference |
| :--- | :--- | :--- | :--- |
| `python:` | A Python script | stdout, parsed as JSON | [Python](/resources/scripting/python) |
| `exec:` | A shell command | stdout, captured as-is | [Exec](/resources/scripting/exec) |

## See also

- [Python data processing tutorial](/examples/python-processing)
- [Shell command API tutorial](/examples/shell-command-api)
- [Inline resources](/concepts/inline-resources) - `python:` / `exec:` in `before:` / `after:`
- [Error handling (onError)](/concepts/error-handling) - retry and fallback on a failed script
- [Resources overview](/resources/overview) - all resource types
