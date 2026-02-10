# Building a Chatbot

This tutorial walks you through building a simple chatbot using KDeps v2. You'll learn how to set up a workflow, configure an LLM resource, and handle API requests.

## Prerequisites

- KDeps installed (see [Installation](../getting-started/installation))
- Ollama installed and running (for local LLM)
- A model pulled in Ollama: `ollama pull llama3.2:1b`

## Step 1: Create the Workflow

Create a new directory for your chatbot:

```bash
mkdir my-chatbot
cd my-chatbot
```

Create `workflow.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: chatbot
  description: Simple LLM chatbot
  version: "1.0.0"
  targetActionId: responseResource

settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
    cors:
      enableCors: true
      allowOrigins:
        - http://localhost:16395

  agentSettings:
    timezone: Etc/UTC
    pythonVersion: "3.12"
    models:
      - llama3.2:1b
```

## Step 2: Create the LLM Resource

Create `resources/llm.yaml`:

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
    jsonResponse: true
    jsonResponseKeys:
      - answer
```

</div>

**Key Points:**
- `get('q')` retrieves the query parameter from the request
- `jsonResponse: true` ensures structured JSON output
- `jsonResponseKeys` defines the expected keys in the response

## Step 3: Create the Response Resource

Create `resources/response.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: responseResource
  name: API Response
  requires:
    - llmResource

run:
  apiResponse:
    success: true
    response:
      data: get('llmResource')
      query: get('q')
```

**Key Points:**
- `requires: [llmResource]` ensures the LLM resource runs first
- `get('llmResource')` accesses the output from the LLM resource
- `get('q')` includes the original query in the response

## Step 4: Run the Chatbot

Start the workflow:

```bash
kdeps run workflow.yaml
```

You should see output indicating the server is running on port 16395.

## Step 5: Test the Chatbot

Send a test request:

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"q": "What is artificial intelligence?"}'
```

Expected response:

```json
{
  "success": true,
  "data": {
    "answer": "Artificial intelligence (AI) is the simulation of human intelligence by machines..."
  },
  "query": "What is artificial intelligence?"
}
```

## Understanding the Unified API

This chatbot demonstrates KDeps' unified API with the `get()` function:

### Data Sources

The `get()` function automatically detects the data source:

<div v-pre>

```yaml
# Query parameters
prompt: "{{ get('q') }}"

# Resource outputs
data: get('llmResource')

# Headers
auth: get('Authorization')

# Session storage
user: get('user_name', 'session')
```

</div>

### Automatic Detection

KDeps automatically determines where to look for data:
- `get('q')` → Query parameter `?q=...`
- `get('llmResource')` → Output from `llmResource`
- `get('Authorization')` → HTTP header

## Adding Validation

Add input validation to ensure the query is not empty:

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: llmResource
  name: LLM Chat

run:
  validations:
    - get('q') != ''
    - len(get('q')) > 3
  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
    jsonResponse: true
    jsonResponseKeys:
      - answer
```

</div>

If validation fails, the resource returns an error before executing.

## Adding Conversation Context

Add system prompts and conversation history:

<div v-pre>

```yaml
run:
  chat:
    model: llama3.2:1b
    scenario:
      - role: system
        prompt: "You are a helpful assistant that provides clear, concise answers."
      - role: user
        prompt: "{{ get('q') }}"
    jsonResponse: true
    jsonResponseKeys:
      - answer
```

</div>

## Adding Session Support

Enable session storage to maintain conversation context:

```yaml
settings:
  session:
    enabled: true
    type: sqlite
    path: ./chatbot.db
```

Then access session data:

<div v-pre>

```yaml
run:
  chat:
    model: llama3.2:1b
    scenario:
      - role: system
        prompt: "You are a helpful assistant."
      - role: assistant
        prompt: "{{ get('previous_response', 'session') }}"
      - role: user
        prompt: "{{ get('q') }}"
```

</div>

## Adding Error Handling

Handle errors gracefully:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: responseResource
  name: API Response
  requires:
    - llmResource

run:
  apiResponse:
    success: true
    response:
      data: get('llmResource')
      query: get('q')
    onError:
      success: false
      response:
        error: "Failed to process request"
        message: get('error')
```

## Next Steps

- **Add Tools**: Learn about [function calling](../concepts/tools) to give your chatbot capabilities
- **Add Memory**: Use [session storage](../configuration/session) for conversation history
- **Add Validation**: Implement input validation and error handling
- **Deploy**: Package your chatbot with [Docker](../deployment/docker)

## Complete Example

See the full example in `examples/chatbot/`:

```bash
kdeps run examples/chatbot/workflow.yaml
```

## Related Documentation

- [LLM Resource](../resources/llm) - Complete LLM configuration reference
- [Unified API](../concepts/unified-api) - Understanding `get()` and `set()`
- [Workflow Configuration](../configuration/workflow) - Full workflow settings
- [Session & Storage](../configuration/session) - Conversation persistence
