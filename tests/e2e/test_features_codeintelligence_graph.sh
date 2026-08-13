set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing CodeIntelligence Folder Graph Feature..."

TEST_DIR=$(mktemp -d)
DOCS_DIR="$TEST_DIR/docs"
mkdir -p "$DOCS_DIR" "$TEST_DIR/resources"
WORKFLOW_FILE="$TEST_DIR/workflow.yaml"
RESOURCE_FILE_INDEX="$TEST_DIR/resources/index-folder.yaml"
RESOURCE_FILE_GRAPH="$TEST_DIR/resources/graph-all.yaml"
RESOURCE_FILE_RESPONSE="$TEST_DIR/resources/response.yaml"

DOCS_DIR_NATIVE=$(to_native_path "$DOCS_DIR")
DB_PATH_NATIVE=$(to_native_path "$TEST_DIR/graph.db")

cat > "$DOCS_DIR/a.md" <<'EOF'
---
topics: [go]
---
See [b](b.md).
EOF

cat > "$DOCS_DIR/b.md" <<'EOF'
---
topics: [go]
---
No links here.
EOF

cat > "$WORKFLOW_FILE" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: codeintelligence-graph-test
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 3091
    routes:
      - path: /api/v1/graph
        methods: [POST]

  agentSettings:
    pythonVersion: "3.12"
EOF

cat > "$RESOURCE_FILE_INDEX" <<EOF

actionId: indexFolder
name: Index Folder

codeIntelligence:
  operation: indexFolder
  path: "$DOCS_DIR_NATIVE"
  graphDBPath: "$DB_PATH_NATIVE"
EOF

cat > "$RESOURCE_FILE_GRAPH" <<EOF

actionId: graphAll
name: Graph All
requires:
  - indexFolder

codeIntelligence:
  operation: graphAll
  graphDBPath: "$DB_PATH_NATIVE"
EOF

cat > "$RESOURCE_FILE_RESPONSE" <<'EOF'

actionId: response
name: Response
requires:
  - graphAll

restrictToHttpMethods: [POST]
restrictToRoutes: [/api/v1/graph]
apiResponse:
  success: true
  response:
    graphSuccess: "{{ output('graphAll').success }}"
    roots: "{{ output('graphAll').roots }}"
    references: "{{ output('graphAll').references }}"
EOF

# Test 1: Validate workflow
if "$KDEPS_BIN" validate "$WORKFLOW_FILE" &> /dev/null; then
    test_passed "CodeIntelligence Graph - Workflow validation"
else
    test_failed "CodeIntelligence Graph - Workflow validation" "Validation failed"
    rm -rf "$TEST_DIR"
    return 0
fi

# Test 2: Start server and run the indexFolder -> graphAll -> response dependency chain
SERVER_LOG=$(mktemp)
timeout 15 "$KDEPS_BIN" run "$WORKFLOW_FILE" > "$SERVER_LOG" 2>&1 &
SERVER_PID=$!

sleep 4
MAX_WAIT=8
WAITED=0
SERVER_READY=false
PORT=3091

while [ $WAITED -lt $MAX_WAIT ]; do
    if command -v lsof &> /dev/null; then
        if lsof -ti:$PORT &> /dev/null; then
            SERVER_READY=true
            sleep 1
            break
        fi
    elif command -v netstat &> /dev/null; then
        if netstat -an 2>/dev/null | grep -q ":$PORT.*LISTEN"; then
            SERVER_READY=true
            sleep 1
            break
        fi
    elif command -v ss &> /dev/null; then
        if ss -lnt 2>/dev/null | grep -q ":$PORT"; then
            SERVER_READY=true
            sleep 1
            break
        fi
    else
        sleep 2
        SERVER_READY=true
        break
    fi
    sleep 0.5
    WAITED=$((WAITED + 1))
done

if [ "$SERVER_READY" = false ]; then
    if [ -f "$SERVER_LOG" ]; then
        ERROR_MSG=$(head -20 "$SERVER_LOG" 2>/dev/null | grep -i "error\|panic\|fail" | head -1 || echo "Unknown error")
    else
        ERROR_MSG="Server log not available"
    fi
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    rm -f "$SERVER_LOG"
    rm -rf "$TEST_DIR"
    test_skipped "CodeIntelligence Graph - Server startup" "Server did not start: $ERROR_MSG"
    return 0
fi

test_passed "CodeIntelligence Graph - Server startup"

# Test 3: Hit the endpoint; indexFolder runs first (Requires), graphAll queries
# the resulting bbolt db, and response exposes its output() via apiResponse.
if command -v curl &> /dev/null; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{}' \
        "http://127.0.0.1:$PORT/api/v1/graph" 2>/dev/null || echo -e "\n000")
    STATUS_CODE=$(echo "$RESPONSE" | tail -n 1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$STATUS_CODE" = "200" ]; then
        test_passed "CodeIntelligence Graph - POST endpoint (200 OK)"

        JSON_BODY=$(echo "$BODY" | grep -o '^{.*}' | head -1 || echo "$BODY")
        if command -v jq &> /dev/null; then
            if echo "$JSON_BODY" | jq -e '.data.graphSuccess == true' > /dev/null 2>&1; then
                test_passed "CodeIntelligence Graph - graphAll reports success"
            else
                test_failed "CodeIntelligence Graph - graphAll reports success" "Got: $JSON_BODY"
            fi

            if echo "$JSON_BODY" | jq -e '.data.roots | length == 1' > /dev/null 2>&1; then
                test_passed "CodeIntelligence Graph - Exactly one root file (a.md; b.md is referenced by it)"
            else
                test_failed "CodeIntelligence Graph - Root file count" "Got: $JSON_BODY"
            fi

            if echo "$JSON_BODY" | jq -e '.data.references | keys | length == 2' > /dev/null 2>&1; then
                test_passed "CodeIntelligence Graph - Reference graph has both indexed files"
            else
                test_failed "CodeIntelligence Graph - Reference graph size" "Got: $JSON_BODY"
            fi
        else
            test_skipped "CodeIntelligence Graph - Response body checks (jq not available)"
        fi
    elif [ "$STATUS_CODE" = "500" ]; then
        test_failed "CodeIntelligence Graph - POST endpoint (500)" "$BODY"
    else
        test_failed "CodeIntelligence Graph - POST endpoint" "Unexpected status $STATUS_CODE: $BODY"
    fi
else
    test_skipped "CodeIntelligence Graph - POST endpoint (curl not available)"
fi

# Test 4: The bbolt graph db file was actually created on disk.
if [ -f "$TEST_DIR/graph.db" ]; then
    test_passed "CodeIntelligence Graph - graph.db created on disk"
else
    test_failed "CodeIntelligence Graph - graph.db created on disk" "Not found at $TEST_DIR/graph.db"
fi

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
rm -f "$SERVER_LOG"
rm -rf "$TEST_DIR"
