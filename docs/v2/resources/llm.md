# LLM Resource

The `chat:` resource sends a prompt to a language model and stores the response as the resource's output. The output is the raw response object -- the reply text is at `get('id').message.content`. With `jsonResponse: true` the output is the parsed JSON object instead.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool.

## Where config lives

Model selection goes in the resource file. Backend and API keys go in `~/.kdeps/config.yaml`. This lets you change backends without touching your workflow.

```yaml
# resources/my-resource.yaml
chat:
  model: llama3.2:1b    # which model to call
  role: user
  prompt: "{{ get('q') }}"
```

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: file                # default: local llamafile, no server install. Also: ollama, openai, anthropic, groq, ...
  # openai_api_key: sk-...
  # anthropic_api_key: sk-ant-...
```

With the default `file` backend, `model: llama3.2:1b` resolves to a local
llamafile that is downloaded once (~1.1 GB) and served automatically - see
[LLM Backends](llm-backends#the-default-llamafile-file-backend).

Set `model: router` to delegate model selection to the router configured in `~/.kdeps/config.yaml`. See [LLM Backends](llm-backends) for routing strategies.

## Basic Usage

<div v-pre>

```yaml

actionId: llmResource
name: LLM Chat
chat:
  prompt: "{{ get('q') }}"
  timeout: 60s
```

</div>

## Complete reference

<div v-pre>

```yaml
# resources/example.yaml
chat:
  model: llama3.2:1b    # model name, or "router" to delegate to config
  role: user            # role for this message: user, assistant, system

  prompt: "{{ get('q') }}"  # the prompt; supports {{ }} interpolation

  contextLength: 8192   # context window in tokens (4096, 8192, 16384, ...)
  temperature: 0.7      # 0.0 = deterministic, 2.0 = very random
  maxTokens: 1000       # hard cap on generated tokens
  topP: 0.9             # nucleus sampling threshold (0.0 to 1.0)
  frequencyPenalty: 0.0 # penalises tokens that have appeared frequently (-2.0 to 2.0)
  presencePenalty: 0.0  # penalises any token that has appeared at all (-2.0 to 2.0)

  # pre-fill the conversation history before the prompt
  scenario:
    - role: system
      prompt: You are a helpful assistant.
    - role: assistant
      prompt: I am ready to help!

  # let the LLM call other resources as functions
  tools:
    - name: calculate
      description: Perform math
      script: calcResource   # actionId of the resource to call
      parameters:
        expression:
          type: string
          required: true

  # attach files for vision-capable models
  files:
    - "{{ get('file', 'filepath') }}"

  jsonResponse: true         # ask the model to return valid JSON
  jsonResponseKeys:          # keys to extract from the JSON response
    - answer
    - confidence

  # strict JSON schema (OpenAI models only) — enforces exact output shape
  # use this instead of jsonResponseKeys when you need guaranteed structure
  jsonSchema:
    type: object
    properties:
      answer:
        type: string
      confidence:
        type: number
    required:
      - answer
      - confidence

  timeout: 60s               # hard stop -- returns error, does not retry
  streaming: true            # Ollama only: stream NDJSON; kdeps accumulates before returning

  # chain-of-thought / few-shot
  chainOfThought: false      # inject step-by-step reasoning prefix into system prompt
  fewShotEmbeddingModel: ""  # embedding model for semantic few-shot selection (requires fewShotSelectK)
  fewShotEmbeddingBackend: "" # backend serving the embedding model above

  # sampling extras
  candidateCount: 1          # number of independent completions (alias: n)
  n: 1                       # alias for candidateCount (OpenAI style)
  minLength: 0               # minimum response length in tokens
  maxLength: 0               # maximum response length in tokens (alias for maxTokens)

  # anthropic-specific
  promptCaching: false           # add prompt-caching beta header; Anthropic only
  anthropicExtendedOutput: false # enable 128K output; adds interleaved-thinking beta header
  anthropicBetaHeaders: []       # additional anthropic-beta header values

  # openai-specific
  openAILegacyMaxTokens: false   # send max_tokens instead of max_completion_tokens

  # google-specific
  googleCachedContent: ""    # name of a Google AI CachedContent resource to attach
  googleHarmThreshold: 0     # safety filter: 0=unspecified, 1=block-none, 2=block-few, 3=block-some, 4=block-most
  googleCloudProject: ""     # Vertex AI GCP project ID
  googleCloudLocation: ""    # Vertex AI region (e.g. "us-central1")

  # ollama-specific
  ollamaThink: false         # enable Ollama extended thinking
  ollamaKeepAlive: ""        # keep model loaded after request (e.g. "5m", "1h")
  ollamaPullModel: false     # auto-pull model if not present
  ollamaPullTimeout: ""      # timeout for model pull (e.g. "10m")
```

</div>

## Advanced Parameters

Full sampling control — all fields are optional and fall back to the model's provider defaults.

```yaml
# resources/example.yaml
chat:
  prompt: "Write a creative story"
  temperature: 0.9          # 0.0 = deterministic, 1.0+ = more random/creative
  maxTokens: 500            # hard cap on generated tokens; 0 = model default
  topP: 0.9                 # nucleus sampling -- lower = less diverse vocabulary
  topK: 40                  # top-K sampling -- limits vocabulary to K most likely tokens
  seed: 42                  # fixed seed for reproducible outputs (provider support varies)
  frequencyPenalty: 0.0     # penalises tokens repeated in the output (-2.0 to 2.0)
  presencePenalty: 0.6      # penalises any token that appeared at all (-2.0 to 2.0)
  repetitionPenalty: 1.1    # multiplicative penalty for repetition (1.0 = none; >1 = penalise)
  stopWords:                # stop generation when any of these strings appear
    - "END"
    - "###"
```

## Context Length

Control the context window:

```yaml
# resources/example.yaml
chat:
  contextLength: 8192  # Options: 4096, 8192, 16384, 32768, 65536, 131072, 262144
```

## Scenario (Conversation History)

Build multi-turn conversations:

<div v-pre>

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  scenario:
    - role: system
      prompt: |
        You are an expert software developer.
        Always provide code examples.
        Be concise and practical.

    - role: user
      prompt: What is a REST API?

    - role: assistant
      prompt: |
        A REST API is an architectural style for web services.
        It uses HTTP methods (GET, POST, PUT, DELETE) to perform operations.
```

</div>

## JSON Response

Get structured JSON output:

<div v-pre>

```yaml
# resources/example.yaml
chat:
  prompt: "Analyze: {{ get('q') }}"
  jsonResponse: true
  jsonResponseKeys:
    - summary
    - sentiment
    - keywords
    - confidence
```

</div>

Output:
```json
{
  "summary": "...",
  "sentiment": "positive",
  "keywords": ["ai", "machine learning"],
  "confidence": 0.95
}
```

## Vision (File Attachments)

Process images (set a vision-capable model in `chat.model` in your resource file):

<div v-pre>

```yaml
# resources/example.yaml
chat:
  model: llama3.2-vision
  prompt: "Describe this image"
  files:
    - "{{ get('file', 'filepath') }}"  # From upload
    - "./images/example.jpg"            # From filesystem
```

</div>

## Tools (Function Calling)

Enable LLMs to call other resources:

<div v-pre>

```yaml
# resources/llm-with-tools.yaml
actionId: llmWithTools
chat:
  prompt: "{{ get('q') }}"
  tools:
    - name: calculate
      description: Perform mathematical calculations
      script: calcTool
      parameters:
        expression:
          type: string
          description: Math expression (e.g., "2 + 2")
          required: true

    - name: search_db
      description: Search the database
      script: dbSearchTool
      parameters:
        query:
          type: string
          description: Search query
          required: true
        limit:
          type: integer
          description: Max results
          required: false
```

</div>

### Component Tools (Opt-In Allowlist)

Expose installed components as LLM function-calling tools:

```bash
kdeps registry install scraper
kdeps registry install search
```

```yaml
# resources/example.yaml
chat:
  prompt: "Research {{ get('q') }} and summarize the findings."
  componentTools:
    - scraper
    - search
```

## Chain-of-Thought

`chainOfThought: true` injects a "think step-by-step" system prompt prefix before the first system message. The model reasons explicitly before answering. Works with any provider.

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  chainOfThought: true   # prepend CoT instruction to the system prompt
```

## Semantic Few-Shot Selection

When you have many examples and want semantic (embedding-based) selection instead of word-overlap, set `fewShotEmbeddingModel` and `fewShotEmbeddingBackend`. The K most similar examples are chosen by embedding cosine similarity.

```yaml
# resources/classifier.yaml
chat:
  prompt: "{{ get('q') }}"
  fewShotSelectK: 3
  fewShotEmbeddingModel: text-embedding-3-small   # embedding model for similarity ranking
  fewShotEmbeddingBackend: openai                 # backend that serves the embedding model
  fewShot:
    - role: user
      prompt: What color is the sky?
    - role: assistant
      prompt: blue
    # ... many more examples
```

Requires `fewShotSelectK` to be set. Falls back to word-overlap if no embedding model is configured.

## Prompt Caching (Anthropic)

`promptCaching: true` enables Anthropic server-side prompt caching. kdeps adds the required `anthropic-beta: prompt-caching-2024-07-31` header automatically. Reduces latency and cost for repeated long system prompts.

```yaml
# resources/example.yaml
chat:
  model: claude-sonnet-4-20250514
  promptCaching: true   # add prompt-caching beta header; Anthropic only
  scenario:
    - role: system
      prompt: |
        You are an expert assistant. [... long system prompt ...]
```

### Per-message cache control

Mark a specific scenario message for caching with `cacheControl: "ephemeral"`:

```yaml
# resources/example.yaml
chat:
  model: claude-sonnet-4-20250514
  scenario:
    - role: system
      prompt: You are a helpful assistant.
      cacheControl: "ephemeral"   # cache this specific message's content
    - role: user
      prompt: What is 2+2?
```

## Extended Output (Anthropic)

`anthropicExtendedOutput: true` enables 128K output tokens via the `interleaved-thinking-2025-05-14` beta header. Use with models that support extended output (e.g. `claude-sonnet-4-20250514`).

```yaml
# resources/example.yaml
chat:
  model: claude-sonnet-4-20250514
  anthropicExtendedOutput: true   # enable 128K output; adds beta header automatically
  maxTokens: 16000
```

## Anthropic Beta Headers

`anthropicBetaHeaders` passes arbitrary beta feature headers to Anthropic. Each string is appended to the `anthropic-beta` header value.

```yaml
# resources/example.yaml
chat:
  model: claude-sonnet-4-20250514
  anthropicBetaHeaders:
    - output-128k-2025-02-19     # example: explicit extended output header
    - interleaved-thinking-2025-05-14
```

## OpenAI Legacy Token Param

Older OpenAI-compatible servers (Azure, self-hosted) use `max_tokens` instead of `max_completion_tokens`. Set `openAILegacyMaxTokens: true` to send the old parameter name.

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  openAILegacyMaxTokens: true   # send max_tokens instead of max_completion_tokens
  maxTokens: 1000
```

## Google AI: Cached Content

`googleCachedContent` specifies the name of a Google AI CachedContent resource to attach to the request. Use with the `google_cache_create` built-in tool to pre-cache large context.

```yaml
# resources/example.yaml
chat:
  model: gemini-1.5-pro
  googleCachedContent: "cachedContents/my-cached-doc"   # CachedContent resource name
  prompt: "{{ get('q') }}"
```

## Google AI: Safety Threshold

`googleHarmThreshold` controls how aggressively Google's safety filters block responses.

| Value | Meaning |
|-------|---------|
| `0` | Unspecified (provider default) |
| `1` | Block none |
| `2` | Block few |
| `3` | Block some |
| `4` | Block most |

```yaml
# resources/example.yaml
chat:
  model: gemini-1.5-pro
  googleHarmThreshold: 1   # block-none: pass all content through filters
  prompt: "{{ get('q') }}"
```

## Vertex AI (Google Cloud)

`googleCloudProject` and `googleCloudLocation` target Google's Vertex AI endpoint instead of the standard AI Studio endpoint.

```yaml
# resources/example.yaml
chat:
  model: gemini-1.5-pro
  googleCloudProject: my-gcp-project      # GCP project ID
  googleCloudLocation: us-central1        # Vertex AI region
  prompt: "{{ get('q') }}"
```

See [LLM Backends - Vertex AI](llm-backends#vertex-ai-google-cloud) for backend configuration.

## Ollama Native Options

Native Ollama options available only when the backend is `ollama`.

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  ollamaThink: true             # enable Ollama extended thinking (model must support it)
  ollamaKeepAlive: "5m"         # keep model loaded in memory for 5 minutes after request
  ollamaPullModel: true         # auto-pull the model if not present locally
  ollamaPullTimeout: "10m"      # timeout for model pull; applies only when ollamaPullModel: true
```

## Sampling: Candidate Count

`candidateCount` (or its alias `n`) requests N independent completions from the model. The response contains all candidates; kdeps merges them into the output object. Not all providers support multiple candidates.

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  candidateCount: 3   # generate 3 independent completions
  # n: 3             # alias for candidateCount (OpenAI style)
```

## Sampling: Length Bounds

`minLength` and `maxLength` set lower and upper bounds on generated token count. `maxLength` is an alias for `maxTokens`.

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  minLength: 50      # minimum response length in tokens
  maxLength: 500     # maximum response length in tokens (alias for maxTokens)
```

## Streaming (Ollama only)

Set `streaming: true` to have Ollama stream the response as NDJSON chunks. KDeps accumulates all chunks and returns the same response shape as non-streaming.

<div v-pre>

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  streaming: true
```

</div>

## Few-Shot Prompting

Inject example user/assistant pairs before the conversation to demonstrate the expected output format. Like calling a function with example inputs and outputs — the model learns the pattern from the examples.

<div v-pre>

```yaml
# resources/classifier.yaml
chat:
  prompt: "{{ get('q') }}"
  fewShot:
    - role: user
      prompt: What color is the sky?
    - role: assistant
      prompt: blue
    - role: user
      prompt: What color is grass?
    - role: assistant
      prompt: green
```

</div>

### Dynamic example selection

When you have many examples, `fewShotSelectK` picks the K most relevant to the current prompt using word-overlap similarity:

<div v-pre>

```yaml
# resources/classifier.yaml
chat:
  prompt: "{{ get('q') }}"
  fewShotSelectK: 3   # pick 3 most similar examples
  fewShot:
    - role: user
      prompt: What color is the sky?
    - role: assistant
      prompt: blue
    # ... many more examples
```

</div>

### Token budget for examples

`fewShotMaxTokens` caps the total tokens used by examples, implementing the LengthBasedExampleSelector pattern. Useful when your example pool is large and you want to stay within context limits:

<div v-pre>

```yaml
# resources/classifier.yaml
chat:
  prompt: "{{ get('q') }}"
  fewShotSelectK: 10          # pick up to 10 similar examples
  fewShotMaxTokens: 500       # but never use more than 500 tokens total
  fewShot:
    - role: user
      prompt: What color is the sky?
    - role: assistant
      prompt: blue
    # ... large example bank
```

</div>

Both fields can be combined: `fewShotSelectK` ranks examples by similarity first, then `fewShotMaxTokens` prunes the result to the token budget. Pairs (user + assistant) are always kept whole.

## RAG Context Injection

Pass pre-fetched retriever chunks into the system prompt with `retrieverContext`. Each chunk becomes a "Retrieved context:" block prepended to the first system message. Populate this from a `vectorStore:` action output.

<div v-pre>

```yaml
# resources/rag-chat.yaml
chat:
  prompt: "{{ get('q') }}"
  retrieverContext: "{{ $actions.search.output.documents }}"
  scenario:
    - role: system
      prompt: Answer using only the retrieved context.
```

</div>

### Contextual compression

`retrieverContextTopK` keeps only the K chunks most relevant to the current prompt (Jaccard word-overlap similarity). Useful when a vectorstore returns many results but you only want the most relevant ones:

<div v-pre>

```yaml
chat:
  prompt: "{{ get('q') }}"
  retrieverContext: "{{ $actions.search.output.documents }}"
  retrieverContextTopK: 5   # keep 5 most relevant chunks
```

</div>

### Token budget for retriever chunks

`retrieverContextMaxTokens` caps the total tokens of all injected chunks. Chunks are added in similarity-score order until the budget is reached:

<div v-pre>

```yaml
chat:
  prompt: "{{ get('q') }}"
  retrieverContext: "{{ $actions.search.output.documents }}"
  retrieverContextTopK: 10           # pick 10 most relevant
  retrieverContextMaxTokens: 1000    # but never use more than 1000 tokens
```

</div>

When combined, `retrieverContextTopK` selects by relevance first, then `retrieverContextMaxTokens` prunes the result to the token budget.

## Examples

### Simple Q&A

<div v-pre>

```yaml
# resources/example.yaml
chat:
  model: llama3.2:1b
  prompt: "{{ get('q') }}"
  scenario:
    - role: system
      prompt: Answer questions concisely.
  jsonResponse: true
  jsonResponseKeys:
    - answer
  timeout: 30s
```

</div>

### Code Generation

<div v-pre>

```yaml
# resources/example.yaml
chat:
  model: llama3.2:1b
  prompt: "Write a Python function that {{ get('task') }}"
  scenario:
    - role: system
      prompt: |
        You are an expert Python developer.
        Write clean, documented code with type hints.
  jsonResponse: true
  jsonResponseKeys:
    - code
    - explanation
  timeout: 60s
```

</div>

### Multi-Resource Pipeline

<div v-pre>

```yaml
# Fast classification resource
actionId: classifier
chat:
  prompt: "Classify this query: {{ get('q') }}"
  jsonResponse: true
  jsonResponseKeys:
    - category
    - confidence

---
# Detailed response (only runs when confidence >= 0.8)
actionId: detailedResponse
requires: [classifier]
validations:
  skip:
  - get('classifier').confidence < 0.8

chat:
  prompt: |
    Category: {{ get('classifier').category }}
    Query: {{ get('q') }}
    Provide a detailed response.
  timeout: 120s
```

</div>

## Accessing Output

```yaml
# resources/example.yaml
requires: [llmResource]
apiResponse:
  response:
    answer: get('llmResource').message.content   # reply text
    raw: get('llmResource')                      # full response object
    parsed: get('llmResource').answer            # if jsonResponse: true with key "answer"
```

## See Also

- [LLM Backends](llm-backends) - Configure model, backend, API keys, and routing
- [Tools](../concepts/tools) - LLM function calling
- [Docker Deployment](../deployment/docker) - Deploying with local models
