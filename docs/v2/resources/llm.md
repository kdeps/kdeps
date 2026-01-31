# LLM Resource

The LLM (chat) resource enables interaction with language models for text generation, question answering, and AI-powered tasks.

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
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
    timeoutDuration: 60s
```

</div>

## Configuration Options

### Complete Reference

<div v-pre>

```yaml
run:
  chat:
    # Model Configuration
    model: llama3.2:1b              # Required: Model name
    backend: ollama                  # Backend: ollama, openai, anthropic, etc.
    baseUrl: http://localhost:11434  # Custom backend URL
    apiKey: "sk-..."                 # API key (or use env var)
    contextLength: 8192              # Context window size

    # Prompt Configuration
    role: user                       # Role: user, assistant, system
    prompt: "{{ get('q') }}"        # The prompt to send

    # Advanced Generation Parameters
    temperature: 0.7                 # 0.0 to 2.0 (default varies by backend)
    maxTokens: 1000                  # Max tokens to generate
    topP: 0.9                        # Nucleus sampling (0.0 to 1.0)
    frequencyPenalty: 0.0            # -2.0 to 2.0
    presencePenalty: 0.0             # -2.0 to 2.0

    # Conversation Context
    scenario:
      - role: system
        prompt: You are a helpful assistant.
      - role: assistant
        prompt: I'm ready to help!

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

    # Timeout
    timeoutDuration: 60s
```

</div>

## Backends

KDeps supports multiple LLM backends:

### Local Backend

| Backend | Default URL | Description |
|---------|-------------|-------------|
| `ollama` | localhost:11434 | Ollama (default) |

### Cloud Backends

| Backend | Environment Variable | Description |
|---------|---------------------|-------------|
| `openai` | `OPENAI_API_KEY` | OpenAI GPT models |
| `anthropic` | `ANTHROPIC_API_KEY` | Claude models |
| `google` | `GOOGLE_API_KEY` | Gemini models |
| `mistral` | `MISTRAL_API_KEY` | Mistral AI |
| `together` | `TOGETHER_API_KEY` | Together AI |
| `groq` | `GROQ_API_KEY` | Groq (fast inference) |
| `perplexity` | `PERPLEXITY_API_KEY` | Perplexity AI |
| `cohere` | `COHERE_API_KEY` | Cohere |
| `deepseek` | `DEEPSEEK_API_KEY` | DeepSeek |

### Backend Examples

**Ollama (Default)**
<div v-pre>

```yaml
chat:
  model: llama3.2:1b
  backend: ollama  # Optional, this is default
  prompt: "{{ get('q') }}"
```

</div>

**OpenAI**
<div v-pre>

```yaml
chat:
  model: gpt-4
  backend: openai
  apiKey: "{{ get('OPENAI_API_KEY', 'env') }}"
  prompt: "{{ get('q') }}"
```

</div>

**Anthropic (Claude)**
<div v-pre>

```yaml
chat:
  model: claude-3-opus-20240229
  backend: anthropic
  prompt: "{{ get('q') }}"
```

</div>

## Advanced Parameters

Fine-tune the model's output generation:

- **temperature**: Controls randomness. Higher values (e.g., 0.8) make output more random, lower values (e.g., 0.2) make it more focused and deterministic.
- **maxTokens**: The maximum number of tokens to generate in the completion.
- **topP**: An alternative to sampling with temperature, called nucleus sampling. The model considers the results of the tokens with top_p probability mass.
- **frequencyPenalty**: Positive values penalize new tokens based on their existing frequency in the text so far, decreasing the model's likelihood to repeat the same line verbatim.
- **presencePenalty**: Positive values penalize new tokens based on whether they appear in the text so far, increasing the model's likelihood to talk about new topics.

```yaml
chat:
  model: llama3.2:1b
  prompt: "Write a creative story"
  temperature: 0.9
  presencePenalty: 0.6
  maxTokens: 500
```

## Context Length

Control the context window size:

```yaml
chat:
  model: llama3.2:1b
  contextLength: 8192  # Options: 4096, 8192, 16384, 32768, 65536, 131072, 262144
```

## Scenario (Conversation History)

Build multi-turn conversations:

<div v-pre>

```yaml
chat:
  model: llama3.2:1b
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
  model: llama3.2:1b
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

Process images with vision-capable models:

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
# Main LLM resource
metadata:
  actionId: llmWithTools

run:
  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
    tools:
      - name: calculate
        description: Perform mathematical calculations
        script: calcTool  # References another resource
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

The LLM automatically decides when to call tools based on the prompt.

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
    timeoutDuration: 30s
```

</div>

### Code Generation

<div v-pre>

```yaml
run:
  chat:
    model: codellama
    prompt: "Write a Python function that {{ get('task') }}"
    scenario:
      - role: system
        prompt: |
          You are an expert Python developer.
          Write clean, documented code.
          Include type hints.
    jsonResponse: true
    jsonResponseKeys:
      - code
      - explanation
    timeoutDuration: 60s
```

</div>

### Multi-Model Workflow

<div v-pre>

```yaml
# Fast model for classification
metadata:
  actionId: classifier

run:
  chat:
    model: llama3.2:1b
    prompt: "Classify this query: {{ get('q') }}"
    jsonResponse: true
    jsonResponseKeys:
      - category
      - confidence

---
# Powerful model for complex queries
metadata:
  actionId: detailedResponse
  requires: [classifier]

run:
  skipCondition:
    - get('classifier').confidence < 0.8

  chat:
    model: llama3.2
    prompt: |
      Category: {{ get('classifier').category }}
      Query: {{ get('q') }}
      Provide a detailed response.
    timeoutDuration: 120s
```

</div>

## Accessing Output

```yaml
# In another resource
metadata:
  requires: [llmResource]

run:
  apiResponse:
    response:
      # Full response
      llm_output: get('llmResource')

      # Specific field (if JSON response)
      answer: get('llmResource').answer
```