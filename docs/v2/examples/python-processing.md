# Process data with a Python script

Build an API that runs a Python script to validate and
convert data formats (JSON, YAML). It shows the `python` component: how it
receives request data, and how its printed output becomes the resource result.

The resource runs a Python script with a package:

<div v-pre>

```yaml
# resources/process.yaml (part)
component:
  name: python
  with:
    packages:
      - pyyaml                      # installed into the script's environment
    script: |
      import json
      data = """{{ input('data') }}"""
```

</div>

## Read next

- [Full walkthrough](/reference/examples/python-processing) - every step, file and command
- [python component](/workflow/resources/python) - script files, packages, virtualenvs
- [Inline resources tutorial](/examples/inline-resources) - `component:` in `after:`
