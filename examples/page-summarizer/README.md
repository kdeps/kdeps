# page-summarizer

A WASM workflow that summarizes the page you are looking at. The build writes one HTML file. Double-click it - no server. Drag **Summarize page** onto the bookmarks bar, then click that bookmark on any tab.

```text
current tab -> bookmarklet (innerText + clipboard) -> page-summarizer.html (WASM) -> OpenAI
```

WASM cannot scrape the open tab. The bookmarklet runs in that tab and copies `url`, `title`, and `text` into the HTML file.

## Build

```bash
kdeps bundle build examples/page-summarizer --wasm
# same as --wasm-output html
```

That writes `examples/page-summarizer/page-summarizer.html`. For a static site plus nginx image: `--wasm --wasm-output server`.

1. Double-click `page-summarizer.html` (Finder / Explorer). No kdeps, no server.
2. Paste an OpenAI API key.
3. Drag **Summarize page** onto the bookmarks bar.
4. Open any article, click the bookmark.

If the browser blocks opening a `file://` window from the bookmark, the page text is on the clipboard - focus the HTML window and it picks it up.

## Input

`window.kdeps.execute` JSON:

```json
{ "url": "https://example.com", "title": "Example", "text": "page body..." }
```

The bookmarklet fills those fields. You can also paste text and click Summarize.

## Layout

```
page-summarizer/
|-- kdeps.pkg.yaml
|-- workflow.yaml
|-- resources/
|   |-- summarize.yaml
|   `-- response.yaml
`-- data/public/index.html   # inlined into page-summarizer.html on --wasm
```
