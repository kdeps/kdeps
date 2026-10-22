# Web resources

Three ways to reach the web. Each has its own reference page.

Works in workflow and agent mode.

| Resource | Use it for | Reference |
| :--- | :--- | :--- |
| `component: http_request` | Call a JSON API, webhook, or REST endpoint | [HTTP client](/workflow/resources/http-client) |

## Which one

- **Just data from an API** - the `http_request` component. Built-in auth, retry, and cache.
- **Text from a page** - the `scraper` component. No browser, optional CSS
  selector.
- **The page only works with JavaScript, or you need to click and type** -
  the `browser` component. Real Chromium/Firefox/WebKit via Playwright.

## See also

- [Authenticated API call tutorial](/examples/http-auth) - `http_request` auth, retry, cache
- Web scraper tutorial - the scraper component plus an LLM summary
- [Search resources](/workflow/resources/search) - find pages to fetch
- [Resources overview](/workflow/resources) - all resource types
