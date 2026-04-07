# file-processor

Demonstrates the **`file` input source** - a single-shot workflow that reads a document from a file and returns a structured AI analysis.

## Structure

```
file-processor/
├── workflow.yaml
└── resources/
    ├── 01-read-file.yaml    # expose file metadata
    ├── 02-summarize.yaml    # LLM summarization
    └── 03-response.yaml     # structured JSON output
```

## Usage

```bash
# Pass a file path via --file flag (highest priority)
kdeps run examples/file-processor --file /path/to/document.txt

# Pipe content via stdin
cat document.txt | kdeps run examples/file-processor

# Use an environment variable
KDEPS_FILE_PATH=/tmp/doc.txt kdeps run examples/file-processor

# Pipe JSON with inline content
echo '{"path":"/tmp/doc.txt","content":"Your text here"}' | kdeps run examples/file-processor
```

## Requirements

- Ollama running locally with `llama3.2` installed (`ollama pull llama3.2`)

## Input Resolution Priority

1. `--file <path>` CLI argument
2. stdin (raw text or JSON `{"path":"...","content":"..."}`)
3. `KDEPS_FILE_PATH` environment variable
4. `input.file.path` config field

## Accessing File Data

Inside resources, use the `input()` expression:

| Expression | Value |
|---|---|
| `input('fileContent')` | The file's text content |
| `input('filePath')` | The source file path |
| `input('content')` | Alias for fileContent |
| `input('path')` | Alias for filePath |

## The `file` Source

```yaml
settings:
  input:
    sources: [file]
    file:
      path: ""   # optional default path; override via --file or stdin
```

The workflow runs **once** and exits after processing the file.

## Sample Output

```json
{
  "file": "/path/to/document.txt",
  "analysis": {
    "title": "Introduction to Machine Learning",
    "summary": "This document covers the fundamentals of ML...",
    "key_points": [
      "Supervised learning uses labeled training data",
      "Neural networks are inspired by the human brain",
      "Regularization prevents overfitting"
    ]
  }
}
```

## Validate

```bash
kdeps validate examples/file-processor
```
