set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing searchLocal graphBoost Feature..."

TEST_DIR=$(mktemp -d)
DOCS_DIR="$TEST_DIR/docs"
mkdir -p "$DOCS_DIR" "$TEST_DIR/resources"
WORKFLOW_FILE="$TEST_DIR/workflow.yaml"
RESOURCE_FILE_SEARCH="$TEST_DIR/resources/search.yaml"

DOCS_DIR_NATIVE=$(to_native_path "$DOCS_DIR")

cat > "$DOCS_DIR/hub.md" <<'EOF'
See [linked](linked.md) for details. needle
EOF

cat > "$DOCS_DIR/linked.md" <<'EOF'
needle
EOF

cat > "$WORKFLOW_FILE" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: search-local-graph-boost-test
  version: "1.0.0"
  targetActionId: search

settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 3093
    routes:
      - path: /api/v1/search
        methods: [POST]

  agentSettings:
    pythonVersion: "3.12"
EOF

cat > "$RESOURCE_FILE_SEARCH" <<EOF

actionId: search
name: Search

restrictToHttpMethods: [POST]
restrictToRoutes: [/api/v1/search]
searchLocal:
  path: "$DOCS_DIR_NATIVE"
  query: "needle"
  index: true
  graphBoost: true
EOF

# Test 1: Validate workflow
if "$KDEPS_BIN" validate "$WORKFLOW_FILE" &> /dev/null; then
    test_passed "searchLocal graphBoost - Workflow validation"
else
    test_failed "searchLocal graphBoost - Workflow validation" "Validation failed"
    rm -rf "$TEST_DIR"
    return 0
fi

# Test 2: Start server
SERVER_LOG=$(mktemp)
timeout 15 "$KDEPS_BIN" run "$WORKFLOW_FILE" > "$SERVER_LOG" 2>&1 &
SERVER_PID=$!

sleep 4
MAX_WAIT=8
WAITED=0
SERVER_READY=false
PORT=3093

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
    test_skipped "searchLocal graphBoost - Server startup" "Server did not start: $ERROR_MSG"
    return 0
fi

test_passed "searchLocal graphBoost - Server startup"

# Test 3: Hit the endpoint; graphBoost re-ranks using hub.md's link to linked.md.
if command -v curl &> /dev/null; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{}' \
        "http://127.0.0.1:$PORT/api/v1/search" 2>/dev/null || echo -e "\n000")
    STATUS_CODE=$(echo "$RESPONSE" | tail -n 1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$STATUS_CODE" = "200" ]; then
        test_passed "searchLocal graphBoost - POST endpoint (200 OK)"

        JSON_BODY=$(echo "$BODY" | grep -o '^{.*}' | head -1 || echo "$BODY")
        if command -v jq &> /dev/null; then
            if echo "$JSON_BODY" | jq -e '.data.count == 2' > /dev/null 2>&1; then
                test_passed "searchLocal graphBoost - both linked and hub docs matched"
            else
                test_failed "searchLocal graphBoost - result count" "Got: $JSON_BODY"
            fi

            if echo "$JSON_BODY" | jq -e '[.data.results[].path] | any(endswith("linked.md"))' > /dev/null 2>&1; then
                test_passed "searchLocal graphBoost - linked.md present in results"
            else
                test_failed "searchLocal graphBoost - linked.md present in results" "Got: $JSON_BODY"
            fi
        else
            test_skipped "searchLocal graphBoost - Response body checks (jq not available)"
        fi
    elif [ "$STATUS_CODE" = "500" ]; then
        test_failed "searchLocal graphBoost - POST endpoint (500)" "$BODY"
    else
        test_failed "searchLocal graphBoost - POST endpoint" "Unexpected status $STATUS_CODE: $BODY"
    fi
else
    test_skipped "searchLocal graphBoost - POST endpoint (curl not available)"
fi

# Test 4: The graph db (separate from the TF-IDF index.db) was created on disk.
if [ -f "$DOCS_DIR/.kdeps/graph.db" ]; then
    test_passed "searchLocal graphBoost - graph.db created on disk"
else
    test_failed "searchLocal graphBoost - graph.db created on disk" "Not found at $DOCS_DIR/.kdeps/graph.db"
fi

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
rm -f "$SERVER_LOG"
rm -rf "$TEST_DIR"
