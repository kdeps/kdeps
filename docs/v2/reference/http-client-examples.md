# HTTP Client Examples

Example `httpClient:` resources for common API integration patterns. See [HTTP Client Resource](/resources/http-client) for the full configuration reference.

## Fetch Data and Process

<div v-pre>

```yaml
# ~/.kdeps/config.yaml
http_connections:
  github:
    auth:
      type: bearer
      token: "${GITHUB_TOKEN}"

# resources/fetch-data.yaml
actionId: fetchData
httpClient:
  method: GET
  url: "https://api.github.com/repos/{{ get('owner') }}/{{ get('repo') }}"
  headers:
    Accept: application/vnd.github.v3+json
  connectionName: github
  timeout: 30s

---

actionId: analyzeRepo
requires: [fetchData]
chat:
  model: llama3.2:1b
  prompt: "Analyze this GitHub repo: {{ get('fetchData') }}"
```

</div>

## Authenticated API with Retry

<div v-pre>

```yaml
# ~/.kdeps/config.yaml
http_connections:
  stripe:
    auth:
      type: bearer
      token: "${STRIPE_SECRET_KEY}"

# resources/example.yaml
httpClient:
  method: POST
  url: "https://api.stripe.com/v1/charges"
  connectionName: stripe
  headers:
    Content-Type: application/x-www-form-urlencoded
  data:
    amount: "{{ get('amount') }}"
    currency: usd
    source: "{{ get('token') }}"
  retry:
    maxAttempts: 3
    backoff: 2s
    retryOn: [500, 502, 503]
  timeout: 30s
```

</div>

## Webhook Call

<div v-pre>

```yaml
# resources/example.yaml
httpClient:
  method: POST
  url: "{{ get('webhook_url') }}"
  headers:
    Content-Type: application/json
    X-Webhook-Secret: "{{ get('WEBHOOK_SECRET', 'env') }}"
  data:
    event: order_completed
    order_id: "{{ get('order_id') }}"
    timestamp: "{{ info('timestamp') }}"
  timeout: 10s
```

</div>

## Cached External API

```yaml
# resources/example.yaml
httpClient:
  method: GET
  url: "https://api.exchangerate.host/latest"
  cache:
    ttl: 1h
    key: "exchange-rates"
  timeout: 30s
```

## See Also

- [HTTP Client Resource](/resources/http-client) - Full configuration reference
