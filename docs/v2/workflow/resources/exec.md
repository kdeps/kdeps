# exec component

The `exec` component runs a shell command and returns a structured result: `{success, exitCode, stdout, stderr, command, result, timedOut}`.

```yaml
# resources/system-info.yaml
actionId: systemInfo
name: System Information
component:
  name: exec                       # from the components asset set; a components/exec/ of your own wins
  with:
    command: "uname -a && date"    # required; multiline scripts allowed
    timeout: 10s                   # hard stop; the call fails after it, no retry
```

## Read next

- [exec component reference](/reference/resources/exec) - every key, option and detail
- [Python resource](/workflow/resources/python) - complex data processing
- [HTTP client](/workflow/resources/http-client) - API calls (preferred over curl)
