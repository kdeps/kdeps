# Request Object

The `request` object provides access to HTTP request data and methods for working with uploaded files. It's available in all expressions and `expr` blocks.

## Overview

The `request` object is automatically available in expressions when running in API server mode. It provides both property access and method-based access to request data.

## Properties

Access request data as properties:

```yaml
run:
  expr:
    - set('method', request.method)
    - set('path', request.path)
    - set('ip', request.IP)
    - set('id', request.ID)
    - set('headers', request.headers)
    - set('query', request.query)
    - set('body', request.body)
```

### Available Properties

| Property | Type | Description |
|----------|------|-------------|
| `request.method` | string | HTTP method (GET, POST, etc.) |
| `request.path` | string | Request URL path |
| `request.IP` | string | Client IP address |
| `request.ID` | string | Unique request identifier |
| `request.headers` | object | All request headers (map) |
| `request.query` | object | Query parameters (map) |
| `request.body` | object | Request body data (map) |

## Methods

### File Access Methods

#### `request.file(name)`

Get the content of an uploaded file by field name.

<div v-pre>

```yaml
run:
  expr:
    - set('fileContent', request.file('document'))
  
  chat:
    prompt: "Analyze this document: {{ get('fileContent') }}"
```

</div>

**Parameters:**
- `name` (string): The form field name of the uploaded file

**Returns:** File content as string (for text files) or nil if not found

#### `request.filepath(name)`

Get the file path of an uploaded file.

<div v-pre>

```yaml
run:
  expr:
    - set('filePath', request.filepath('image'))
  
  python:
    script: |
      from PIL import Image
      img = Image.open("{{ get('filePath') }}")
      # Process image...
```

</div>

**Parameters:**
- `name` (string): The form field name of the uploaded file

**Returns:** File path (string) or nil if not found

#### `request.filetype(name)`

Get the MIME type of an uploaded file.

```yaml
run:
  expr:
    - set('mimeType', request.filetype('upload'))
    - set('isImage', get('mimeType') startsWith 'image/')
```

**Parameters:**
- `name` (string): The form field name of the uploaded file

**Returns:** MIME type (string) or nil if not found

#### `request.filesByType(mimeType)`

Get all uploaded files matching a specific MIME type.

```yaml
run:
  expr:
    - set('images', request.filesByType('image/png'))
    - set('pdfs', request.filesByType('application/pdf'))
```

**Parameters:**
- `mimeType` (string): MIME type to filter by (e.g., "image/png", "image/*")

**Returns:** Array of file paths matching the MIME type

**Example:**
```yaml
run:
  expr:
    - set('allImages', request.filesByType('image/*'))
    - set('imageCount', len(get('allImages')))
```

### File Information Methods

#### `request.filecount()`

Get the total number of uploaded files.

```yaml
run:
  expr:
    - set('fileCount', request.filecount())
  
  preflightCheck:
    validations:
      - request.filecount() > 0
    error:
      code: 400
      message: At least one file is required
```

**Returns:** Number of uploaded files (integer)

#### `request.files()`

Get a list of all uploaded file paths.

```yaml
run:
  expr:
    - set('allFiles', request.files())
    - set('fileList', join(get('allFiles'), ', '))
```

**Returns:** Array of file paths (strings)

#### `request.filetypes()`

Get a list of MIME types for all uploaded files.

```yaml
run:
  expr:
    - set('types', request.filetypes())
    - set('hasImages', contains(get('types'), 'image/png'))
```

**Returns:** Array of MIME types (strings)

### Request Data Methods

#### `request.data()`

Get the entire request body as an object.

```yaml
run:
  expr:
    - set('requestData', request.data())
    - set('userId', get('requestData').userId)
```

**Returns:** Request body object (map) or empty object if no body

**Note:** This is equivalent to accessing `request.body` directly.

#### `request.params(name)`

Get a query parameter value.

```yaml
run:
  expr:
    - set('userId', request.params('userId'))
    - set('page', request.params('page'))
```

**Parameters:**
- `name` (string): Query parameter name

**Returns:** Parameter value (string) or nil if not found

**Note:** This is equivalent to `get('name', 'param')` or accessing `request.query.name`.

#### `request.header(name)`

Get a request header value.

```yaml
run:
  expr:
    - set('auth', request.header('Authorization'))
    - set('contentType', request.header('Content-Type'))
```

**Parameters:**
- `name` (string): Header name (case-insensitive)

**Returns:** Header value (string) or nil if not found

**Note:** This is equivalent to `get('HeaderName', 'header')` or accessing `request.headers.HeaderName`.

## Examples

### File Upload Processing

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: processUpload
run:
  expr:
    # Check file count
    - set('fileCount', request.filecount())
    - set('hasFiles', get('fileCount') > 0)
  
  preflightCheck:
    validations:
      - request.filecount() > 0
    error:
      code: 400
      message: At least one file is required
  
  expr:
    # Get file information
    - set('filePath', request.filepath('document'))
    - set('fileType', request.filetype('document'))
    - set('isPDF', get('fileType') == 'application/pdf')
  
  chat:
    model: llama3.2-vision
    prompt: |
      Analyze this {{ get('fileType') }} file.
      Path: {{ get('filePath') }}
    files:
      - "{{ get('filePath') }}"
```

</div>

### Multi-File Processing

<div v-pre>

```yaml
run:
  expr:
    # Get all images
    - set('images', request.filesByType('image/*'))
    - set('imageCount', len(get('images')))
    
    # Get all PDFs
    - set('pdfs', request.filesByType('application/pdf'))
    
    # Process each image
    - set('imageList', join(get('images'), '\n'))
  
  chat:
    model: llama3.2-vision
    prompt: |
      Process {{ get('imageCount') }} images:
      {{ get('imageList') }}
```

</div>

### Request Metadata

<div v-pre>

```yaml
run:
  expr:
    - set('requestInfo', {
        "method": request.method,
        "path": request.path,
        "ip": request.IP,
        "id": request.ID,
        "fileCount": request.filecount()
      })
  
  apiResponse:
    response:
      request: get('requestInfo')
      timestamp: info('timestamp')
```

</div>

### Conditional Processing Based on Request

```yaml
run:
  expr:
    - set('isPost', request.method == 'POST')
    - set('isApiPath', request.path startsWith '/api/')
    - set('hasAuth', request.header('Authorization') != nil)
  
  skipCondition:
    - !get('isPost')
    - !get('isApiPath')
  
  preflightCheck:
    validations:
      - request.header('Authorization') != ''
    error:
      code: 401
      message: Authorization required
```

### File Type Validation

```yaml
run:
  expr:
    - set('fileTypes', request.filetypes())
    - set('allowedTypes', ['image/png', 'image/jpeg', 'application/pdf'])
    - set('isValid', all(get('fileTypes'), . in get('allowedTypes')))
  
  preflightCheck:
    validations:
      - get('isValid')
    error:
      code: 400
      message: Only PNG, JPEG, and PDF files are allowed
```

## Comparison with Unified API

The `request` object provides an alternative way to access request data. Both approaches work:

| Request Object | Unified API | Description |
|----------------|------------|-------------|
| `request.method` | `info('method')` | HTTP method |
| `request.path` | `info('path')` | Request path |
| `request.IP` | `info('clientIp')` | Client IP |
| `request.ID` | `info('requestId')` | Request ID |
| `request.params('key')` | `get('key', 'param')` | Query parameter |
| `request.header('Name')` | `get('Name', 'header')` | Request header |
| `request.data()` | `request.body` | Request body |
| `request.file('name')` | `get('name', 'file')` | File content |
| `request.filepath('name')` | `get('name', 'filepath')` | File path |
| `request.filetype('name')` | `get('name', 'filetype')` | File MIME type |
| `request.filecount()` | `info('filecount')` | File count |
| `request.files()` | `info('files')` | All file paths |
| `request.filetypes()` | `info('filetypes')` | All MIME types |

**Recommendation:** Use the Unified API (`get()`, `info()`) for consistency, but `request` object methods are available for convenience and backward compatibility.

## Best Practices

1. **Use for file operations** - `request.file()`, `request.filepath()`, etc. are convenient for file handling
2. **Check file count first** - Use `request.filecount()` before accessing files
3. **Validate file types** - Use `request.filetypes()` or `request.filetype()` to validate uploads
4. **Access request metadata** - Use `request.method`, `request.path`, etc. for routing logic
5. **Combine with expressions** - Use in `expr` blocks for pre-processing

## See Also

- [Unified API](unified-api) - Primary API for data access
- [File Upload Tutorial](../tutorials/file-upload) - Working with file uploads
- [Info Function](unified-api#info-function) - Request metadata access
