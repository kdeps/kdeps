# PDF Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install pdf`
>
> Usage: `run: { component: { name: pdf, with: { content: "...", outputFile: "/tmp/output.pdf" } } }`

The PDF component generates a PDF file from HTML or plain text content using `pdfkit`.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `content` | string | yes | — | HTML or plain text source to render |
| `outputFile` | string | no | `/tmp/output.pdf` | Destination path for the generated PDF |

## Using the PDF Component

```yaml
run:
  component:
    name: pdf
    with:
      content: "<h1>Hello, World!</h1><p>This is a PDF report.</p>"
      outputFile: "/tmp/report.pdf"
```

Access the result via `output('<callerActionId>')`.

---

## Result Map

| Key | Type | Description |
|-----|------|-------------|
| `success` | bool | `true` if the PDF was generated successfully. |
| `outputFile` | string | Absolute path to the generated PDF file. |

---

## Using Expressions in `content`

The `content` field supports [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: pdf
    with:
      content: |
        <h1>Report for {{ get("name") }}</h1>
        <p>Score: <strong>{{ get("score") }}</strong> / 100</p>
        <p>Generated: {{ info("timestamp") }}</p>
      outputFile: /tmp/report.pdf
```

</div>

---

## Example: CV/JD Match Report

Full pipeline: scrape a job description, score the match, generate a PDF report.

<div v-pre>

```yaml
# resources/scrapeJD.yaml
metadata:
  actionId: scrapeJD
run:
  component:
    name: scraper
    with:
      url: "{{ get('jd_url') }}"

# resources/scoreMatch.yaml
metadata:
  actionId: scoreMatch
  requires: [scrapeJD]
run:
  chat:
    model: gpt-4o
    prompt: |
      CV: {{ get('cv_text') }}
      Job Description: {{ output('scrapeJD').content }}
      Rate the match from 0-100 and explain the top 3 strengths and gaps. Output JSON.

# resources/generateReport.yaml
metadata:
  actionId: generateReport
  requires: [scoreMatch]
run:
  component:
    name: pdf
    with:
      content: |
        <html>
        <head><style>
          body { font-family: Arial, sans-serif; max-width: 800px; margin: auto; padding: 2em; }
          h1 { color: #1a1a2e; }
          .score { font-size: 2em; font-weight: bold; color: #16213e; }
        </style></head>
        <body>
          <h1>Match Report</h1>
          <p class="score">Score: {{ output('scoreMatch').score }} / 100</p>
          <h2>Strengths</h2>
          <ul>{{ output('scoreMatch').strengths }}</ul>
          <h2>Gaps</h2>
          <ul>{{ output('scoreMatch').gaps }}</ul>
        </body>
        </html>
      outputFile: /tmp/match-report.pdf
```

</div>

---

## Example: Motivation Letter

Generate a personalized motivation letter as a PDF using an LLM:

<div v-pre>

```yaml
# resources/writeLetter.yaml
metadata:
  actionId: writeLetter
run:
  chat:
    model: gpt-4o
    prompt: |
      Write a professional motivation letter for this candidate applying to this role.
      CV: {{ get('cv_text') }}
      Job: {{ get('jd_text') }}
      Format as clean HTML with <h1>, <p> tags only.

# resources/letterPDF.yaml
metadata:
  actionId: letterPDF
  requires: [writeLetter]
run:
  component:
    name: pdf
    with:
      content: "{{ output('writeLetter') }}"
      outputFile: /tmp/motivation-letter.pdf
```

</div>

---

## Next Steps

- [Scraper Resource](scraper) - Extract content from web pages and documents
- [LLM Resource](llm) - Generate content with AI models
- [TTS Resource](tts) - Text-to-Speech synthesis
- [API Response](api-response) - Return data to the HTTP caller
