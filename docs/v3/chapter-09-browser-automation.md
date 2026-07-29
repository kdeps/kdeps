# Chapter 9: Browser Automation

The `browser:` resource drives a real browser engine via Playwright. Use it when the page you need to interact with renders content dynamically via JavaScript — which is most of the modern web — or when you need to fill forms, click buttons, log in, or capture screenshots.

## Why a Real Browser

Web scrapers that fetch HTML over HTTP see the raw server response, before JavaScript runs. For single-page applications, dashboards, and any site that loads data asynchronously, the raw HTML is empty or nearly so. A real browser executes JavaScript, renders the DOM, handles authentication flows, and gives you the page as the user actually sees it.

kdeps compiles Playwright support directly into its resource execution. You do not need to install a separate Playwright server or manage browser binaries separately — kdeps handles that.

## Basic Usage

```yaml
# resources/capture.yaml
actionId: capture
browser:
  engine: chromium
  url: "https://example.com"
  actions:
    - action: evaluate
      script: "document.title"
```

After execution, `get('capture')` contains the result of the last action (here, the page title as a string).

## Browser Engines

Three engines are available:

```yaml
browser:
  engine: chromium    # Chromium (default, most compatible)
  engine: firefox     # Firefox
  engine: webkit      # WebKit (Safari engine)
```

Use `chromium` unless you have a specific reason to test against a different engine. Some sites render differently in different browsers; if your target site behaves oddly in Chromium, try `webkit`.

## Actions

Actions are the steps the browser takes after loading the URL. They execute sequentially.

### navigate

Load a URL:

```yaml
actions:
  - action: navigate
    url: "&#123;&#123; get('targetUrl') &#125;&#125;"
```

### click

Click an element matching a CSS selector:

```yaml
actions:
  - action: click
    selector: "button.submit"
```

### fill

Fill an input with a value:

```yaml
actions:
  - action: fill
    selector: "input[name='email']"
    value: "&#123;&#123; env('TEST_EMAIL') &#125;&#125;"
  - action: fill
    selector: "input[name='password']"
    value: "&#123;&#123; env('TEST_PASSWORD') &#125;&#125;"
```

### evaluate

Run JavaScript and capture its return value:

```yaml
actions:
  - action: evaluate
    script: "document.querySelector('.price').innerText"
```

The JavaScript is executed in the page context. The return value of the last expression is captured as the resource's output.

For extracting multiple values, return a JSON-serializable object:

```yaml
actions:
  - action: evaluate
    script: |
      JSON.stringify({
        title: document.title,
        price: document.querySelector('.price')?.innerText,
        stock: document.querySelector('.stock-status')?.innerText
      })
```

### screenshot

Capture a screenshot as base64 PNG:

```yaml
actions:
  - action: screenshot
    selector: ".dashboard-widget"    # optional; capture full page if omitted
    fullPage: true
```

The screenshot is stored as a base64-encoded string in the resource's output.

### wait / waitFor

Pause execution for a fixed duration or until a CSS selector appears:

```yaml
# Wait for a fixed duration
- action: wait
  wait: "500ms"

# Wait until an element is visible
- action: wait
  selector: ".results-loaded"

# waitFor is an alias — both forms work
- action: waitFor
  selector: ".results-loaded"
  timeout: 10000    # milliseconds
```

Use `wait` / `waitFor` when you need to wait for dynamic content to render before capturing it.

### select

Select a value in a `<select>` dropdown:

```yaml
actions:
  - action: select
    selector: "select#country"
    value: "NL"
```

### type

Type text character by character (simulates keyboard input, useful for autocomplete fields):

```yaml
actions:
  - action: type
    selector: "input.search-autocomplete"
    value: "Amsterdam"
    delay: 50    # milliseconds between keystrokes
```

### check / uncheck

Check or uncheck a checkbox or radio button:

```yaml
- action: check
  selector: "#agree-terms"

- action: uncheck
  selector: "#newsletter"
```

### hover

Hover the mouse cursor over an element (triggers tooltips and dropdown menus):

```yaml
- action: hover
  selector: ".dropdown-trigger"
```

### scroll

Scroll the page by a pixel offset, or scroll a specific element into view:

```yaml
# Scroll page down 500 pixels
- action: scroll
  value: "500"

# Scroll element into view
- action: scroll
  selector: "#footer"
```

### press

Press a keyboard key, optionally scoped to a focused element:

```yaml
# Press Enter on a specific input
- action: press
  selector: "#search-input"
  key: "Enter"

# Press Escape globally
- action: press
  key: "Escape"
```

Key names: `Enter`, `Tab`, `Escape`, `ArrowDown`, `ArrowUp`, `Backspace`, etc.

### clear

Clear the contents of a text input:

```yaml
- action: clear
  selector: "#notes"
```

### upload

Upload one or more local files to a `<input type="file">` element:

```yaml
- action: upload
  selector: "#file-input"
  files:
    - /tmp/report.pdf
    - /tmp/image.png
```

## Authentication: Logging In

For sites requiring authentication, sequence the login steps before navigating to the protected content:

```yaml
# resources/dashboard.yaml
actionId: dashboard
browser:
  engine: chromium
  url: "https://app.example.com/login"
  actions:
    # Fill login form
    - action: fill
      selector: "input[name='email']"
      value: "&#123;&#123; env('APP_EMAIL') &#125;&#125;"
    - action: fill
      selector: "input[name='password']"
      value: "&#123;&#123; env('APP_PASSWORD') &#125;&#125;"
    - action: click
      selector: "button[type='submit']"
    
    # Wait for redirect to dashboard
    - action: waitFor
      selector: ".dashboard-content"
      timeout: 15000
    
    # Navigate to the specific page we want
    - action: navigate
      url: "https://app.example.com/reports"
    
    # Wait for data to load
    - action: waitFor
      selector: ".report-table"
      timeout: 10000
    
    # Extract the data
    - action: evaluate
      script: |
        const rows = Array.from(document.querySelectorAll('.report-table tr'));
        JSON.stringify(rows.map(row => ({
          date: row.querySelector('.date')?.innerText,
          value: row.querySelector('.value')?.innerText
        })).filter(r => r.date));
```

## Session Persistence

For workflows that make multiple browser calls and need to reuse an authenticated session, configure session storage:

```yaml
browser:
  engine: chromium
  sessionPath: "/data/browser-session.json"    # saved cookies and storage
  url: "https://app.example.com/dashboard"
  actions:
    - action: evaluate
      script: "document.querySelector('.user-name').innerText"
```

On the first run, kdeps saves the browser session (cookies, localStorage) to `sessionPath`. On subsequent runs, it loads the saved session — skipping login if the session is still valid.

## Full Example: Extracting a Dynamic Dashboard

```yaml
# resources/extract-metrics.yaml
actionId: extractMetrics
browser:
  engine: chromium
  headless: true
  url: "https://analytics.internal/dashboard"
  viewport:
    width: 1920
    height: 1080
  actions:
    - action: waitFor
      selector: ".metrics-loaded"
      timeout: 30000
    - action: evaluate
      script: |
        JSON.stringify({
          dau: document.querySelector('[data-metric="dau"]')?.innerText,
          revenue: document.querySelector('[data-metric="revenue"]')?.innerText,
          conversions: document.querySelector('[data-metric="conversions"]')?.innerText,
          timestamp: new Date().toISOString()
        })
```

```yaml
# resources/report.yaml
actionId: report
requires: [extractMetrics]
chat:
  model: llama3.2:1b
  prompt: |
    Generate a daily metrics summary from these numbers:
    DAU: &#123;&#123; get('extractMetrics').dau &#125;&#125;
    Revenue: &#123;&#123; get('extractMetrics').revenue &#125;&#125;
    Conversions: &#123;&#123; get('extractMetrics').conversions &#125;&#125;
    
    Write 3 bullet points highlighting anything notable.
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [report]
apiResponse:
  success: true
  response:
    metrics: get('extractMetrics')
    summary: get('report')
```

## Headless vs. Headed Mode

By default, `browser:` runs headless (no visible browser window). For debugging, you can run headed:

```yaml
browser:
  engine: chromium
  headless: false    # opens a visible browser window
  url: "https://example.com"
```

Headed mode is useful when developing browser automation steps — you can watch what the browser does. Always use headless mode in production.

## Performance Considerations

Browser automation is the most expensive resource type. Each `browser:` resource:
- Starts a browser process
- Loads the target page (JavaScript, assets, API calls)
- Executes your action sequence

This can take 2–30 seconds depending on the page. Design browser resources to extract everything you need in one session rather than making multiple browser resource calls for the same site.

For high-throughput workflows where browser automation is a bottleneck, consider:
- Caching results with `validations.skip` (skip the browser call if we already have recent data)
- Using `scraper:` for static pages (5-10x faster than a full browser render)
- Batching multiple data extractions in a single `evaluate` script

## When to Use Browser vs. Scraper vs. httpClient

| Resource | Use when |
|---|---|
| `httpClient:` | You are calling an API — JSON or XML response, known endpoint |
| `scraper:` | You need page text from a mostly-static HTML page |
| `browser:` | The page requires JavaScript to render, or you need to interact with UI elements |

As a rule: try `scraper:` first. If you get an empty or incomplete response, the page is dynamic — switch to `browser:`.

X> ## Exercise
X>
X> Build a workflow that extracts the current price of a product from an e-commerce page that renders prices via JavaScript (try a public site like books.toscrape.com which is safe for scraping practice).
X>
X> 1. Start with a `scraper:` resource pointing at a product page. Inspect what `get('page')` returns — it will likely be empty or incomplete for a JavaScript-rendered page.
X> 2. Replace or add a `browser:` resource using `engine: chromium`. Add a `waitFor` action waiting for the price element to appear, then an `evaluate` action to extract the price text.
X> 3. Compare the two outputs. Confirm the `browser:` resource returns the price where `scraper:` did not.
X> 4. Add a `screenshot` action after the evaluate to capture the rendered page as a base64 PNG. Include it in the response so you can verify visually what the browser saw.
X>
X> ```bash
X> curl -X POST localhost:16395/api/v1/price \
X>   -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
X>   -H "Content-Type: application/json" \
X>   -d '{"url":"http://books.toscrape.com/catalogue/a-light-in-the-attic_1000/index.html"}'
X> ```
X>
X> **Stretch goal:** Add session persistence so a second request to the same domain reuses the browser session without reloading the page from scratch. Measure the latency difference.
