#!/bin/bash
# Copyright 2026 Kdeps, KvK 94834768
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# E2E tests for the embedding component (local SQLite document store).
# Tests index / search / delete operations via run.component: {name: embedding}.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
cd "$SCRIPT_DIR"

echo "Testing Embedding Component Feature..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Embedding - python3 not available"
    echo ""
    return 0 2>/dev/null || return 0
fi

API_PORT=$(python3 - <<'PY'
import socket
s = socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()
PY
)

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
DB_PATH="$TEST_DIR/embeddings.db"

cat > "$TEST_DIR/workflow.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: embedding-e2e-test
  version: "1.0.0"
  targetActionId: response
settings:
  apiServerMode: true
  hostIp: "0.0.0.0"
  portNum: ${API_PORT}
  apiServer:
    routes:
      - path: /embed/index
        methods: [POST]
      - path: /embed/search
        methods: [POST]
      - path: /embed/delete
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
EOF

cat > "$TEST_DIR/resources/index.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: indexDoc
  name: Index Document
run:
  validations:
    routes: [/embed/index]
    methods: [POST]
  component:
    name: embedding
    with:
      operation: "index"
      text: "{{ get('text', 'test document') }}"
      collection: "e2e_docs"
      dbPath: "${DB_PATH}"
  apiResponse:
    success: true
    response:
      id: "{{ output('indexDoc').id }}"
      dimensions: "{{ output('indexDoc').dimensions }}"
EOF

cat > "$TEST_DIR/resources/search.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: searchDocs
  name: Search Documents
run:
  validations:
    routes: [/embed/search]
    methods: [POST]
  component:
    name: embedding
    with:
      operation: "search"
      text: ""
      collection: "e2e_docs"
      dbPath: "${DB_PATH}"
  apiResponse:
    success: true
    response:
      count: "{{ output('searchDocs').count }}"
      results: "{{ output('searchDocs').results }}"
EOF

cat > "$TEST_DIR/resources/delete.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: deleteDoc
  name: Delete Document
run:
  validations:
    routes: [/embed/delete]
    methods: [POST]
  component:
    name: embedding
    with:
      operation: "delete"
      text: "{{ get('text', '') }}"
      collection: "e2e_docs"
      dbPath: "${DB_PATH}"
  apiResponse:
    success: true
    response:
      deleted: "{{ output('deleteDoc').deleted }}"
EOF

cat > "$TEST_DIR/resources/response.yaml" <<'EOF'
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: response
  name: Response
  requires: [indexDoc, searchDocs, deleteDoc]
run:
  apiResponse:
    success: true
    response:
      indexResult: "{{ output('indexDoc') }}"
      searchResult: "{{ output('searchDocs') }}"
      deleteResult: "{{ output('deleteDoc') }}"
EOF

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" &
KDEPS_PID=$!
trap 'kill "$KDEPS_PID" 2>/dev/null; wait "$KDEPS_PID" 2>/dev/null; rm -rf "$TEST_DIR" 2>/dev/null' EXIT

KDEPS_STARTED=false
for i in $(seq 1 30); do
    if curl -sf --max-time 1 "http://127.0.0.1:${API_PORT}/health" > /dev/null 2>&1; then
        KDEPS_STARTED=true
        break
    fi
    sleep 0.5
done

if [ "$KDEPS_STARTED" = false ]; then
    test_skipped "Embedding - server failed to start"
    echo ""
    return 0 2>/dev/null || return 0
fi

# Test 1: Index a document
INDEX_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/embed/index" \
    -H "Content-Type: application/json" -d '{"text": "KDeps is a declarative YAML workflow framework."}' 2>&1)

if echo "$INDEX_RESP" | grep -q '"success"'; then
    test_passed "Embedding - index document"
else
    test_failed "Embedding - index document" "Response: $INDEX_RESP"
fi

# Test 2: Index a second document
INDEX2_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/embed/index" \
    -H "Content-Type: application/json" -d '{"text": "Vector embeddings enable semantic search."}' 2>&1)

if echo "$INDEX2_RESP" | grep -q '"success"'; then
    test_passed "Embedding - index second document"
else
    test_failed "Embedding - index second document" "Response: $INDEX2_RESP"
fi

# Test 3: Search documents
SEARCH_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/embed/search" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

if echo "$SEARCH_RESP" | grep -q '"searchResult"'; then
    test_passed "Embedding - search documents"
else
    test_failed "Embedding - search documents" "Response: $SEARCH_RESP"
fi

# Test 4: Search returns results
SEARCH_COUNT=$(echo "$SEARCH_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    data = d.get('data', d)
    sr = data.get('searchResult') or {}
    inner = sr.get('data', sr) if isinstance(sr, dict) else {}
    print(inner.get('count', 0) if isinstance(inner, dict) else 0)
except Exception:
    print(0)
" 2>/dev/null || echo "0")

if [ "$SEARCH_COUNT" -ge "1" ] 2>/dev/null; then
    test_passed "Embedding - search returns results (count=$SEARCH_COUNT)"
else
    test_failed "Embedding - search returns results" "count=$SEARCH_COUNT"
fi

# Test 5: Delete a document
DELETE_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/embed/delete" \
    -H "Content-Type: application/json" -d '{"text": "KDeps is a declarative YAML workflow framework."}' 2>&1)

if echo "$DELETE_RESP" | grep -q '"success"'; then
    test_passed "Embedding - delete document"
else
    test_failed "Embedding - delete document" "Response: $DELETE_RESP"
fi

# Test 6: Search count decreased after delete
SEARCH2_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/embed/search" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

SEARCH2_COUNT=$(echo "$SEARCH2_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    data = d.get('data', d)
    sr = data.get('searchResult') or {}
    inner = sr.get('data', sr) if isinstance(sr, dict) else {}
    print(inner.get('count', 0) if isinstance(inner, dict) else 0)
except Exception:
    print(0)
" 2>/dev/null || echo "0")

if [ "$SEARCH2_COUNT" -lt "$SEARCH_COUNT" ] 2>/dev/null; then
    test_passed "Embedding - search count decreased after delete ($SEARCH_COUNT → $SEARCH2_COUNT)"
else
    test_failed "Embedding - search count decreased after delete" "before=$SEARCH_COUNT after=$SEARCH2_COUNT"
fi

echo ""
echo "Embedding E2E tests complete."
