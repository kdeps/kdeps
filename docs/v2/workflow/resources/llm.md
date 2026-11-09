# LLM resource

The `chat:` resource sends a prompt to a language model and stores the response as the resource's output. The output is the raw response object - the reply text is at `get('id').message.content`. With `json: true` under `harness.output` the output is the parsed JSON object instead.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool.

## Key groups

A chat's keys sit in six groups:

| Group | Holds |
|---|---|
| `model` | `name`, `backend`, `timeout`, `streaming`, `thinking`, caching, sampling (`temperature`, `maxTokens`, ...), provider fields (`ollama*`, `google*`, ...) |
| `context` | `prompt`, `role`, `messages`, `scenario`, `files`, `promptVars`, `goTemplate`, `contextLength`, `fewShot` |
| `harness` | `sections`, `refine`, `chainOfThought`, `output` (`json`, `keys`, `schema`, `parser`) |
| `tools` | `list`, `components`, `choice`, `maxRounds`, `skills` |
| `rag` | `knowledge`, `retriever` (`context`, `topK`, `maxTokens`), `chunk` |
| `guardrails` | `judges`, `judgePreset` |

## Where config lives

Model selection goes in the resource file. Backend and API keys go in `~/.kdeps/config.yaml`. This lets you change backends without touching your workflow.

```yaml
# resources/my-resource.yaml
chat:
  model:
    name: llama3.2:1b    # which model to call
  context:
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

With the default `file` backend, `name: llama3.2:1b` resolves to a local
llamafile that is downloaded once (~1.1 GB) and served automatically - see
[LLM backends](/workflow/resources/llm-backends#the-default-llamafile-file-backend).

Set `name: router` to delegate model selection to the router configured in `~/.kdeps/config.yaml`. See [LLM backends](/workflow/resources/llm-backends) for routing strategies.

Set `name: system` (and/or `backend: system`) to follow whatever the machine is configured to use in `~/.kdeps/config.yaml`. It resolves exactly like an omitted field, but says so on purpose.

```yaml
# resources/chat.yaml
chat:
  model:
    backend: system   # use llm.backend from ~/.kdeps/config.yaml
    name: system     # use llm.models / the router
  context:
    prompt: "{{ get('q') }}"
```

If a `chat` resource uses a cloud model (e.g. `name: deepseek-chat`) whose provider key is missing from `config.yaml`, `kdeps run` prompts for the key at startup and saves it as `<provider>_api_key` under `llm:`. With no default backend set, it also points `backend:` at that provider. Interactive only; CI and pipes skip the prompt.

## Model resolution order

`model.name` is optional. When omitted, kdeps resolves it in this order:

```text
resource model:  ->  config router (llm.models + strategy)  ->  first llm.models entry
  ->  best installed model by hardware fit (llmfit)  ->  built-in llama3.2:1b (file backend)
```

- A resource with an explicit `name:` always wins - unless it is the literal `system`, which resolves down this same list.
- Omit `name:` to inherit the machine default, first match wins:
  1. the LLM router, when `strategy:` is set under `llm:`
  2. the first entry of `models:` under `llm:`
  3. on the local file backend, the downloaded model that best fits this machine, when [`llmfit`](/agent/models#how-a-model-is-picked-when-none-is-configured) is installed and a local model is downloaded
  4. the built-in `llama3.2:1b` on the local file backend
- A cloud, gguf or ollama backend with no model and no `models:` list errors at run time; kdeps will not guess a model that backend cannot serve. Set `name:` on the resource or `models:` under `llm:` in `config.yaml`.

## Basic usage

<div v-pre>

```yaml

actionId: llmResource
name: LLM Chat
chat:
  model:
    timeout: 60s                  # fail the resource if the model has not answered in 60s
  context:
    prompt: "{{ get('q') }}"
```

</div>

## Complete reference

<div v-pre>

```yaml
# resources/example.yaml
chat:
  model:
    name: llama3.2:1b    # model name, or "router" to delegate to config
    temperature: 0.7      # 0.0 = deterministic, 2.0 = very random
    maxTokens: 1000       # hard cap on generated tokens
    topP: 0.9             # nucleus sampling threshold (0.0 to 1.0)
    frequencyPenalty: 0.0 # penalises tokens that have appeared frequently (-2.0 to 2.0)
    presencePenalty: 0.0  # penalises any token that has appeared at all (-2.0 to 2.0)
  context:
    role: user            # role for this message: user, assistant, system

    prompt: "{{ get('q') }}"  # the prompt; supports {{ }} interpolation

    contextLength: 8192   # context window in tokens (4096, 8192, 16384, ...)
```

</div>

## History, tools, and files

<div v-pre>

```yaml
# resources/example.yaml
chat:
  context:
    # pre-fill the conversation history before the prompt
    scenario:
      - role: system
        prompt: You are a helpful assistant.
      - role: assistant
        prompt: I am ready to help!

    # attach files for vision-capable models
    files:
      - "{{ get('file', 'filepath') }}"
  tools:
    # let the LLM call other resources as functions
    list:
      - name: calculate
        description: Perform math
        script: calcResource   # actionId of the resource to call
        parameters:
          expression:
            type: string
            required: true

```

</div>

## JSON shape, timeout, and streaming

```yaml
# resources/example.yaml
chat:
  model:
    timeout: 60s               # hard stop -- returns error, does not retry
    streaming: true            # Ollama only: stream NDJSON; kdeps accumulates before returning
  harness:
    output:
      json: true         # ask the model to return valid JSON
      keys:          # keys to extract from the JSON response
        - answer
        - confidence

      # strict JSON schema (OpenAI models only) - enforces exact output shape
      # use this instead of keys when you need guaranteed structure
      schema:
        type: object
        properties:
          answer:
            type: string
          confidence:
            type: number
        required:
          - answer
          - confidence

```

## Reasoning, few-shot, and extra sampling

```yaml
# resources/example.yaml
chat:
  model:
    # sampling extras
    candidateCount: 1          # number of independent completions (alias: n)
    n: 1                       # alias for candidateCount (OpenAI style)
    minLength: 0               # minimum response length in tokens
    maxLength: 0               # maximum response length in tokens (alias for maxTokens)
  context:
    fewShot:
      embeddingModel: ""  # embedding model for semantic few-shot selection (requires fewShotSelectK)
      embeddingBackend: "" # backend serving the embedding model above

  harness:
    # chain-of-thought / few-shot
    chainOfThought: false      # inject step-by-step reasoning prefix into system prompt
```

## Provider fields

```yaml
# resources/example.yaml
chat:
  model:
    # anthropic-specific
    promptCaching: false           # add prompt-caching beta header; Anthropic only
    anthropicExtendedOutput: false # enable 128K output; adds interleaved-thinking beta header
    anthropicBetaHeaders: []       # additional anthropic-beta header values

    # openai-specific
    openAILegacyMaxTokens: false   # send max_tokens instead of max_completion_tokens
    openAIWebSearch: false         # OpenAI server-side web search; search-capable models only

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

## Advanced parameters

Full sampling control - all fields are optional and fall back to the model's provider defaults.

```yaml
# resources/example.yaml
chat:
  model:
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
  context:
    prompt: "Write a creative story"
```

## Context length

Control the context window:

```yaml
# resources/example.yaml
chat:
  context:
    contextLength: 8192  # Options: 4096, 8192, 16384, 32768, 65536, 131072, 262144
```

## Knowledge recall

Every `chat` call adds [knowledge vault](/agent/knowledge) notes related to its prompt by default. kdeps refreshes the structural graph (links, tags, frontmatter; no model call), walks it from the prompt's words, and puts the best few note paths and summaries, within the recall token budget, in a system message after the system prompt. Nothing is added when the vault is off, the resource sets `knowledge: false`, or no note matches. Calls write to the vault only with `workflow_notes: true` under `knowledge:`, which logs each call to the workflow's project (see [Workflow mode](/agent/knowledge#workflow-mode)).

<div v-pre>

```yaml
# resources/answer.yaml
actionId: answer
name: Answer
chat:
  model:
    name: llama3.2:1b
  context:
    prompt: "{{ get('q') }}"
  rag:
    knowledge: false     # skip recall for this call; default true
```

</div>

`knowledge.enabled: false` in `~/.kdeps/config.yaml` turns recall off for every call.

## Scenario (conversation history)

Build multi-turn conversations:

<div v-pre>

```yaml
# resources/example.yaml
chat:
  context:
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

## JSON response

Get structured JSON output:

<div v-pre>

```yaml
# resources/example.yaml
chat:
  context:
    prompt: "Analyze: {{ get('q') }}"
  harness:
    output:
      json: true
      keys:
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

## Judges

`guardrails.judges` reviews the answer before the resource finishes. The judges run in parallel; if any rejects, the chat runs again with their feedback, and the panel reviews the revision. Agent mode's [judge panel](/agent/judges) runs on the same engine.

```text
chat -> judges (parallel) -> all approve? -> output
              | any rejects
              v
        chat again with the feedback -> judges again
```

<div v-pre>

```yaml
# resources/notes.yaml
chat:
  context:
    prompt: "Write release notes for {{ get('version') }}"
  tools:
    list:
      - name: changelog            # a tool of this chat; judges may borrow it
        description: Read the changelog
        script: changelogTool      # the resource that runs the tool
  guardrails:
    judges:
      - name: complete
        criteria: Every change in the changelog is listed.   # the rubric the judge reviews against
        tools: [changelog]         # only this chat's tools or tools.components
        maxRounds: 5               # its tool rounds; 0 = the judge-max-rounds event
      - name: format
        graders:                   # checks without a model call; a failure rejects
          - kind: not_contains     # exact | contains | not_contains | regex | json | expr
            pattern: TODO
```

</div>

- Each judge uses the chat's model and replies with a JSON verdict; it sees the rendered prompt and the answer, not the conversation.
- Reviews per answer: the `judge-iterations` event (2). The last revision is the output when the budget runs out.
- A judge that fails or replies without a verdict approves: judges never fail the resource.

`judgePreset` uses a preset's panel instead of listing judges (not both):

```yaml
chat:
  context:
    prompt: "Write a design for {{ get('feature') }}."
  tools:
    list:
      - name: read_file          # the preset's judges may use the chat tools they name
  guardrails:
    judgePreset: design-review   # assets/presets/design-review.yaml: scope, clarity, failure-modes, readiness
```

An unknown preset, or one with no judges, fails the resource.

## Harness sections

`harness.sections` adds agent mode's system-prompt sections to the chat, in the order listed, ahead of its other instructions:

```yaml
chat:
  context:
    prompt: "Summarize {{ get('url') }}."
  harness:
    sections: [accuracy, honesty, output]   # assets/harness/<name>.yaml, kind: preamble-section
```

Any `preamble-section` entry works (`accuracy`, `honesty`, `output`, `scope`, `safety`, ...), including your overrides in `~/.kdeps/harness/`. A section you disabled adds nothing; an unknown name fails the resource.

## Prompt refine

`refine: true` rewrites the prompt into a clearer, self-contained request before the call, as agent mode's `/refine` does:

```yaml
chat:
  context:
    prompt: "{{ get('q') }}"   # e.g. "fix it" -> "Fix the failing build in ./cmd and report what changed."
  harness:
    refine: true                 # one extra model call with the refine-system harness prompt
```

The chat's history is context for the rewrite. A trivial prompt, a failed refine call or an unusable reply keeps the prompt as written.

## Skills

`tools.skills` gives the chat agent mode's skills: each listed folder of `SKILL.md` files (or one skill file) is read, the skill names and descriptions join the system prompt, and the model reads one in full with the `load_skill` tool:

```yaml
chat:
  context:
    prompt: "Release {{ get('version') }}."
  tools:
    skills: [skills]   # skills/deploy/SKILL.md -> skill "deploy"; relative to the run's working folder
```

A path with no skills fails the resource. A skill marked `hidden: true` is not offered.

## Vision (file attachments)

Process images (set a vision-capable `name:` under the resource's `model:`):

<div v-pre>

```yaml
# resources/example.yaml
chat:
  model:
    name: llama3.2-vision
  context:
    prompt: "Describe this image"
    files:
      - "{{ get('file', 'filepath') }}"  # From upload
      - "./images/example.jpg"            # From filesystem
```

</div>

## Function calling

Enable LLMs to call other resources:

<div v-pre>

```yaml
# resources/llm-with-tools.yaml
actionId: llmWithTools
chat:
  context:
    prompt: "{{ get('q') }}"
  tools:
    list:
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

### Tool loop behavior

kdeps runs the tool loop one call at a time, regardless of backend: if a model
returns several tool calls in one turn, only the first is executed. A batched
call is executed on state the model only guessed at - it never saw the earlier
calls' real results - so trimming to one call per turn forces a real
step-by-step loop where each call reacts to the previous tool's actual output.
Set `KDEPS_ALLOW_MULTI_TOOL=1` to restore batched execution. `tools.maxRounds` on
the `chat:` resource still bounds how many one-at-a-time rounds the loop runs;
without it, the `chat-max-rounds` event does (5).

Agent mode's call guard applies per chat call: a [call-budget event](/agent/events-guards#call-budget-events) (`web-call-budget`, or your own with `tools:` and `distinctCalls:`) blocks new distinct calls to its tools once spent. A call the model repeats gets its first result back without running again, and the chat answers once a call repeats past `identical-tool-calls` (3). A tool result the model sees is cut to the `tool-result-truncate` event (8192 bytes, start and end kept) and a tool error to `tool-error-truncate` (500).

A model that writes a tool call as text instead of a structured call (`<tool_call>`, `<invoke>`, `<function=...>`, DeepSeek DSML, or a bare JSON call) still runs it, parsed as in [agent mode](/agent/tools-fenced). A call inside a markdown fence is an example and does not run, and a call to a tool the chat does not list is ignored.

The answer of a chat with tools is cleaned as in agent mode: `[MEMORY: ...]` markers, tool-call markup, a self-written `<tool_response>` and `<File>` wrappers are stripped. A `harness.output.json` chat is left to its JSON parser.

Each model call is checked against the backend's [request body limit](/reference/security#request-body-size-preflight) first; a request over it fails with an actionable error and never reaches the model.

A failed tool call gets a failure brief after its error, as in agent mode: the folder of a missing path or the lines around a failed `edit_file`, and an instruction to fix the cause and call again. The `efficiency-failures` event (5) caps the briefed failures per chat call; later failures return the plain error. `failure-context-bytes`, `failure-dir-entries` and `failure-error-bytes` size the brief.

A call that overflows the model's context is compacted and retried once, as in agent mode: everything after the system messages is summarized with the `compaction-system` and `compaction-user` harness prompts, and the model resumes from that summary and the original prompt. A prompt that overflows on its own still fails.

### kdeps tools (by name)

A tool with no `script` or `mcp` is the kdeps tool of that name, the same one agent mode uses. Its description, parameters and code come from its asset, and the call runs through the kdeps API.

```yaml
# resources/research.yaml
chat:
  context:
    prompt: "{{ get('q') }}"
  tools:
    list:
      - name: web_search     # asset tool: no script needed
      - name: calculator
      - name: http_request
        description: Fetch one of our status pages   # optional: overrides the asset's description
```

An unknown name, or a tool that needs a live agent session (`session_ops`), fails the resource.

`subagent` hands a task to a fresh chat call on the same model. It has not seen this chat and gets only the chat's read-only tools (`read_file`, `web_search`, `calculator`, ...), never a write tool or another subagent. It returns a plain-text report as the tool result.

```yaml
chat:
  context:
    prompt: "Review {{ get('path') }} for security issues."
  tools:
    list:
      - name: subagent     # the model calls it with task, optional instructions and tools
      - name: read_file    # read-only: the subagent may use it
      - name: write_file   # not read-only: only this chat may use it
```

Its prompt is the `subagent-system` harness; its tool rounds are the `subagent-max-rounds` event, as in agent mode.

### Component tools (opt-in allowlist)

Expose shipped or installed components as LLM function-calling tools:

```yaml
# resources/example.yaml
chat:
  context:
    prompt: "Research {{ get('q') }} and summarize the findings."
  tools:
    components:
      - web_scraper
      - web_search
```

## Chain-of-thought

`chainOfThought: true` injects a "think step-by-step" system prompt prefix before the first system message. The model reasons explicitly before answering. Works with any provider.

```yaml
# resources/example.yaml
chat:
  context:
    prompt: "{{ get('q') }}"
  harness:
    chainOfThought: true   # prepend CoT instruction to the system prompt
```

## Semantic few-shot selection

When you have many examples and want semantic (embedding-based) selection instead of word-overlap, set `fewShot.embeddingModel` and `fewShot.embeddingBackend`. The K most similar examples are chosen by embedding cosine similarity.

```yaml
# resources/classifier.yaml
chat:
  context:
    prompt: "{{ get('q') }}"
    fewShot:
      selectK: 3
      embeddingModel: text-embedding-3-small   # embedding model for similarity ranking
      embeddingBackend: openai                 # backend that serves the embedding model
      examples:
        - role: user
          prompt: What color is the sky?
        - role: assistant
          prompt: blue
        # ... many more examples
```

Requires `fewShot.selectK` to be set. Falls back to word-overlap if no embedding model is configured.

## Provider-specific options

Some `chat:` fields apply to one backend only: Anthropic prompt caching and beta headers, Google cached content and safety thresholds, Vertex AI routing, Ollama native options and streaming, the OpenAI legacy token parameter. They are set per resource, not in `config.yaml`, and documented with their backend:

- [LLM backends](/workflow/resources/llm-backends) - `promptCaching`, `cacheControl`, `anthropicExtendedOutput`, `anthropicBetaHeaders`, `googleCachedContent`, `googleHarmThreshold`, `googleCloudProject` / `googleCloudLocation`, `ollamaThink` / `ollamaKeepAlive` / `ollamaPullModel`, `openAILegacyMaxTokens`, `openAIWebSearch`, Ollama `streaming`
- [LLM provider reference](/llm-server/providers) - per-provider config snippets and model names

## Sampling: candidate count

`candidateCount` (or its alias `n`) requests N independent completions from the model. The response contains all candidates; kdeps merges them into the output object. Not all providers support multiple candidates.

```yaml
# resources/example.yaml
chat:
  model:
    candidateCount: 3   # generate 3 independent completions
  context:
    prompt: "{{ get('q') }}"
  # n: 3             # alias for candidateCount (OpenAI style)
```

## Sampling: length bounds

`minLength` and `maxLength` set lower and upper bounds on generated token count. `maxLength` is an alias for `maxTokens`.

```yaml
# resources/example.yaml
chat:
  model:
    minLength: 50      # minimum response length in tokens
    maxLength: 500     # maximum response length in tokens (alias for maxTokens)
  context:
    prompt: "{{ get('q') }}"
```

## Few-shot prompting

Inject example user/assistant pairs before the conversation to demonstrate the expected output format. The model follows the pattern in the examples.

<div v-pre>

```yaml
# resources/classifier.yaml
chat:
  context:
    prompt: "{{ get('q') }}"
    fewShot:
      examples:
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

When you have many examples, `fewShot.selectK` picks the K most relevant to the current prompt using word-overlap similarity:

<div v-pre>

```yaml
# resources/classifier.yaml
chat:
  context:
    prompt: "{{ get('q') }}"
    fewShot:
      selectK: 3   # pick 3 most similar examples
      examples:
        - role: user
          prompt: What color is the sky?
        - role: assistant
          prompt: blue
        # ... many more examples
```

</div>

### Token budget for examples

`fewShot.maxTokens` caps the total tokens used by examples, implementing the LengthBasedExampleSelector pattern. Useful when your example pool is large and you want to stay within context limits:

<div v-pre>

```yaml
# resources/classifier.yaml
chat:
  context:
    prompt: "{{ get('q') }}"
    fewShot:
      selectK: 10          # pick up to 10 similar examples
      maxTokens: 500       # but never use more than 500 tokens total
      examples:
        - role: user
          prompt: What color is the sky?
        - role: assistant
          prompt: blue
        # ... large example bank
```

</div>

Both fields can be combined: `fewShot.selectK` ranks examples by similarity first, then `fewShot.maxTokens` prunes the result to the token budget. Pairs (user + assistant) are always kept whole.

## RAG context injection

Pass pre-fetched retriever chunks into the system prompt with `rag.retriever.context`. Each chunk becomes a "Retrieved context:" block prepended to the first system message. Populate this from the `vector_store` component's output.

<div v-pre>

```yaml
# resources/rag-chat.yaml
chat:
  context:
    prompt: "{{ get('q') }}"
    scenario:
      - role: system
        prompt: Answer using only the retrieved context.
  rag:
    retriever:
      context: "{{ $actions.search.output.documents }}"
```

</div>

### Contextual compression

`retriever.topK` keeps only the K chunks most relevant to the current prompt (Jaccard word-overlap similarity). Useful when a vectorstore returns many results but you only want the most relevant ones:

<div v-pre>

```yaml
chat:
  context:
    prompt: "{{ get('q') }}"
  rag:
    retriever:
      context: "{{ $actions.search.output.documents }}"
      topK: 5   # keep 5 most relevant chunks
```

</div>

### Token budget for retriever chunks

`retriever.maxTokens` caps the total tokens of all injected chunks. Chunks are added in similarity-score order until the budget is reached:

<div v-pre>

```yaml
chat:
  context:
    prompt: "{{ get('q') }}"
  rag:
    retriever:
      context: "{{ $actions.search.output.documents }}"
      topK: 10           # pick 10 most relevant
      maxTokens: 1000    # but never use more than 1000 tokens
```

</div>

When combined, `retriever.topK` selects by relevance first, then `retriever.maxTokens` prunes the result to the token budget.

## Examples

### Simple q&A

<div v-pre>

```yaml
# resources/example.yaml
chat:
  model:
    name: llama3.2:1b
    timeout: 30s
  context:
    prompt: "{{ get('q') }}"
    scenario:
      - role: system
        prompt: Answer questions concisely.
  harness:
    output:
      json: true
      keys:
        - answer
```

</div>

### Code generation

<div v-pre>

```yaml
# resources/example.yaml
chat:
  model:
    name: llama3.2:1b
    timeout: 60s
  context:
    prompt: "Write a Python function that {{ get('task') }}"
    scenario:
      - role: system
        prompt: |
          You are an expert Python developer.
          Write clean, documented code with type hints.
  harness:
    output:
      json: true
      keys:
        - code
        - explanation
```

</div>

### Multi-resource pipeline

<div v-pre>

```yaml
# Fast classification resource
actionId: classifier
chat:
  context:
    prompt: "Classify this query: {{ get('q') }}"
  harness:
    output:
      json: true
      keys:
        - category
        - confidence

---
# Detailed response (only runs when confidence >= 0.8)
actionId: detailedResponse
workflow:
  requires: [classifier]
guardrails:
  skip:
  - get('classifier').confidence < 0.8

chat:
  model:
    timeout: 120s
  context:
    prompt: |
      Category: {{ get('classifier').category }}
      Query: {{ get('q') }}
      Provide a detailed response.
```

</div>

## Accessing output

```yaml
workflow:
  # resources/example.yaml
  requires: [llmResource]
apiResponse:
  response:
    answer: get('llmResource').message.content   # reply text
    raw: get('llmResource')                      # full response object
    parsed: get('llmResource').answer            # if output.json: true with key "answer"
```

A failed call fails the resource. If the backend returns an HTTP error, times
out, or is unreachable, the resource errors with `LLM call failed: <cause>` (the
provider's own error text when it sends one) - it never yields an empty result,
so `get('llmResource').message.content` never resolves to `<nil>`. Use `onError:`
to retry or substitute a fallback.

## See also

- [LLM backends](/workflow/resources/llm-backends) - Configure model, backend, API keys, and routing
- [Function calling](/workflow/tools) - LLM function calling
- [Docker deployment](/deploy/docker) - Deploying with local models
