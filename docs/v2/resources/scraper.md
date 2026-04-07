# Scraper Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install scraper`
>
> Usage: `run: { component: { name: scraper, with: { url: "...", selector: "...", timeout: 30 } } }`

The Scraper component extracts text content from 15 source types — web pages, documents,
spreadsheets, images, and structured data files — without requiring external services for most
formats. The source type is inferred automatically from the `url` input.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | yes | — | URL or file path to scrape |
| `selector` | string | no | — | CSS selector to scope extraction (URL type only) |
| `timeout` | integer | no | `30` | Maximum fetch time in seconds |

## Using the Scraper Component

```yaml
run:
  component:
    name: scraper
    with:
      url: "https://example.com"
      selector: ".article"
      timeout: 15
```

Access the result via `output('<callerActionId>')`:

<div v-pre>

```yaml
metadata:
  actionId: fetch-page
run:
  component:
    name: scraper
    with:
      url: "https://example.com"

---

metadata:
  actionId: summarize
  requires: [fetch-page]
run:
  chat:
    model: llama3.2:1b
    prompt: "Summarize: {{ output('fetch-page').content }}"
```

</div>

---

## Inferred Source Types

The component selects the extraction method from the URL extension or scheme. Explicit `type` selection is not required.

| Type | Inferred from | External Dependency |
|------|---------------|---------------------|
| Web page | `http://` or `https://` URL | None |
| PDF | `.pdf` extension | `pdftotext` (poppler-utils) preferred; falls back to raw scan |
| Word document | `.docx` extension | None |
| Excel spreadsheet | `.xlsx` extension | None |
| Image OCR | `.png`, `.jpg`, `.tiff`, `.bmp` | `tesseract` CLI required |
| Plain text | `.txt` extension | None |
| HTML file | `.html` extension | None |
| CSV | `.csv` extension | None |
| Markdown | `.md` extension | None |
| PowerPoint | `.pptx` extension | None |
| JSON | `.json` extension | None |
| XML | `.xml` extension | None |
| OpenDocument Text | `.odt` extension | None |
| OpenDocument Spreadsheet | `.ods` extension | None |
| OpenDocument Presentation | `.odp` extension | None |

---

## Examples by Type

### Web Page

```yaml
run:
  component:
    name: scraper
    with:
      url: "https://example.com/page"
      timeout: 15
```

### PDF

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/report.pdf
```

### Word Document

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/contract.docx
```

### Excel Spreadsheet

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/budget.xlsx
```

### Image OCR

Requires the `tesseract` CLI. Supported formats: PNG, JPEG, TIFF, BMP.

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/scanned-invoice.png
```

### Plain Text

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/notes.txt
```

### CSV

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/records.csv
```

### Markdown

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/README.md
```

### PowerPoint

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/presentation.pptx
```

### JSON

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/config.json
```

### XML

```yaml
run:
  component:
    name: scraper
    with:
      url: /data/feed.xml
```

---

## Accessing the Result

The scraper stores its result under the resource's `actionId`. Use `output()` in downstream
resources to access the extracted content.

<div v-pre>

```yaml
# Scrape the page
metadata:
  actionId: fetchPage
run:
  component:
    name: scraper
    with:
      url: "https://example.com"

---

# Use the content in an LLM prompt
metadata:
  actionId: summarize
  requires:
    - fetchPage
run:
  chat:
    model: llama3.2:1b
    prompt: "Summarize this page: {{ output('fetchPage').content }}"
```

</div>

The result map returned by the scraper contains:

| Key | Type | Description |
|-----|------|-------------|
| `content` | string | The extracted text. |
| `source` | string | The evaluated source URL or path. |
| `type` | string | The inferred type used. |
| `success` | bool | `true` if extraction succeeded. |

Access individual fields with `output('actionId').content` or the full result with `output('actionId')`.

---

## Dynamic Sources with Expressions

The `url` field supports [expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: scraper
    with:
      url: "{{ get('baseUrl') }}/page/{{ get('pageId') }}"
```

</div>

---

## External Dependencies

| Type | Requirement |
|------|-------------|
| Image OCR | `tesseract` CLI (`apt install tesseract-ocr` or `brew install tesseract`) |
| PDF | `pdftotext` from poppler-utils (optional, improves quality: `apt install poppler-utils` or `brew install poppler`) |

All other types use Go standard library only.

---

## Error Handling

When scraping fails, the error is propagated to the engine. Use `onError` to control behavior:

```yaml
run:
  component:
    name: scraper
    with:
      url: "https://example.com"
  onError:
    action: continue
    fallback: ""
```

---

## Next Steps

- [Expressions](../concepts/expressions) - Dynamic values in `url`
- [LLM Resource](llm) - Feed scraped content into LLM prompts
- [Exec Resource](exec) - Run shell commands to pre-process files before scraping
