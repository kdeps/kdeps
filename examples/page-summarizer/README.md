# page-summarizer

A WASM workflow that summarizes the page you are looking at. A bookmarklet copies `document.body.innerText` and `postMessage`s it into the app. The DAG is `chat:` (OpenAI) plus `apiResponse:`.

*Applies to workflow mode. Built with `kdeps bundle build --wasm`.*

```text
current tab -> bookmarklet (innerText) -> dist/index.html (WASM) -> OpenAI -> summary
```

WASM cannot scrape the open tab. The bookmarklet runs in that tab and sends the text in.

## Build

```bash
make build-wasm
export OPENAI_API_KEY=sk-...   # used if you also `kdeps run`; the WASM UI stores its own key
kdeps bundle build examples/page-summarizer --wasm
```

Serve `dist/` over HTTP (cloud `chat:` from `file://` is blocked by provider CORS):

```bash
# after the docker build, the bundle is also in the image; or copy dist/ from the build output
docker run -p 8080:80 kdeps-wasm:latest
# or: python3 -m http.server 8080  (from the generated dist/)
```

Open `http://127.0.0.1:8080/`, paste an OpenAI API key, drag **Summarize page** to the bookmarks bar. Visit any site, click the bookmark.

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
`-- data/public/index.html   # copied into the WASM dist/
```
