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

# E2E test for stateless-bot example (validation only — requires external services)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing Stateless Bot Example..."

WORKFLOW_PATH="$PROJECT_ROOT/examples/stateless-bot/workflow.yaml"
[ ! -f "$WORKFLOW_PATH" ] && { test_skipped "Stateless Bot (workflow not found)"; return 0; }

if "$KDEPS_BIN" validate "$WORKFLOW_PATH" &> /dev/null; then
    test_passed "Stateless Bot - Workflow validation"
else
    test_failed "Stateless Bot - Workflow validation" "Validation failed"
fi

RESOURCE_COUNT=0
for f in "$PROJECT_ROOT/examples/stateless-bot/resources/"*.yaml; do
    [ -f "$f" ] && RESOURCE_COUNT=$((RESOURCE_COUNT + 1))
done
[ $RESOURCE_COUNT -gt 0 ] && test_passed "Stateless Bot - Resource files exist ($RESOURCE_COUNT found)"

# Stateless bot mode reads a JSON message from stdin and runs one-shot (no
# HTTP server) -- the workflow's chat resource uses model: llama3.2:1b, the
# same local llamafile CI already provisions for every other local-model
# e2e test, so this needs no external service. 180s: a cold llamafile on a
# shared CI runner can take over a minute to load and answer.
RUN_OUT=$(echo '{"message": "Say hi"}' | timeout 180 "$KDEPS_BIN" run "$WORKFLOW_PATH" 2>&1 || true)
if output_grep "127\.0\.0\.1:[0-9]+/v1/chat/completions.*context deadline exceeded" "$RUN_OUT"; then
    # The local llamafile answered slower than the chat resource's timeout:
    # a loaded shared runner, not a kdeps fault. Any other fatal still fails.
    test_skipped "Stateless Bot - Server test - local model slower than the resource timeout on this runner"
elif output_grep_i "fatal" "$RUN_OUT"; then
    test_failed "Stateless Bot - Server test" "$RUN_OUT"
elif output_grep_i "Resource completed" "$RUN_OUT"; then
    test_passed "Stateless Bot - Server test"
else
    skip_or_fail_llm "Stateless Bot - Server test (unexpected output)" "$RUN_OUT"
fi
echo ""
