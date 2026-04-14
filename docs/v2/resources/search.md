# Search Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps registry install search`
>
> Usage: `run: { component: { name: search, with: { query: "...", apiKey: "...", maxResults: 5 } } }`

The Search component discovers content from the web via the [Tavily](https://tavily.com/) AI-optimized search API, returning a list of results (title, URL, snippet) for downstream processing.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `query` | string | yes | — | Search query string |
| `apiKey` | string | yes | — | Tavily API key |
| `maxResults` | integer | no | `5` | Maximum number of results to return |

## Using the Search Component

```yaml
run:
  component:
    name: search
    with:
      query: "kdeps AI agent framework"
      apiKey: "tvly-your-api-key"
      maxResults: 10
```

Access the result via `output('<callerActionId>')`. The result is a list of objects with `title`, `url`, and `snippet` fields.

---

## Basic Usage

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: findArticles
  name: Find Articles

run:
  component:
    name: search
    with:
      query: "{{ get('topic') }}"
      apiKey: "{{ env('TAVILY_API_KEY') }}"
      maxResults: 5
```

</div>

---

## Result Map

```json
{
  "query": "kdeps AI workflow",
  "results": [
    {
      "title": "Introduction to kdeps",
      "url": "https://kdeps.io/docs/intro",
      "snippet": "kdeps is an open-source AI workflow engine..."
    }
  ],
  "count": 1,
  "success": true
}
```

| Field | Type | Description |
|---|---|---|
| `query` | string | The query string that was used. |
| `results` | []object | Array of result items. |
| `count` | int | Number of results returned. |
| `success` | bool | `true` when the search completed without error. |
| `error` | string | Error message (only present when `success` is `false`). |

**Result item fields:**

| Field | Type | Description |
|---|---|---|
| `title` | string | Page title. |
| `url` | string | Page URL. |
| `snippet` | string | Excerpt or description. |

---

## Expression Support

All `with:` fields support [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: search
    with:
      query: "{{ get('user_query') }}"
      apiKey: "{{ env('TAVILY_API_KEY') }}"
      maxResults: "{{ get('result_count', 5) }}"
```

</div>

---

## Full Example: Web Research Pipeline

This pipeline searches the web for a topic, scrapes the top result, and asks an LLM to
summarize it.

<div v-pre>

```yaml
# Step 1: Search for the topic
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: webSearch
    name: Web Search

  run:
    component:
      name: search
      with:
        query: "{{ get('research_topic') }}"
        apiKey: "{{ env('TAVILY_API_KEY') }}"
        maxResults: 3

# Step 2: Scrape the first result
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: scrapeTop
    name: Scrape Top Result
    requires:
      - webSearch

  run:
    component:
      name: scraper
      with:
        url: "{{ output('webSearch').results[0].url }}"
        timeout: 30

# Step 3: LLM summarizes the scraped content
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: summarise
    name: Summarise Research
    requires:
      - scrapeTop

  run:
    llm:
      prompt: |
        You are a research assistant. Summarise the following content about
        "{{ get('research_topic') }}":

        {{ output('scrapeTop').content }}

        Provide a concise 3-paragraph summary.

# Step 4: Return the summary
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: researchResponse
    name: Research Response
    requires:
      - summarise

  run:
    apiResponse:
      success: true
      response:
        summary: "{{ output('summarise') }}"
        source_url: "{{ output('webSearch').results[0].url }}"
        source_title: "{{ output('webSearch').results[0].title }}"
```

</div>
