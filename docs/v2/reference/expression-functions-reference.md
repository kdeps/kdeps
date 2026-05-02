# Expression Functions Reference

This reference lists all available functions in KDeps expressions. These functions can be used in any field that supports string interpolation (<span v-pre>`{{ }}`</span>) or in `expr` blocks.

## Core Functions

### get(key, typeHint?)
Retrieves data from any source.

- **key**: The key to look up.
- **typeHint** (optional): Force a specific source (`param`, `header`, `session`, `memory`, `item`, `env`, `file`, `filepath`, `filetype`).

**Examples:**
```yaml
get('q')                     # Auto-detect source
get('Authorization', 'header') # Get from headers
get('user_id', 'session')    # Get from session
get('API_KEY', 'env')        # Get environment variable
```

### set(key, value, storage?)
Stores data in memory or session.

- **key**: The key to store.
- **value**: The value to store.
- **storage** (optional): Storage type (`memory` or `session`). Default is `memory`.

**Examples:**
```yaml
set('count', 1)              # Store in memory (request-scoped)
set('user', data, 'session') # Store in session (persistent)
```

### file(pattern, selector?)
Accesses uploaded files or local files.

- **pattern**: Glob pattern or file name.
- **selector** (optional): `first`, `last`, `count`, `all`, `mime:<type>`.

**Examples:**
```yaml
file('*.jpg')                # Get all JPG files content
file('*.pdf', 'first')       # Get content of first PDF
file('*', 'count')           # Get total file count
```

### info(field)
Retrieves request metadata.

- **field**: One of `requestId`, `timestamp`, `path`, `method`, `clientIp`, `sessionId`, `filecount`, `files`, `filetypes`.

**Examples:**
```yaml
info('requestId')            # Get request ID
info('clientIp')             # Get client IP
```

## Data Handling Functions

### json(data)
Converts data to a JSON string.

- **data**: The data to stringify.

**Examples:**
```yaml
json(get('userData'))        # Convert object to JSON string
```

### safe(obj, path)
Safely accesses nested properties without panicking on nil values.

- **obj**: The object to access.
- **path**: Dot-notation path to the property.

**Examples:**
```yaml
safe(user, "profile.address.city") # Returns city or nil if path invalid
```

### debug(obj)
Returns a pretty-printed JSON string representation of an object for debugging.

- **obj**: The object to inspect.

**Examples:**
```yaml
debug(get('httpResponse'))   # Inspect complex object structure
```

### default(value, fallback)
Returns a fallback value if the primary value is nil or empty.

- **value**: The value to check.
- **fallback**: The value to return if primary is nil/empty.

**Examples:**
```yaml
default(get('limit'), 10)    # Return 10 if limit is missing
```

## Input/Output Functions

### input(name, type?)
Accesses input data (similar to `get` but strictly for inputs).

- **name**: Input name.
- **type** (optional): `param`, `header`, `body`.

**Examples:**
```yaml
input('q')                   # Get query param
input('body')                # Get request body
```

### output(resourceId)
Accesses the output of a completed resource.

- **resourceId**: The actionId of the resource.

**Examples:**
```yaml
output('llmResource')        # Get LLM output
```

## Iteration Functions

### item(type?)
Accesses current iteration context.

- **type** (optional): `current`, `prev`, `next`, `index`, `count`, `values`.

**Examples:**
```yaml
item('current')              # Current item value
item('index')                # Current index (0-based)
item('count')                # Total items count
```

## Session Functions

### session()
Returns the entire session data object.

**Examples:**
```yaml
session()                    # Get all session data
```