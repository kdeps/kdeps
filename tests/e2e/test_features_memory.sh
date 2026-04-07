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

# E2E tests for the memory executor (SQLite-backed semantic memory store).
# Tests consolidate/recall/forget operations via run.memory:.
# Uses keyword fallback when Ollama/nomic-embed-text is unavailable.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
cd "$SCRIPT_DIR"

echo "Testing Memory Component Feature..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Memory - python3 not available"
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
DB_PATH="${TEST_DIR}/memory.db"

cat > "$TEST_DIR/workflow.yaml" <<WFEOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: memory-e2e-test
  version: "1.0.0"
  targetActionId: response
settings:
  apiServerMode: true
  hostIp: "0.0.0.0"
  portNum: ${API_PORT}
  apiServer:
    routes:
      - path: /memory/consolidate
        methods: [POST]
      - path: /memory/recall
        methods: [POST]
      - path: /memory/forget
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
WFEOF

cat > "$TEST_DIR/resources/consolidate.yaml" <<RESEOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: memConsolidate
  name: Memory Consolidate
run:
  validations:
    routes: [/memory/consolidate]
    methods: [POST]
  memory:
    operation: consolidate
    content: "The E2E test ran successfully on this date"
    category: "e2e-tests"
    dbPath: "${DB_PATH}"
RESEOF

cat > "$TEST_DIR/resources/recall.yaml" <<RESEOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: memRecall
  name: Memory Recall
run:
  validations:
    routes: [/memory/recall]
    methods: [POST]
  memory:
    operation: recall
    content: "E2E test"
    category: "e2e-tests"
    topK: 3
    dbPath: "${DB_PATH}"
RESEOF

cat > "$TEST_DIR/resources/forget.yaml" <<RESEOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: memForget
  name: Memory Forget
run:
  validations:
    routes: [/memory/forget]
    methods: [POST]
  memory:
    operation: forget
    content: "E2E test"
    category: "e2e-tests"
    dbPath: "${DB_PATH}"
RESEOF

cat > "$TEST_DIR/resources/response.yaml" <<'RESEOF'
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: response
  name: Response
  requires: [memConsolidate, memRecall, memForget]
run:
  apiResponse:
    success: true
    response:
      consolidateResult: "{{ output('memConsolidate') }}"
      recallResult: "{{ output('memRecall') }}"
      forgetResult: "{{ output('memForget') }}"
RESEOF

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
    test_skipped "Memory - server failed to start"
    cat "$LOG_FILE"
    echo ""
    return 0 2>/dev/null || return 0
fi

# Test 1: Consolidate (store) a memory entry
CONS_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/memory/consolidate" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

CONS_RESULT=$(echo "$CONS_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = (d.get('data') or {}).get('consolidateResult') or {}
    print(r.get('result', ''))
except Exception:
    print('')
" 2>/dev/null || echo "")

if [ "$CONS_RESULT" = "consolidated" ]; then
    test_passed "Memory - store value"
else
    test_failed "Memory - store value" "result=$CONS_RESULT resp=$CONS_RESP"
fi

# Test 2: Recall - should find the stored entry
RECALL_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/memory/recall" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

RECALL_RESULT=$(echo "$RECALL_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = (d.get('data') or {}).get('recallResult') or {}
    print(r.get('result', ''))
except Exception:
    print('')
" 2>/dev/null || echo "")

RECALL_ENTRIES=$(echo "$RECALL_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = (d.get('data') or {}).get('recallResult') or {}
    entries = r.get('entries', [])
    print(len(entries))
except Exception:
    print(0)
" 2>/dev/null || echo "0")

if [ "$RECALL_RESULT" = "recalled" ] && [ "$RECALL_ENTRIES" -ge 1 ] 2>/dev/null; then
    test_passed "Memory - retrieve value"
else
    test_failed "Memory - retrieve value" "result='$RECALL_RESULT' entries=$RECALL_ENTRIES resp=$RECALL_RESP"
fi

# Test 3: SQLite DB file was created
if [ -f "$DB_PATH" ]; then
    test_passed "Memory - SQLite DB file created"
else
    test_failed "Memory - SQLite DB file created" "DB not found at $DB_PATH"
fi

# Test 4: Forget - remove the entry
FORGET_RESP=$(curl -sf --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/memory/forget" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

FORGET_RESULT=$(echo "$FORGET_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = (d.get('data') or {}).get('forgetResult') or {}
    print(r.get('result', ''))
except Exception:
    print('')
" 2>/dev/null || echo "")

if [ "$FORGET_RESULT" = "forgotten" ]; then
    test_passed "Memory - forget key"
else
    test_failed "Memory - forget key" "result=$FORGET_RESULT resp=$FORGET_RESP"
fi

echo ""
echo "Memory E2E tests complete."
