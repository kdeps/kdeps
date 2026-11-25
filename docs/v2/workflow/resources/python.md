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

- [python component reference](/reference/resources/python) - configuration, script or script file, packages and virtualenvs, output, environment variables, errors and timeouts
- [Python examples](/reference/python-examples) - data transformation, ML inference, image processing, error handling, debugging
- [exec component](/workflow/resources/exec) - shell command execution
- [Inline resources](/workflow/inline-resources) - `component:` in `before:` / `after:`
- [LLM resource](/workflow/resources/llm) - combine with AI
