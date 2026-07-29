# Chapter 6: LLM Resources

The `chat:` resource is where your workflow calls a language model. It handles prompt construction, model selection, timeout management, structured output, tool registration, and multi-turn conversation. This chapter covers the full `chat:` API and the backend configuration that powers it.

## Basic Usage

```yaml
# resources/llm.yaml
actionId: llm
chat:
  model: llama3.2:1b
  role: user
  prompt: "&#123;&#123; get('q') &#125;&#125;"
  timeout: 60s
```

After execution, the model's response is stored under `get('llm')` (the `actionId`). Downstream resources read it with `get('llm')`.

## Full Configuration Reference

```yaml
# resources/llm.yaml
actionId: myLlm

chat:
  model: llama3.2:1b           # model name, or "router" to delegate to config routing
  role: user                   # "user" or "system"; defaults to "user"
  prompt: "&#123;&#123; get('q') &#125;&#125;"     # the prompt; supports expression interpolation
  timeout: 60s                 # max wait for model response; default 60s
  jsonResponse: false          # if true, prompt the model for JSON and parse response
  
  # Sampling parameters (optional)
  temperature: 0.7             # 0.0 = deterministic, 2.0 = very random; default varies by model
  maxTokens: 1000              # hard cap on generated tokens
  topP: 0.9                    # nucleus sampling threshold (0.0 to 1.0)
  frequencyPenalty: 0.0        # penalise repeated tokens (-2.0 to 2.0)
  presencePenalty: 0.0         # penalise any already-seen token (-2.0 to 2.0)
  contextLength: 8192          # context window in tokens
  streaming: false             # Ollama only: stream NDJSON; kdeps accumulates before returning

  # Optional: system message sets context before the user prompt
  systemPrompt: |
    You are a concise technical assistant. Answer in 3 sentences or fewer.
    Never speculate beyond what is asked.
  
  # Optional: pre-seed the conversation before the user prompt
  scenario:
    - role: system
      prompt: "You are a helpful assistant."
      cacheControl: ephemeral    # Anthropic only: mark this message for server-side caching
    - role: assistant
      prompt: "I am ready to help!"
    
  # Optional: conversation history for multi-turn
  messages:
    - role: user
      content: "&#123;&#123; get('userMessage') &#125;&#125;"
    - role: assistant
      content: "&#123;&#123; get('assistantReply') &#125;&#125;"
  
  # Optional: attach files for vision-capable models
  files:
    - "&#123;&#123; get('image', 'filepath') &#125;&#125;"
      
  # Optional: expose components as LLM tools
  componentTools:
    - scraper
    - search

  # Optional: chain-of-thought reasoning prefix
  chainOfThought: false          # if true, injects a CoT system prompt prefix before the user prompt

  # Optional: semantic few-shot example selection
  fewShotEmbeddingModel: ""      # embedding model for selecting few-shot examples by similarity
  fewShotEmbeddingBackend: ""    # backend for that embedding model (openai, ollama, etc.)

  # Optional: additional candidate/length controls
  candidateCount: 1              # number of candidate responses to generate (provider-dependent)
  n: 1                           # alias for candidateCount (OpenAI-style)
  minLength: 0                   # minimum response length in tokens (provider-dependent)
  maxLength: 0                   # maximum response length in tokens (alias for maxTokens on some backends)
```

## Structured JSON Output

When you need structured data from the model — not free text — use `jsonResponse: true`:

```yaml
# resources/extract.yaml
actionId: extract
chat:
  model: llama3.2:1b
  jsonResponse: true
  prompt: |
    Extract the following fields from this text as JSON:
    - company_name (string)
    - founded_year (integer)
    - employee_count (integer or null if not mentioned)
    
    Text:
    &#123;&#123; get('rawText') &#125;&#125;
    
    Return only valid JSON. No explanation.
```

With `jsonResponse: true`, kdeps:
1. Appends "Respond with valid JSON only" context to the system prompt
2. Parses the model's response as JSON
3. Stores the parsed object in the data store

Downstream expressions can then access fields directly:

```yaml
after:
  - set('company', get('extract').company_name)
  - set('year', get('extract').founded_year)
```

If the model returns malformed JSON, the resource fails with a parse error.

### jsonResponseKeys

When you only need specific fields from the JSON response, use `jsonResponseKeys:` to tell the model exactly which keys to include:

```yaml
chat:
  model: llama3.2:1b
  jsonResponse: true
  jsonResponseKeys:
    - label
    - confidence
    - reason
  prompt: "Classify this email as spam/not-spam: &#123;&#123; get('email') &#125;&#125;"
```

kdeps instructs the model to return only the listed keys. This reduces token usage, eliminates extraneous fields, and makes downstream `get()` calls more predictable.

The output is still a parsed JSON object — `get('classify').label`, `get('classify').confidence` — but limited to the declared keys.

## System Prompts

Use `systemPrompt:` for persistent context that should frame all responses from this resource:

```yaml
# resources/classifier.yaml
actionId: classifier
chat:
  model: llama3.2:1b
  systemPrompt: |
    You are a document classifier. Given a document, you return exactly one of these labels:
    INVOICE, CONTRACT, REPORT, EMAIL, OTHER
    
    You return only the label. No explanation. No punctuation.
  prompt: "Classify this document:\n\n&#123;&#123; get('content') &#125;&#125;"
```

System prompts are good for:
- Persona definition ("You are a...")
- Output format constraints ("Return only JSON")
- Domain context ("The following questions are about our product catalog")
- Behavioral constraints ("Do not speculate beyond the provided data")

## Multi-Turn Conversations

To maintain conversation history, store message pairs in the data store and feed them back into subsequent calls:

```yaml
# resources/chat-turn.yaml
actionId: chatTurn
chat:
  model: llama3.2:1b
  systemPrompt: "You are a helpful assistant."
  messages:
    - role: user
      content: "&#123;&#123; get('input') &#125;&#125;"
  
after:
  # Append this turn to history stored in session (Chapter 15 covers sessions)
  - set('history', get('history') + [{"role": "user", "content": get('input')}, {"role": "assistant", "content": get('chatTurn')}], 'session')
```

For stateless APIs where the client maintains history, accept the message array in the request body:

```yaml
chat:
  model: llama3.2:1b
  messages: "&#123;&#123; get('messages') &#125;&#125;"    # client sends [{role, content}, ...] array
```

## tools: — Function Calling (Resource-Based Tools)

`tools:` lets the LLM call other resources mid-response. When the LLM decides it needs a tool, kdeps runs the target resource, feeds the result back, and the LLM continues. This is native function calling — the LLM can invoke your pipeline logic, not just pre-built components.

```yaml
# resources/chat.yaml
actionId: chat
chat:
  model: llama3.2:1b
  prompt: "&#123;&#123; get('q') &#125;&#125;"
  tools:
    - name: calculate
      description: "Perform mathematical calculations. Use for any arithmetic."
      script: calcTool          # actionId of the resource to run
      parameters:
        expression:
          type: string
          description: "Math expression to evaluate, e.g. '(2 + 3) * 10'"
          required: true
    - name: database_lookup
      description: "Look up a user record by ID"
      script: userLookup
      parameters:
        user_id:
          type: string
          description: "The user's UUID"
          required: true
```

The target resources (`calcTool`, `userLookup`) are ordinary resources in `resources/`. When the LLM calls a tool, kdeps runs that resource with the LLM-supplied parameters available via `get()`, and returns the resource's output to the LLM.

**Tool definition fields:**

| Field | Description |
|---|---|
| `name` | Tool name the LLM uses (must be unique within the `tools:` list) |
| `description` | What the LLM reads to decide when to call this tool. Be specific. |
| `script` | `actionId` of the resource to execute when the tool is called |
| `parameters` | Map of parameter name → `{type, description, required}` |

**Parameter types:** `string`, `number`, `integer`, `boolean`, `object`, `array`

**Example: calculator tool**

```yaml
# resources/calcTool.yaml
actionId: calcTool
python:
  script: |
    import json, math
    expression = """&#123;&#123; get('expression') &#125;&#125;"""
    safe = {k: getattr(math, k) for k in dir(math) if not k.startswith('_')}
    safe.update({'abs': abs, 'round': round, 'min': min, 'max': max})
    try:
        result = eval(expression, {"__builtins__": {&#125;&#125;, safe)
        print(json.dumps({"result": result}))
    except Exception as e:
        print(json.dumps({"error": str(e)}))
```

The LLM can call `calculate` as many times as needed during its response. Each call executes `calcTool` and returns the result. The LLM's final output is what gets stored as the `chat` resource's output.

### MCP External Tools

Instead of `script:`, use `mcp:` to call a tool on an external MCP (Model Context Protocol) server. kdeps spawns the server as a subprocess, performs the JSON-RPC initialize handshake, calls the tool, and shuts the process down. This lets you use any MCP-compatible tool server without writing a kdeps resource.

```yaml
# resources/chat.yaml
chat:
  model: llama3.2:1b
  prompt: "&#123;&#123; get('q') &#125;&#125;"
  tools:
    - name: read_file
      description: "Read the contents of a file from the workspace"
      mcp:
        server: npx
        args: ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]
        transport: stdio     # only "stdio" supported
        env:
          HOME: /tmp         # env vars injected into the subprocess
      parameters:
        path:
          type: string
          description: "Absolute path of the file to read"
          required: true
```

**MCP tool fields:**

| Field | Description |
|---|---|
| `server` | Executable to start the MCP server (`npx`, `uvx`, path to binary) |
| `args` | Arguments passed to the executable |
| `transport` | Transport type — only `stdio` is supported |
| `env` | Environment variables injected into the subprocess |

`mcp:` and `script:` are mutually exclusive per tool entry. A fresh subprocess is started for each tool invocation.

**When to use `mcp:` vs `script:`:**
- Use `script:` when the tool logic is a kdeps resource you control — the code lives in your workflow.
- Use `mcp:` when the tool is a pre-built MCP server you want to consume without writing glue code (filesystem access, GitHub API, Stripe, Slack, etc.).

`tools:` is for resource-based or MCP-based function calling. `componentTools:` (below) is for registry-installed components.

## componentTools: — Component-Based Tools

In agent mode, `componentTools:` registers installed components as function-calling tools the LLM can invoke directly during a `chat:` call:

```yaml
# resources/research.yaml
actionId: research
chat:
  model: llama3.2:1b
  prompt: "Research this topic thoroughly: &#123;&#123; get('topic') &#125;&#125;"
  componentTools:
    - scraper
    - search
    - embedding
```

When the LLM processes this prompt, it has access to three tools. It can call `scraper` to fetch URLs, `search` to find relevant pages, and `embedding` to look up similar stored content — all within a single `chat:` resource execution.

The component's `interface.inputs` schema becomes the tool's parameter schema. The LLM uses this to pass correct arguments. See Chapter 12 for components in depth.

## Timeouts

The default timeout is 60 seconds. For models that require longer processing (large context windows, complex reasoning tasks), increase it:

```yaml
chat:
  model: llama3.2:70b     # larger model = slower response
  timeout: 300s            # 5 minutes
  prompt: "&#123;&#123; get('longDocument') &#125;&#125;"
```

If the model does not respond within the timeout, the resource fails with a timeout error. Set timeouts that reflect the actual worst-case response time for your model and context size.

## Configuring Backends

The backend — where the LLM call actually goes — is configured in `~/.kdeps/config.yaml`, separate from the workflow.

### Llamafile (Local, the Default)

With no configuration at all, chat resources run on the `file` backend. The
model is a [llamafile](https://github.com/Mozilla-Ocho/llamafile): a single
self-contained binary that kdeps downloads to `~/.kdeps/models/`, makes
executable, and serves as a local OpenAI-compatible server. Known aliases map
to Mozilla's HuggingFace builds, with the quantization encoded in the alias:

| Alias | Model | Quant | Size |
|-------|-------|-------|------|
| `llama3.2` / `llama3.2:1b` | Llama 3.2 1B Instruct | Q4_K_M | ~1.1 GB |
| `llama3.2:1b-q6` | Llama 3.2 1B Instruct | Q6_K | ~1.5 GB |
| `llama3.2:1b-q8` | Llama 3.2 1B Instruct | Q8_0 | ~2.1 GB |
| `llama3.2:3b` | Llama 3.2 3B Instruct | Q4_K_M | ~2.2 GB |
| `llama3.1:8b` | Llama 3.1 8B Instruct | Q4_K_M | ~5.2 GB |

`kdeps llamafile list` prints the full registry (100+ models);
`kdeps llamafile update` refreshes it from HuggingFace. The `model:` field
also accepts a direct URL, an absolute or relative path to a `.llamafile`,
or a bare filename looked up in `~/.kdeps/models/`.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: file   # this is the default - the line can be omitted entirely
```

No API key, no server install. Each distinct model gets one long-lived local
server process, started on demand and reused across requests. When building
Docker images, the referenced llamafiles are pre-baked into the image so
containers run offline.

### GGUF (llama.cpp)

The `gguf` backend serves GGUF model files via `llama-server` (llama.cpp). Same download-and-cache flow as `file`, but requires `llama-server` installed separately (override binary path with `KDEPS_LLAMA_SERVER_BIN`). Set `KDEPS_GGUF_CTX_SIZE` to change the context window.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: gguf
```

Known aliases:

| Alias | Model | Quant | Size |
|-------|-------|-------|------|
| `qwen3.5-4b` | Qwen3.5 4B | Q5_K_S | ~3.1 GB |
| `qwen3.5-8b` | Qwen3.5 8B | Q4_K_M | ~5.0 GB |
| `llama3.2-3b` | Llama 3.2 3B Instruct | Q5_K_M | ~2.4 GB |
| `llama3.1-8b` | Llama 3.1 8B Instruct | Q4_K_M | ~4.9 GB |
| `phi4-mini` | Phi-4 Mini | Q5_K_M | ~2.7 GB |
| `gemma3-4b` | Gemma 3 4B | Q5_K_M | ~3.1 GB |
| `mistral-7b` | Mistral 7B v0.3 | Q4_K_M | ~4.4 GB |
| `deepseek-r1-7b` | DeepSeek-R1 Distill 7B | Q4_K_M | ~5.0 GB |

The `model:` field also accepts a direct HuggingFace URL, an absolute/relative path to a `.gguf`, or a bare filename in `~/.kdeps/models/`.

### Ollama (Local, Opt-In)

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: ollama
  # base_url: http://custom-ollama:11434   # optional override
```

No API key required. Models must be pulled first with `ollama pull <model>`. When building Docker images, Ollama is automatically installed when `backend: ollama` is set, or when the workflow sets `installOllama: true`.

### OpenAI

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  openai_api_key: "sk-..."
```

Model names: `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `gpt-3.5-turbo`.

### Anthropic

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: anthropic
  anthropic_api_key: "sk-ant-..."
```

Model names: `claude-sonnet-4-20250514`, `claude-3-5-sonnet-20241022`, `claude-3-opus-20240229`, `claude-3-haiku-20240307`.

### Groq

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: groq
  groq_api_key: "gsk_..."
```

Groq offers ultra-fast inference. Current models: `llama-3.1-70b-versatile`, `llama-3.1-8b-instant`, `mixtral-8x7b-32768`, `gemma2-9b-it`.

### Google (Gemini)

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: google
  google_api_key: "..."
```

Model names: `gemini-1.5-pro`, `gemini-1.5-flash`, `gemini-pro`.

### Mistral

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: mistral
  mistral_api_key: "..."
```

Model names: `mistral-large-latest`, `mistral-medium-latest`, `mistral-small-latest`, `open-mistral-7b`, `open-mixtral-8x7b`.

### Cohere

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: cohere
  cohere_api_key: "..."
```

Model names: `command-r-plus`, `command-r`, `command`, `command-light`.

### DeepSeek

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: deepseek
  deepseek_api_key: "..."
```

Model names: `deepseek-chat`, `deepseek-coder`.

### Together AI

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: together
  together_api_key: "..."
```

Hosts many open-source models: `meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo`, `mistralai/Mixtral-8x7B-Instruct-v0.1`, `Qwen/Qwen2-72B-Instruct`.

### Perplexity

Search-augmented responses — Perplexity models have live web access built in.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: perplexity
  perplexity_api_key: "..."
```

Models: `llama-3.1-sonar-large-128k-online` (large, with web search), `llama-3.1-sonar-small-128k-online` (fast, with web search), `llama-3.1-sonar-large-128k-chat` (large, chat only).

### xAI (Grok)

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: xai
  xai_api_key: "xai-..."
```

Model names: `grok-2`, `grok-beta`, `grok-vision-beta`.

### OpenRouter

Access 100+ models from multiple providers through a single API key.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openrouter
  openrouter_api_key: "sk-or-..."
```

Model names use the `provider/model` format: `openai/gpt-4o`, `anthropic/claude-3.5-sonnet`, `meta-llama/llama-3.1-70b-instruct`. Full list at openrouter.ai/models.

### Any OpenAI-Compatible Endpoint

Any server that implements the OpenAI chat completions API works with kdeps — vLLM, Text Generation Inference (TGI), LocalAI, LlamaCpp Server:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  openai_api_key: "your-key"
  base_url: "http://your-vllm-server:8000/v1"
  # or http://localhost:8000/v1  (local vLLM)
```


### kdeps LLM server appliance

To **provision** a self-hosted OpenAI-compatible server (Docker, Kubernetes, or ISO) instead of only pointing at someone else's endpoint, use `kdeps llm`. No workflow path — select an engine recipe and model, then configure clients with `backend: openai` and `base_url`.

Stock engines include `ollama`, `llamafile`, `llama-server` / `gguf`, `llamacpp`, `vllm`, `tgi`, `sglang`, and `localai`.

```bash
$ kdeps llm list
$ kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
$ kdeps llm build --engine vllm --model facebook/opt-125m --gpu cuda --tag myorg/vllm:1
$ kdeps llm client-config --url http://192.168.1.50:8000/v1
```

Full guide: **Chapter 26**. Docs: `docs/v2/deployment/llm-server.md`.


### Per-Resource Backend Overrides

In some workflows, different resources should use different backends. Override at the resource level:

```yaml
# resources/fast-classify.yaml
actionId: classify
chat:
  model: llama3.2:1b
  backend:
    provider: ollama
    baseUrl: http://localhost:11434
  prompt: "Classify: &#123;&#123; get('input') &#125;&#125;"

# resources/deep-analyze.yaml
actionId: analyze
requires: [classify]
chat:
  model: gpt-4o
  backend:
    provider: openai
    apiKey: "&#123;&#123; env('OPENAI_API_KEY') &#125;&#125;"
  prompt: "Analyze in depth: &#123;&#123; get('classify') &#125;&#125;"
```

This lets you use a fast local model for cheap classification and a large frontier model for deep analysis — in the same workflow, controlled per resource.

## Environment Variables in Config

Never hardcode API keys in workflow files. Use `env()` to read from the environment:

```yaml
chat:
  backend:
    provider: openai
    apiKey: "&#123;&#123; env('OPENAI_API_KEY') &#125;&#125;"
```

Or set them in `agentSettings.envVars` in `workflow.yaml`:

```yaml
# workflow.yaml
settings:
  agentSettings:
    envVars:
      OPENAI_API_KEY: "${OPENAI_API_KEY}"   # propagated from shell environment
```

See Chapter 14 for the full `agentSettings` reference.

## Sampling Parameters

When you need fine-grained control over the model's output distribution, use the sampling fields:

```yaml
chat:
  model: llama3.2:7b
  temperature: 0.2       # low = focused and deterministic; good for extraction/classification
  maxTokens: 500         # hard cap on response length; use to control cost
  topP: 0.9              # nucleus sampling; lower = more conservative vocabulary
  frequencyPenalty: 0.5  # reduces repetition; useful for longer text generation
  presencePenalty: 0.3   # encourages topic diversity; useful for creative tasks
  contextLength: 16384   # extend context window for long document tasks
```

**Practical guidance:**
- `temperature: 0.0–0.3` for extraction, classification, and code generation where correctness matters
- `temperature: 0.7–1.0` for creative tasks, diverse responses, or brainstorming
- `maxTokens` is a hard stop — the model will truncate mid-sentence if it hits the limit; set it higher than your worst case
- Not all parameters are supported by all backends; unsupported fields are silently ignored

## Vision (Multimodal Input)

For vision-capable models, attach files to the prompt using `files:`:

```yaml
# resources/describe-image.yaml
actionId: describeImage
chat:
  model: llama3.2-vision   # must be a vision-capable model
  prompt: "Describe what is in this image in detail."
  files:
    - "&#123;&#123; get('image', 'filepath') &#125;&#125;"    # uploaded file path
```

The `files:` list accepts file paths. Files are sent to the model alongside the text prompt. Use `get('fieldName', 'filepath')` to reference uploaded files from the request.

Models with vision support: `llama3.2-vision` (Ollama), `gpt-4o` and `gpt-4-turbo` (OpenAI), all Claude 3+ models, `gemini-1.5-pro` (Google).

## Model Routing

For production workflows, `model: router` delegates model selection from the resource file to the router configured in `~/.kdeps/config.yaml`. This lets you change which model handles a resource without editing resource files:

```yaml
# resources/analyze.yaml
chat:
  model: router    # delegate selection to ~/.kdeps/config.yaml
  prompt: "&#123;&#123; get('q') &#125;&#125;"
```

```yaml
# ~/.kdeps/config.yaml
llm:
  strategy: fallback
  models:
    - model: claude-sonnet-4-20250514
      backend: anthropic
      priority: 1
    - model: gpt-4o
      backend: openai
      priority: 2
    - model: llama3.2:7b
      backend: ollama
      priority: 3
      default: true
```

### Routing Strategies

**`fallback`** — tries models in priority order; on error, automatically retries the next:

```yaml
llm:
  strategy: fallback
  models:
    - model: gpt-4o
      backend: openai
      priority: 1
    - model: llama3.2:7b    # local fallback if API is unreachable
      backend: ollama
      priority: 2
      default: true
```

**`token_threshold`** — routes by estimated prompt token count; first matching range wins:

```yaml
llm:
  strategy: token_threshold
  models:
    - model: gpt-4o-mini
      backend: openai
      max_tokens: 500        # short prompts use this
      default: true
    - model: gpt-4o
      backend: openai
      min_tokens: 501        # long prompts need the larger context window
```

**`cost_optimized`** — selects the cheapest model based on cost per 1K input tokens:

```yaml
llm:
  strategy: cost_optimized
  models:
    - model: gpt-4o-mini
      backend: openai
      cost_per_input_token: 0.00015    # $0.15/1M tokens
    - model: gpt-4o
      backend: openai
      cost_per_input_token: 0.0025     # $2.50/1M tokens
      default: true
```

**`round_robin`** — distributes requests evenly across all configured models:

```yaml
llm:
  strategy: round_robin
  models:
    - model: gpt-4o
      backend: openai
    - model: claude-sonnet-4-20250514
      backend: anthropic
```

### Allowlist Mode

Without a `strategy:`, `llm.models` is a simple allowlist. Any request for an unlisted model is overridden to the first model and a warning is logged:

```yaml
llm:
  backend: ollama
  models:
    - llama3.2:1b
    - llama3.2:7b
    - nomic-embed-text
```

Models in the allowlist are pre-pulled into Docker/ISO artifacts when building deployment images.

## Streaming (Ollama)

Set `streaming: true` to have Ollama stream the response as NDJSON chunks. kdeps accumulates all chunks internally and returns the same response shape as a non-streaming call — your downstream resources see no difference:

```yaml
chat:
  model: llama3.2:7b
  prompt: "&#123;&#123; get('q') &#125;&#125;"
  streaming: true      # Ollama only; silently ignored for other backends
```

## Chain-of-Thought and Few-Shot Selection

### Chain-of-Thought

Set `chainOfThought: true` to inject a reasoning prefix into the system prompt. The model is instructed to work through the problem step by step before producing its final answer. This is useful for tasks that benefit from explicit intermediate reasoning -- math problems, multi-step logic, or classification with edge cases:

```yaml
chat:
  model: gpt-4o
  chainOfThought: true
  prompt: "&#123;&#123; get('problem') &#125;&#125;"
```

The prefix is injected automatically. You do not need to write "think step by step" in your own system prompt.

### Semantic Few-Shot Selection

When you maintain a library of example prompt-response pairs, `fewShotEmbeddingModel` and `fewShotEmbeddingBackend` let kdeps select the most relevant examples for each request by semantic similarity rather than static ordering:

```yaml
chat:
  model: gpt-4o
  fewShotEmbeddingModel: text-embedding-3-small   # embedding model for ranking examples
  fewShotEmbeddingBackend: openai                  # backend for that model
  prompt: "&#123;&#123; get('q') &#125;&#125;"
```

The embedding model encodes the incoming prompt and scores each candidate example. The top-ranked examples are injected into the conversation as few-shot context. This is more token-efficient than including all examples and more accurate than random selection.

## Provider-Specific Options

### Google AI (Gemini and Vertex AI)

**Cached content** -- Google AI lets you pre-cache large context objects (documents, system instructions, tool descriptions) as named `CachedContent` resources. Reference a cached object by name so it is not re-sent on every request:

```yaml
chat:
  model: gemini-1.5-pro
  googleCachedContent: "cachedContents/my-doc-cache-v1"   # Google AI CachedContent resource name
  prompt: "Summarize the key findings."
```

This reduces per-request token cost when the same large context is reused across many calls.

**Safety filter threshold** -- Controls how aggressively the safety filters block content. The default is `0` (unspecified, uses the model's built-in defaults):

```yaml
chat:
  model: gemini-1.5-pro
  googleHarmThreshold: 1   # 0=unspecified, 1=block-none, 2=block-few, 3=block-some, 4=block-most
```

Use `1` (block-none) only in server-to-server workflows where you control the input. Use `4` (block-most) for consumer-facing applications.

**Vertex AI** -- To use Vertex AI instead of the standard Google AI API, provide your GCP project and region:

```yaml
chat:
  model: gemini-1.5-pro
  googleCloudProject: my-gcp-project    # GCP project ID
  googleCloudLocation: us-central1      # Vertex AI region
```

Authentication uses Application Default Credentials (`gcloud auth application-default login`).

### Anthropic

**Prompt caching** -- Server-side caching reduces cost and latency for large, reused context blocks (system prompts, tool lists, document prefixes). Enable with `promptCaching: true`:

```yaml
chat:
  model: claude-sonnet-4-20250514
  promptCaching: true      # sends the anthropic-beta: prompt-caching-2024-07-31 header
  systemPrompt: |
    You are an expert legal analyst with access to a large case library.
    [... thousands of tokens ...]
```

For per-message granularity, mark individual `scenario` items with `cacheControl: ephemeral`:

```yaml
scenario:
  - role: system
    prompt: "&#123;&#123; get('longSystemDoc') &#125;&#125;"
    cacheControl: ephemeral    # cache this message block server-side
```

**Extended output** -- Enable up to 128K output tokens (instead of the default 8K) for long document generation:

```yaml
chat:
  model: claude-sonnet-4-20250514
  anthropicExtendedOutput: true   # enables anthropic-beta: max-tokens-3-5-sonnet-2024-07-15
  maxTokens: 16000
  prompt: "Write a detailed technical specification for &#123;&#123; get('project') &#125;&#125;"
```

**Custom beta headers** -- Pass additional Anthropic beta feature headers when you need features not yet covered by a dedicated field:

```yaml
chat:
  model: claude-sonnet-4-20250514
  anthropicBetaHeaders:
    - "interleaved-thinking-2025-05-14"
    - "output-128k-2025-02-19"
```

### OpenAI and Compatible Endpoints

**Legacy max tokens** -- Older OpenAI-compatible servers (some vLLM builds, LocalAI) use the deprecated `max_tokens` parameter name instead of `max_completion_tokens`. Enable compatibility mode:

```yaml
chat:
  model: my-model
  backend:
    provider: openai
    baseUrl: http://localhost:8000/v1
  openAILegacyMaxTokens: true    # sends max_tokens instead of max_completion_tokens
  maxTokens: 2000
```

Only set this when you get errors about unrecognized `max_completion_tokens` from a local server. Do not set it for the standard OpenAI API.

### Ollama Native Options

Ollama exposes several options not available in its OpenAI-compatibility layer. Access them with the `ollama*` fields:

```yaml
chat:
  model: qwen3:14b
  ollamaThink: true             # enable extended thinking (models that support it)
  ollamaKeepAlive: 10m          # how long to keep the model loaded; "0" = unload immediately
  ollamaPullModel: true         # pull the model from the registry if not already present
  ollamaPullTimeout: 10m        # max time to wait for a pull to complete
  prompt: "&#123;&#123; get('q') &#125;&#125;"
```

`ollamaKeepAlive` is useful in low-memory environments where you want the model evicted after a burst of requests. `ollamaPullModel: true` lets you reference models that may not be pre-pulled yet -- kdeps will pull them before running the first call.

## Choosing the Right Model

A few guidelines for model selection in workflow resources:

**Use small local models for classification, routing, and extraction.** A 1B-7B model is fast, cheap, and accurate enough for "classify this as X or Y" or "extract these fields from text." Avoid using large models for tasks a small model handles well.

**Use medium models for summarization and rewriting.** 7B-13B models (Llama 3 8B, Mistral 7B) handle most summarization and text transformation tasks well.

**Reserve large models for complex reasoning.** GPT-4o, Claude Sonnet, Llama 3 70B for tasks requiring synthesis across large contexts, multi-step reasoning, or high-stakes output quality.

**Match timeout to model size.** A 1B model on CPU might respond in 2-3 seconds. A 70B model might need 60-120 seconds. Set timeouts that reflect actual hardware, not theoretical throughput.

## The Output

After a `chat:` resource executes, its output is stored in the data store under its `actionId`. By default, the output is a string (the model's response text). With `jsonResponse: true`, it is a parsed JSON object.

```yaml
# reads string output
answer: get('myLlm')

# reads JSON field from structured output
company: get('myExtract').company_name
score: get('myClassifier').confidence
```

In the next chapter, we look at the data resources: SQL, HTTP, Python, and Exec — the resources that integrate your AI workflows with the rest of your infrastructure.

X> ## Exercise
X>
X> Build a workflow that extracts structured data from a freeform product review. The endpoint accepts `POST /api/v1/review` with a `text` field and returns a JSON object with these exact fields: `sentiment` (positive/negative/neutral), `score` (1–5 integer), `topics` (array of strings), `summary` (one sentence).
X>
X> Requirements:
X>
X> 1. Use `responseFormat:` with a JSON schema that enforces the exact field types above. The LLM must return valid structured output — not freeform text.
X> 2. Add a `validations.check` that rejects reviews shorter than 10 characters.
X> 3. Test with at least two reviews — one clearly positive, one mixed.
X>
X> ```bash
X> curl -X POST localhost:16395/api/v1/review \
X>   -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
X>   -H "Content-Type: application/json" \
X>   -d '{"text":"Great headphones, amazing sound quality but the ear cups hurt after 2 hours."}'
X> ```
X>
X> Verify that `sentiment`, `score`, `topics`, and `summary` are all present in the response and match the correct types. If the LLM returns a string for `score` instead of an integer, adjust the schema and prompt until it is correct.
X>
X> **Stretch goal:** Add a `systemPrompt:` that instructs the model to be conservative with 5-star scores, then compare output with and without it for the same review.
