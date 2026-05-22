# Python Resource

The `python:` resource runs a Python script and stores its stdout (parsed as JSON) as the resource's output.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-mode). In workflow mode it executes as a DAG step. In agent mode it is auto-registered as a callable tool.

## Complete reference

<div v-pre>

```yaml
python:
  script: |                  # inline Python -- must print JSON to stdout
    import json
    data = {{ get('inputData') }}
    print(json.dumps({"processed": len(data)}))

  scriptFile: "./scripts/process.py"  # alternative: path to a .py file

  args:                      # command-line arguments passed to the script
    - "--input"
    - "{{ get('input_file') }}"

  venvName: "my-project-env" # isolated venv -- resources sharing the same name share packages
  timeout: 60s               # hard stop; non-zero exit code also counts as failure
```

</div>

`script` and `scriptFile` are mutually exclusive. The script must write valid JSON to stdout -- that output becomes `get('actionId')` for downstream resources.

## Inline Scripts

<div v-pre>

```yaml
python:
  script: |
    import json
    import pandas as pd

    raw_data = {{ get('httpResource') }}
    df = pd.DataFrame(raw_data)
    summary = df.describe()

    result = {
        "rows": len(df),
        "columns": list(df.columns),
        "summary": summary.to_dict()
    }
    print(json.dumps(result))
  timeout: 120s
```

</div>

## Script Files

<div v-pre>

```yaml
python:
  scriptFile: "./scripts/data_processor.py"
  args:
    - "--mode"
    - "analyze"
    - "--data"
    - "{{ get('data') }}"
  timeout: 60s
```

</div>

The script receives arguments via `sys.argv`:

```python
# scripts/data_processor.py
import sys
import json
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("--mode", required=True)
parser.add_argument("--data", required=True)
args = parser.parse_args()

data = json.loads(args.data)
# Process data...

result = {"status": "success", "mode": args.mode}
print(json.dumps(result))
```

## Python Packages

Configure Python packages in your workflow:

```yaml
# workflow.yaml
settings:
  agentSettings:
    pythonVersion: "3.12"

    # Option 1: List packages
    pythonPackages:
      - pandas>=2.0
      - numpy
      - scikit-learn
      - requests

    # Option 2: Requirements file
    requirementsFile: "requirements.txt"

    # Option 3: pyproject.toml (for uv)
    pyprojectFile: "pyproject.toml"
    lockFile: "uv.lock"
```

KDeps uses [uv](https://github.com/astral-sh/uv) for fast Python package management (97% smaller than Anaconda).

## Virtual Environment Isolation

Use separate virtual environments for different resources:

```yaml
# Resource 1: Data science packages
actionId: dataScience
python:
  venvName: "datascience-env"
  script: |
    import pandas as pd
    import numpy as np
    # ...

---
# Resource 2: Web scraping packages
actionId: webScraper
python:
  venvName: "scraper-env"
  script: |
    import requests
    from bs4 import BeautifulSoup
    # ...
```

## Output Handling

Python scripts must output JSON to stdout:

```python
import json

result = {
    "status": "success",
    "data": [1, 2, 3]
}

# This is how KDeps captures the output
print(json.dumps(result))
```

Access the output in other resources:

```yaml
requires: [pythonResource]
apiResponse:
  response:
    # Full output
    python_result: get('pythonResource')

    # Specific fields
    status: get('pythonResource').status
    data: get('pythonResource').data
```

## Environment Variables

Access environment variables in scripts:

```yaml
# In workflow.yaml
settings:
  agentSettings:
    env:
      API_KEY: "secret-key"
      DEBUG: "true"
```

```python
# In Python script
import os

api_key = os.environ.get('API_KEY')
debug = os.environ.get('DEBUG') == 'true'
```

## Accessing Output Details

Access stdout, stderr, and exit codes from other resources:

```yaml
requires: [pythonResource]
after:
  # Check if Python script succeeded
  - set('script_success', python.exitCode('pythonResource') == 0)
  - set('error_output', python.stderr('pythonResource'))

apiResponse:
  response:
    output: get('pythonResource')  # stdout (default)
    errors: get('error_output')
    success: get('script_success')
```

See [Unified API](../concepts/unified-api.md#resource-specific-accessors) for details.

## See Also

- [Python Examples](/reference/python-examples) - Data transformation, ML inference, image processing, error handling, debugging
- [Exec Resource](exec) -- shell command execution
- [LLM Resource](llm) -- combine with AI
- [Workflow Configuration](../configuration/workflow) -- Python settings
