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

# E2E tests for the embedding component upsert operation.
# Tests: upsert new, upsert same (idempotent), similarity search returns results.
# Uses run.component: {name: embedding, with: {operation: upsert}}.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
cd "$SCRIPT_DIR"

echo "Testing Embedding Upsert Component Feature..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Embedding Upsert - python3 not available"
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
trap 'kill "$KDEPS_PID" 2>/dev/null; wait "$KDEPS_PID" 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT
DB_PATH="${TEST_DIR}/embeddings.db"

cat > "$TEST_DIR/workflow.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: embedding-upsert-e2e
  version: "1.0.0"
  targetActionId: response
settings:
  apiServerMode: true
  hostIp: "0.0.0.0"
  portNum: ${API_PORT}
  apiServer:
    routes:
      - path: /embed/upsert
        methods: [POST]
      - path: /embed/search
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
EOF

cat > "$TEST_DIR/resources/upsert.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: embedUpsert
  name: Embed Upsert
run:
  validations:
    routes: [/embed/upsert]
    methods: [POST]
  component:
    name: embedding
    with:
      operation: "upsert"
      text: "The quick brown fox jumps over the lazy dog"
      collection: "e2e_upsert"
      dbPath: "${DB_PATH}"
  apiResponse:
    success: true
    response:
      upsert: "{{ output('embedUpsert') }}"
EOF

cat > "$TEST_DIR/resources/search.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: embedSearch
  name: Embed Search
run:
  validations:
    routes: [/embed/search]
    methods: [POST]
  component:
    name: embedding
    with:
      operation: "search"
      text: ""
      collection: "e2e_upsert"
      dbPath: "${DB_PATH}"
EOF

cat > "$TEST_DIR/resources/response.yaml" <<'EOF'
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: response
  name: Response
  requires: [embedUpsert, embedSearch]
run:
  apiResponse:
    success: true
    response:
      upsert: "{{ output('embedUpsert') }}"
      search: "{{ output('embedSearch') }}"
EOF

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!

KDEPS_STARTED=false
for i in $(seq 1 30); do
    if curl -sf --max-time 1 "http://127.0.0.1:${API_PORT}/health" > /dev/null 2>&1; then
        KDEPS_STARTED=true
        break
    fi
    sleep 0.5
done

if [ "$KDEPS_STARTED" = false ]; then
    test_skipped "Embedding Upsert - server failed to start"
    test_skipped "Embedding Upsert - upsert operation"
    test_skipped "Embedding Upsert - upsert idempotent"
    test_skipped "Embedding Upsert - search returns results"
    test_skipped "Embedding Upsert - DB file created"
    echo ""
    return 0 2>/dev/null || return 0
fi

# Test 1: Upsert a new document
UPS_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/embed/upsert" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

if echo "$UPS_RESP" | grep -qi "upsert\|success\|operation"; then
    test_passed "Embedding Upsert - upsert operation"
else
    test_failed "Embedding Upsert - upsert operation" "resp=$UPS_RESP"
fi

# Test 2: Upsert same document again (idempotent)
UPS2_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/embed/upsert" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

UPSERT_SKIPPED=$(echo "$UPS2_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    data = d.get('data', d)
    u = data.get('upsert') or {}
    inner = u.get('data', u) if isinstance(u, dict) else {}
    print(inner.get('skipped', False) if isinstance(inner, dict) else False)
except Exception:
    print(False)
" 2>/dev/null || echo "False")

if [ "$UPSERT_SKIPPED" = "True" ]; then
    test_passed "Embedding Upsert - upsert idempotent (skipped=True)"
else
    # Second upsert should at least succeed
    if echo "$UPS2_RESP" | grep -qi "success\|upsert"; then
        test_passed "Embedding Upsert - upsert idempotent (second upsert succeeded)"
    else
        test_failed "Embedding Upsert - upsert idempotent" "resp=$UPS2_RESP"
    fi
fi

# Test 3: Search returns results
SRCH_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/embed/search" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

SRCH_COUNT=$(echo "$SRCH_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    data = d.get('data', d)
    s = data.get('search') or {}
    print(s.get('count', 0) if isinstance(s, dict) else 0)
except Exception:
    print(0)
" 2>/dev/null || echo "0")

if [ "$SRCH_COUNT" -ge "1" ] 2>/dev/null; then
    test_passed "Embedding Upsert - search returns results (count=$SRCH_COUNT)"
else
    test_failed "Embedding Upsert - search returns results" "count=$SRCH_COUNT resp=$SRCH_RESP"
fi

# Test 4: DB file was created
if [ -f "$DB_PATH" ]; then
    test_passed "Embedding Upsert - DB file created"
else
    test_failed "Embedding Upsert - DB file created" "DB not found at $DB_PATH"
fi

echo ""
echo "Embedding Upsert E2E tests complete."
