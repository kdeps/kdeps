# python component

The `python` component runs a Python script in a managed virtualenv and returns its stdout, parsed as JSON when possible and as raw text otherwise.

```yaml
# resources/analyze.yaml
actionId: analyze
name: Analyze Data
component:
  name: python                     # from the components asset set
  with:
    script: |                      # inline Python; print JSON for structured output
      import json
      print(json.dumps({"rows": 3}))
    timeout: 120s                  # hard stop; the call fails after it, no retry
```

## Read next

- [python component reference](/reference/resources/python) - every key, option and detail
- [exec component](/workflow/resources/exec) - shell command execution
- [Inline resources](/workflow/inline-resources) - `component:` in `before:` / `after:`
