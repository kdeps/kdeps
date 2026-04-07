# pdf

> Generate a PDF file from HTML or plain text content via Python pdfkit.

The pdf component converts HTML or plain text to a PDF file using `pdfkit` (a Python wrapper around `wkhtmltopdf`). Use it to generate reports, invoices, letters, or any document that needs to be distributed as a PDF.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `content` | string | yes | — | HTML or plain text to convert to PDF |
| `outputFile` | string | no | `/tmp/output.pdf` | Path to write the generated PDF |

## Outputs

Accessed via `output('<your-action-id>')`:

| Field | Type | Description |
|-------|------|-------------|
| `generated` | string | Absolute path to the generated PDF file |

## Usage

Generate a simple PDF:

```yaml
run:
  component:
    name: pdf
    with:
      content: "<h1>Invoice #1042</h1><p>Amount due: $500</p>"
      outputFile: /tmp/invoice-1042.pdf
  apiResponse:
    success: true
    response:
      pdfPath: "{{ output('generatePdf').generated }}"
```

Generate a report from a previous step's output:

```yaml
run:
  component:
    name: pdf
    with:
      content: "{{ output('reportStep').html }}"
      outputFile: "/tmp/report-{{ session('id') }}.pdf"
```

Plain text (no HTML):

```yaml
run:
  component:
    name: pdf
    with:
      content: "Dear John,\n\nYour order has shipped.\n\nRegards,\nSupport Team"
```

## Requirements

- Python package `pdfkit`: `pip install pdfkit`
- System dependency `wkhtmltopdf` must be installed and available on `$PATH`:
  - macOS: `brew install wkhtmltopdf`
  - Ubuntu/Debian: `apt-get install wkhtmltopdf`
  - CentOS/RHEL: `yum install wkhtmltopdf`
- Write access to the directory containing `outputFile`.

## Notes

- HTML input is fully supported including CSS styles, images, and tables. External CSS/images must be accessible at runtime.
- For best results with HTML, use absolute URLs for images and stylesheets, or embed them inline.
- The generated PDF path is returned so it can be passed to the `email` component as an attachment path (in a custom send step) or served via a file response.
- Page size defaults to A4 — use wkhtmltopdf options by extending with a `run.exec:` resource if you need custom paper sizes.

## Automatic Setup

This component automatically installs its dependencies when first used:

```yaml
setup:
  pythonPackages:
    - pdfkit
  osPackages:
    - wkhtmltopdf
```

No manual `agentSettings.pythonPackages` declaration is needed. `wkhtmltopdf` is installed via the system package manager (apt-get / apk / brew).
