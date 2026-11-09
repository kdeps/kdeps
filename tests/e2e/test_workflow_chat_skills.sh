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

# E2E: chat.skills lists a workflow's SKILL.md folders in the chat's system
# prompt and gives it load_skill; the model loads a skill and answers from it.
# Relative skill paths resolve against the run's working folder.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat skills..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow chat skills - python3 not available"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources" "$TEST_DIR/skills/deploy"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

cat > "$TEST_DIR/skills/deploy/SKILL.md" <<'MD'
---
name: deploy
description: How this team ships a release
---

SKILLBODY build, tag, then ship.
MD

PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chat-skills-e2e
  version: "1.0.0"
  targetActionId: response
settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${PORT}
    routes:
      - path: /ask
        methods: [GET]
YAML

cat > "$TEST_DIR/resources/ask.yaml" <<'YAML'
actionId: ask
name: Ask
validations:
  routes: [/ask]
chat:
  model: fake
  prompt: "SKILLJSON deploy"
  skills: [skills]
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
requires: [ask]
apiResponse:
  success: true
  response:
    answer: "{{ output('ask') }}"
YAML

(cd "$TEST_DIR" && exec "$KDEPS_BIN" run workflow.yaml) > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow chat skills: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

if echo "$RESP" | grep -q "USED .*SKILLBODY build, tag, then ship"; then
    test_passed "Workflow chat skills - the chat loads a workflow skill with load_skill"
else
    test_failed "Workflow chat skills - the chat loads a workflow skill with load_skill" "resp=$RESP"
fi
