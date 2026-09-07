# Web resources

Three resources for reaching the web. Each has its own reference page.

*Applies to both workflow mode and agent mode.*

| Resource | Use it for | Reference |
| :--- | :--- | :--- |
| `httpClient:` | Call a JSON API, webhook, or REST endpoint | [HTTP client](/workflow/resources/http-client) |
| `scraper:` | Get the readable text of a static page | [Scraper](/workflow/resources/scraper) |
| `browser:` | A page that needs JavaScript, a login, or form interaction | [Browser](/workflow/resources/browser) |

## Which one

- **Just data from an API** - `httpClient:`. Built-in auth, retry, and cache.
- **Text from a page** - `scraper:`. In-process, no browser, optional CSS
  selector.
- **The page only works with JavaScript, or you need to click and type** -
  `browser:`. Real Chromium/Firefox/WebKit via Playwright.

## See also

- [Authenticated API call tutorial](/examples/http-auth) - `httpClient:` auth, retry, cache
- [Web scraper tutorial](/examples/web-scraper) - `scraper:` plus an LLM summary
- [Search resources](/workflow/resources/search) - find pages to fetch
- [Resources overview](/workflow/resources) - all resource types
