# CORS Configuration

Cross-Origin Resource Sharing (CORS) configuration allows you to control how the KDeps API Server handles cross-origin HTTP requests. By defining CORS settings, you can specify which origins, methods, and headers are allowed, ensuring secure and controlled access to your API resources.

CORS settings are defined within the `apiServer` configuration under the `cors` block. These settings are particularly useful when your API is accessed by web applications hosted on different domains.

## Basic Configuration

To configure CORS, you define the `cors` block inside the `apiServer` configuration:

```yaml
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
        - https://myapp.com
      allowMethods:
        - GET
        - POST
        - OPTIONS
      allowHeaders:
        - Content-Type
        - Authorization
      allowCredentials: true
      maxAge: "24h"
```

## Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `enableCors` | boolean | Enables or disables CORS support (default: `false`) |
| `allowOrigins` | array | List of allowed origin domains. Use `"*"` for all origins. If unset, no origins are allowed unless CORS is disabled. |
| `allowMethods` | array | List of HTTP methods allowed for CORS requests. Must be one of: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`. If unset, defaults to route methods. |
| `allowHeaders` | array | List of request headers allowed in CORS requests (e.g., `Content-Type`, `Authorization`). If unset, no additional headers are allowed. |
| `exposeHeaders` | array | List of response headers exposed to clients (e.g., `X-Request-Id`). If unset, no headers are exposed beyond defaults. |
| `allowCredentials` | boolean | Allows credentials (e.g., cookies, HTTP authentication) in CORS requests (default: `true`) |
| `maxAge` | string | Maximum duration for caching CORS preflight responses (e.g., `"24h"`, `"12h"`). Default: `"12h"` |

## Common Scenarios

### Development: Allow All Origins

For development purposes, you might want to allow all origins temporarily:

```yaml
cors:
  enableCors: true
  allowOrigins:
    - "*"
  allowMethods:
    - GET
    - POST
    - OPTIONS
  allowHeaders:
    - Content-Type
  allowCredentials: false
  maxAge: "12h"
```

**Note**: When using `"*"` for `allowOrigins`, you must set `allowCredentials: false` as browsers don't allow credentials with wildcard origins.

### Production: Specific Origins

For production, restrict to specific domains:

```yaml
cors:
  enableCors: true
  allowOrigins:
    - https://myapp.com
    - https://www.myapp.com
    - https://admin.myapp.com
  allowMethods:
    - GET
    - POST
    - PUT
    - DELETE
  allowHeaders:
    - Content-Type
    - Authorization
    - X-Requested-With
  exposeHeaders:
    - X-Request-Id
    - X-Rate-Limit
  allowCredentials: true
  maxAge: "24h"
```

### Multiple Environments

You can configure different CORS settings for different environments:

```yaml
# Development
cors:
  enableCors: true
  allowOrigins:
    - http://localhost:3000
    - http://localhost:8080
  allowCredentials: false

# Production
cors:
  enableCors: true
  allowOrigins:
    - https://myapp.com
  allowCredentials: true
  maxAge: "24h"
```

## How CORS Works

1. **Simple Requests**: For simple requests (GET, POST with certain content types), the browser sends the request directly with an `Origin` header.

2. **Preflight Requests**: For complex requests (PUT, DELETE, custom headers), the browser first sends an OPTIONS request (preflight) to check if the actual request is allowed.

3. **Response Headers**: The server responds with CORS headers:
   - `Access-Control-Allow-Origin`: Which origins are allowed
   - `Access-Control-Allow-Methods`: Which HTTP methods are allowed
   - `Access-Control-Allow-Headers`: Which headers can be sent
   - `Access-Control-Expose-Headers`: Which response headers can be read
   - `Access-Control-Allow-Credentials`: Whether credentials are allowed
   - `Access-Control-Max-Age`: How long to cache preflight responses

## Best Practices

### Security

- **Restrict Origins in Production**: Use specific domains in `allowOrigins` instead of `"*"` to enhance security.
- **Limit Methods and Headers**: Only allow the HTTP methods and headers required by your API to minimize the attack surface.
- **Use HTTPS**: Always use HTTPS in production to protect credentials and data in transit.

### Performance

- **Set Appropriate MaxAge**: Set a reasonable `maxAge` (e.g., `"12h"` or `"24h"`) to balance performance and flexibility for preflight requests.
- **Minimize Exposed Headers**: Only expose headers that clients actually need.

### Compatibility

- **Disable Credentials When Possible**: Set `allowCredentials: false` if your API doesn't require cookies or authentication headers to simplify CORS handling.
- **Handle Preflight Requests**: Ensure your routes support `OPTIONS` method for preflight requests.

## Troubleshooting

### CORS Errors in Browser

If you see CORS errors in the browser console:

1. **Check `enableCors`**: Ensure `enableCors: true` is set.
2. **Verify Origins**: Make sure your frontend origin is in the `allowOrigins` list.
3. **Check Methods**: Ensure the HTTP method is in `allowMethods`.
4. **Check Headers**: Verify custom headers are in `allowHeaders`.
5. **Credentials Mismatch**: If using credentials, ensure `allowCredentials: true` and origins are not `"*"`.

### Common Error Messages

- **"No 'Access-Control-Allow-Origin' header"**: CORS is disabled or origin not in `allowOrigins`.
- **"Method not allowed"**: HTTP method not in `allowMethods`.
- **"Header not allowed"**: Custom header not in `allowHeaders`.
- **"Credentials not allowed"**: Using credentials with wildcard origin (`"*"`).

## Example: Full Configuration

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: cors-example
  version: "1.0.0"
  targetActionId: apiHandler

settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 3000
    routes:
      - path: /api/v1/data
        methods: [GET, POST, PUT, DELETE]
    cors:
      enableCors: true
      allowOrigins:
        - https://myapp.com
        - https://admin.myapp.com
      allowMethods:
        - GET
        - POST
        - PUT
        - DELETE
        - OPTIONS
      allowHeaders:
        - Content-Type
        - Authorization
        - X-Requested-With
        - X-API-Key
      exposeHeaders:
        - X-Request-Id
        - X-Rate-Limit-Remaining
      allowCredentials: true
      maxAge: "24h"
```

## Related Documentation

- [Workflow Configuration](workflow.md) - Full workflow settings reference
- [API Server Settings](workflow.md#api-server-settings) - Complete API server configuration
- [WebServer Mode](../deployment/webserver.md) - Serving static files and proxying
