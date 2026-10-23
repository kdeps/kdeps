# Web resources

Three components reach the web: `http_request` for APIs, `web_scraper` for page text, and `web_search` for finding pages. Each has its own reference page.

Works in workflow and agent mode.

| Resource | Use it for | Reference |
| :--- | :--- | :--- |
| `component: http_request` | Call a JSON API, webhook, or REST endpoint | [HTTP client](/workflow/resources/http-client) |
| `component: web_scraper` | Read the readable text of one page (title, headers, body, links) | [Web scraper](/workflow/resources/web-scraper) |
| `component: web_search` | Find pages for a query (DuckDuckGo) | [Search resources](/workflow/resources/search) |

## Which one

- **Just data from an API** - the `http_request` component. Built-in auth, retry, and cache.
- **Text from a page you already have a URL for** - the `web_scraper` component. No browser: the page is fetched and converted to markdown.
- **You need to find the right URL first** - the `web_search` component, then `web_scraper` on the result.

## See also

- [Authenticated API call tutorial](/examples/http-auth) - `http_request` auth, retry, cache
- [Search resources](/workflow/resources/search) - find pages to fetch
- [Resources overview](/workflow/resources) - all resource types
