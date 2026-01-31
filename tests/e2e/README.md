# KDeps E2E Tests

End-to-end tests for KDeps that run the actual `kdeps` binary to verify functionality.

## Structure

The E2E tests are modularized into separate shell scripts for each test scenario:

- **`e2e.sh`** - Main test runner that orchestrates all test scenarios
- **`common.sh`** - Common functions and setup (binary detection, test helpers, counters)
- **`test_validation.sh`** - Tests workflow validation
- **`test_scaffolding.sh`** - Tests scaffolding commands (`new`, `scaffold`)
- **`test_packaging.sh`** - Tests package command
- **`test_build.sh`** - Tests build command (requires Docker)
- **`test_docker_services.sh`** - Tests that kdeps services are running in Docker containers (includes full container build/run/test workflow)
- **`test_execution.sh`** - Tests run command
- **`test_examples.sh`** - Orchestrates individual example tests
- **`test_examples_chatbot.sh`** - Comprehensive tests for chatbot example
- **`test_examples_http_advanced.sh`** - Comprehensive tests for http-advanced example
- **`test_examples_shell_exec.sh`** - Comprehensive tests for shell-exec example
- **`test_examples_sql_advanced.sh`** - Comprehensive tests for sql-advanced example
- **`test_features_file_upload.sh`** - E2E tests for file upload handling feature
- **`test_features_input_validation.sh`** - E2E tests for input validation framework
- **`test_features_session.sh`** - E2E tests for session persistence and TTL
- **`test_features_error_handling.sh`** - E2E tests for error handling scenarios (404, 405, validation errors)
- **`test_features_cors.sh`** - E2E tests for CORS configuration (preflight, origin headers)
- **`test_features_health_check.sh`** - E2E tests for health check endpoint (`/health`)
- **`test_features_memory_storage.sh`** - E2E tests for memory storage (persistent across requests)
- **`test_features_python_executor.sh`** - E2E tests for Python executor and script execution
- **`test_features_multi_resource.sh`** - E2E tests for workflows with multiple resources and dependencies
- **`test_features_expression_eval.sh`** - E2E tests for complex expression evaluation (arithmetic, comparisons)
- **`test_features_items_iteration.sh`** - E2E tests for items iteration (processing arrays of items)
- **`test_features_workflow_metadata.sh`** - E2E tests for workflow metadata access via `info()` functions
- **`test_features_route_methods.sh`** - E2E tests for route method restrictions (GET, POST, PUT, DELETE, PATCH)

## Usage

### Run all tests

```bash
./tests/e2e/e2e.sh
```

### Run individual test scenarios

Each test script can be run independently:

```bash
./tests/e2e/test_validation.sh
./tests/e2e/test_scaffolding.sh
./tests/e2e/test_packaging.sh
./tests/e2e/test_build.sh
./tests/e2e/test_execution.sh
```

## Requirements

- `kdeps` binary must be available (either in project root or in PATH)
- `timeout` command (usually available on Linux/macOS)
- For build tests: Docker must be installed and running
- For example tests: `curl` command (for HTTP endpoint testing)
- For example tests: `lsof`, `netstat`, or `ss` (for port checking)

## Test Output

Tests provide color-coded output:
- ✓ **Green** - Test passed
- ✗ **Red** - Test failed
- ⊘ **Yellow** - Test skipped (usually due to missing dependencies)

## Adding New Tests

To add a new test scenario:

1. Create a new script `test_<scenario>.sh`
2. Source `common.sh` at the top:
   ```bash
   source "$SCRIPT_DIR/common.sh"
   ```
3. Use the test helper functions:
   - `test_passed "test name"`
   - `test_failed "test name" "error message"`
   - `test_skipped "test name"`
4. Add the script to `e2e.sh`:
   ```bash
   source "$SCRIPT_DIR/test_<scenario>.sh"
   ```

## Notes

- Tests are designed to be lenient with environment-specific failures
- Build tests will skip if Docker is not available
- Run tests will handle timeouts gracefully for server processes
- Example tests verify that servers start and endpoints are accessible, but may pass even if dependencies (LLM models, databases) are missing
- All temporary files are cleaned up automatically

## Example Tests

Each example has its own comprehensive test script that verifies:

### Chatbot Example (`test_examples_chatbot.sh`)
- ✅ Workflow validation
- ✅ Resource files exist
- ✅ Server startup
- ✅ POST endpoint to `/api/v1/chat` with query parameter
- ✅ GET endpoint rejection (method restriction)
- ✅ Response structure validation

### HTTP Advanced Example (`test_examples_http_advanced.sh`)
- ✅ Workflow validation
- ✅ Resource files exist (4 resources)
- ✅ Server startup
- ✅ GET endpoint to `/api/v1/http-demo`
- ✅ POST endpoint testing
- ✅ Response structure validation

### Shell Exec Example (`test_examples_shell_exec.sh`)
- ✅ Workflow validation
- ✅ Resource files exist
- ✅ Server startup
- ✅ GET endpoint to `/api/v1/exec`
- ✅ POST endpoint testing
- ✅ System info in response

### SQL Advanced Example (`test_examples_sql_advanced.sh`)
- ✅ Workflow validation
- ✅ Resource files exist (3 resources)
- ✅ SQL connections configuration
- ✅ Server startup (may skip if database unavailable)
- ✅ GET endpoint testing (if server starts)

All example tests verify that:
1. Workflows can be parsed and validated
2. Required resource files exist
3. Servers start successfully on configured ports
4. API endpoints are accessible and respond correctly
5. Method restrictions work (GET vs POST)
6. Response JSON structures match expected formats from READMEs
7. The examples work end-to-end (even if some dependencies like LLM models or databases are missing)

## Feature Tests

In addition to example tests, the suite includes comprehensive feature tests:

### File Upload Handling (`test_features_file_upload.sh`)
- ✅ Workflow validation with file upload support
- ✅ Server startup with file upload configuration
- ✅ Single file upload via POST with `multipart/form-data`
- ✅ Multiple file uploads
- ✅ File count, names, types accessible via `info()` functions
- ✅ File content, path, and MIME type accessible via `get()` functions

### Input Validation Framework (`test_features_input_validation.sh`)
- ✅ Workflow validation with validation rules
- ✅ Server startup with validation configuration
- ✅ Required field validation (returns 400 on missing fields)
- ✅ Field type validation (string, number)
- ✅ Pattern validation (email regex)
- ✅ Min/max length and value validation
- ✅ Valid input acceptance (returns 200)

### Session Persistence (`test_features_session.sh`)
- ✅ Workflow validation with session configuration
- ✅ Server startup with session storage (SQLite in-memory)
- ✅ Session ID generation and return
- ✅ Session value storage and retrieval
- ✅ Session cookies (if applicable)
- ✅ Query parameter accessibility via `get()`

### Error Handling (`test_features_error_handling.sh`)
- ✅ Workflow validation
- ✅ Server startup
- ✅ 404 Not Found error handling
- ✅ Method not allowed (405) error handling
- ✅ Invalid JSON body handling
- ✅ Success response structure (200 OK)
- ✅ Error response structure validation:
  - `{"success": false, "error": {"code": "...", "message": "..."}, "meta": {...}}`
  - Meta fields: `requestID`, `timestamp`, `path`, `method`

### CORS Configuration (`test_features_cors.sh`)
- ✅ Workflow validation with CORS enabled
- ✅ Server startup with CORS configuration
- ✅ OPTIONS preflight request handling
- ✅ Access-Control-Allow-Origin header presence
- ✅ Access-Control-Allow-Methods header
- ✅ Access-Control-Allow-Headers header
- ✅ Origin header validation (allowed origins)
- ✅ Wildcard origin support (`*`)
- ✅ CORS headers in actual requests (GET, POST)

### Health Check Endpoint (`test_features_health_check.sh`)
- ✅ Workflow validation
- ✅ Server startup
- ✅ GET `/health` endpoint (200 OK)
- ✅ Response structure: `{"status": "ok", "workflow": {"name": "...", "version": "..."}}`
- ✅ Workflow name and version in response
- ✅ Accessible without authentication

### Memory Storage (`test_features_memory_storage.sh`)
- ✅ Workflow validation with memory storage
- ✅ Server startup
- ✅ Set value in memory endpoint
- ✅ Get value from memory endpoint
- ✅ Memory persistence across requests (if implemented)
- ✅ Response structure validation

### Python Executor (`test_features_python_executor.sh`)
- ✅ Workflow validation with Python resource
- ✅ Server startup with Python configuration
- ✅ Python script execution via POST endpoint
- ✅ Python stdout capture and return
- ✅ Python result accessible via `get()` in dependent resources
- ✅ Response structure with Python execution results

### Multi-Resource Workflows (`test_features_multi_resource.sh`)
- ✅ Workflow validation with multiple resources
- ✅ Resource dependency definitions (`requires` field)
- ✅ All resource files exist
- ✅ Server startup
- ✅ Dependency chain execution (firstStep → secondStep → finalStep)
- ✅ Resource outputs accessible via `get()` in dependent resources
- ✅ Combined results from multiple resources

### Expression Evaluation (`test_features_expression_eval.sh`)
- ✅ Workflow validation with complex expressions
- ✅ Server startup
- ✅ Arithmetic expressions (addition, multiplication)
- ✅ Comparison expressions (greater than)
- ✅ Expression results in response
- ✅ Input values accessible in expressions
- ✅ Type validation (rejects strings for number fields)

### Items Iteration (`test_features_items_iteration.sh`)
- ✅ Workflow validation with items iteration
- ✅ Items field defined in resource
- ✅ Multiple items defined (array of items)
- ✅ Server startup
- ✅ Items iteration endpoint execution
- ✅ Returns array of results (one per item)
- ✅ Iteration context variables accessible (`item`, `index`, `count`)

### Workflow Metadata (`test_features_workflow_metadata.sh`)
- ✅ Workflow validation
- ✅ Server startup
- ✅ GET endpoint for metadata
- ✅ Workflow name accessible via `info('name')`
- ✅ Workflow version accessible via `info('version')`
- ✅ Current time accessible via `info('current_time')`
- ✅ Response structure validation

### Route Method Restrictions (`test_features_route_methods.sh`)
- ✅ Workflow validation with multiple method handlers
- ✅ Multiple resource files (GET handler, POST handler)
- ✅ Method restrictions defined (`restrictToHttpMethods`)
- ✅ Server startup
- ✅ GET method handling
- ✅ POST method handling
- ✅ PUT method restriction (405/404/400 when not configured)
- ✅ DELETE method restriction
- ✅ PATCH method restriction (not in allowed methods)

### Docker Services Verification (`test_docker_services.sh`)
- ✅ **Service Detection**: Checks for running containers with kdeps/ollama services
- ✅ **Process Verification**: Confirms both kdeps and ollama processes are running in containers
- ✅ **Health Endpoint Testing**: Validates health check endpoints when available
- ✅ **Full Container E2E Testing**: Complete end-to-end workflow that:
  - Packages a KDeps workflow example with Ollama
  - Builds a Docker container image
  - Runs the container and verifies startup
  - Checks that BOTH kdeps AND ollama processes are running inside the container
  - Tests health check endpoints
  - Performs complete cleanup of containers and images
- ✅ **Port Mapping Verification**: Checks container port exposure and accessibility
- ✅ **Environment Compatibility**: Handles different Docker configurations gracefully

All feature tests verify:
1. Workflows can be created and validated with feature-specific configurations
2. Servers start successfully with feature configurations
3. Features work as expected (where applicable)
4. Error handling is graceful and structured
5. Response formats match expected API structures
6. Dependencies between resources are resolved correctly
7. Method restrictions work as expected
8. Metadata and info functions return correct values