# Browser Action Types Reference

Complete reference for all action types available in the `actions:` list of a [`browser:` resource](/resources/browser).

Each item in the `actions` list requires an `action` field that selects the operation.

---

## `navigate`

Navigate to a URL mid-sequence.

```yaml
# resources/example.yaml
- action: navigate
  url: "https://example.com/login"
```

| Field | Description |
|-------|-------------|
| `url` | URL to navigate to. Also accepted via the `value` field. |

---

## `click`

Click on an element.

```yaml
# resources/example.yaml
- action: click
  selector: "#submit-button"
```

| Field | Description |
|-------|-------------|
| `selector` | **Required.** CSS selector of the element to click. |

---

## `fill`

Fill a text input field (replaces existing content atomically).

```yaml
# resources/example.yaml
- action: fill
  selector: "#email"
  value: "user@example.com"
```

| Field | Description |
|-------|-------------|
| `selector` | **Required.** CSS selector of the input field. |
| `value` | Text to fill in. |

---

## `type`

Type text character-by-character (useful for fields with key handlers).

```yaml
# resources/example.yaml
- action: type
  selector: "#search"
  value: "my query"
```

| Field | Description |
|-------|-------------|
| `selector` | **Required.** CSS selector of the element. |
| `value` | Text to type. |

---

## `upload`

Upload one or more local files to a file input element.

```yaml
# resources/example.yaml
- action: upload
  selector: "#file-input"
  files:
    - /tmp/report.pdf
    - /tmp/image.png
```

| Field | Description |
|-------|-------------|
| `selector` | **Required.** CSS selector of the `<input type="file">` element. |
| `files` | **Required.** List of absolute file paths to upload. |

---

## `select`

Choose an option in a `<select>` dropdown by value.

```yaml
# resources/example.yaml
- action: select
  selector: "#country"
  value: "US"
```

| Field | Description |
|-------|-------------|
| `selector` | **Required.** CSS selector of the `<select>` element. |
| `value` | The `value` attribute of the option to select. |

---

## [`check`](/reference/glossary#check)

Check a checkbox or radio button.

```yaml
# resources/example.yaml
- action: check
  selector: "#agree-terms"
```

| Field | Description |
|-------|-------------|
| `selector` | **Required.** CSS selector of the checkbox. |

---

## `uncheck`

Uncheck a checkbox.

```yaml
# resources/example.yaml
- action: uncheck
  selector: "#newsletter"
```

| Field | Description |
|-------|-------------|
| `selector` | **Required.** CSS selector of the checkbox. |

---

## `hover`

Hover the mouse cursor over an element (useful for triggering tooltips / menus).

```yaml
# resources/example.yaml
- action: hover
  selector: ".dropdown-trigger"
```

| Field | Description |
|-------|-------------|
| `selector` | **Required.** CSS selector of the element. |

---

## `scroll`

Scroll the page or scroll a specific element into view.

```yaml
# Scroll the page down by 500 pixels
- action: scroll
  value: "500"

# Scroll a specific element into view
- action: scroll
  selector: "#footer"
```

| Field | Description |
|-------|-------------|
| `selector` | *(Optional)* Scroll this element into view. |
| `value` | Pixel offset for page scrolling when no selector is given. |

---

## `press`

Press a keyboard key, optionally scoped to a focused element.

```yaml
# Press Enter on a focused element
- action: press
  selector: "#search-input"
  key: "Enter"

# Press a global key (no selector)
- action: press
  key: "Escape"
```

| Field | Description |
|-------|-------------|
| `selector` | *(Optional)* Element to press the key on. |
| `key` | Key name (e.g. `Enter`, `Tab`, `Escape`, `ArrowDown`). Also accepted via `value`. |

---

## `clear`

Clear the contents of a text input field.

```yaml
# resources/example.yaml
- action: clear
  selector: "#notes"
```

| Field | Description |
|-------|-------------|
| `selector` | **Required.** CSS selector of the input to clear. |

---

## `evaluate`

Execute a JavaScript expression and capture the return value.

```yaml
# resources/example.yaml
- action: evaluate
  script: "document.title"
```

| Field | Description |
|-------|-------------|
| `script` | **Required.** JavaScript expression or statement to execute. |

The return value is stored in the resource output and accessible via `get()`.

---

## `screenshot`

Capture a screenshot of the full page or a specific element.

```yaml
# Full page screenshot
- action: screenshot
  outputFile: /tmp/page.png
  fullPage: true

# Element screenshot
- action: screenshot
  selector: "#chart"
  outputFile: /tmp/chart.png
```

| Field | Description |
|-------|-------------|
| `outputFile` | File path for the PNG image. Auto-generated under `/tmp/kdeps-browser/` if omitted. |
| `fullPage` | Capture the full scrollable page (`true`/`false`). Default: `false`. |
| `selector` | *(Optional)* Capture only this element. |

---

## `wait`

Pause execution for a duration or until a CSS selector appears.

```yaml
# Wait for a fixed duration
- action: wait
  wait: "500ms"

# Wait until an element is visible
- action: wait
  selector: ".loading-spinner"
```

| Field | Description |
|-------|-------------|
| `wait` | Duration string (e.g. `"500ms"`, `"2s"`) **or** CSS selector. |
| `selector` | CSS selector to wait for when `wait` is not set. |
| `value` | Fallback: duration or selector when neither `wait` nor `selector` is set. |

---

## See Also

- [Browser Resource](/resources/browser) - Configuration, stealth mode, sessions, and examples
- [Scraper Resource](/resources/scraper) - Text extraction from already-fetched pages
