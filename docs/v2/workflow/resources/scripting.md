# Scripting resources

Two resources for running code kdeps has no native resource for. Each has its
own reference page.

Works in workflow and agent mode.

| Resource | Runs | Output | Reference |
| :--- | :--- | :--- | :--- |
| `component: python` | A Python script | stdout, parsed as JSON | [Python](/workflow/resources/python) |
| `component: exec` | A shell command | structured result map | [exec](/workflow/resources/exec) |

## See also

- [Python data processing tutorial](/examples/python-processing)
- [Shell command API tutorial](/examples/shell-command-api)
- [Inline resources](/workflow/inline-resources) - `component:` in `before:` / `after:`
- [Error handling (onError)](/workflow/error-handling) - retry and fallback on a failed script
- [Resources overview](/workflow/resources) - all resource types
