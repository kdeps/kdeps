# LLM Resource

The LLM (chat) resource enables interaction with language models for text generation, question answering, and AI-powered tasks.

## Model and Backend Configuration

**Model is set per resource** in `run.chat.model`. Set `model: router` to delegate model selection to the LLM router.

```yaml
# resources/my-resource.yaml
run:
  chat:
    model: llama3.2:1b          # Per-resource model selection
    role: user
    prompt: "{{ get('q') }}"
```

**Backend, base URL, and API keys** are configured in `~/.kdeps/config.yaml`:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: ollama              # Default backend (ollama, openai, anthropic, ...)
  # base_url: http://localhost:11434
  # openai_api_key: sk-...
  # anthropic_api_key: sk-ant-...
```

For router configuration and multi-backend routing, see [LLM Backends](llm-backends).

## Basic Usage

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: llmResource
  name: LLM Chat

run:
  chat:
    prompt: "{{ get('q') }}"
    timeout: 60s
```

</div>

## Configuration Options

### Complete Reference

<div v-pre>

```yaml
run:
  chat:
    model: llama3.2:1b              # Model name, or "router" to delegate to config
    # Prompt Configuration
    role: user                       # Role: user, assistant, system
    prompt: "{{ get('q') }}"        # The prompt to send

    # Generation Parameters
    contextLength: 8192              # Context window size (tokens)
    temperature: 0.7                 # 0.0 to 2.0
    maxTokens: 1000                  # Max tokens to generate
    topP: 0.9                        # Nucleus sampling (0.0 to 1.0)
    frequencyPenalty: 0.0            # -2.0 to 2.0
    presencePenalty: 0.0             # -2.0 to 2.0

    # Conversation Context
    scenario:
      - role: system
        prompt: You are a helpful assistant.
      - role: assistant
        prompt: I am ready to help!

    # Tools (Function Calling)
    tools:
      - name: calculate
        description: Perform math
        script: calcResource
        parameters:
          expression:
            type: string
            required: true

    # File Attachments (Vision)
    files:
      - "{{ get('file', 'filepath') }}"

    # Response Formatting
    jsonResponse: true
    jsonResponseKeys:
      - answer
      - confidence

    # Timeout and Streaming
    timeout: 60s
    streaming: true              # Ollama only: stream NDJSON chunks
```

</div>

## Advanced Parameters

- **temperature**: Controls randomness. Higher (e.g. 0.8) = more random, lower (e.g. 0.2) = more focused.
- **maxTokens**: Maximum tokens to generate.
- **topP**: Nucleus sampling - considers tokens with top_p probability mass.
- **frequencyPenalty**: Penalizes repeated tokens, reducing verbatim repetition.
- **presencePenalty**: Penalizes any token that has appeared, encouraging new topics.

```yaml
chat:
  prompt: "Write a creative story"
  temperature: 0.9
  presencePenalty: 0.6
  maxTokens: 500
```

## Context Length

Control the context window:

```yaml
chat:
  contextLength: 8192  # Options: 4096, 8192, 16384, 32768, 65536, 131072, 262144
```

## Scenario (Conversation History)

Build multi-turn conversations:

<div v-pre>

```yaml
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

Process images (set a vision-capable model in `run.chat.model` in your resource YAML):

<div v-pre>

```yaml
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
metadata:
  actionId: llmWithTools

run:
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
run:
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
run:
  chat:
    prompt: "{{ get('q') }}"
    streaming: true
```

</div>

## Examples

### Simple Q&A

<div v-pre>

```yaml
run:
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
run:
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
metadata:
  actionId: classifier

run:
  chat:
    prompt: "Classify this query: {{ get('q') }}"
    jsonResponse: true
    jsonResponseKeys:
      - category
      - confidence

---
# Detailed response (only runs when confidence >= 0.8)
metadata:
  actionId: detailedResponse
  requires: [classifier]

run:
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
metadata:
  requires: [llmResource]

run:
  apiResponse:
    response:
      llm_output: get('llmResource')
      answer: get('llmResource').answer  # If jsonResponse: true
```

## See Also

- [LLM Backends](llm-backends) - Configure model, backend, API keys, and routing
- [Tools](../concepts/tools) - LLM function calling
- [Docker Deployment](../deployment/docker) - Deploying with local models
