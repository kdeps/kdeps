# browser

> Browser automation via Python Playwright (headless Chromium).

The browser component drives a real Chromium browser to navigate URLs, capture screenshots, or extract text from pages. Use it to automate web interactions, visual testing, or content extraction that requires JavaScript rendering.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `url` | string | yes | — | URL to navigate to |
| `action` | string | no | `navigate` | Action to perform: `navigate`, `screenshot`, or `getText` |
| `selector` | string | no | — | CSS selector for `getText` action |
| `screenshotPath` | string | no | `/tmp/screenshot.png` | Output file path for `screenshot` action |

## Actions

| Action | Description | Output |
|--------|-------------|--------|
| `navigate` | Load the page and return the final URL | Final URL string |
| `screenshot` | Take a full-page screenshot | Path to the saved PNG file |
| `getText` | Extract text from the page or a CSS selector | Extracted text content |

## Outputs

Accessed via `output('<your-action-id>')`:

| Field | Type | Description |
|-------|------|-------------|
| `result` | string | The action result — final URL, screenshot path, or extracted text |

## Usage

Navigate and return the resolved URL:

```yaml
run:
  component:
    name: browser
    with:
      url: "https://example.com"
      action: navigate
  apiResponse:
    success: true
    response:
      finalUrl: "{{ output('loadPage').result }}"
```

Take a screenshot:

```yaml
run:
  component:
    name: browser
    with:
      url: "https://dashboard.example.com"
      action: screenshot
      screenshotPath: /tmp/dashboard.png
```

Extract text using a CSS selector:

```yaml
run:
  component:
    name: browser
    with:
      url: "https://news.ycombinator.com"
      action: getText
      selector: ".titleline > a"
```

## Requirements

- [Playwright for Python](https://playwright.dev/python/) must be installed: `pip install playwright && playwright install chromium`
- Chromium browser binaries must be present (installed via `playwright install chromium`).
- The component runs with `headless=True` — no display required.

## Notes

- For `getText` without a `selector`, the full page text is returned (all visible text, whitespace-normalized).
- Screenshots are always PNG format.
- Playwright launches a new browser and page for each invocation; it does not maintain session state between calls.
- JavaScript-heavy SPAs are fully supported since a real browser is used.

## Automatic Setup

This component automatically installs its dependencies when first used:

```yaml
setup:
  pythonPackages:
    - playwright
  commands:
    - "playwright install chromium"
```

No manual `agentSettings.pythonPackages` declaration is needed.
