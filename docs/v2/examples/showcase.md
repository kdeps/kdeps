# Real-World Workflows Showcase

These examples show how complex agents can be built in ~20 lines of YAML with no custom code. Each example is a complete workflow: POST a JSON body, get a structured JSON response.

The pattern for all examples:

```
POST /api/v1/run  {"q": "...", ...other fields...}
        |
        v
  chat resource reads fields via get('field')
        |
        v
{"success": true, "response": {"data": {...structured output...}}}
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
settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/run
        methods: [POST]
```

</div>

---

### 1. Email — sort, draft, unsubscribe

<div v-pre>

```yaml
actionId: respond
chat:
  prompt: |
    Email: {{ get('email') }}
    Classify as urgent / normal / unsubscribe.
    If urgent: draft a reply.
    If unsubscribe: return the sender address.
  jsonResponse: true
  jsonResponseKeys: [label, draft, unsubscribe_from]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 2. Meetings — agenda + action items

<div v-pre>

```yaml
actionId: respond
chat:
  prompt: |
    Meeting request: {{ get('request') }}
    Attendees: {{ get('attendees') }}
    Write a concise agenda and post-meeting action items.
  jsonResponse: true
  jsonResponseKeys: [agenda, action_items, suggested_time]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 3. Late bills — cash-flow-aware due dates

<div v-pre>

```yaml
actionId: respond
before:
  - httpClient:
      method: GET
      url: "{{ env('BANK_API_URL') }}/transactions"
      headers:
        Authorization: "Bearer {{ env('BANK_TOKEN') }}"
chat:
  prompt: |
    Transactions: {{ get('httpClient') }}
    Today: {{ info('current_date') }}
    List bills due in 7 days. Flag anything overdue.
  jsonResponse: true
  jsonResponseKeys: [due_soon, overdue, balance_after]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 4. Subscription leaks — find what you forgot

<div v-pre>

```yaml
actionId: respond
before:
  - httpClient:
      method: GET
      url: "{{ env('BANK_API_URL') }}/transactions?days=90"
      headers:
        Authorization: "Bearer {{ env('BANK_TOKEN') }}"
chat:
  prompt: |
    Transactions: {{ get('httpClient') }}
    Find all recurring charges. Flag any not used in 30+ days.
  jsonResponse: true
  jsonResponseKeys: [subscriptions, unused, monthly_total]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 5. Finding your own files — RAG search

<div v-pre>

```yaml
actionId: respond
before:
  - component:
      name: embedding
      with:
        text: "{{ get('q') }}"
chat:
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

### 6. Grocery waste — meal plan from what's expiring

<div v-pre>

```yaml
actionId: respond
before:
  - httpClient:
      method: GET
      url: "{{ env('PANTRY_API') }}/inventory"
chat:
  prompt: |
    Pantry: {{ get('httpClient') }}
    Suggest 5 meals using items expiring soonest.
    List what needs reordering.
  jsonResponse: true
  jsonResponseKeys: [meals, reorder_list]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 7. Travel planning — one pass

<div v-pre>

```yaml
actionId: respond
before:
  - component:
      name: scraper
      with:
        url: "https://www.kayak.com/flights/{{ get('from') }}-{{ get('to') }}/{{ get('date') }}"
chat:
  prompt: |
    Trip: {{ get('from') }} → {{ get('to') }} on {{ get('date') }}
    Data: {{ get('scraper') }}
    Best flight option, hotel, and 3-day itinerary.
  jsonResponse: true
  jsonResponseKeys: [flight, hotel, itinerary]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

### 8. Admin overhead — generate invoice

<div v-pre>

```yaml
actionId: respond
chat:
  prompt: |
    Client: {{ get('client') }}
    Work done: {{ get('description') }}
    Hours: {{ get('hours') }} at {{ get('rate') }}/hr
    Generate a professional invoice.
  jsonResponse: true
  jsonResponseKeys: [invoice_number, line_items, subtotal, due_date]
apiResponse:
  success: true
  response:
    data: get('respond')
```

</div>

## See Also

- [Quick Start](/getting-started/quickstart) - Walkthrough with a running example
- [Resources Overview](/resources/overview) - All resource types
- [LLM Resource](/resources/llm) - Chat and JSON response
