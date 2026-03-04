# Real-World Workflows Showcase

These examples demonstrate how complex agents can be built in ~20 lines of YAML with no custom code.

Every agent shares the same entry point:

<div v-pre>

```yaml
# workflow.yaml (same for all agents)
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: respond
settings:
  apiServerMode: true
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  chat:
    model: gpt-4o-mini
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  chat:
    model: gpt-4o-mini
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - httpClient:
        method: GET
        url: "{{ env('BANK_API_URL') }}/transactions"
        headers:
          Authorization: "Bearer {{ env('BANK_TOKEN') }}"
  chat:
    model: gpt-4o-mini
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - httpClient:
        method: GET
        url: "{{ env('BANK_API_URL') }}/transactions?days=90"
        headers:
          Authorization: "Bearer {{ env('BANK_TOKEN') }}"
  chat:
    model: gpt-4o-mini
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - embedding:
        model: nomic-embed-text
        input: "{{ get('q') }}"
        collection: my-docs
        operation: search
        topK: 5
  chat:
    model: llama3.2
    prompt: |
      Context: {{ get('embedding').results }}
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - httpClient:
        method: GET
        url: "{{ env('PANTRY_API') }}/inventory"
  chat:
    model: gpt-4o-mini
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - scraper:
        url: "https://www.kayak.com/flights/{{ get('from') }}-{{ get('to') }}/{{ get('date') }}"
  chat:
    model: gpt-4o-mini
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  chat:
    model: gpt-4o-mini
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
