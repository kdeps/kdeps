# Handle a file upload

*Applies to workflow mode.*

## Overview

In this short tutorial you build an endpoint that accepts one or more file
uploads and returns their metadata: count, names, MIME types, and the path of
the first file on disk.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- How to send a multipart upload with `curl`

By the end you will be able to:

- Accept `multipart/form-data` on a route
- Read upload metadata with `info('filecount')`, `info('files')`, `info('filetypes')`
- Get an uploaded file's path and type with `get(field, 'filepath')` and `get(field, 'filetype')`

## Background

When a request is `multipart/form-data`, kdeps writes each uploaded file to a
temporary path and exposes it. A resource can then read the file, pass its path
to an `exec:` or `python:` step, or attach it to an LLM prompt.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory and a couple of files to upload.

## Step 1: create the project

```bash
mkdir upload-api
cd upload-api
mkdir resources
```

## Step 2: define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: upload-api
  version: "1.0.0"
  targetActionId: fileProcessor

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/upload
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
```

## Step 3: report the upload

Create `resources/file-processor.yaml`:

<div v-pre>

```yaml
# resources/file-processor.yaml
actionId: fileProcessor
name: File processor
validations:
  methods: [POST]
  routes: [/api/v1/upload]
apiResponse:
  success: true
  response:
    message: "file processed"
    file_count: "{{ info('filecount') }}"
    files: "{{ info('files') }}"
    file_types: "{{ info('filetypes') }}"
    first_file:
      path: "{{ get('file', 'filepath') }}"    # 'file' = the form field name
      type: "{{ get('file', 'filetype') }}"
```

</div>

## Step 4: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

```bash
curl -X POST http://localhost:16395/api/v1/upload \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -F "file=@./report.pdf" \
  -F "file=@./notes.txt"
```

Response:

```json
{
  "success": true,
  "data": {
    "message": "file processed",
    "file_count": 2,
    "files": ["report.pdf", "notes.txt"],
    "file_types": ["application/pdf", "text/plain"],
    "first_file": {
      "path": "/tmp/kdeps-uploads/abc123/report.pdf",
      "type": "application/pdf"
    }
  }
}
```

## Summary

You built an endpoint that:

- Accepts `multipart/form-data`
- Reads upload metadata with `info('filecount')`, `info('files')`, `info('filetypes')`
- Gets a file's path and type with `get(field, 'filepath')` / `get(field, 'filetype')`

## Next steps

- [Data access](/concepts/unified-api#the-request-object) - `request.file()`, `filesByType()`, `filecount()`
- [Image analysis tutorial](/examples/vision) - attach the upload to an LLM
- [Document summarizer tutorial](/examples/file-processor) - the `file` input source
- [Exec resource](/resources/scripting/exec) - process the file with a shell command
