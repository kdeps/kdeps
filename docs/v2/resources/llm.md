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

  timeout: 60s               # hard stop -- returns error, does not retry
  streaming: true            # Ollama only: stream NDJSON; kdeps accumulates before returning
```

</div>

## Advanced Parameters

```yaml
# resources/example.yaml
chat:
  prompt: "Write a creative story"
  temperature: 0.9      # 0.0 = deterministic, 1.0+ = more random/creative
  maxTokens: 500        # hard cap on generated tokens; 0 = model default
  topP: 0.9             # nucleus sampling -- lower = less diverse vocabulary
  frequencyPenalty: 0.0 # penalises tokens repeated in the output (-2.0 to 2.0)
  presencePenalty: 0.6  # penalises any token that appeared at all (-2.0 to 2.0)
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
