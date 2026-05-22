# Request Object

The `request` object gives you direct access to the HTTP request -- method, path, headers, query params, body, and uploaded files. It is available in all expressions and `expr` blocks when running in workflow mode.

## Properties

Access request data as properties:

```yaml
after:
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
after:
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
after:
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
after:
  - set('mimeType', request.filetype('upload'))
  - set('isImage', get('mimeType') startsWith 'image/')
```

**Parameters:**
- `name` (string): The form field name of the uploaded file

**Returns:** MIME type (string) or nil if not found

#### `request.filesByType(mimeType)`

Get all uploaded files matching a specific MIME type.

```yaml
after:
  - set('images', request.filesByType('image/png'))
  - set('pdfs', request.filesByType('application/pdf'))
```

**Parameters:**
- `mimeType` (string): MIME type to filter by (e.g., "image/png", "image/*")

**Returns:** Array of file paths matching the MIME type

**Example:**
```yaml
after:
  - set('allImages', request.filesByType('image/*'))
  - set('imageCount', len(get('allImages')))
```

### File Information Methods

#### `request.filecount()`

Get the total number of uploaded files.

```yaml
after:
  - set('fileCount', request.filecount())

validations:
  check:
    - request.filecount() > 0
  error:
    code: 400
    message: At least one file is required
```

**Returns:** Number of uploaded files (integer)

#### `request.files()`

Get a list of all uploaded file paths.

```yaml
after:
  - set('allFiles', request.files())
  - set('fileList', join(get('allFiles'), ', '))
```

**Returns:** Array of file paths (strings)

#### `request.filetypes()`

Get a list of MIME types for all uploaded files.

```yaml
after:
  - set('types', request.filetypes())
  - set('hasImages', contains(get('types'), 'image/png'))
```

**Returns:** Array of MIME types (strings)

### Request Data Methods

#### `request.data()`

Get the entire request body as an object.

```yaml
after:
  - set('requestData', request.data())
  - set('userId', get('requestData').userId)
```

**Returns:** Request body object (map) or empty object if no body

**Note:** This is equivalent to accessing `request.body` directly.

#### `request.params(name)`

Get a query parameter value.

```yaml
after:
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
after:
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
actionId: processUpload
after:
  # Check file count
  - set('fileCount', request.filecount())
  - set('hasFiles', get('fileCount') > 0)

validations:
  check:
    - request.filecount() > 0
  error:
    code: 400
    message: At least one file is required

after:
  # Get file information
  - set('filePath', request.filepath('document'))
  - set('fileType', request.filetype('document'))
  - set('isPDF', get('fileType') == 'application/pdf')

chat:
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
after:
  # Get all images
  - set('images', request.filesByType('image/*'))
  - set('imageCount', len(get('images')))

  # Get all PDFs
  - set('pdfs', request.filesByType('application/pdf'))

  # Process each image
  - set('imageList', join(get('images'), '\n'))

chat:
  prompt: |
    Process {{ get('imageCount') }} images:
    {{ get('imageList') }}
```

</div>

### Request Metadata

<div v-pre>

```yaml
after:
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
after:
  - set('isPost', request.method == 'POST')
  - set('isApiPath', request.path startsWith '/api/')
  - set('hasAuth', request.header('Authorization') != nil)

validations:
  skip:
  - "!get('isPost')"
  - "!get('isApiPath')"
  check:
    - request.header('Authorization') != ''
  error:
    code: 401
    message: Authorization required
```

### File Type Validation

```yaml
after:
  - set('fileTypes', request.filetypes())
  - set('allowedTypes', ['image/png', 'image/jpeg', 'application/pdf'])
  - set('isValid', all(get('fileTypes'), . in get('allowedTypes')))

validations:
  check:
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

**Recommendation:** Use the Unified API (`get()`, `info()`) for consistency, but `request` object methods are available as a convenience shorthand.

## Best Practices

1. **Use for file operations** - `request.file()`, `request.filepath()`, etc. are convenient for file handling
2. **Check file count first** - Use `request.filecount()` before accessing files
3. **Validate file types** - Use `request.filetypes()` or `request.filetype()` to validate uploads
4. **Access request metadata** - Use `request.method`, `request.path`, etc. for routing logic
5. **Combine with expressions** - Use in `expr` blocks for pre-processing

## See Also

- [Unified API](/concepts/unified-api) - Primary API for data access
- [Input Sources](input-sources) - Configuring API, bot, and file input sources
- [Info Function](unified-api#info-function) - Request metadata access
