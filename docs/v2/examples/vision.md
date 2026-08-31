# Analyze an uploaded image

*Applies to workflow mode.*

## Overview

In this tutorial you build an API that accepts an image upload and a question,
sends both to a multimodal LLM, and returns a structured description of what is
in the image.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- How to send a multipart form upload with `curl`

By the end you will be able to:

- Accept a file upload on an API route
- Attach the uploaded file to a `chat:` prompt with `files:`
- Read upload metadata with `info('files')` and `info('filetypes')`

## Background

A `chat:` resource can attach files to the prompt. For a vision model, an
attached image is analyzed alongside the text. Vision needs the Ollama backend
- the default llamafile backend is text-only - so this workflow enables Ollama
and pulls a multimodal model.

## Before you start

- kdeps installed (`kdeps --version`).
- [Ollama](https://ollama.com) installed and running.
- A multimodal model pulled: `ollama pull llama3.2-vision` (or `llava`).
- An image file to test with.

## Step 1: create the project

```bash
mkdir image-analyzer
cd image-analyzer
mkdir resources
```

## Step 2: enable Ollama and define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: image-analyzer
  version: "1.0.0"
  targetActionId: visionResponse

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/vision
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
    installOllama: true                 # vision needs the Ollama backend
    env:
      KDEPS_DEFAULT_BACKEND: ollama
```

## Step 3: send the image to the model

Create `resources/vision-llm.yaml`:

<div v-pre>

```yaml
# resources/vision-llm.yaml
actionId: visionLLM
name: Vision LLM
chat:
  model: llama3.2-vision
  role: user
  prompt: "{{ get('q') }}"
  files:
    - "{{ get('file', 'filepath') }}"   # the uploaded file, by form field name
  jsonResponse: true
  jsonResponseKeys:
    - description
    - objects
    - scene
```

</div>

`get('file', 'filepath')` returns the path of the file uploaded under the form
field `file`.

## Step 4: return the analysis

Create `resources/vision-response.yaml`:

<div v-pre>

```yaml
# resources/vision-response.yaml
actionId: visionResponse
name: Vision response
requires: [visionLLM]
validations:
  methods: [POST]
  routes: [/api/v1/vision]
apiResponse:
  success: true
  response:
    query: "{{ get('q') }}"
    analysis: "{{ get('visionLLM') }}"
    file_info:
      filename: "{{ info('files') }}"
      filetype: "{{ info('filetypes') }}"
```

</div>

## Step 5: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

Send an image and a question as a multipart form:

```bash
curl -X POST http://localhost:16395/api/v1/vision \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -F "q=What is in this picture?" \
  -F "file=@./photo.jpg"
```

Response:

```json
{
  "success": true,
  "data": {
    "query": "What is in this picture?",
    "analysis": {
      "description": "A wooden desk with a laptop, a mug, and a potted plant.",
      "objects": ["laptop", "mug", "plant", "desk"],
      "scene": "indoor workspace"
    },
    "file_info": { "filename": "photo.jpg", "filetype": "image/jpeg" }
  }
}
```

## Summary

You built an API that:

- Accepts a multipart image upload
- Attaches the file to the LLM prompt with `files:`
- Forces a structured reply with `jsonResponse` and `jsonResponseKeys`
- Reads upload metadata with `info('files')` and `info('filetypes')`

## Next steps

- [LLM resource](/resources/llm/) - vision, files, streaming, tools
- [Request object](/concepts/request-object) - `request.file()`, `filesByType()`
- [LLM backends](/resources/llm/backends) - Ollama configuration
- [File processor tutorial](/examples/file-processor) - text files instead of images
