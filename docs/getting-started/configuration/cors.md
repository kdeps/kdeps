---
outline: deep
---

# CORS Configuration

Cross-Origin Resource Sharing (CORS) configuration allows you to control how the Kdeps API Server handles cross-origin
HTTP requests. By defining CORS settings, you can specify which origins, methods, and headers are allowed, ensuring
secure and controlled access to your API resources.

CORS settings are defined within the `APIServer` configuration under the `cors` block. These settings are particularly
useful when your API is accessed by web applications hosted on different domains.

## Defining a `cors` Configuration

To configure CORS, you define the `cors` block inside the `APIServer` configuration. The `cors` block includes fields to
enable CORS, specify allowed origins, methods, headers, and other settings. Below are examples of common CORS
configurations.

### Example 1: Enabling CORS for a Specific Origin

In this scenario, CORS is enabled for a specific origin, allowing only requests from `https://example.com` with specific
HTTP methods.


```apl
APIServer {
    CORS {
        EnableCORS = true
        AllowOrigins {
            "https://example.com"
        }
        AllowMethods {
            "GET"
            "POST"
        }
        AllowHeaders {
            "Content-Type"
            "Authorization"
        }
        AllowCredentials = true
        MaxAge = 24.h
    }
}
```

This configuration allows `https://example.com` to make `GET` and `POST` requests to the API, including credentials
(e.g., cookies), with a preflight cache duration of 24 hours.

### Example 2: Allowing All Origins for Development

For development purposes, you might want to allow all origins temporarily. This configuration enables CORS for any
origin but restricts the allowed methods and headers.

```apl
APIServer {
    CORS {
        EnableCORS = true
        AllowOrigins {
            "*"
        }
        AllowMethods {
            "GET"
            "OPTIONS"
        }
        AllowHeaders {
            "Content-Type"
        }
        ExposeHeaders {
            "X-Custom-Header"
        }
        AllowCredentials = false
        MaxAge = 12.h
    }
}
```

This setup allows any origin to make `GET` and `OPTIONS` requests, exposes a custom response header, and disables
credentials for broader compatibility.

## CORS Configuration Fields

The `cors` block supports several fields to customize cross-origin request handling. Below is a table of available
fields and their descriptions:

| **Field**            | **Description**                                                                 |
|----------------------|---------------------------------------------------------------------------------|
| `EnableCORS`         | Enables or disables CORS support (Boolean, default: `false`).                   |
| `AllowOrigins`       | List of allowed origin domains (e.g., `"https://example.com"`). Use `"*"` for all origins. If unset, no origins are allowed unless CORS is disabled. |
| `AllowMethods`       | List of HTTP methods allowed for CORS requests (e.g., `"GET"`, `"POST"`). Must be one of: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`. If unset, defaults to route methods. |
| `AllowHeaders`       | List of request headers allowed in CORS requests (e.g., `"Content-Type"`). If unset, no additional headers are allowed. |
| `ExposeHeaders`      | List of response headers exposed to clients (e.g., `"X-Custom-Header"`). If unset, no headers are exposed beyond defaults. |
| `AllowCredentials`   | Allows credentials (e.g., cookies, HTTP authentication) in CORS requests (Boolean, default: `true`). |
| `MaxAge`             | Maximum duration for caching CORS preflight responses (Duration, default: `12.h`). |

## Best Practices

- **Restrict Origins in Production**: Use specific domains in `AllowOrigins` (e.g., `"https://yourapp.com"`) instead of `"*"` to enhance security.
- **Limit Methods and Headers**: Only allow the HTTP methods and headers required by your API to minimize the attack surface.
- **Adjust `MaxAge` Carefully**: Set a reasonable `MaxAge` (e.g., `12.h` or `24.h`) to balance performance and flexibility for preflight requests.
- **Disable Credentials When Possible**: Set `AllowCredentials = false` if your API doesn't require cookies or authentication headers to simplify CORS handling.

By tailoring the `cors` configuration to your API’s requirements, you can ensure secure and efficient cross-origin request handling.
