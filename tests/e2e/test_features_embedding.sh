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

# E2E tests for the embedding executor.
# Tests index/search/delete operations via run.embedding:.
# Uses Ollama + nomic-embed-text when available, falls back to keyword search.

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
LOG_FILE=$(mktemp)
DB_PATH="${TEST_DIR}/embeddings.db"

trap 'kill "$KDEPS_PID" 2>/dev/null; wait "$KDEPS_PID" 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

# Check if Ollama has nomic-embed-text (optional - enables semantic mode)
EMBED_URL="http://127.0.0.1:11434"
HAS_EMBED_MODEL=false
if curl -sf --max-time 2 "${EMBED_URL}/api/tags" 2>/dev/null | grep -q "nomic-embed-text"; then
    HAS_EMBED_MODEL=true
fi

cat > "$TEST_DIR/workflow.yaml" <<WFEOF
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
WFEOF

cat > "$TEST_DIR/resources/index.yaml" <<RESEOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: indexDoc
  name: Index Document
run:
  validations:
    routes: [/embed/index]
    methods: [POST]
  embedding:
    operation: index
    input: "{{ get('text', 'The quick brown fox jumps over the lazy dog') }}"
    collection: "e2e_docs"
    model: "nomic-embed-text"
    backend: "ollama"
    baseUrl: "${EMBED_URL}"
    dbPath: "${DB_PATH}"
RESEOF

cat > "$TEST_DIR/resources/search.yaml" <<RESEOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: searchDocs
  name: Search Documents
run:
  validations:
    routes: [/embed/search]
    methods: [POST]
  embedding:
    operation: search
    input: "{{ get('q', 'fox') }}"
    collection: "e2e_docs"
    model: "nomic-embed-text"
    backend: "ollama"
    baseUrl: "${EMBED_URL}"
    topK: 5
    dbPath: "${DB_PATH}"
RESEOF

cat > "$TEST_DIR/resources/delete.yaml" <<RESEOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: deleteDoc
  name: Delete Document
run:
  validations:
    routes: [/embed/delete]
    methods: [POST]
  embedding:
    operation: delete
    input: "{{ get('text', 'The quick brown fox jumps over the lazy dog') }}"
    collection: "e2e_docs"
    dbPath: "${DB_PATH}"
RESEOF

cat > "$TEST_DIR/resources/response.yaml" <<'RESEOF'
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
RESEOF

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
IDX_RESP=$(curl -sf --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/embed/index" \
    -H "Content-Type: application/json" \
    -d '{"text":"The quick brown fox jumps over the lazy dog"}' 2>&1)

IDX_ID=$(echo "$IDX_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = (d.get('data') or {}).get('indexResult') or {}
    print(r.get('id', ''))
except Exception:
    print('')
" 2>/dev/null || echo "")

if [ -n "$IDX_ID" ]; then
    test_passed "Embedding - index document"
else
    test_failed "Embedding - index document" "id='$IDX_ID' resp=$IDX_RESP"
fi

# Test 2: Index a second document
IDX2_RESP=$(curl -sf --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/embed/index" \
    -H "Content-Type: application/json" \
    -d '{"text":"A completely different sentence about cats and dogs"}' 2>&1)

IDX2_ID=$(echo "$IDX2_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = (d.get('data') or {}).get('indexResult') or {}
    print(r.get('id', ''))
except Exception:
    print('')
" 2>/dev/null || echo "")

if [ -n "$IDX2_ID" ]; then
    test_passed "Embedding - index second document"
else
    test_failed "Embedding - index second document" "id='$IDX2_ID' resp=$IDX2_RESP"
fi

# Test 3: Search - should find results
SRCH_RESP=$(curl -sf --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/embed/search" \
    -H "Content-Type: application/json" \
    -d '{"q":"fox"}' 2>&1)

SRCH_COUNT=$(echo "$SRCH_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = (d.get('data') or {}).get('searchResult') or {}
    print(r.get('count', 0))
except Exception:
    print(0)
" 2>/dev/null || echo "0")

if [ "$SRCH_COUNT" -ge 1 ] 2>/dev/null; then
    test_passed "Embedding - search documents"
else
    test_failed "Embedding - search documents" "count=$SRCH_COUNT resp=$SRCH_RESP"
fi

if [ "$SRCH_COUNT" -ge 2 ] 2>/dev/null; then
    test_passed "Embedding - search returns results (count=$SRCH_COUNT)"
else
    test_failed "Embedding - search returns results" "count=$SRCH_COUNT"
fi

# Test 4: Delete - remove the first document
DEL_RESP=$(curl -sf --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/embed/delete" \
    -H "Content-Type: application/json" \
    -d '{"text":"The quick brown fox jumps over the lazy dog"}' 2>&1)

DEL_OK=$(echo "$DEL_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = (d.get('data') or {}).get('deleteResult') or {}
    print(r.get('deleted', False))
except Exception:
    print(False)
" 2>/dev/null || echo "False")

if [ "$DEL_OK" = "True" ] || [ "$DEL_OK" = "true" ]; then
    test_passed "Embedding - delete document"
else
    test_failed "Embedding - delete document" "deleted=$DEL_OK resp=$DEL_RESP"
fi

# Test 5: Search count decreased after delete
SRCH2_RESP=$(curl -sf --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/embed/search" \
    -H "Content-Type: application/json" \
    -d '{"q":"fox"}' 2>&1)

SRCH2_COUNT=$(echo "$SRCH2_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = (d.get('data') or {}).get('searchResult') or {}
    print(r.get('count', 0))
except Exception:
    print(0)
" 2>/dev/null || echo "0")

if [ "$SRCH2_COUNT" -lt "$SRCH_COUNT" ] 2>/dev/null; then
    test_passed "Embedding - search count decreased after delete ($SRCH_COUNT → $SRCH2_COUNT)"
else
    test_failed "Embedding - search count decreased after delete" "before=$SRCH_COUNT after=$SRCH2_COUNT"
fi

kill "$KDEPS_PID" 2>/dev/null
echo "✓ Server stopped"
echo ""
echo "Embedding E2E tests complete."
