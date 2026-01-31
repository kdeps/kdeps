# Resources Overview

Resources are the fundamental building blocks of KDeps workflows. Each resource performs a specific action and can depend on other resources.

## Resource Structure

Every resource follows this structure:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: myResource        # Unique identifier
  name: My Resource           # Human-readable name
  description: What it does   # Optional description
  category: api               # Optional: for organization
  requires:                   # Dependencies
    - otherResource

items:                        # Optional: for iteration
  - item1
  - item2

run:
  # Request restrictions
  restrictToHttpMethods: [POST]
  restrictToRoutes: [/api/v1/endpoint]
  allowedHeaders: [Authorization]
  allowedParams: [q, limit]

  # Validation
  skipCondition:
    - get('skip') == true

  preflightCheck:
    validations:
      - get('q') != ''
    error:
      code: 400
      message: Query required

  # Processing expressions
  exprBefore:                 # Runs BEFORE the main action
    - set('pre', 'value')
  expr:                       # Runs AFTER the main action (default)
    - set('post', 'value')
  exprAfter:                  # Alias for expr
    - set('also_post', 'value')

  # Action (only one per resource)
  chat: { ... }        # LLM chat
  httpClient: { ... }  # HTTP request
  sql: { ... }         # Database query
  python: { ... }      # Python script
  exec: { ... }        # Shell command
  apiResponse: { ... } # API response
```

## Resource Types

| Type | Description | Use Case |
|------|-------------|----------|
| `chat` | LLM interaction | AI responses, text generation |
| `httpClient` | HTTP requests | External APIs, webhooks |
| `sql` | Database queries | Data retrieval, updates |
| `python` | Python scripts | Data processing, ML |
| `exec` | Shell commands | System operations |
| `apiResponse` | Response formatting | Final output |

## Metadata

### actionId (Required)
Unique identifier for the resource. Used to reference output from other resources.

```yaml
metadata:
  actionId: llmResource

# Access output in another resource:
data: get('llmResource')
```

### description (Optional)
Human-readable description of what the resource does.

```yaml
metadata:
  actionId: llmResource
  name: LLM Chat
  description: Processes user queries using language models
```

### category (Optional)
Organize resources by category for better management.

```yaml
metadata:
  actionId: userAuth
  name: User Authentication
  category: auth

metadata:
  actionId: dataProcessor
  name: Data Processor
  category: processing
```

Common categories: `api`, `auth`, `processing`, `storage`, `ai`, `utils`.

### requires (Dependencies)
List of resources that must execute before this one.

```yaml
metadata:
  actionId: responseResource
  requires:
    - llmResource
    - httpResource
```

KDeps automatically builds a dependency graph and executes resources in the correct order.

## Request Restrictions

### restrictToHttpMethods
Limit which HTTP methods trigger this resource:

```yaml
run:
  restrictToHttpMethods: [GET, POST]
```

### restrictToRoutes
Limit which routes trigger this resource:

```yaml
run:
  restrictToRoutes: [/api/v1/chat, /api/v1/query]
```

### allowedHeaders / allowedParams
Whitelist specific headers or parameters:

```yaml
run:
  allowedHeaders:
    - Authorization
    - X-API-Key
  allowedParams:
    - q
    - limit
    - offset
```

## Validation

### skipCondition
Skip resource execution based on conditions:

```yaml
run:
  skipCondition:
    - get('skip') == true
    - get('mode') == 'fast'
```

If any condition evaluates to `true`, the resource is skipped.

### preflightCheck
Validate inputs before execution:

```yaml
run:
  preflightCheck:
    validations:
      - get('q') != ''
      - get('limit') <= 100
    error:
      code: 400
      message: Invalid request parameters
```

If validation fails, the error response is returned immediately.

## Processing with Expressions

Execute logic before or after the main action:

### exprBefore
Runs **before** the main action. Use this to prepare data used in the resource's own configuration (like prompts or URLs).

<div v-pre>

```yaml
run:
  exprBefore:
    - set('full_name', get('first') + ' ' + get('last'))
  chat:
    prompt: "Hello {{ get('full_name') }}"
```

</div>

### expr (or exprAfter)
Runs **after** the main action. Use this to process results or update state for subsequent resources.

<div v-pre>

```yaml
run:
  chat:
    prompt: "Summary of {{ get('q') }}"
  expr:
    - set('summary', get('myResourceId'))
    - set('processed_at', info('timestamp'))
```

</div>

See [Expressions](../concepts/expressions.md) for detailed documentation.

## Items Iteration

Process multiple items in sequence:

<div v-pre>

```yaml
items:
  - "First item"
  - "Second item"
  - "Third item"

run:
  chat:
    prompt: "Process: {{ get('current') }}"
```

</div>

Access iteration context:
- `get('current')` - Current item
- `get('prev')` - Previous item
- `get('next')` - Next item
- `get('index')` - Current index (0-based)
- `get('count')` - Total item count

## Resource Output

Each resource produces output that can be accessed by dependent resources:

<div v-pre>

```yaml
# LLM resource output
metadata:
  actionId: llmResource
run:
  chat:
    model: llama3.2:1b
    prompt: "Answer: {{ get('q') }}"

# Access in another resource
metadata:
  requires: [llmResource]
run:
  apiResponse:
    response:
      answer: get('llmResource')  # Get the LLM response
```

</div>

## Execution Flow

```
Request
    ↓
┌─────────────────┐
│ Route Matching  │
└────────┬────────┘
         ↓
┌─────────────────┐
│ Build Dep Graph │
└────────┬────────┘
         ↓
For each resource (in order):
    ┌─────────────────┐
    │ Check Route     │ → Skip if not matching
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Check Skip      │ → Skip if condition true
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Preflight Check │ → Error if validation fails
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Execute exprBefore │
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Execute Action  │
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Execute expr    │
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Store Output    │
    └─────────────────┘
         ↓
┌─────────────────┐
│ Return Target   │
└─────────────────┘
```

## Best Practices

### 1. Use Descriptive actionIds
```yaml
# Good
actionId: fetchUserProfile
actionId: validatePayment

# Avoid
actionId: resource1
actionId: r2
```

### 2. Single Responsibility
Each resource should do one thing well. Split complex logic into multiple resources.

### 3. Validate Early
Use `preflightCheck` to catch errors before expensive operations.

### 4. Handle Dependencies
Only list direct dependencies in `requires`. KDeps handles transitive dependencies.

### 5. Use Appropriate Timeouts
Set realistic `timeoutDuration` values based on expected execution time.

## Next Steps

- [LLM Resource](llm) - AI model integration
- [HTTP Client](http-client) - External API calls
- [SQL Resource](sql) - Database operations
- [Python Resource](python) - Script execution
- [Exec Resource](exec) - Shell commands
- [API Response](api-response) - Response formatting
