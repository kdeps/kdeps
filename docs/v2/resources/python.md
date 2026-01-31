# Python Resource

The Python resource enables execution of Python scripts for data processing, ML inference, and custom logic.

## Basic Usage

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: pythonResource
  name: Data Processing

run:
  python:
    script: |
      import json
      <span v-pre>data = {{ get('inputData') }}</span>
      result = {"processed": len(data)}
      print(json.dumps(result))
    timeoutDuration: 60s

```

## Configuration Options

```yaml
run:
  python:
    # Script content (inline)
    script: |
      print("Hello, World!")

    # Or script file path
    scriptFile: "./scripts/process.py"

    # Command line arguments
    args:
      - "--input"
      - <span v-pre>"{{ get('input_file') }}"</span>


    # Custom virtual environment name (for isolation)
    # Each venvName creates a separate virtual environment
    # Resources with the same venvName share the same environment
    venvName: "my-project-env"

    # Timeout
    timeoutDuration: 60s
```

## Configuration Options

| Option | Description |
|--------|-------------|
| `script` | Inline Python code to execute. |
| `scriptFile` | Path to a `.py` file to execute. |
| `args` | List of command-line arguments. |
| `venvName` | Name of the virtual environment to use. Defaults to "default". |
| `timeoutDuration` | Maximum time allowed for execution. |

## Inline Scripts

Write Python code directly in YAML:

```yaml
run:
  python:
    script: |
      import json
      import pandas as pd

      # Access input data
      <span v-pre>raw_data = {{ get('httpResource') }}</span>

      # Process with pandas

      df = pd.DataFrame(raw_data)
      summary = df.describe()

      # Output result (must print JSON)
      result = {
          "rows": len(df),
          "columns": list(df.columns),
          "summary": summary.to_dict()
      }
      print(json.dumps(result))
    timeoutDuration: 120s
```

## Script Files

Reference external Python files:

```yaml
run:
  python:
    scriptFile: "./scripts/data_processor.py"
    args:
      - "--mode"
      - "analyze"
      - "--data"
      - <span v-pre>"{{ get('data') }}"</span>
    timeoutDuration: 60s

```

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
metadata:
  actionId: dataScience

run:
  python:
    venvName: "datascience-env"
    script: |
      import pandas as pd
      import numpy as np
      # ...

---
# Resource 2: Web scraping packages
metadata:
  actionId: webScraper

run:
  python:
    venvName: "scraper-env"
    script: |
      import requests
      from bs4 import BeautifulSoup
      # ...
```

## Examples

### Data Transformation

<div v-pre>

```yaml
metadata:
  actionId: transformData
  requires: [fetchData]

run:
  python:
    script: |
      import json
      import pandas as pd

      # Get input from previous resource
      raw_data = {{ get('fetchData') }}

      # Transform with pandas
      df = pd.DataFrame(raw_data)
      df['processed_at'] = pd.Timestamp.now().isoformat()
      df['value_normalized'] = df['value'] / df['value'].max()

      # Group and aggregate
      summary = df.groupby('category').agg({
          'value': ['sum', 'mean', 'count'],
          'value_normalized': 'mean'
      }).reset_index()

      # Output as JSON
      result = {
          "original_count": len(raw_data),
          "processed_count": len(df),
          "summary": summary.to_dict(orient='records')
      }
      print(json.dumps(result))
    timeoutDuration: 60s
```

</div>

### ML Inference

<div v-pre>

```yaml
metadata:
  actionId: mlPredict

run:
  preflightCheck:
    validations:
      - get('features') != ''
    error:
      code: 400
      message: Features are required

  python:
    script: |
      import json
      import numpy as np
      from sklearn.ensemble import RandomForestClassifier
      import joblib

      # Load pre-trained model
      model = joblib.load('/models/classifier.pkl')

      # Get input features
      features = {{ get('features') }}
      X = np.array(features).reshape(1, -1)

      # Predict
      prediction = model.predict(X)[0]
      probability = model.predict_proba(X)[0].max()

      result = {
          "prediction": int(prediction),
          "confidence": float(probability),
          "model_version": "1.0.0"
      }
      print(json.dumps(result))
    timeoutDuration: 30s
```

</div>

### Text Processing

<div v-pre>

```yaml
metadata:
  actionId: textAnalysis

run:
  python:
    script: |
      import json
      import re
      from collections import Counter

      text = """{{ get('text') }}"""

      # Clean text
      words = re.findall(r'\b\w+\b', text.lower())

      # Analysis
      word_count = len(words)
      unique_words = len(set(words))
      word_freq = Counter(words).most_common(10)

      # Sentence count
      sentences = re.split(r'[.!?]+', text)
      sentence_count = len([s for s in sentences if s.strip()])

      result = {
          "word_count": word_count,
          "unique_words": unique_words,
          "sentence_count": sentence_count,
          "avg_words_per_sentence": word_count / max(sentence_count, 1),
          "top_words": [{"word": w, "count": c} for w, c in word_freq]
      }
      print(json.dumps(result))
    timeoutDuration: 30s
```

</div>

### Image Processing

<div v-pre>

```yaml
metadata:
  actionId: imageProcess

run:
  python:
    script: |
      import json
      from PIL import Image
      import base64
      from io import BytesIO

      # Load uploaded image
      image_path = "{{ get('file', 'filepath') }}"
      img = Image.open(image_path)

      # Get metadata
      width, height = img.size
      format = img.format

      # Resize for thumbnail
      img.thumbnail((200, 200))

      # Convert to base64
      buffer = BytesIO()
      img.save(buffer, format='PNG')
      thumbnail_b64 = base64.b64encode(buffer.getvalue()).decode()

      result = {
          "original_size": {"width": width, "height": height},
          "format": format,
          "thumbnail": f"data:image/png;base64,{thumbnail_b64}"
      }
      print(json.dumps(result))
    timeoutDuration: 60s
```

</div>

### API Integration

<div v-pre>

```yaml
metadata:
  actionId: externalApi

run:
  python:
    script: |
      import json
      import requests
      import os

      api_key = os.environ.get('EXTERNAL_API_KEY')
      query = """{{ get('q') }}"""

      response = requests.post(
          'https://api.example.com/analyze',
          headers={'Authorization': f'Bearer {api_key}'},
          json={'query': query},
          timeout=30
      )
      response.raise_for_status()

      result = response.json()
      print(json.dumps(result))
    timeoutDuration: 60s
```

</div>

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
metadata:
  requires: [pythonResource]

run:
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
metadata:
  requires: [pythonResource]

run:
  expr:
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

## Error Handling

Handle errors gracefully:

```python
import json
import sys

try:
    # Your processing logic
    result = process_data(input_data)
    output = {"success": True, "result": result}
except ValueError as e:
    output = {"success": False, "error": str(e), "type": "validation"}
except Exception as e:
    output = {"success": False, "error": str(e), "type": "unknown"}
    sys.exit(1)  # Non-zero exit code indicates failure

print(json.dumps(output))
```

**Note**: Errors written to stderr are accessible via `python.stderr('resourceId')` in other resources.

## Best Practices

1. **Always output JSON** - KDeps parses stdout as JSON
2. **Handle errors gracefully** - Return error info in JSON
3. **Use virtual environments** - Isolate dependencies per project
4. **Set appropriate timeouts** - Prevent runaway scripts
5. **Keep scripts focused** - One task per resource
6. **Use environment variables** - Don't hardcode secrets

## Debugging

Enable debug output:

```python
import json
import sys

# Debug info goes to stderr (not captured as output)
print("Debug: Starting processing...", file=sys.stderr)

result = {"data": "value"}

# Only stdout is captured as the result
print(json.dumps(result))
```

## Next Steps

- [Exec Resource](exec) - Shell command execution
- [LLM Resource](llm) - Combine with AI
- [Workflow Configuration](../configuration/workflow) - Python settings
