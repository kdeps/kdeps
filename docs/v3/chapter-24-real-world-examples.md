# Chapter 24: Real-World Examples

This chapter is a collection of complete, deployable workflow examples. Each addresses a real use case you can adapt directly. They draw on everything covered in the book.

## Example 1: Customer Support Bot (Multi-Turn)

A support bot that maintains conversation history and can look up order information from a database.

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: support-bot
  version: "1.0.0"
  targetActionId: respond
settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 8080
    routes:
      - path: /api/v1/chat
        methods: [POST]
  session:
    type: sqlite
    path: "/data/sessions.db"
    ttl: "2h"
  sqlConnections:
    orders: {}  # DSN: set sql_connections.orders.connection in ~/.kdeps/config.yaml
```

```yaml
# resources/validate.yaml
actionId: validate
validations:
  methods: [POST]
  check:
    - get('message') != ''
  error:
    code: 400
    message: "message is required"
exec:
  command: "echo ok"
```

```yaml
# resources/load-history.yaml
actionId: loadHistory
requires: [validate]
before:
  - set('history', get('history') or [])
exec:
  command: "echo loaded"
```

```yaml
# resources/order-lookup.yaml
actionId: orderLookup
requires: [loadHistory]
validations:
  skip:
    - get('order_id') == ''    # skip if no order_id in the message
before:
  - set('order_id', get('order_id'))
sql:
  connectionName: orders
  query: "SELECT id, status, items, total FROM orders WHERE id = $1 AND customer_email = $2"
  params:
    - get('order_id')
    - get('customer_email')
```

```yaml
# resources/reply.yaml
actionId: reply
requires: [loadHistory, orderLookup]
before:
  - set('context', get('orderLookup') != null and len(get('orderLookup')) > 0
      ? 'Order found: ' + json(get('orderLookup')[0])
      : 'No order context available')
  - set('history', get('history') + [{"role": "user", "content": get('message')}], 'session')
chat:
  model: llama3.2:7b
  systemPrompt: |
    You are a helpful customer support assistant for an e-commerce company.
    Be concise, friendly, and accurate.
    If asked about an order and you have order data, use it. Otherwise say you need the order ID.
    Never make up order information.
  messages: "&#123;&#123; get('history') &#125;&#125;"
  prompt: |
    &#123;&#123; get('context') &#125;&#125;
    
    Customer message: &#123;&#123; get('message') &#125;&#125;
after:
  - set('history', get('history') + [{"role": "assistant", "content": get('reply')}], 'session')
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [reply]
apiResponse:
  success: true
  response:
    reply: get('reply')
    session_turns: len(get('history'))
```

---

## Example 2: Document Processing Pipeline

Accepts a document URL, extracts text, generates a structured summary, and stores both in a database.

```yaml
# workflow.yaml
metadata:
  name: doc-processor
  targetActionId: respond
settings:
  apiServer:
    routes:
      - path: /api/v1/process
        methods: [POST]
  sqlConnections:
    main: {}  # DSN: set sql_connections.main.connection in ~/.kdeps/config.yaml
```

```yaml
# resources/validate.yaml
actionId: validate
validations:
  check:
    - get('url') != ''
    - get('url') startsWith 'https://'
  error:
    code: 400
    message: "valid https URL required"
exec:
  command: "echo ok"
```

```yaml
# resources/fetch.yaml
actionId: fetch
requires: [validate]
scraper:
  url: "&#123;&#123; get('url') &#125;&#125;"
  selector: "article, main, .content"
  timeout: 30
```

```yaml
# resources/check-length.yaml
actionId: checkLength
requires: [fetch]
validations:
  check:
    - len(get('fetch')) > 100
  error:
    code: 422
    message: "document content too short or could not be extracted"
exec:
  command: "echo ok"
```

```yaml
# resources/extract.yaml
actionId: extract
requires: [checkLength]
chat:
  model: llama3.2:7b
  jsonResponse: true
  prompt: |
    Extract the following from this document as JSON:
    - title (string): document title or first heading
    - summary (string): 2-3 sentence summary
    - topics (array of strings): main topics covered
    - sentiment (string): positive/neutral/negative
    - word_count (integer): approximate word count
    
    Document:
    &#123;&#123; get('fetch')[0:8000] &#125;&#125;
    
    Return only valid JSON.
```

```yaml
# resources/store.yaml
actionId: store
requires: [extract]
sql:
  connectionName: main
  query: |
    INSERT INTO processed_documents 
      (url, title, summary, topics, sentiment, raw_text, created_at)
    VALUES ($1, $2, $3, $4, $5, $6, NOW())
    RETURNING id
  params:
    - get('url')
    - get('extract').title
    - get('extract').summary
    - json(get('extract').topics)
    - get('extract').sentiment
    - get('fetch')[0:50000]
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [store]
apiResponse:
  success: true
  response:
    id: get('store')[0].id
    url: get('url')
    extracted: get('extract')
```

---

## Example 3: Autonomous Research Agency

A three-agent agency that searches, reads sources, and writes a research brief.

```yaml
# agency.yaml
apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: research-agency
  version: "1.0.0"
  targetAgentId: coordinator
```

**Coordinator agent** (entry point, orchestrates):

```yaml
# agents/coordinator/workflow.yaml
metadata:
  name: coordinator
  description: "Orchestrates research: receives a question, delegates to searcher and writer"
  targetActionId: respond
settings:
  apiServer:
    routes:
      - path: /api/v1/research
        methods: [POST]
```

```yaml
# agents/coordinator/resources/delegate-search.yaml
actionId: delegateSearch
validations:
  check:
    - get('question') != ''
agent:
  target: searcher
  input: "&#123;&#123; get('question') &#125;&#125;"
```

```yaml
# agents/coordinator/resources/delegate-write.yaml
actionId: delegateWrite
requires: [delegateSearch]
agent:
  target: writer
  input: |
    Question: &#123;&#123; get('question') &#125;&#125;
    Research findings:
    &#123;&#123; get('delegateSearch') &#125;&#125;
```

```yaml
# agents/coordinator/resources/respond.yaml
actionId: respond
requires: [delegateWrite]
apiResponse:
  success: true
  response:
    brief: get('delegateWrite')
    sources_consulted: get('delegateSearch')
```

**Searcher agent:**

```yaml
# agents/searcher/workflow.yaml
metadata:
  name: searcher
  description: "Searches the web and returns summarized findings with source URLs"
  targetActionId: respond
```

```yaml
# agents/searcher/resources/search.yaml
actionId: search
searchWeb:
  query: "&#123;&#123; get('input') &#125;&#125;"
  maxResults: 5
```

```yaml
# agents/searcher/resources/scrape.yaml
actionId: scrape
requires: [search]
scraper:
  url: "&#123;&#123; get('search')[0].url &#125;&#125;"
  timeout: 20
```

```yaml
# agents/searcher/resources/summarize.yaml
actionId: summarize
requires: [search, scrape]
chat:
  model: llama3.2:1b
  prompt: |
    Summarize what you found about: &#123;&#123; get('input') &#125;&#125;
    
    Top search results:
    {% for r in get('search') %}
    - &#123;&#123; r.title &#125;&#125; (&#123;&#123; r.url &#125;&#125;): &#123;&#123; r.snippet &#125;&#125;
    {% endfor %}
    
    Full content of top result:
    &#123;&#123; get('scrape')[0:3000] &#125;&#125;
```

```yaml
# agents/searcher/resources/respond.yaml
actionId: respond
requires: [summarize]
apiResponse:
  success: true
  response:
    summary: get('summarize')
    sources: map(get('search'), {.url})
```

**Writer agent:**

```yaml
# agents/writer/workflow.yaml
metadata:
  name: writer
  description: "Takes a research brief as input and writes a structured 400-word report"
  targetActionId: respond
```

```yaml
# agents/writer/resources/write.yaml
actionId: write
chat:
  model: llama3.2:7b
  systemPrompt: |
    You write clear, factual research briefs. Use the provided findings.
    Structure: Executive Summary (2 sentences), Key Findings (3-5 bullets), Conclusion (2 sentences).
    Total: approximately 400 words.
  prompt: "&#123;&#123; get('input') &#125;&#125;"
```

```yaml
# agents/writer/resources/respond.yaml
actionId: respond
requires: [write]
apiResponse:
  success: true
  response: get('write')
```

Usage:

```bash
$ kdeps run agency.yaml

$ curl -X POST http://localhost:16395/api/v1/research \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"question": "What are the economic impacts of remote work adoption?"}'
```

---

## Example 4: Content Moderation API

A classification API that uses a fast local model to categorize content, flags high-risk items, and stores audit records.

```yaml
# resources/classify.yaml
actionId: classify
validations:
  check:
    - get('content') != ''
    - len(get('content')) <= 5000
  error:
    code: 400
    message: "content required (max 5000 chars)"
chat:
  model: llama3.2:1b
  jsonResponse: true
  systemPrompt: "You classify content. Respond only with valid JSON."
  prompt: |
    Classify this content and return JSON:
    {
      "category": "<one of: safe, spam, hate_speech, adult, violence, off_topic>",
      "confidence": <0.0-1.0>,
      "reasons": ["<reason 1>", "<reason 2>"]
    }
    
    Content: &#123;&#123; get('content')[0:1000] &#125;&#125;
```

```yaml
# resources/flag-high-risk.yaml
actionId: flagHighRisk
requires: [classify]
validations:
  skip:
    - get('classify').confidence < 0.8
    - get('classify').category == 'safe'
sql:
  connectionName: main
  query: "INSERT INTO flagged_content (content_hash, category, confidence, content) VALUES ($1, $2, $3, $4)"
  params:
    - "&#123;&#123; sha256(get('content')) &#125;&#125;"
    - get('classify').category
    - get('classify').confidence
    - get('content')[0:1000]
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [classify, flagHighRisk]
apiResponse:
  success: true
  response:
    category: get('classify').category
    confidence: get('classify').confidence
    flagged: get('classify').category != 'safe' and get('classify').confidence >= 0.8
    reasons: get('classify').reasons
```

---

## Example 5: Telegram Bot

```yaml
# workflow.yaml
metadata:
  name: telegram-bot
  targetActionId: respond
settings:
  apiServer:
    routes:
      - path: /webhook
        methods: [POST]
```

```yaml
# resources/extract-message.yaml
actionId: extractMessage
before:
  - set('text', get('message').text or get('callback_query').data or '')
  - set('chat_id', string(get('message').chat.id or get('callback_query').message.chat.id))
  - set('username', get('message').from.username or 'unknown')
validations:
  check:
    - get('chat_id') != ''
    - get('text') != ''
exec:
  command: "echo extracted"
```

```yaml
# resources/generate-reply.yaml
actionId: generateReply
requires: [extractMessage]
chat:
  model: llama3.2:1b
  systemPrompt: "You are a helpful Telegram bot. Keep replies short (under 200 chars)."
  prompt: "&#123;&#123; get('text') &#125;&#125;"
```

```yaml
# resources/send-reply.yaml
actionId: sendReply
requires: [generateReply]
httpClient:
  method: POST
  url: "https://api.telegram.org/bot&#123;&#123; env('TELEGRAM_BOT_TOKEN') &#125;&#125;/sendMessage"
  body:
    chat_id: get('chat_id')
    text: get('generateReply')
    parse_mode: Markdown
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [sendReply]
apiResponse:
  success: true
  statusCode: 200
  response:
    ok: true
```

Register the webhook with Telegram:

```bash
$ curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
  -d "url=https://your-domain.com/webhook"
```

---

These five examples cover the range of real-world use cases kdeps is built for: conversational bots, document pipelines, multi-agent research, high-volume classification APIs, and messenger integrations. All use the same building blocks. All deploy anywhere kdeps deploys.

The patterns are yours to combine.

X> ## Exercise
X>
X> Choose one of the five examples from this chapter and adapt it for a domain you know well. The adaptation must change at least three of the following:
X>
X> - The data source (database schema, API endpoint, or file format)
X> - The LLM prompt (different task or persona)
X> - The validation rules (different required fields or constraints)
X> - The response shape (different fields in `apiResponse`)
X> - The routing (different HTTP path or methods)
X>
X> Document what you changed and why in a `CHANGES.md` file inside the project directory.
X>
X> Then:
X> 1. Run `kdeps validate workflow.yaml` — it must pass cleanly.
X> 2. Test the happy path with curl.
X> 3. Test at least one error path (invalid input, missing field, or bad data) and verify the correct HTTP status and error message.
X> 4. Package the adapted workflow: `kdeps bundle package workflow.yaml`.
X>
X> This is a capstone exercise — it uses resources, validation, expressions, configuration, and packaging from across the book.
X>
X> **Stretch goal:** Deploy your adapted workflow to Docker (Chapter 17), generate Kubernetes manifests (Chapter 18), and verify it runs identically in both environments with only environment variables changing between deployments.
