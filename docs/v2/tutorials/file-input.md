# Tutorial: File Input — CLI Document Processor

This tutorial walks through building a workflow that processes file content piped via stdin — or read from a path — and returns an LLM-generated summary. This pattern is useful for document analysis, batch processing, ETL pipelines, and any scenario where you want to drive a KDeps workflow from the command line.

## Prerequisites

- `kdeps` CLI installed (`kdeps version`)
- Ollama installed locally (or `installOllama: true` in `agentSettings`)

---

## How the File Source Works

When `sources: [file]` is configured:

1. KDeps reads content from **stdin** (raw text or JSON `{"path":"…","content":"…"}`).
2. Falls back to the **`KDEPS_FILE_PATH`** environment variable.
3. Falls back to the configured **`input.file.path`** field.
4. If only a path is provided, the file is read from disk.
5. The workflow executes **once** and exits.

Resources access the content via `input("fileContent")` and the path via `input("filePath")`.

---

## Step 1 — Create the Workflow File

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: doc-summarizer
  description: Summarise a document piped via stdin
  version: "1.0.0"
  targetActionId: summarize

settings:
  agentSettings:
    timezone: Etc/UTC
    installOllama: true
    models:
      - llama3.2:3b

  input:
    sources: [file]
    # Optional: default file path when stdin and KDEPS_FILE_PATH are not set
    # file:
    #   path: /tmp/default-document.txt
```

Key points:
- `sources: [file]` enables the file input subsystem.
- `targetActionId: summarize` — the workflow ends by executing the `summarize` resource.
- No API server is started; the process reads input, runs once, and exits.

---

## Step 2 — Create the LLM Resource

```yaml
# resources/summarize.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: summarize
  name: Summarize Document

run:
  chat:
    model: llama3.2:3b
    prompt: |
      You are a concise document summarizer.
      Summarize the following document in 3–5 bullet points:

      {{ input('fileContent') }}
```

The `input('fileContent')` expression injects the file's text content into the LLM prompt. You can also access `input('filePath')` if you need to reference the source path.

---

## Step 3 — Run the Workflow

### Option A — Pipe raw text from stdin

```bash
cat report.txt | ./kdeps run workflow.yaml
```

### Option B — Pipe a JSON object with a file path

The file is read from disk automatically:

```bash
echo '{"path":"/tmp/report.txt"}' | ./kdeps run workflow.yaml
```

### Option C — Pipe a JSON object with inline content

```bash
echo '{"path":"/tmp/report.txt","content":"Q1 revenue exceeded targets by 12%..."}' \
  | ./kdeps run workflow.yaml
```

### Option D — Use an environment variable

```bash
KDEPS_FILE_PATH=/tmp/report.txt ./kdeps run workflow.yaml
```

### Option E — Use the configured default path

Set `input.file.path` in `workflow.yaml` and run without stdin:

```yaml
settings:
  input:
    sources: [file]
    file:
      path: /tmp/report.txt
```

```bash
./kdeps run workflow.yaml
```

---

## Step 4 — Using the File Path in Resources

If you need to reference the source file path (for example, to log it or pass it to another resource):

```yaml
run:
  exec:
    command: echo
    args:
      - "Processing file: {{ input('filePath') }}"
```

---

## Step 5 — Chaining Resources

You can chain multiple resources. The file content flows through the pipeline via `get()`:

```yaml
# resources/extract.yaml
metadata:
  actionId: extract
run:
  exec:
    command: bash
    args:
      - "-c"
      - "echo '{{ input('fileContent') | replace('\n', ' ') }}' | wc -w"
```

```yaml
# resources/summarize.yaml
metadata:
  actionId: summarize
  dependencies: [extract]
run:
  chat:
    model: llama3.2:3b
    prompt: |
      Document word count: {{ get('extract') }}

      Summarize this document:

      {{ input('fileContent') }}
```

---

## Full Working Example

**workflow.yaml:**

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: doc-summarizer
  version: "1.0.0"
  targetActionId: summarize

settings:
  agentSettings:
    installOllama: true
    models:
      - llama3.2:3b
  input:
    sources: [file]
```

**resources/summarize.yaml:**

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: summarize

run:
  chat:
    model: llama3.2:3b
    prompt: |
      Summarize this document in 3 bullet points:

      {{ input('fileContent') }}
```

**Run it:**

```bash
cat /path/to/document.txt | ./kdeps run workflow.yaml
```

---

## Integration with Shell Scripts

The file source is designed for scripting. Here is a shell script that processes every `.txt` file in a directory:

```bash
#!/bin/bash
for f in /docs/*.txt; do
  echo "=== Summarising $f ==="
  KDEPS_FILE_PATH="$f" ./kdeps run workflow.yaml
done
```

---

## See Also

- [Input Sources](../concepts/input-sources.md) — Full reference for all input source types
- [LLM Resource](../resources/llm.md) — Language model configuration
- [Exec Resource](../resources/exec.md) — Shell command execution
- [Bot Tutorial](bot.md) — Chat bot with stateless stdin input (similar single-shot pattern)
