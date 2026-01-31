# Quickstart

This guide walks you through building your first AI agent with KDeps. You'll create a simple chatbot that responds to user queries.

## Prerequisites

Make sure KDeps is installed on your system:

```bash
# Install via script (Mac/Linux)
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh

# Or via Homebrew (Mac)
brew install kdeps/tap/kdeps
```

For more options, see the [Installation Guide](installation).

## Option 1: Use the New Command (Easiest)

The `new` command creates a complete agent project interactively:

```bash
kdeps new my-agent
```

Follow the prompts to:
1. Select an agent template
2. Choose required resources
3. Configure basic settings (port, models, etc.)

Or use a template directly:

```bash
kdeps new my-agent --template api-service
```

## Option 2: Create Manually

### Step 1: Create Project Structure

```bash
mkdir my-agent
cd my-agent
mkdir resources
```

### Step 2: Create the Workflow

Create `workflow.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent
  description: My first AI agent
  version: "1.0.0"
  targetActionId: responseResource

settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 3000
    routes:
      - path: /api/v1/chat
        methods: [POST]
    cors:
      enableCors: true
      allowOrigins:
        - http://localhost:8080

  agentSettings:
    timezone: Etc/UTC
    models:
      - llama3.2:1b
```

### Step 3: Create the LLM Resource

Create `resources/llm.yaml`:

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: llmResource
  name: LLM Chat

run:
  restrictToHttpMethods: [POST]
  restrictToRoutes: [/api/v1/chat]

  preflightCheck:
    validations:
      - get('q') != ''
    error:
      code: 400
      message: Query parameter 'q' is required

  chat:
    model: llama3.2:1b
    role: user
    prompt: "{{ get('q') }}"
    scenario:
      - role: assistant
        prompt: You are a helpful AI assistant.
    jsonResponse: true
    jsonResponseKeys:
      - answer
    timeoutDuration: 60s
```

</div>

### Step 4: Create the Response Resource

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
  restrictToHttpMethods: [POST]
  restrictToRoutes: [/api/v1/chat]

  apiResponse:
    success: true
    response:
      data: get('llmResource')
      query: get('q')
    meta:
      headers:
        Content-Type: application/json
```

### Step 5: Validate Your Workflow

```bash
kdeps validate workflow.yaml
```

You should see:
```
Workflow validated successfully
```

### Step 6: Run the Agent

```bash
kdeps run workflow.yaml
```

You'll see output like:
```
Starting API server on 127.0.0.1:3000
Routes:
  POST /api/v1/chat
```

### Step 7: Test the API

In another terminal:

```bash
curl -X POST http://localhost:3000/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"q": "What is artificial intelligence?"}'
```

Response:
```json
{
  "success": true,
  "response": {
    "data": {
      "answer": "Artificial intelligence (AI) refers to..."
    },
    "query": "What is artificial intelligence?"
  }
}
```

## Development Mode

For faster development with auto-reload:

```bash
kdeps run workflow.yaml --dev
```

Changes to your YAML files will automatically reload the server.

## Understanding the Flow

1. **Request arrives** at `/api/v1/chat` with `{"q": "..."}`
2. **Preflight check** validates that `q` is not empty
3. **LLM resource** sends the prompt to the model
4. **Response resource** formats the output as JSON
5. **API responds** with the formatted result

```
Request → Preflight → LLM → Response → Output
              ↓
         (if invalid)
              ↓
         400 Error
```

## Project Structure

```
my-agent/
├── workflow.yaml         # Main workflow configuration
└── resources/
    ├── llm.yaml         # LLM chat resource
    └── response.yaml    # API response formatting
```

## Key Concepts

### Workflow
The `workflow.yaml` defines:
- Agent metadata (name, version)
- Target action to execute
- API server settings
- LLM models to use

### Resources
Resources are the building blocks:
- Each resource has a unique `actionId`
- Resources can depend on other resources via `requires`
- Resources execute in dependency order

### Unified API
Access data with `get()`:
- `get('q')` - Get query parameter or body field
- `get('llmResource')` - Get output from another resource
<span v-pre>`{{ get('q') }}`</span> - String interpolation

## Next Steps

- [CLI Reference](cli-reference) - Complete command reference
- [Workflow Configuration](../configuration/workflow) - Deep dive into settings
- [LLM Resource](../resources/llm) - Advanced LLM configuration
- [HTTP Client](../resources/http-client) - Make external API calls
- [SQL Resource](../resources/sql) - Query databases
- [Unified API](../concepts/unified-api) - Master get() and set()

## Examples

Explore more examples in the [examples directory](https://github.com/kdeps/kdeps/tree/main/examples):

| Example | Description |
|---------|-------------|
| chatbot | Basic LLM chatbot |
| http-advanced | Authentication, caching, retries |
| sql-advanced | Database queries with pooling |
| file-upload | File upload handling |
| vision | Image analysis with vision models |
| tools | LLM function calling |
| session-auth | Session management |
| webserver-static | Static file serving |
