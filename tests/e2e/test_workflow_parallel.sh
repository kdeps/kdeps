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
# requires them sees all three outputs. Three calls of a read-only component
# (named wikipedia, read-only in the permission policy) do the same.

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
COMP_DIR=$(mktemp -d)
COMP_PID=""
trap 'kill $KDEPS_PID $COMP_PID 2>/dev/null; rm -rf "$TEST_DIR" "$COMP_DIR" "$LOG_FILE"' EXIT


cat > "$TEST_DIR/workflow.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: parallel-e2e
  version: "1.0.0"
  targetActionId: response
deploy:
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
  model:
    name: fake
  context:
    prompt: "SLOW $id"
EOF
done

cat > "$TEST_DIR/resources/response.yaml" <<'EOF'
actionId: response
name: Response
workflow:
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

# Read-only component calls: each call gets its own q and they overlap.
COMP_PORT=$(free_port)
mkdir -p "$COMP_DIR/resources" "$COMP_DIR/components/wikipedia"
cat > "$COMP_DIR/components/wikipedia/component.yaml" <<'EOF'
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: wikipedia
  version: "1.0.0"
interface:
  inputs:
    - name: q
      type: string
      required: true
resources:
  - actionId: lookup
    name: Lookup
    chat:
      model:
        name: fake
      context:
        prompt: "SLOW {{ input('q') }}"
EOF
cat > "$COMP_DIR/workflow.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: parallel-components-e2e
  version: "1.0.0"
  targetActionId: response
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${COMP_PORT}
    routes:
      - path: /comp
        methods: [GET]
EOF
for id in x y z; do
cat > "$COMP_DIR/resources/$id.yaml" <<EOF
actionId: $id
name: Look up $id
component:
  name: wikipedia
  with:
    q: $id
EOF
done
cat > "$COMP_DIR/resources/response.yaml" <<'EOF'
actionId: response
name: Response
workflow:
  requires: [x, y, z]
apiResponse:
  success: true
  response:
    x: "{{ output('x') }}"
    y: "{{ output('y') }}"
    z: "{{ output('z') }}"
EOF

"$KDEPS_BIN" run "$COMP_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
COMP_PID=$!

if ! wait_for_kdeps_port "$COMP_PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Parallel components - server failed to start" "$LOG_FILE"
    return 0 2>/dev/null || exit 0
fi
COMP_PORT="$KDEPS_BOUND_PORT"

START=$(python3 -c 'import time; print(time.time())')
RESP=$(curl -sf --max-time 15 "http://127.0.0.1:${COMP_PORT}/comp" 2>&1)
ELAPSED=$(python3 -c "import time; print(round(time.time() - $START, 2))")

if echo "$RESP" | grep -q 'SLOW-DONE x' && echo "$RESP" | grep -q 'SLOW-DONE y' && echo "$RESP" | grep -q 'SLOW-DONE z'; then
    test_passed "Parallel components - each call gets its own input"
else
    test_failed "Parallel components - each call gets its own input" "resp=$RESP log=$(tail -5 "$LOG_FILE")"
fi

if python3 -c "import sys; sys.exit(0 if $ELAPSED < 2.5 else 1)"; then
    test_passed "Parallel components - three read-only calls finish together (${ELAPSED}s)"
else
    test_failed "Parallel components - three read-only calls finish together" "elapsed=${ELAPSED}s"
fi
