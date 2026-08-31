# Build a document summarizer

*Applies to workflow mode.*

## Overview

In this tutorial you build a single-shot workflow that reads a document from a
file, sends it to a local LLM, and returns a structured JSON summary. It runs
once and exits - no server, no polling.

This tutorial is for developers who have installed kdeps and run the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- How to run a shell command and pipe input

By the end you will be able to:

- Configure the `file` input source
- Read file content in a resource with `input()`
- Chain an LLM resource to a response resource
- Run the workflow three different ways (flag, stdin, env var)

## Background

kdeps workflows usually run as an HTTP API. The `file` input source is the
exception: the workflow reads one file, processes it, prints the result, and
exits. This is the shape you want for a cron job, a CI step, or a
`kdeps run ... | jq` one-liner.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.
- A text file to summarize. Any `.txt` or `.md` file works.

No LLM server is required. The default model runs as a local llamafile,
downloaded on first run.

## Step 1: create the project

```bash
mkdir -p doc-summarizer/resources
cd doc-summarizer
```

## Step 2: configure the file input source

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: doc-summarizer
  version: "1.0.0"
  targetActionId: response      # the resource whose output is returned

settings:
  agentSettings:
    pythonVersion: "3.12"
  input:
    sources: [file]             # read one file, then exit
    file:
      path: ""                  # empty: supply the path at run time
```

With `sources: [file]` the workflow does not start a server. It resolves the
file, runs the resource graph once, and prints the response.

## Step 3: expose the file content

Create `resources/01-read.yaml`:

<div v-pre>

```yaml
# resources/01-read.yaml
actionId: readFile
name: Read file content

exec:
  command: echo "file loaded"   # a no-op action; the real work is the response block
apiResponse:
  success: true
  response:
    path: "{{ input('filePath') }}"
    preview: "{{ input('fileContent') | truncate(200) }}"
```

</div>

`input('fileContent')` is the file's text. `input('filePath')` is its path.
Both are populated by the `file` input source before any resource runs.

## Step 4: summarize with an LLM

Create `resources/02-summarize.yaml`:

<div v-pre>

```yaml
# resources/02-summarize.yaml
actionId: summarize
name: Summarize document
requires: [readFile]            # run after readFile

chat:
  model: llama3.2:1b
  prompt: |
    You are a document analyst. Return a JSON object with exactly these keys:
      - "title": a short descriptive title (max 10 words)
      - "summary": a 2-3 sentence summary
      - "key_points": an array of 3-5 bullet-point strings

    Document:
    {{ input('fileContent') }}

    Return only valid JSON. No markdown, no explanation.
```

</div>

## Step 5: return the structured result

Create `resources/03-response.yaml`:

<div v-pre>

```yaml
# resources/03-response.yaml
actionId: response
name: Final response
requires: [summarize]

apiResponse:
  success: true
  response:
    file: "{{ input('filePath') }}"
    analysis: "{{ get('summarize') }}"   # the LLM output
```

</div>

`get('summarize')` reads the output of the `summarize` resource by its
`actionId`.

## Step 6: validate and run

```bash
kdeps validate .
```

Run it three ways - all equivalent:

```bash
# 1. Pass the path with --file
kdeps run . --file ./notes.txt

# 2. Pipe content on stdin
cat notes.txt | kdeps run .

# 3. Use an environment variable
KDEPS_FILE_PATH=./notes.txt kdeps run .
```

Output:

```json
{
  "file": "./notes.txt",
  "analysis": {
    "title": "Quarterly planning notes",
    "summary": "The team agreed to ship the API in March and defer the mobile client.",
    "key_points": [
      "API ships in March",
      "Mobile client deferred to Q3",
      "Hire one backend engineer"
    ]
  }
}
```

## Summary

You built a single-shot workflow that:

- Uses the `file` input source to run once and exit
- Reads file content with `input('fileContent')`
- Chains three resources with `requires:` and `get()`
- Returns structured JSON

## Next steps

- [Input sources](/concepts/input-sources) - the `api` and `bot` sources
- [Batch processing](/examples/batch-processing) - process many items in one run
- [LLM resource](/resources/llm/) - JSON mode, vision, streaming, tools
- [Unified API](/concepts/unified-api) - `get()`, `set()`, `input()`, `output()`
