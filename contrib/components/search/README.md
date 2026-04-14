# search

> Web search via the Tavily API.

The search component queries the [Tavily](https://tavily.com) search API and returns structured web search results. Use it to give your AI agents real-time access to web information without building your own search infrastructure.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | yes | — | Search query string |
| `apiKey` | string | yes | — | Tavily API key |
| `maxResults` | integer | no | `5` | Maximum number of results to return |

## Outputs

Accessed via `output('<your-action-id>')`:

The output is the raw Tavily API response, parsed as JSON:

```json
{
  "query": "...",
  "follow_up_questions": null,
  "answer": "...",
  "images": [],
  "results": [
    {
      "title": "...",
      "url": "...",
      "content": "...",
      "score": 0.97,
      "raw_content": null
    }
  ],
  "response_time": 1.23
}
```

| Field | Type | Description |
|-------|------|-------------|
| `answer` | string | AI-generated answer summary (if available) |
| `results` | array | List of search results with title, URL, content snippet, and relevance score |
| `query` | string | The search query that was executed |

## Usage

Basic web search:

```yaml
run:
  component:
    name: search
    with:
      query: "latest news about large language models"
      apiKey: "{{ get('env.TAVILY_API_KEY') }}"
      maxResults: 5
  apiResponse:
    success: true
    response:
      results: "{{ output('webSearch').results }}"
      answer: "{{ output('webSearch').answer }}"
```

Research agent — search then summarise:

```yaml
- actionId: searchWeb
  run:
    component:
      name: search
      with:
        query: "{{ input('topic') }}"
        apiKey: "{{ get('env.TAVILY_API_KEY') }}"
        maxResults: 10

- actionId: summariseResults
  run:
    chat:
      prompt: |
        Based on these search results, answer the question: {{ input('topic') }}

        Results:
        {{ output('searchWeb').results }}
```

## Requirements

- A [Tavily API key](https://app.tavily.com). Sign up at tavily.com for a free tier.
- Network access to `api.tavily.com`.

## Notes

- Store `apiKey` in an environment variable — never hardcode it in workflow YAML.
- The `answer` field is populated by Tavily's AI synthesis and may be empty depending on the query.
- Tavily returns results with `content` (summary snippet) and optionally `raw_content` (full page text). Full content requires a Tavily Pro plan.
- For higher result limits or advanced search options (search depth, domains), use `run.httpClient:` directly against the Tavily API with a custom request body.
