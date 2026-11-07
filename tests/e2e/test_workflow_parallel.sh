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

# E2E: independent workflow resources run in parallel. Three chat calls the
# mock LLM answers after 1s each finish in about 1s, and the response that
# requires them sees all three outputs.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
cd "$SCRIPT_DIR"

echo "Testing parallel workflow resources..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Parallel resources - python3 not available"
    return 0 2>/dev/null || exit 0
fi

free_port() {
    python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()'
}
API_PORT=$(free_port)

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT


cat > "$TEST_DIR/workflow.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: parallel-e2e
  version: "1.0.0"
  targetActionId: response
settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${API_PORT}
    routes:
      - path: /par
        methods: [GET]
EOF

for id in a b c; do
cat > "$TEST_DIR/resources/$id.yaml" <<EOF
actionId: $id
name: Call $id
chat:
  model: fake
  prompt: "SLOW $id"
EOF
done

cat > "$TEST_DIR/resources/response.yaml" <<'EOF'
actionId: response
name: Response
requires: [a, b, c]
apiResponse:
  success: true
  response:
    a: "{{ output('a') }}"
    b: "{{ output('b') }}"
    c: "{{ output('c') }}"
EOF

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!

if ! wait_for_kdeps_port "$API_PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Parallel resources - server failed to start" "$LOG_FILE"
    return 0 2>/dev/null || exit 0
fi
API_PORT="$KDEPS_BOUND_PORT"

START=$(python3 -c 'import time; print(time.time())')
RESP=$(curl -sf --max-time 15 "http://127.0.0.1:${API_PORT}/par" 2>&1)
ELAPSED=$(python3 -c "import time; print(round(time.time() - $START, 2))")

if echo "$RESP" | grep -q 'SLOW-DONE a' && echo "$RESP" | grep -q 'SLOW-DONE b' && echo "$RESP" | grep -q 'SLOW-DONE c'; then
    test_passed "Parallel resources - response sees every output"
else
    test_failed "Parallel resources - response sees every output" "resp=$RESP"
fi

if python3 -c "import sys; sys.exit(0 if $ELAPSED < 2.5 else 1)"; then
    test_passed "Parallel resources - three 1s model calls finish together (${ELAPSED}s)"
else
    test_failed "Parallel resources - three 1s model calls finish together" "elapsed=${ELAPSED}s"
fi
