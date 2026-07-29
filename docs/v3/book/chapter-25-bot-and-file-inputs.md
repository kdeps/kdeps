# Chapter 25: Bot and File Input Sources

Every example so far has used the `api` input source — an HTTP server that accepts JSON requests. kdeps supports two more input sources: `bot` for native chat platform integration, and `file` for single-shot file processing. You configure the source in `workflow.yaml`; your resources do not need to change.

## Three Input Sources

```yaml
# workflow.yaml
settings:
  input:
    sources: [api]    # default — HTTP REST server
    sources: [bot]    # chat platform integration
    sources: [file]   # single-shot file/stdin processing
```

Sources can also be combined:

```yaml
sources: [api, bot]    # serve HTTP API and bot simultaneously
```

The default is `[api]`. If you omit `settings.input`, kdeps starts the HTTP server.

## Bot Source

The `bot` source connects your workflow to chat platforms. Instead of handling webhook payloads manually (as in the Telegram example in Chapter 24), the bot source abstracts the platform protocol. You write one workflow; kdeps handles the platform-specific handshake.

### Supported Platforms

- **Discord** — slash commands and DMs
- **Slack** — slash commands, DMs, app mentions
- **Telegram** — messages and commands
- **WhatsApp** — messages via WhatsApp Business API

### Configuration

Bot credentials belong in `~/.kdeps/config.yaml` — they are machine-local secrets, not version-controlled workflow config. The workflow only declares which platforms to enable and their non-sensitive settings.

```yaml
# ~/.kdeps/config.yaml — credentials, never committed to version control
bot_connections:
  discord:
    botToken: "${DISCORD_BOT_TOKEN}"
  slack:
    botToken: "${SLACK_BOT_TOKEN}"
    appToken: "${SLACK_APP_TOKEN}"         # required for Socket Mode
    signingSecret: "${SLACK_SIGNING_SECRET}"
  telegram:
    botToken: "${TELEGRAM_BOT_TOKEN}"
  whatsapp:
    phoneNumberId: "${WHATSAPP_PHONE_NUMBER_ID}"
    accessToken: "${WHATSAPP_ACCESS_TOKEN}"
    webhookSecret: "${WHATSAPP_WEBHOOK_SECRET}"

# workflow.yaml — platform selection and non-sensitive settings
settings:
  input:
    sources: [bot]
    bot:
      executionType: polling    # "polling" or "stateless"
      discord: {}               # presence enables the platform
      slack: {}
      telegram:
        pollIntervalSeconds: 1  # optional; default 1
      whatsapp:
        webhookPort: 16396      # optional; default 16396
```

You only need to configure the platforms you are using. Always use environment variables for credentials — never hardcode them.

### Execution Types

**`polling`** (default) — kdeps maintains a persistent connection to the platform and processes messages as they arrive. The process runs indefinitely until stopped. This is the mode for long-running bot deployments.

**`stateless`** — reads one message from stdin as JSON, runs the workflow once, writes the reply to stdout, and exits. Useful for:
- Serverless/FaaS deployments (AWS Lambda, Google Cloud Run)
- Testing bot logic locally without a live connection
- Webhook-based integrations where an external system calls your workflow

### Reading Bot Input

In the workflow, the user's message is available via `get('message')` or `input.message`. Platform metadata is also available:

```yaml
# resources/handle-message.yaml
actionId: handleMessage
before:
  - set('text', get('message').text)
  - set('user_id', get('message').from.id)
  - set('platform', get('message').platform)   # "discord", "slack", "telegram", "whatsapp"
  - set('channel', get('message').chat.id)
```

The `message` object structure varies slightly by platform, but `message.text` contains the user's text on all platforms.

### Sending Replies

In bot mode, use the `botReply:` resource type — not `apiResponse:` — to send the reply back to the user on their platform:

```yaml
# resources/respond.yaml
actionId: respond
requires: [generateReply]
botReply:
  text: "&#123;&#123; get('generateReply') &#125;&#125;"
```

`botReply:` routes the message back to the originating platform and channel automatically. kdeps handles the platform-specific delivery (Telegram `sendMessage`, Slack `chat.postMessage`, Discord message, WhatsApp reply).

**`botReply:` vs `apiResponse:`:**

| Resource | Use in |
|---|---|
| `apiResponse:` | API (`sources: [api]`) — builds HTTP response |
| `botReply:` | Bot (`sources: [bot]`) — sends chat platform message |

When you combine sources (`sources: [api, bot]`), use both resource types in your workflow — each handles its own source type.

### Complete Bot Example: Slack Assistant

```yaml
# workflow.yaml
metadata:
  name: slack-assistant
  version: "1.0.0"
  targetActionId: respond

settings:
  input:
    sources: [bot]
    bot:
      executionType: polling
      slack: {}   # credentials in ~/.kdeps/config.yaml bot_connections.slack
  
  sqlConnections:
    main: {}  # DSN: set sql_connections.main.connection in ~/.kdeps/config.yaml
  
  session:
    type: sqlite
    path: "/data/sessions.db"
    ttl: "4h"
```

```yaml
# resources/extract-intent.yaml
actionId: extractIntent
before:
  - set('user_message', get('message').text)
  - set('user_id', string(get('message').from.id))
  - set('history', get('history') or [])
  - set('history', get('history') + [{"role": "user", "content": get('user_message')}], 'session')
chat:
  model: llama3.2:1b
  jsonResponse: true
  systemPrompt: "Classify the user's intent as one of: question, task, greeting, feedback. Return JSON: {intent: string}"
  prompt: "&#123;&#123; get('user_message') &#125;&#125;"
```

```yaml
# resources/lookup-context.yaml
actionId: lookupContext
requires: [extractIntent]
validations:
  skip:
    - get('extractIntent').intent == 'greeting'
sql:
  connectionName: main
  query: "SELECT content FROM knowledge_base WHERE relevance_to($1) > 0.7 LIMIT 3"
  params:
    - get('user_message')
```

```yaml
# resources/reply.yaml
actionId: reply
requires: [extractIntent, lookupContext]
chat:
  model: llama3.2:7b
  systemPrompt: |
    You are a helpful Slack assistant. Be concise. Use Slack markdown formatting.
    Answer based on context if available. Otherwise use your knowledge.
  prompt: |
    Context from knowledge base: &#123;&#123; get('lookupContext') or 'none' &#125;&#125;
    
    User message: &#123;&#123; get('user_message') &#125;&#125;
after:
  - set('history', get('history') + [{"role": "assistant", "content": get('reply')}], 'session')
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [reply]
botReply:
  text: "&#123;&#123; get('reply') &#125;&#125;"
```

Run it:

```bash
$ kdeps run workflow.yaml
kdeps: bot source active
kdeps: slack connected — listening for messages
```

The assistant maintains conversation history per user (via session keyed on `user_id`), looks up context from a knowledge base for non-greeting messages, and replies using Slack markdown.

### Multi-Platform Bot

A single workflow can handle multiple platforms simultaneously:

```yaml
settings:
  input:
    sources: [bot]
    bot:
      telegram: {}   # credentials in ~/.kdeps/config.yaml bot_connections.telegram
      discord: {}    # credentials in ~/.kdeps/config.yaml bot_connections.discord
      slack: {}      # credentials in ~/.kdeps/config.yaml bot_connections.slack
```

kdeps handles each platform's connection independently. Your resources receive messages from all platforms via the same `message` object.

---

## The botReply: Resource

`botReply:` is the primary executor type for sending a reply back to the originating chat platform. It is the bot-mode counterpart to `apiResponse:` — where `apiResponse:` builds an HTTP response for API callers, `botReply:` routes a text message to whoever sent the triggering message.

The resource that uses `botReply:` is the terminal node of your bot workflow's DAG — the last resource to execute, the one that delivers the result.

### Configuration Reference

```yaml
# resources/respond.yaml
actionId: respond
requires: [generateReply]
botReply:
  text: "&#123;&#123; get('generateReply') &#125;&#125;"    # required — the message text to send
```

`text` is the only required field. It supports full expression interpolation: any `&#123;&#123; &#125;&#125;` block is evaluated against the data store before the message is sent.

**Plain text (all platforms):**

```yaml
botReply:
  text: "&#123;&#123; get('reply') &#125;&#125;"
```

**Static fallback with expression guard:**

```yaml
botReply:
  text: "&#123;&#123; get('reply') or 'Sorry, I could not generate a response.' &#125;&#125;"
```

**Multi-line with formatting:**

```yaml
botReply:
  text: |
    Here is your summary:

    &#123;&#123; get('summary') &#125;&#125;

    Generated from &#123;&#123; get('source_url') &#125;&#125;
```

### How Delivery Works

When `botReply:` executes, kdeps reads the `BotSend` function injected into the execution context at request time. This function was bound to the originating platform and channel when the message arrived — you never specify the platform or channel in the resource. kdeps routes the reply back automatically.

```
User sends message on Telegram
        |
        v
kdeps receives message, binds BotSend -> telegram.sendMessage(chatId, ...)
        |
        v
Workflow DAG executes (validate -> llm -> respond)
        |
        v
botReply: executor calls ctx.BotSend(text)
        |
        v
Message delivered to original Telegram chat
```

This means the same `botReply:` resource works identically regardless of whether the message came from Telegram, Slack, Discord, or WhatsApp. You write one resource; kdeps handles the platform protocol.

### Expression Evaluation in text:

The `text:` field evaluates `&#123;&#123; expr &#125;&#125;` blocks the same way `prompt:` and `url:` fields do in other resource types. Any expression valid in the data store is valid here.

Read from a previous LLM resource:

```yaml
botReply:
  text: "&#123;&#123; get('llm') &#125;&#125;"
```

Read a specific field from a structured result:

```yaml
botReply:
  text: "&#123;&#123; get('extract').summary &#125;&#125;"
```

Compose a message with multiple data points:

```yaml
botReply:
  text: "Order &#123;&#123; get('order_id') &#125;&#125; is &#123;&#123; get('lookup').status &#125;&#125;. Expected delivery: &#123;&#123; get('lookup').delivery_date &#125;&#125;"
```

Use conditionals:

```yaml
botReply:
  text: "&#123;&#123; get('found') ? 'Found it: ' + get('result') : 'Nothing matched your query.' &#125;&#125;"
```

### Complete Request/Response Lifecycle

Here is the full lifecycle of a bot interaction using `botReply:`, from message arrival to reply delivery:

```yaml
# workflow.yaml
metadata:
  name: faq-bot
  version: "1.0.0"
  targetActionId: respond

settings:
  input:
    sources: [bot]
    bot:
      executionType: polling
      telegram: {}   # credentials in ~/.kdeps/config.yaml bot_connections.telegram
```

```yaml
# resources/validate.yaml
actionId: validate
before:
  - set('text', trim(get('message').text or ''))
validations:
  check:
    - get('text') != ''
  error:
    code: 400
    message: "empty message"
exec:
  command: "echo ok"
```

```yaml
# resources/answer.yaml
actionId: answer
requires: [validate]
chat:
  model: llama3.2:3b
  systemPrompt: |
    You are a concise FAQ bot. Answer questions in 2-3 sentences maximum.
    If you do not know, say so clearly.
  prompt: "&#123;&#123; get('text') &#125;&#125;"
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [answer]
botReply:
  text: "&#123;&#123; get('answer') &#125;&#125;"
```

Run:

```bash
$ TELEGRAM_BOT_TOKEN=your-token kdeps run workflow.yaml
kdeps: bot source active
kdeps: telegram connected — listening for messages
```

Send any message to the bot in Telegram. The workflow validates it, calls the LLM, and sends the reply back to the same chat.

### botReply: with onError

If the LLM call fails or any upstream resource errors, you can catch the error and still deliver a reply to the user instead of silently failing:

```yaml
# resources/answer.yaml
actionId: answer
requires: [validate]
chat:
  model: llama3.2:3b
  prompt: "&#123;&#123; get('text') &#125;&#125;"
onError:
  action: continue
  fallback: "I ran into a problem. Please try again."
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [answer]
botReply:
  text: "&#123;&#123; get('answer') or 'Something went wrong. Please try again.' &#125;&#125;"
```

When `onError: action: continue` fires and returns the fallback string, `get('answer')` holds that string. The `or` guard also catches any case where `get('answer')` is empty.

### Combining API and Bot Sources

When `sources: [api, bot]` is set, the workflow handles both HTTP requests and bot messages. Use `botReply:` for bot replies and `apiResponse:` for HTTP responses. Both can coexist in the same DAG — each fires only for its source type:

```yaml
settings:
  input:
    sources: [api, bot]
    bot:
      executionType: polling
      slack: {}   # credentials in ~/.kdeps/config.yaml bot_connections.slack
    api:
      apiServer:
        routes:
          - path: /api/v1/chat
            methods: [POST]
```

```yaml
# resources/answer.yaml
actionId: answer
chat:
  model: llama3.2:3b
  prompt: "&#123;&#123; get('q') or get('message').text &#125;&#125;"
```

```yaml
# resources/http-respond.yaml
actionId: httpRespond
requires: [answer]
validations:
  routes: [/api/v1/chat]    # only fires for HTTP requests
apiResponse:
  success: true
  response:
    answer: get('answer')
```

```yaml
# resources/bot-respond.yaml
actionId: botRespond
requires: [answer]
validations:
  skip:
    - get('message') == null    # skip when request is HTTP, not bot
botReply:
  text: "&#123;&#123; get('answer') &#125;&#125;"
```

Both resources depend on `answer`. For an HTTP request, `bot-respond` is skipped (no `message` in the data store). For a bot message, `http-respond` is skipped (route restriction does not match). The LLM resource runs in both cases.

### Testing botReply: Without a Live Bot Connection

In `stateless` mode, kdeps reads a single message from stdin as JSON and writes the reply to stdout. This is the fastest way to test your bot workflow locally:

```yaml
settings:
  input:
    sources: [bot]
    bot:
      executionType: stateless
      telegram: {}   # no credentials needed in stateless mode
```

```bash
# Pipe a simulated message in, capture the reply
$ echo '{"message": {"text": "What is kdeps?", "from": {"id": 123}, "chat": {"id": 123}, "platform": "telegram"&#125;&#125;' \
  | kdeps run workflow.yaml

# Output: {"reply": "kdeps is a YAML-based framework for building production AI agents..."}
```

No live Telegram connection is needed. The reply is printed to stdout and the process exits. This makes `stateless` mode ideal for automated testing and CI pipelines.

---

## File Input Source

The `file` source runs the workflow once with file content as input, then exits. It is designed for batch processing, document ingestion pipelines, and scripted automation where kdeps is invoked as a command-line tool rather than as a server.

### Configuration

```yaml
# workflow.yaml
settings:
  input:
    sources: [file]
    file:
      path: "${KDEPS_FILE_PATH}"     # from environment variable
      # or
      path: "/data/input.txt"        # hardcoded path
      # or omit path entirely: reads from stdin
```

### Reading File Content

The file content is available via `file()` in expressions:

```yaml
# resources/process-file.yaml
before:
  - set('content', file('input.txt'))    # reads the configured file
  - set('filename', file('input.txt', 'filepath'))   # just the path
  - set('filetype', file('input.txt', 'filetype'))   # MIME type
```

Or read from stdin (when no path is configured):

```bash
$ cat document.txt | kdeps run workflow.yaml
```

```yaml
before:
  - set('content', file('stdin'))
```

### The file() Function — Full Reference

```yaml
file('pattern')                # content of first matching file (default)
file('pattern', 'first')       # content of first matching file
file('pattern', 'last')        # content of last matching file
file('pattern', 'all')         # array of contents of all matching files
file('pattern', 'count')       # integer count of matching files
file('pattern', 'filepath')    # path of first matching file (string)
file('pattern', 'filetype')    # MIME type of first matching file

# Filter by MIME type
file('mime:application/pdf')   # first PDF
file('mime:image/*')           # first image (any type)
file('mime:text/plain', 'all') # all plain-text files
```

**Patterns:**

```yaml
file('document.pdf')    # exact filename
file('*.pdf')           # any PDF
file('report-*.txt')    # glob pattern
file('stdin')           # stdin (when no path is configured)
```

### Glob Patterns for Multiple Files

```yaml
before:
  - set('first_pdf', file('*.pdf'))              # first matching file content
  - set('all_csvs', file('*.csv', 'all'))        # array of all CSV contents
  - set('pdf_count', file('*.pdf', 'count'))     # count of matching PDFs
  - set('pdf_path', file('*.pdf', 'filepath'))   # path of first PDF
  - set('pdf_type', file('*.pdf', 'filetype'))   # MIME type: "application/pdf"
```

### Complete File Processing Pipeline

```yaml
# workflow.yaml
metadata:
  name: doc-ingestor
  version: "1.0.0"
  targetActionId: confirm

settings:
  input:
    sources: [file]
  sqlConnections:
    main: {}  # DSN: set sql_connections.main.connection in ~/.kdeps/config.yaml
```

```yaml
# resources/read.yaml
actionId: read
before:
  - set('content', file('*.txt', 'first') or file('stdin'))
  - set('filename', file('*.txt', 'filepath') or 'stdin')
validations:
  check:
    - get('content') != ''
  error:
    code: 400
    message: "no input file found"
exec:
  command: "echo read"
```

```yaml
# resources/extract.yaml
actionId: extract
requires: [read]
chat:
  model: llama3.2:1b
  jsonResponse: true
  prompt: |
    Extract: title, summary (2 sentences), topics (array), word_count.
    Return JSON only.
    
    &#123;&#123; get('content')[0:8000] &#125;&#125;
```

```yaml
# resources/store.yaml
actionId: store
requires: [extract]
sql:
  connectionName: main
  query: "INSERT INTO documents (filename, title, summary, topics, raw_text) VALUES ($1, $2, $3, $4, $5)"
  params:
    - get('filename')
    - get('extract').title
    - get('extract').summary
    - json(get('extract').topics)
    - get('content')[0:50000]
```

```yaml
# resources/confirm.yaml
actionId: confirm
requires: [store]
apiResponse:
  success: true
  response:
    ingested: get('filename')
    title: get('extract').title
```

Run it:

```bash
# Single file via stdin
$ cat report.txt | kdeps run workflow.yaml

# Single file via path
$ KDEPS_FILE_PATH=./report.txt kdeps run workflow.yaml

# Batch processing from a shell loop
$ for f in ./docs/*.txt; do
    KDEPS_FILE_PATH="$f" kdeps run workflow.yaml
  done
```

### Combining API and File Sources

For a workflow that can be triggered both via HTTP API and via file input (useful during development and for batch backfill):

```yaml
settings:
  input:
    sources: [api, file]
    file:
      path: "${KDEPS_FILE_PATH}"
    api:
      apiServer:
        routes:
          - path: /ingest
            methods: [POST]
```

When `KDEPS_FILE_PATH` is set, the file source runs. When it is not set, the HTTP server starts. Resources can check which source triggered them:

```yaml
before:
  - set('content', file('${KDEPS_FILE_PATH}') or get('content'))
```

## Choosing an Input Source

| Source | Use when |
|---|---|
| `api` (default) | Building HTTP APIs, REST services, webhook handlers |
| `bot` | Building chat assistants for Slack, Discord, Telegram, WhatsApp |
| `file` | Document ingestion, batch processing, CLI usage, scripted automation |
| `api + bot` | Service that handles both HTTP requests and chat messages |
| `api + file` | Service that can be triggered by HTTP or run in batch mode |

X> ## Exercise
X>
X> Build a simple FAQ bot that runs in three configurations — Telegram bot, API, and file processor — sharing one `resources/` directory across all three.
X>
X> **Part 1 — botReply: in stateless mode (no live connection needed):**
X> 1. Create a workflow with `executionType: stateless` and a `respond` resource that uses `botReply:`.
X> 2. Write the resources: `validate.yaml` (check that `message.text` is non-empty), `answer.yaml` (LLM chat), `respond.yaml` (`botReply: text: "&#123;&#123; get('answer') &#125;&#125;"`).
X> 3. Test locally without any bot token:
X>    ```bash
X>    $ echo '{"message": {"text": "What is kdeps?", "from": {"id": 1}, "chat": {"id": 1}, "platform": "telegram"&#125;&#125;' \
X>      | kdeps run workflow.yaml
X>    ```
X> 4. Verify the reply appears in stdout and the process exits cleanly.
X>
X> **Part 2 — Telegram bot (polling):**
X> 1. Duplicate `workflow.yaml` as `workflow-telegram.yaml`. Change `executionType` to `polling`. Add your `TELEGRAM_BOT_TOKEN` to `~/.kdeps/config.yaml` under `bot_connections.telegram.botToken`.
X> 2. Start: `TELEGRAM_BOT_TOKEN=<your-token> kdeps run workflow-telegram.yaml`.
X> 3. Send a message in Telegram. Verify `botReply:` delivers it back to the same chat.
X>
X> **Part 3 — File processor:**
X> 1. Duplicate `workflow.yaml` as `workflow-file.yaml`. Change `sources: [file]` with `file.path: /tmp/question.txt`. Replace `botReply:` with `apiResponse:` in `respond.yaml` (or add a separate file-mode respond resource with a route skip condition).
X> 2. Run: `echo "What is the capital of France?" > /tmp/question.txt && kdeps run workflow-file.yaml`.
X> 3. Confirm the answer is printed and the process exits.
X>
X> Confirm `resources/answer.yaml` and `resources/validate.yaml` are unchanged across all three versions. Only `workflow.yaml` (and the terminal respond resource) differs.
X>
X> **Stretch goal:** Add `onError` to `answer.yaml` with `action: continue` and `fallback: "I could not answer that."`. Test that a forced error (set an invalid model name) causes `botReply:` to deliver the fallback message rather than silently dropping the reply.
