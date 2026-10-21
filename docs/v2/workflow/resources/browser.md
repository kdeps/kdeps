# Browser component

The `browser` component drives a real browser (Chromium, Firefox, or WebKit) with [Playwright](https://playwright.dev/): navigate pages, fill forms, run JavaScript, capture screenshots, and keep a logged-in session between calls. It is a kdeps component (YAML, no Go code) from the `components` asset set: call it with a `component:` resource, no install step. A `components/browser/` of your own, or an installed registry component named `browser`, takes its place.

## Requirements

None beyond kdeps itself. Playwright is compiled into the binary; the first launch of an engine downloads its browser (`chromium` about 150 MB) and later runs reuse it. On Linux, Chromium needs system libraries - Playwright's error names the missing library. Set `CHROMIUM_PATH` to use a browser you already have instead of downloading one.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode, the workflow containing it runs as a single callable tool.

## Basic usage

```yaml
# resources/browse.yaml
actionId: captureTitle
name: Capture Page Title
component:
  name: browser
  with:
    engine: chromium
    url: "https://example.com"
    actions:
      - action: evaluate
        script: "document.title"
apiResponse:
  success: true
  response:
    title: "{{ output('captureTitle').actionResults[0].result }}"
```

## Configuration options

| Option | Description | Default |
|--------|-------------|---------|
| `engine` | Browser engine: `chromium`, `firefox`, or `webkit` | `chromium` |
| `url` | Initial URL to navigate to | - |
| `waitFor` | CSS selector to wait for before running actions | - |
| `headless` | Run browser in headless mode | `true` |
| `sessionId` | Reuse a named persistent browser session | - |
| `timeout` | Global action timeout (e.g. `"30s"`) | `30s` |
| `viewport.width` | Browser viewport width in pixels | `1280` |
| `viewport.height` | Browser viewport height in pixels | `720` |
| `userAgent` | Custom User-Agent string for the browser | *(default Mozilla/5.0)* |
| `stealthMode` | Enable anti-bot detection features | `false` |
| `actions` | Ordered list of browser actions | `[]` |

## Action types

15 action types are available: `navigate`, `click`, `fill`, `type`, `upload`, `select`, [`check`](/reference/glossary#check), `uncheck`, `hover`, `scroll`, `press`, `clear`, `evaluate`, `screenshot`, `wait`. See the [Browser Action Types Reference](/reference/browser-actions) for the complete field-level documentation of each action.

## Stealth mode

Enable `stealthMode: true` to evade bot detection on websites like LinkedIn that block headless browsers. Stealth mode configures the browser with anti-detection settings including:

- Disables `AutomationControlled` blink features
- Adds realistic viewport, locale (`en-US`), and timezone (`America/New_York`)
- Removes automation flags (`--disable-blink-features=AutomationControlled`)
- Sets a realistic User-Agent string

```yaml
# resources/example.yaml
component:
  name: browser
  with:
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

## Persistent sessions

By default each call runs in a fresh browser. Set `sessionId` to keep the session between calls: cookies and local storage are saved to `~/.kdeps/browser-sessions/<sessionId>.json` and the current page's URL beside it. The next call with that `sessionId` starts from the saved storage and, when it sets no `url`, reopens the last page. Each call starts its own browser, so unsaved form input and in-page JavaScript state do not carry over - only what the site stores in cookies or local storage.

<div v-pre>

```yaml
# Resource 1 - log in and save the session
component:
  name: browser
  with:
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

# Resource 2 - reuse the authenticated session
component:
  name: browser
  with:
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

## Stealth mode login (bot detection evasion)

For websites that block headless browsers, enable `stealthMode` and consider using non-headless mode to appear more human-like:

```yaml
# resources/linkedin-login.yaml
actionId: linkedinLogin
component:
  name: browser
  with:
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

## Form fill and submit

<div v-pre>

```yaml
# resources/submit-form.yaml
actionId: submitForm
component:
  name: browser
  with:
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

## Screenshot of a dynamic dashboard

```yaml
# resources/dashboard-shot.yaml
actionId: dashboardShot
component:
  name: browser
  with:
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

## Extract JavaScript-rendered data

```yaml
# resources/extract-data.yaml
actionId: extractData
component:
  name: browser
  with:
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

## Multi-step login with persistent session

<div v-pre>

```yaml
# resources/login-step.yaml
actionId: loginStep
component:
  name: browser
  with:
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

## File upload

```yaml
# resources/upload-document.yaml
actionId: uploadDocument
component:
  name: browser
  with:
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

```json
{
  "success": true,
  "url": "https://example.com/",
  "title": "Example Domain",
  "sessionId": "",
  "actionResults": [
    {"action": "evaluate", "result": "Example Domain", "success": true}
  ]
}
```

Each action result carries what it did: `value` (fill, type, select), `key` (press), `result` (evaluate), `file` (screenshot), `files` (upload), `waited` (wait), `url` (navigate). Read one with `output('captureTitle').actionResults[0].result`. A failing action fails the call with `browser: action[N] 'click' failed: ...`; add `onError:` to the calling resource to continue. With a `sessionId`, the session is saved either way.

## Supported engines

| Engine | Notes |
|--------|-------|
| `chromium` | Default. Chromium-based (Chrome/Edge compatible). |
| `firefox` | Gecko-based. Good for Firefox-specific testing. |
| `webkit` | WebKit-based. Useful for Safari compatibility testing. |

## How it is built

```text
component: browser -> POST /apis/tools.kdeps.io/v1/tools/browser_ops/call -> {success, url, title, sessionId, actionResults}
```

One `httpClient` POST carries the ten inputs to the `browser_ops` tool in the kdeps API. The tool launches the engine with Playwright's Go library inside the kdeps process - no python, no uv - runs the actions in order, and returns the result as JSON. Copy `components/browser/` into your project to change defaults; the actions themselves are interpreted by the tool.

## See also

- [HTTP client](/workflow/resources/http-client) - simple HTTP requests without a browser
- [Scraper component](/workflow/resources/scraper) - text extraction from already-fetched pages
- [Python](/workflow/resources/python) - Custom Python automation scripts
