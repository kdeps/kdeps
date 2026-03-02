# Scraper Resource

The Scraper resource extracts text content from 15 source types — web pages, documents,
spreadsheets, images, and structured data files — without requiring external services for most
formats. It can be used as a primary resource or as an [inline resource](../concepts/inline-resources) inside `before` / `after` blocks.

## Basic Usage

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: scrapeWebPage
  name: Scrape Web Page

run:
  scraper:
    type: url
    source: "https://example.com"
    timeoutDuration: 30s
```

---

## Configuration Options

| Option | Type | Description |
|--------|------|-------------|
| `type` | string | **Required.** Content type to scrape (see [Supported Types](#supported-types)). |
| `source` | string | **Required.** URL or file path to scrape. Supports [expressions](../concepts/expressions). |
| `timeoutDuration` | string | Maximum time for URL fetching (e.g. `30s`, `1m`). Default: `30s`. |
| `timeout` | string | Alias for `timeoutDuration`. |
| `ocr` | object | OCR options (only for `type: image`). |
| `ocr.language` | string | Tesseract language code (e.g. `eng`, `deu`). Default: `eng`. |

---

## Supported Types

| Type | Description | External Dependency |
|------|-------------|---------------------|
| `url` | Fetches a web page and extracts its visible text | None |
| `pdf` | Extracts text from a PDF file | `pdftotext` (poppler-utils) preferred; falls back to raw scan |
| `word` | Extracts text from a `.docx` file | None |
| `excel` | Extracts cell values from a `.xlsx` file | None |
| `image` | Runs OCR on an image file | `tesseract` CLI required |
| `text` | Reads a plain-text file as-is | None |
| `html` | Reads a local HTML file and extracts visible text | None |
| `csv` | Reads a CSV file and returns rows as tab-separated text | None |
| `markdown` | Reads a Markdown file and returns plain text (markup stripped) | None |
| `pptx` | Extracts text from a PowerPoint `.pptx` file | None |
| `json` | Reads a JSON file and returns its pretty-printed content | None |
| `xml` | Reads a local XML file and extracts all text nodes | None |
| `odt` | Extracts text from an OpenDocument Text `.odt` file | None |
| `ods` | Extracts text from an OpenDocument Spreadsheet `.ods` file | None |
| `odp` | Extracts text from an OpenDocument Presentation `.odp` file | None |

---

## Examples by Type

### URL

Fetches a web page and strips HTML tags, scripts, and styles, returning plain visible text.

```yaml
run:
  scraper:
    type: url
    source: "https://example.com/page"
    timeoutDuration: 15s
```

### PDF

Extracts text from a PDF file. Uses `pdftotext` (from poppler-utils) when available; otherwise
falls back to a raw ASCII scan of the PDF binary.

```yaml
run:
  scraper:
    type: pdf
    source: /data/report.pdf
```

### Word Document

Extracts text from a Word `.docx` file by parsing its internal XML.

```yaml
run:
  scraper:
    type: word
    source: /data/contract.docx
```

### Excel Spreadsheet

Extracts cell values from an Excel `.xlsx` file. Each row is returned as a tab-separated line, with rows separated by newlines (tabs and newlines are preserved in the output).

```yaml
run:
  scraper:
    type: excel
    source: /data/budget.xlsx
```

### Image OCR

Runs Tesseract OCR on an image to extract text. Requires the `tesseract` CLI to be installed.

```yaml
run:
  scraper:
    type: image
    source: /data/scanned-invoice.png
    ocr:
      language: eng     # Tesseract language code; default: eng
```

**Supported image formats:** PNG, JPEG, TIFF, BMP, and any other format that Tesseract accepts.

### Plain Text

Reads a plain-text file and returns its content as-is.

```yaml
run:
  scraper:
    type: text
    source: /data/notes.txt
```

### HTML File

Reads a local HTML file and returns its visible text (scripts, styles, and tags removed).

```yaml
run:
  scraper:
    type: html
    source: /data/page.html
```

### CSV

Reads a CSV file and returns each row as a tab-separated line.

```yaml
run:
  scraper:
    type: csv
    source: /data/records.csv
```

### Markdown

Reads a Markdown file and returns plain text with most markup (headers, bold, links) stripped.

```yaml
run:
  scraper:
    type: markdown
    source: /data/README.md
```

### PowerPoint

Extracts text from the slides of a `.pptx` file.

```yaml
run:
  scraper:
    type: pptx
    source: /data/presentation.pptx
```

### JSON

Reads a JSON file, validates it, and returns its pretty-printed content.

```yaml
run:
  scraper:
    type: json
    source: /data/config.json
```

### XML

Reads a local XML file and concatenates all text node content.

```yaml
run:
  scraper:
    type: xml
    source: /data/feed.xml
```

### OpenDocument Text (ODT)

Extracts text from a LibreOffice/OpenOffice Writer `.odt` file.

```yaml
run:
  scraper:
    type: odt
    source: /data/document.odt
```

### OpenDocument Spreadsheet (ODS)

Extracts cell text from a LibreOffice/OpenOffice Calc `.ods` file.

```yaml
run:
  scraper:
    type: ods
    source: /data/spreadsheet.ods
```

### OpenDocument Presentation (ODP)

Extracts slide text from a LibreOffice/OpenOffice Impress `.odp` file.

```yaml
run:
  scraper:
    type: odp
    source: /data/slides.odp
```

---

## Accessing the Result

The scraper stores its result under the resource's `actionId`. Use `get()` in downstream
resources to access the extracted content.

<div v-pre>

```yaml
# Scrape the page
metadata:
  actionId: fetchPage
run:
  scraper:
    type: url
    source: "https://example.com"

---

# Use the content in an LLM prompt
metadata:
  actionId: summarize
  requires:
    - fetchPage
run:
  chat:
    model: llama3.2:1b
    prompt: "Summarize this page: {{ get('fetchPage') }}"
```

</div>

The result map returned by the scraper contains:

| Key | Type | Description |
|-----|------|-------------|
| `content` | string | The extracted text. |
| `source` | string | The evaluated source URL or path. |
| `type` | string | The scraper type used. |
| `success` | bool | `true` if extraction succeeded. |

Access individual fields with `get('actionId', 'content')` or the full map with `get('actionId')`.

<div v-pre>

```yaml
run:
  expr:
    - set('pageText', get('fetchPage', 'content'))
    - set('didSucceed', get('fetchPage', 'success'))
```

</div>

---

## Dynamic Sources with Expressions

The `source` field supports [expressions](../concepts/expressions), so you can build file paths
or URLs at runtime.

<div v-pre>

```yaml
run:
  scraper:
    type: url
    source: "{{ get('baseUrl') }}/page/{{ get('pageId') }}"
```

</div>

---

## Using Scraper as an Inline Resource

The scraper can be embedded inside `before` / `after` blocks of any resource:

```yaml
run:
  before:
    - scraper:
        type: text
        source: /data/prompt_prefix.txt
  chat:
    model: llama3.2:1b
    prompt: "Context loaded. Answer the query."
```

---

## External Dependencies

| Type | Requirement |
|------|-------------|
| `image` | `tesseract` CLI (install: `apt install tesseract-ocr` or `brew install tesseract`) |
| `pdf` | `pdftotext` from `poppler-utils` (optional, but improves quality). Install: `apt install poppler-utils` or `brew install poppler` |

All other types use Go standard library only and have no external dependencies.

---

## Error Handling

When scraping fails, the error is propagated to the engine. The engine returns early and no
output is stored unless `onError.action: continue` is configured. Use `onError` to control
behavior:

```yaml
run:
  scraper:
    type: url
    source: "https://example.com"
  onError:
    action: continue     # continue, fail (default), or retry
    fallback: ""         # Value to use when action is "continue"
```

---

## Next Steps

- [Expressions](../concepts/expressions) - Dynamic values in `source`
- [Inline Resources](../concepts/inline-resources) - Embed scrapers in `before`/`after` blocks
- [LLM Resource](llm) - Feed scraped content into LLM prompts
- [Exec Resource](exec) - Run shell commands to pre-process files before scraping
