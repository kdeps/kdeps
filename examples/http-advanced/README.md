# Advanced HTTP Client Features

This example demonstrates advanced HTTP client capabilities in KDeps v2:

## 🚀 Advanced Features

### 🔐 **Authentication**
- **Bearer Token**: Automatic `Authorization: Bearer <token>` headers
- **API Key**: Custom header-based API key authentication
- **Basic Auth**: Username/password with base64 encoding
- **OAuth2**: Token-based authentication (extensible)

### 🔄 **Retry Logic**
- **Exponential Backoff**: Configurable retry delays
- **Status Code Retries**: Retry on specific HTTP status codes (5xx, 429, etc.)
- **Max Attempts**: Configurable retry limits
- **Backoff Limits**: Maximum delay caps

### 💾 **Response Caching**
- **Memory Caching**: Fast in-memory response caching
- **TTL Support**: Time-based cache expiration
- **Custom Keys**: Configurable cache keys
- **Request Deduplication**: Avoid duplicate API calls

### 🌐 **Advanced Configuration**
- **Proxy Support**: HTTP/SOCKS proxy configuration
- **TLS Configuration**: Custom certificates and verification
- **Redirect Control**: Follow or reject redirects
- **Timeout Management**: Per-request timeouts

## 📋 API Endpoints

### GET /api/v1/http-demo
**Bearer Token Authentication with Caching:**
- Calls `https://httpbin.org/bearer` with Bearer token auth
- Responses cached for 5 minutes
- Automatic retries on server errors

**Response:**
```json
{
  "statusCode": 200,
  "headers": {
    "content-type": "application/json",
    "authorization": "[REDACTED]"
  },
  "body": {
    "authenticated": true,
    "token": "your-bearer-token-here"
  },
  "url": "https://httpbin.org/bearer",
  "method": "GET"
}
```

### POST /api/v1/http-demo
**API Key Authentication with POST Data:**
- Sends JSON payload to `https://httpbin.org/post`
- Uses `X-API-Key` header for authentication
- Includes custom headers and request body
- Aggressive retry strategy for resilience

**Request Body:**
```json
{
  "message": "Hello from KDeps!",
  "timestamp": "2024-01-15T10:30:00Z",
  "user_id": "12345"
}
```

**Response:**
```json
{
  "statusCode": 200,
  "headers": {
    "content-type": "application/json"
  },
  "body": {
    "data": {
      "message": "Hello from KDeps!",
      "timestamp": "2024-01-15T10:30:00Z",
      "user_id": "12345"
    },
    "headers": {
      "X-Api-Key": "[REDACTED]"
    }
  }
}
```

## ⚙️ Configuration Examples

### Bearer Token Authentication
```yaml
httpClient:
  auth:
    type: bearer
    token: "{{ get('api_token') }}"
```

### API Key Authentication
```yaml
httpClient:
  auth:
    type: api_key
    key: "X-API-Key"
    value: "{{ get('api_key') }}"
```

### Advanced Retry Configuration
```yaml
httpClient:
  retry:
    maxAttempts: 5
    backoff: 500ms      # Initial delay
    maxBackoff: 10s     # Maximum delay
    retryOn: [429, 500, 502, 503]  # Status codes to retry
```

### Response Caching
```yaml
httpClient:
  cache:
    enabled: true
    ttl: 5m
    key: "custom_cache_key"
```

### Proxy and TLS Configuration
```yaml
httpClient:
  proxy: "http://proxy.company.com:8080"
  tls:
    insecureSkipVerify: false
    certFile: "/path/to/client.crt"
    keyFile: "/path/to/client.key"
    caFile: "/path/to/ca.crt"
  followRedirects: false
```

## 🧪 Testing the Features

1. **Bearer Token Authentication:**
```bash
curl "http://localhost:16395/api/v1/http-demo"
# Uses cached response on subsequent calls
```

2. **API Key Authentication:**
```bash
curl -X POST "http://localhost:16395/api/v1/http-demo" \
  -H "Content-Type: application/json" \
  -d '{"custom_header": "test-value"}'
```

3. **Simulate Failures** (for retry testing):
```bash
# Temporarily change URLs to non-existent endpoints
# Watch logs for retry attempts with exponential backoff
```

## 🔧 Environment Variables

Set these environment variables for authentication:

```bash
export API_TOKEN="your-bearer-token-here"
export API_KEY="your-api-key-here"
```

Or pass them as query parameters:
```bash
curl "http://localhost:16395/api/v1/http-demo?api_token=your-token&api_key=your-key"
```

## 🚀 Performance Benefits

- **Caching**: Reduces API calls and improves response times
- **Retries**: Automatic recovery from transient failures
- **Connection Reuse**: Efficient HTTP client pooling
- **Timeout Management**: Prevents hanging requests

## 🔒 Security Features

- **Token Masking**: Sensitive headers are redacted in logs
- **TLS Verification**: Configurable certificate validation
- **Proxy Support**: Enterprise proxy compatibility
- **Header Sanitization**: Safe header value handling

## 📊 Monitoring & Debugging

Each HTTP response includes:
- Status code and headers
- Request URL and method
- Response body (JSON parsed or raw)
- Timing information
- Retry attempt counts

This enables comprehensive monitoring of API interactions and debugging of integration issues.
