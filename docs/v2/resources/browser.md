# Browser Automation Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install browser`
>
> Usage: `run: { component: { name: browser, with: { url: "...", action: "navigate", selector: "...", screenshotPath: "..." } } }`

The Browser component enables browser automation via [Playwright](https://playwright.dev/), supporting page navigation, text extraction, and screenshots.

> **Note**: The component exposes three actions: `navigate`, `screenshot`, and `getText`. For advanced automation (form fill, click, evaluate JavaScript, sessions, stealth mode), use a Python resource with `playwright` directly.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | yes | — | URL to navigate to |
| `action` | string | no | `navigate` | Action: `navigate`, `screenshot`, or `getText` |
| `selector` | string | no | — | CSS selector for `getText` action |
| `screenshotPath` | string | no | `/tmp/screenshot.png` | File path for `screenshot` action output |

## Using the Browser Component

**Navigate and get page text:**

```yaml
run:
  component:
    name: browser
    with:
      url: "https://example.com"
      action: getText
```

**Take a screenshot:**

```yaml
run:
  component:
    name: browser
    with:
      url: "https://example.com"
      action: screenshot
      screenshotPath: "/tmp/page.png"
```

**Get text from a specific element:**

```yaml
run:
  component:
    name: browser
    with:
      url: "https://example.com"
      action: getText
      selector: ".article-content"
```

Access the result via `output('<callerActionId>')`.

---

## Actions

| Action | Description |
|--------|-------------|
| `navigate` | Navigate to the URL and return the final page URL |
| `screenshot` | Capture a PNG screenshot of the page |
| `getText` | Extract visible text from the page or from a CSS selector |

---

## Result Map

| Key | Type | Description |
|-----|------|-------------|
| `success` | bool | `true` if the action completed without error. |
| `text` | string | Extracted text (`getText` action). |
| `screenshotPath` | string | Path to saved screenshot (`screenshot` action). |
| `url` | string | Final page URL after navigation. |

---

## Expression Support

All fields support [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: browser
    with:
      url: "{{ get('target_url') }}"
      action: getText
      selector: "{{ get('css_selector') }}"
```

</div>

---

## Full Example: Screenshot Pipeline

<div v-pre>

```yaml
# Step 1: Take a screenshot of a dashboard
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: capture
    name: Capture Dashboard

  run:
    component:
      name: browser
      with:
        url: "{{ get('dashboard_url') }}"
        action: screenshot
        screenshotPath: /tmp/dashboard.png

# Step 2: Send the screenshot by email
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: notify
    name: Send Screenshot
    requires:
      - capture

  run:
    component:
      name: email
      with:
        to: "{{ get('recipient') }}"
        subject: "Dashboard snapshot"
        body: "Screenshot saved at: {{ output('capture').screenshotPath }}"
        smtpHost: "{{ env('SMTP_HOST') }}"
        smtpUser: "{{ env('SMTP_USER') }}"
        smtpPass: "{{ env('SMTP_PASS') }}"
```

</div>

---

## Installation

Playwright browsers must be installed on the host running kdeps:

```bash
npx playwright install chromium
# or install all browsers:
npx playwright install
```

---

## Related Resources

- [HTTP Client](http-client) - Simple HTTP requests without a browser
- [Scraper](scraper) - Text extraction from already-fetched pages
- [Python](python) - Custom Python automation scripts (for advanced Playwright use)
