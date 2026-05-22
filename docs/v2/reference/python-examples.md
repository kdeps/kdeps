# Python Resource Examples

Examples, error handling, and debugging guidance for the [`python:` resource](/resources/python).

## Examples

### Data Transformation

<div v-pre>

```yaml
# resources/transform-data.yaml
actionId: transformData
requires: [fetchData]
python:
  script: |
    import json
    import pandas as pd

    raw_data = {{ get('fetchData') }}
    df = pd.DataFrame(raw_data)
    df['processed_at'] = pd.Timestamp.now().isoformat()
    df['value_normalized'] = df['value'] / df['value'].max()

    summary = df.groupby('category').agg({
        'value': ['sum', 'mean', 'count'],
        'value_normalized': 'mean'
    }).reset_index()

    result = {
        "original_count": len(raw_data),
        "processed_count": len(df),
        "summary": summary.to_dict(orient='records')
    }
    print(json.dumps(result))
  timeout: 60s
```

</div>

### ML Inference

<div v-pre>

```yaml
# resources/ml-predict.yaml
actionId: mlPredict
validations:
  check:
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

    model = joblib.load('/models/classifier.pkl')
    features = {{ get('features') }}
    X = np.array(features).reshape(1, -1)

    prediction = model.predict(X)[0]
    probability = model.predict_proba(X)[0].max()

    result = {
        "prediction": int(prediction),
        "confidence": float(probability),
        "model_version": "1.0.0"
    }
    print(json.dumps(result))
  timeout: 30s
```

</div>

### Text Processing

<div v-pre>

```yaml
# resources/text-analysis.yaml
actionId: textAnalysis
python:
  script: |
    import json
    import re
    from collections import Counter

    text = """{{ get('text') }}"""
    words = re.findall(r'\b\w+\b', text.lower())
    word_count = len(words)
    unique_words = len(set(words))
    word_freq = Counter(words).most_common(10)
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
  timeout: 30s
```

</div>

### Image Processing

<div v-pre>

```yaml
# resources/image-process.yaml
actionId: imageProcess
python:
  script: |
    import json
    from PIL import Image
    import base64
    from io import BytesIO

    image_path = "{{ get('file', 'filepath') }}"
    img = Image.open(image_path)
    width, height = img.size
    fmt = img.format
    img.thumbnail((200, 200))
    buffer = BytesIO()
    img.save(buffer, format='PNG')
    thumbnail_b64 = base64.b64encode(buffer.getvalue()).decode()

    result = {
        "original_size": {"width": width, "height": height},
        "format": fmt,
        "thumbnail": f"data:image/png;base64,{thumbnail_b64}"
    }
    print(json.dumps(result))
  timeout: 60s
```

</div>

### API Integration

<div v-pre>

```yaml
# resources/external-api.yaml
actionId: externalApi
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
    print(json.dumps(response.json()))
  timeout: 60s
```

</div>

## Error Handling

```python
import json
import sys

try:
    result = process_data(input_data)
    output = {"success": True, "result": result}
except ValueError as e:
    output = {"success": False, "error": str(e), "type": "validation"}
except Exception as e:
    output = {"success": False, "error": str(e), "type": "unknown"}
    sys.exit(1)  # Non-zero exit code signals failure to kdeps

print(json.dumps(output))
```

Errors written to stderr are accessible via `python.stderr('resourceId')` in downstream resources.

## Debugging

```python
import json
import sys

# Stderr is not captured as output -- safe for debug logging
print("Debug: Starting processing...", file=sys.stderr)

result = {"data": "value"}
print(json.dumps(result))  # Only stdout is the resource output
```

## Best Practices

- Always output JSON to stdout - kdeps parses it as the resource result
- Handle exceptions and output `{"success": false, "error": "..."}` rather than crashing
- Use `venvName` to isolate dependencies per resource
- Set realistic `timeout` values - the default may not be enough for ML inference
- Store secrets in env vars, never inline in scripts

## See Also

- [Python Resource](/resources/python) - Core configuration reference
- [Exec Resource](/resources/exec) - Shell command execution
- [LLM Resource](/resources/llm) - Combine Python with AI
