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

# E2E: kdeps run scales a workflow to zero on one machine. The front holds
# the port, stops the workflow after deploy.scale.idle, and the next request
# starts it again. A fixed count serves directly. kdeps validate rejects
# scale.min 0 when no request can wake the workflow.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
cd "$SCRIPT_DIR"

echo "Testing deploy.scale on one machine..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Run scale - python3 not available"
    return 0 2>/dev/null || exit 0
fi

free_port() {
    python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()'
}

# write_scale_wf DIR SCALE_BLOCK: an API workflow on a free port (SC_PORT).
write_scale_wf() {
    local dir="$1" scale="$2"
    SC_PORT=$(free_port)
    mkdir -p "$dir/resources"
    cat > "$dir/workflow.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: scale-e2e
  version: "1.0.0"
  targetActionId: a
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${SC_PORT}
    routes:
      - path: /hi
        methods: [GET]
${scale}
EOF
    cat > "$dir/resources/a.yaml" <<'EOF'
actionId: a
name: A
apiResponse:
  success: true
  response: scale-ok
EOF
}

# wait_log FILE TEXT COUNT SECONDS: wait until TEXT is in FILE COUNT times.
wait_log() {
    local i
    for i in $(seq 1 "$4"); do
        [ "$(grep -c "$2" "$1" 2>/dev/null)" -ge "$3" ] && return 0
        sleep 1
    done
    return 1
}

Z_DIR=$(mktemp -d)
F_DIR=$(mktemp -d)
V_DIR=$(mktemp -d)
Z_PID=""
F_PID=""
trap 'kill $Z_PID $F_PID 2>/dev/null; rm -rf "$Z_DIR" "$F_DIR" "$V_DIR"' EXIT

# Scale to zero: start, idle stop, wake again.
write_scale_wf "$Z_DIR" "  scale:
    idle: 2s"
"$KDEPS_BIN" run "$Z_DIR/workflow.yaml" > "$Z_DIR/log" 2>&1 &
Z_PID=$!
if wait_for_kdeps_port "$SC_PORT" 30 "$Z_PID" "$Z_DIR/log"; then
    PORT="$KDEPS_BOUND_PORT"
    RESP=$(curl -s --max-time 30 "http://127.0.0.1:${PORT}/hi" 2>&1)
    if echo "$RESP" | grep -q 'scale-ok' && grep -q 'Scale to zero on' "$Z_DIR/log"; then
        test_passed "Run scale - the front serves the workflow"
    else
        test_failed "Run scale - the front serves the workflow" "resp=$RESP log=$(tail -20 "$Z_DIR/log")"
    fi
    if wait_log "$Z_DIR/log" 'No request for 2s: stopping the workflow' 1 20; then
        test_passed "Run scale - idle stops the workflow"
    else
        test_failed "Run scale - idle stops the workflow" "$(tail -20 "$Z_DIR/log")"
    fi
    RESP=$(curl -s --max-time 60 "http://127.0.0.1:${PORT}/hi" 2>&1)
    if echo "$RESP" | grep -q 'scale-ok' && wait_log "$Z_DIR/log" 'Starting the workflow on' 2 5; then
        test_passed "Run scale - the next request starts the workflow again"
    else
        test_failed "Run scale - the next request starts the workflow again" "resp=$RESP log=$(tail -20 "$Z_DIR/log")"
    fi
    kill "$Z_PID" 2>/dev/null
    wait "$Z_PID" 2>/dev/null
    Z_PID=""
    if command -v pgrep &> /dev/null && [[ "$(uname -s)" != MINGW* && "$(uname -s)" != MSYS* ]]; then
        sleep 2
        if pgrep -f "run $Z_DIR/workflow.yaml" > /dev/null; then
            test_failed "Run scale - stopping kdeps run stops the workflow" "$(pgrep -fl "run $Z_DIR/workflow.yaml")"
        else
            test_passed "Run scale - stopping kdeps run stops the workflow"
        fi
    fi
else
    fail_server_startup "Run scale - the front failed to start" "$Z_DIR/log"
fi

# A fixed count serves directly, with no front.
write_scale_wf "$F_DIR" "  scale:
    min: 1
    max: 1"
"$KDEPS_BIN" run "$F_DIR/workflow.yaml" > "$F_DIR/log" 2>&1 &
F_PID=$!
if wait_for_kdeps_port "$SC_PORT" 30 "$F_PID" "$F_DIR/log"; then
    RESP=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/hi" 2>&1)
    if echo "$RESP" | grep -q 'scale-ok' && ! grep -q 'Scale to zero on' "$F_DIR/log"; then
        test_passed "Run scale - min equal to max serves with no front"
    else
        test_failed "Run scale - min equal to max serves with no front" "resp=$RESP log=$(tail -20 "$F_DIR/log")"
    fi
else
    fail_server_startup "Run scale - the fixed server failed to start" "$F_DIR/log"
fi

# kdeps validate rejects scale.min 0 with no HTTP server.
mkdir -p "$V_DIR/resources"
cat > "$V_DIR/workflow.yaml" <<'EOF'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: scale-validate
  version: "1.0.0"
  targetActionId: a
deploy:
  scale:
    max: 3
EOF
cat > "$V_DIR/resources/a.yaml" <<'EOF'
actionId: a
name: A
apiResponse:
  success: true
  response: ok
EOF
if OUT=$("$KDEPS_BIN" validate "$V_DIR/workflow.yaml" 2>&1); then
    test_failed "Run scale - validate rejects scale.min 0 with no server" "validate passed: $OUT"
elif echo "$OUT" | grep -q 'scale to zero needs apiServer or webServer'; then
    test_passed "Run scale - validate rejects scale.min 0 with no server"
else
    test_failed "Run scale - validate rejects scale.min 0 with no server" "$OUT"
fi
