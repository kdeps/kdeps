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

# E2E: a chat that names no model runs on pybot. kdeps serves it from the
# models folder (a fake bot package stands in for PyBot here). A chat with
# loop: on pybot logs a warning, because pybot does not call tools.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the pybot default model..."
without_llm_override   # pybot is the model under test, not the mock

if ! command -v python3 > /dev/null 2>&1; then
    test_skipped "pybot default - python3 not on PATH"
    exit 0
fi

PB_DIR=$(mktemp -d)
trap 'rm -rf "$PB_DIR"' EXIT
PB_DIRNAME=$(python3 -c 'import sys,yaml; print(yaml.safe_load(open(sys.argv[1]))["pybot"]["dir"])' \
    "$SCRIPT_DIR/../../assets/models/pybot.yaml" 2>/dev/null \
    || sed -n 's/^  dir: "\(.*\)"/\1/p' "$SCRIPT_DIR/../../assets/models/pybot.yaml")
mkdir -p "$PB_DIR/models/$PB_DIRNAME/bot" "$PB_DIR/proj/resources"
: > "$PB_DIR/models/$PB_DIRNAME/bot/__init__.py"
printf "def turn(text, memory):\n    return {'reply': 'PYBOT ' + text}\n" > "$PB_DIR/models/$PB_DIRNAME/bot/runtime.py"

cat > "$PB_DIR/proj/workflow.yaml" <<'WFEOF'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: pybot-default
  version: "1.0.0"
  targetActionId: llm
deploy:
  input:
    sources: [interactive]
WFEOF

run_pybot() {
    (cd "$PB_DIR/proj" && printf 'hello\n/quit\n' | KDEPS_MODELS_DIR="$PB_DIR/models" \
        KDEPS_DEFAULT_BACKEND= KDEPS_LLM_MODELS= timeout 120 "$KDEPS_BIN" run workflow.yaml 2>&1 || true)
}

cat > "$PB_DIR/proj/resources/llm.yaml" <<'REOF'
actionId: llm
name: Chat
chat:
  context:
    prompt: "{{ input('message') }}"
REOF
OUT=$(run_pybot)
if echo "$OUT" | grep -q "PYBOT hello"; then
    test_passed "pybot default - a chat with no model answers from pybot"
else
    test_failed "pybot default - a chat with no model answers from pybot" "$(echo "$OUT" | tail -15)"
fi

cat >> "$PB_DIR/proj/resources/llm.yaml" <<'REOF'
  loop:
    tools: [read_file]
    maxRounds: 2
REOF
OUT=$(run_pybot)
if echo "$OUT" | grep -q "PYBOT hello" && echo "$OUT" | grep -q "does not call tools"; then
    test_passed "pybot default - a chat with loop: answers and warns that pybot does not call tools"
else
    test_failed "pybot default - a chat with loop: answers and warns that pybot does not call tools" "$(echo "$OUT" | tail -15)"
fi

restore_llm_override
echo ""
echo "pybot default E2E tests complete."
