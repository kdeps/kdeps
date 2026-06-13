# Browser Automation Resource

The `browser:` resource drives a real browser (Chromium, Firefox, or WebKit) via [Playwright](https://playwright.dev/). Use it to navigate pages, fill forms, run JavaScript, capture screenshots, and maintain authenticated sessions across resources.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool.

## Basic Usage

```yaml
# resources/browse.yaml
actionId: captureTitle
name: Capture Page Title
browser:
  engine: chromium
  url: "https://example.com"
  actions:
    - action: evaluate
      script: "document.title"
apiResponse:
  success: true
  response:
    title: "{{ get('captureTitle') }}"
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `engine` | Browser engine: `chromium`, `firefox`, or `webkit` | `chromium` |
| `url` | Initial URL to navigate to | — |
| `waitFor` | CSS selector to wait for before running actions | — |
| `headless` | Run browser in headless mode | `true` |
| `sessionId` | Reuse a named persistent browser session | — |
| `timeout` | Global action timeout (e.g. `"30s"`) | `30s` |
| `viewport.width` | Browser viewport width in pixels | `1280` |
| `viewport.height` | Browser viewport height in pixels | `720` |
| `userAgent` | Custom User-Agent string for the browser | *(default Mozilla/5.0)* |
| `stealthMode` | Enable anti-bot detection features | `false` |
| `actions` | Ordered list of browser actions | `[]` |

## Action Types

16 action types are available: `navigate`, `click`, `fill`, `type`, `upload`, `select`, [`check`](/reference/glossary#check), `uncheck`, `hover`, `scroll`, `press`, `clear`, `evaluate`, `screenshot`, `wait`. See the [Browser Action Types Reference](/reference/browser-actions) for the complete field-level documentation of each action.

## Stealth Mode

Enable `stealthMode: true` to evade bot detection on websites like LinkedIn that block headless browsers. Stealth mode configures the browser with anti-detection settings including:

- Disables `AutomationControlled` blink features
- Adds realistic viewport, locale (`en-US`), and timezone (`America/New_York`)
- Removes automation flags (`--disable-blink-features=AutomationControlled`)
- Sets a realistic User-Agent string

```yaml
# resources/example.yaml
browser:
  engine: chromium
  headless: true
  stealthMode: true
  url: "https://www.linkedin.com/login"
  actions:
    - action: fill
      selector: "#username"
      value: "{{ get('email') }}"
```

For sites with sophisticated bot detection, also consider:
- Setting `headless: false` to run with a visible browser window
- Using a custom `userAgent` that matches your OS/browser
- Reusing `sessionId` with pre-authenticated sessions instead of logging in fresh

## Persistent Sessions

By default each resource runs in a fresh, ephemeral browser context. Set `sessionId` to share a browser context across multiple resources or API calls — cookies, local storage, and page state are preserved.

<div v-pre>

```yaml
# Resource 1 – log in and save the session
browser:
  engine: chromium
  url: "https://app.example.com/login"
  sessionId: "user-session"
  actions:
    - action: fill
      selector: "#username"
      value: "{{ get('username') }}"
    - action: fill
      selector: "#password"
      value: "{{ get('password') }}"
    - action: click
      selector: "#login-button"

# Resource 2 – reuse the authenticated session
browser:
  engine: chromium
  sessionId: "user-session"
  actions:
    - action: navigate
      url: "https://app.example.com/dashboard"
    - action: evaluate
      script: "document.querySelector('.user-greeting').textContent"
```

</div>

## Examples

### Stealth Mode Login (Bot Detection Evasion)

For websites that block headless browsers, enable `stealthMode` and consider using non-headless mode to appear more human-like:

```yaml
# resources/linkedin-login.yaml
actionId: linkedinLogin
browser:
    engine: chromium
    headless: true
    stealthMode: true
    sessionId: "linkedin-session"
    url: "https://www.linkedin.com/login"
    waitFor: "#username"
    actions:
      - action: fill
        selector: "#username"
        value: "{{ get('linkedin_email') }}"
      - action: fill
        selector: "#password"
        value: "{{ get('linkedin_password') }}"
      - action: click
        selector: "button[type='submit']"
      - action: wait
        wait: "3000ms"
```

### Form Fill and Submit

<div v-pre>

```yaml
# resources/submit-form.yaml
actionId: submitForm
browser:
    engine: chromium
    url: "https://forms.example.com/contact"
    waitFor: "#name"
    actions:
      - action: fill
        selector: "#name"
        value: "{{ get('contact_name') }}"
      - action: fill
        selector: "#email"
        value: "{{ get('contact_email') }}"
      - action: fill
        selector: "#message"
        value: "{{ get('message') }}"
      - action: click
        selector: "#submit"
      - action: wait
        selector: ".confirmation-message"
      - action: evaluate
        script: "document.querySelector('.confirmation-message').textContent"
  apiResponse:
    success: true
    response:
      confirmation: "{{ get('submitForm') }}"
```

</div>

### Screenshot of a Dynamic Dashboard

```yaml
# resources/dashboard-shot.yaml
actionId: dashboardShot
browser:
    engine: chromium
    url: "https://dashboard.example.com"
    viewport:
      width: 1920
      height: 1080
    actions:
      - action: wait
        selector: ".chart-loaded"
      - action: screenshot
        outputFile: /tmp/dashboard.png
        fullPage: true
  apiResponse:
    success: true
    response:
      screenshot: "/tmp/dashboard.png"
```

### Extract JavaScript-Rendered Data

```yaml
# resources/extract-data.yaml
actionId: extractData
browser:
    engine: chromium
    url: "https://spa.example.com/products"
    waitFor: ".product-list"
    actions:
      - action: evaluate
        script: |
          Array.from(document.querySelectorAll('.product-item')).map(el => ({
            name: el.querySelector('.name').textContent.trim(),
            price: el.querySelector('.price').textContent.trim()
          }))
  apiResponse:
    success: true
    response:
      products: "{{ get('extractData') }}"
```

### Multi-Step Login with Persistent Session

<div v-pre>

```yaml
# resources/login-step.yaml
actionId: loginStep
browser:
    engine: firefox
    url: "https://secure.example.com/login"
    sessionId: "{{ get('session_id') }}"
    actions:
      - action: fill
        selector: "[name=username]"
        value: "{{ get('user') }}"
      - action: fill
        selector: "[name=password]"
        value: "{{ get('pass') }}"
      - action: click
        selector: "[type=submit]"
      - action: wait
        selector: ".dashboard"
      - action: evaluate
        script: "document.cookie"
  apiResponse:
    success: true
    response:
      cookies: "{{ get('loginStep') }}"
```

</div>

### File Upload

```yaml
# resources/upload-document.yaml
actionId: uploadDocument
browser:
    engine: chromium
    url: "https://docs.example.com/upload"
    actions:
      - action: upload
        selector: "#document-input"
        files:
          - /tmp/document.pdf
      - action: click
        selector: "#upload-button"
      - action: wait
        selector: ".upload-success"
  apiResponse:
    success: true
    response:
      status: "uploaded"
```

## Output

After execution the resource output contains the **result of the last `evaluate` action** (or the final page URL when no evaluate action is present).
Access it with `get('actionId')`.

```yaml
# Capture and use document.title
actions:
  - action: evaluate
    script: "document.title"

# In a subsequent resource before block:
before:
  - "set('title', get('captureTitle'))"
```

## Supported Engines

| Engine | Notes |
|--------|-------|
| `chromium` | Default. Chromium-based (Chrome/Edge compatible). |
| `firefox` | Gecko-based. Good for Firefox-specific testing. |
| `webkit` | WebKit-based. Useful for Safari compatibility testing. |

## Installation

Playwright browsers must be installed on the host running kdeps:

```bash
npx playwright install chromium
# or install all browsers:
npx playwright install
```

## See Also

- [HTTP Client](http-client) -- simple HTTP requests without a browser
- [Scraper](scraper) -- text extraction from already-fetched pages
- [Python](python) — Custom Python automation scripts
