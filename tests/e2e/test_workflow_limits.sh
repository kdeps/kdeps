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
# E2E: deploy.limits bounds each run. A 1s timeout stops a 2s mock LLM call
# and names deploy.limits.timeout. concurrency: 1 makes two requests that each
# take 2s finish one after the other (about 4s, not 2s).

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
cd "$SCRIPT_DIR"

echo "Testing deploy.limits..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Run limits - python3 not available"
    return 0 2>/dev/null || exit 0
fi

free_port() {
    python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()'
}

# start_limits_server DIR LIMITS: write a one-chat workflow with the given
# deploy.limits block, start it, and set LIM_PORT and LIM_PID.
start_limits_server() {
    local dir="$1" limits="$2" port
    port=$(free_port)
    mkdir -p "$dir/resources"
    cat > "$dir/workflow.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: limits-e2e
  version: "1.0.0"
  targetActionId: slow
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${port}
    routes:
      - path: /lim
        methods: [GET]
  limits:
${limits}
EOF
    cat > "$dir/resources/slow.yaml" <<'EOF'
actionId: slow
name: Slow
chat:
  model:
    name: fake
  context:
    prompt: "SLOW {{ get('id') }}"
EOF
    "$KDEPS_BIN" run "$dir/workflow.yaml" > "$dir/log" 2>&1 &
    LIM_PID=$!
    if ! wait_for_kdeps_port "$port" 20 "" "$dir/log"; then
        return 1
    fi
    LIM_PORT="$KDEPS_BOUND_PORT"
}

T_DIR=$(mktemp -d)
C_DIR=$(mktemp -d)
T_PID=""
C_PID=""
trap 'kill $T_PID $C_PID 2>/dev/null; rm -rf "$T_DIR" "$C_DIR"' EXIT

# Timeout: the run stops after 1s and the error names the limit.
if start_limits_server "$T_DIR" "    timeout: 1s"; then
    T_PID=$LIM_PID
    START=$(python3 -c 'import time; print(time.time())')
    RESP=$(curl -s --max-time 15 "http://127.0.0.1:${LIM_PORT}/lim?id=t" 2>&1)
    ELAPSED=$(python3 -c "import time; print(round(time.time() - $START, 2))")
    if echo "$RESP$(cat "$T_DIR/log")" | grep -q 'deploy.limits.timeout'; then
        test_passed "Run limits - timeout stops the run and names deploy.limits.timeout"
    else
        test_failed "Run limits - timeout stops the run and names deploy.limits.timeout" "resp=$RESP"
    fi
    if python3 -c "import sys; sys.exit(0 if $ELAPSED < 1.9 else 1)"; then
        test_passed "Run limits - timeout answers before the 2s model call ends (${ELAPSED}s)"
    else
        test_failed "Run limits - timeout answers before the 2s model call ends" "elapsed=${ELAPSED}s"
    fi
else
    fail_server_startup "Run limits - timeout server failed to start" "$T_DIR/log"
fi

# Concurrency 1: two 2s requests run one after the other.
if start_limits_server "$C_DIR" "    concurrency: 1"; then
    C_PID=$LIM_PID
    START=$(python3 -c 'import time; print(time.time())')
    curl -sf --max-time 20 "http://127.0.0.1:${LIM_PORT}/lim?id=x" > "$C_DIR/x" 2>&1 &
    PX=$!
    curl -sf --max-time 20 "http://127.0.0.1:${LIM_PORT}/lim?id=y" > "$C_DIR/y" 2>&1 &
    PY=$!
    wait $PX $PY
    ELAPSED=$(python3 -c "import time; print(round(time.time() - $START, 2))")
    if grep -q 'SLOW-DONE x' "$C_DIR/x" && grep -q 'SLOW-DONE y' "$C_DIR/y"; then
        test_passed "Run limits - both queued requests get an answer"
    else
        test_failed "Run limits - both queued requests get an answer" "x=$(cat "$C_DIR/x") y=$(cat "$C_DIR/y")"
    fi
    if python3 -c "import sys; sys.exit(0 if $ELAPSED >= 3.8 else 1)"; then
        test_passed "Run limits - concurrency 1 runs them one after the other (${ELAPSED}s)"
    else
        test_failed "Run limits - concurrency 1 runs them one after the other" "elapsed=${ELAPSED}s"
    fi
else
    fail_server_startup "Run limits - concurrency server failed to start" "$C_DIR/log"
fi

# kdeps validate rejects a timeout that is not a duration.
V_DIR=$(mktemp -d)
mkdir -p "$V_DIR/resources"
cat > "$V_DIR/workflow.yaml" <<'EOF'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: limits-validate
  version: "1.0.0"
  targetActionId: a
deploy:
  limits:
    timeout: soon
EOF
cat > "$V_DIR/resources/a.yaml" <<'EOF'
actionId: a
name: A
apiResponse:
  success: true
  response: ok
EOF
if OUT=$("$KDEPS_BIN" validate "$V_DIR/workflow.yaml" 2>&1); then
    test_failed "Run limits - validate rejects a bad timeout" "validate passed: $OUT"
elif echo "$OUT" | grep -q 'deploy.limits.timeout'; then
    test_passed "Run limits - validate rejects a bad timeout"
else
    test_failed "Run limits - validate rejects a bad timeout" "out=$OUT"
fi
rm -rf "$V_DIR"
