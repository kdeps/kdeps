# Real-world workflows showcase

These examples show how to build complex agents in about 20 lines of YAML, with no custom code. Each example is a complete workflow: POST a JSON body, get a structured JSON response.

Works in workflow and agent mode.

The pattern for all examples:

```
POST /api/v1/run  {"q": "...", ...other fields...}
        |
        v
  chat resource reads fields via get('field')
        |
        v
{"success": true, "data": {...structured output...}}
```

Every agent shares the same entry point:

<div v-pre>

```yaml
# workflow.yaml (same for all agents)
apiVersion: kdeps.io/v1
kind: Workflow
name: my-agent
version: "1.0.0"
targetActionId: respond
deploy:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/run
        methods: [POST]
```

</div>

---

### 1. Email - sort, draft, unsubscribe

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
chat:
  context:
    prompt: |
      Email: {{ get('email') }}
      Classify as urgent / normal / unsubscribe.
      If urgent: draft a reply.
      If unsubscribe: return the sender address.
  harness:
    output:
      json: true
      keys: [label, draft, unsubscribe_from]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 2. Meetings - agenda + action items

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
chat:
  context:
    prompt: |
      Meeting request: {{ get('request') }}
      Attendees: {{ get('attendees') }}
      Write a concise agenda and post-meeting action items.
  harness:
    output:
      json: true
      keys: [agenda, action_items, suggested_time]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 3. Late bills - cash-flow-aware due dates

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
before:
  - component:
      name: http_request
      with:
        method: GET
        url: "{{ env('BANK_API_URL') }}/transactions"
        headers:
          Authorization: "Bearer {{ env('BANK_TOKEN') }}"
chat:
  context:
    prompt: |
      Transactions: {{ get('http_request') }}
      Today: {{ info('timestamp') }}
      List bills due in 7 days. Flag anything overdue.
  harness:
    output:
      json: true
      keys: [due_soon, overdue, balance_after]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 4. Subscription leaks - find what you forgot

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
before:
  - component:
      name: http_request
      with:
        method: GET
        url: "{{ env('BANK_API_URL') }}/transactions?days=90"
        headers:
          Authorization: "Bearer {{ env('BANK_TOKEN') }}"
chat:
  context:
    prompt: |
      Transactions: {{ get('http_request') }}
      Find all recurring charges. Flag any not used in 30+ days.
  harness:
    output:
      json: true
      keys: [subscriptions, unused, monthly_total]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 5. Finding your own files - RAG search

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
before:
  - component:
      name: embedding
      with:
        text: "{{ get('q') }}"
chat:
  context:
    prompt: |
      Context: {{ get('embedding').embedding }}
      Question: {{ get('q') }}
      Answer using only the context above.
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 6. Grocery waste - meal plan from what is expiring

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
before:
  - component:
      name: http_request
      with:
        method: GET
        url: "{{ env('PANTRY_API') }}/inventory"
chat:
  context:
    prompt: |
      Pantry: {{ get('http_request') }}
      Suggest 5 meals using items expiring soonest.
      List what needs reordering.
  harness:
    output:
      json: true
      keys: [meals, reorder_list]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 7. Travel planning - one pass

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
before:
  - component:
      name: web_scraper
      with:
        url: "https://www.kayak.com/flights/{{ get('from') }}-{{ get('to') }}/{{ get('date') }}"
chat:
  context:
    prompt: |
      Trip: {{ get('from') }} -> {{ get('to') }} on {{ get('date') }}
      Data: {{ get('web_scraper') }}
      Best flight option, hotel, and 3-day itinerary.
  harness:
    output:
      json: true
      keys: [flight, hotel, itinerary]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 8. Admin overhead - generate invoice

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
chat:
  context:
    prompt: |
      Client: {{ get('client') }}
      Work done: {{ get('description') }}
      Hours: {{ get('hours') }} at {{ get('rate') }}/hr
      Generate a professional invoice.
  harness:
    output:
      json: true
      keys: [invoice_number, line_items, subtotal, due_date]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

## See also

- [Quick start](/workflow/quickstart) - Walkthrough with a running example
- [Resources overview](/workflow/resources) - All resource types
- [LLM resource](/workflow/resources/llm) - Chat and JSON response
