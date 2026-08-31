# Scrape a web page and summarize it

*Applies to workflow mode.*

## Overview

In this tutorial you build an API that takes a URL, fetches the page with the
built-in `scraper:` resource, sends the extracted text to a local LLM, and
returns a short summary as JSON.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- How to send a POST request with `curl`
- What a CSS selector is

By the end you will be able to:

- Fetch and extract page text with `scraper:`
- Pass one resource's output to another with `output()`
- Force a JSON reply from the LLM with `jsonResponse`

## Background

The `scraper:` resource fetches a URL and returns its readable text. It runs
in-process - no browser, no external service. For pages that need JavaScript
rendering, use the [`browser:` resource](/resources/web/browser) instead.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.
- Network access.

## Step 1: create the project

```bash
mkdir page-summarizer
cd page-summarizer
mkdir resources
```

## Step 2: define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: page-summarizer
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    portNum: 16402
    routes:
      - path: /summarize
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
```

## Step 3: fetch the page

Create `resources/fetch.yaml`:

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch
name: Fetch page
validations:
  methods: [POST]
  routes: [/summarize]
  check:
    - get('url') != ''
  error:
    code: 400
    message: "url is required"
scraper:
  url: "{{ get('url') }}"
  selector: "{{ get('selector', '') }}"   # optional CSS selector; empty = whole page
  timeout: 30
```

</div>

`output('fetch').content` will hold the extracted text.

## Step 4: summarize with the LLM

Create `resources/summarize.yaml`:

<div v-pre>

```yaml
# resources/summarize.yaml
actionId: summarize
name: Summarize page
requires: [fetch]
chat:
  model: llama3.2:1b
  role: user
  prompt: |
    Summarize the following web page content in 3-5 sentences.

    URL: {{ get('url') }}
    Content:
    {{ output('fetch').content }}
  jsonResponse: true          # force a JSON object as the reply
  jsonResponseKeys:
    - summary                 # the object must contain a "summary" key
  timeout: 60s
```

</div>

## Step 5: return the summary

Create `resources/response.yaml`:

<div v-pre>

```yaml
# resources/response.yaml
actionId: response
name: API response
requires: [summarize]
apiResponse:
  success: true
  response:
    url: "{{ get('url') }}"
    summary: "{{ output('summarize').summary }}"
```

</div>

## Step 6: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run . --dev
```

```bash
curl -X POST http://localhost:16402/summarize \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/doc/"}'
```

Response:

```json
{
  "success": true,
  "data": {
    "url": "https://go.dev/doc/",
    "summary": "The page is the documentation hub for the Go language. It links to the tour, effective Go, and the standard library reference..."
  }
}
```

## Summary

You built an API that:

- Fetches and extracts page text with `scraper:`
- Passes the text to the LLM with `output('fetch').content`
- Forces a structured reply with `jsonResponse` and `jsonResponseKeys`

## Next steps

- [Scraper resource](/resources/web/scraper) - selectors, timeouts, output shape
- [Browser resource](/resources/web/browser) - JavaScript-rendered pages
- [LLM resource](/resources/llm/) - JSON mode, streaming, tools
- [searchWeb](/resources/search/searchweb) - find pages to scrape
