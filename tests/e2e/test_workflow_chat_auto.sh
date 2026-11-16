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

# E2E: a workflow chat's auto values. tools.list: auto offers the built-in
# kdeps tools; guardrails.judges: auto makes a roster call before review.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat auto values..."

if ! command -v python3 &> /dev/null || [ -z "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "Workflow chat auto - needs python3 and the mock LLM"
    return 0 2>/dev/null || exit 0
fi

CA_DIR=$(mktemp -d)
mkdir -p "$CA_DIR/resources"
trap 'rm -rf "$CA_DIR"' EXIT

cat > "$CA_DIR/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chat-auto-e2e
  version: "1.0.0"
  targetActionId: ask
YAML
cat > "$CA_DIR/resources/ask.yaml" <<'YAML'
actionId: ask
name: Ask
chat:
  model:
    name: llama3.2:1b
    retries: auto
  context:
    prompt: "CHATAUTOE2E hello"
    contextLength: auto
  tools:
    list: auto
    maxRounds: auto
  guardrails:
    judges: auto
YAML

if ! "$KDEPS_BIN" validate "$CA_DIR/workflow.yaml" > /dev/null 2>&1; then
    test_failed "Workflow chat auto - auto values validate" "$("$KDEPS_BIN" validate "$CA_DIR/workflow.yaml" 2>&1 | tail -3)"
    return 1 2>/dev/null || exit 1
fi
test_passed "Workflow chat auto - auto values validate"

(cd "$CA_DIR" && timeout 120 "$KDEPS_BIN" run workflow.yaml > /dev/null 2>&1) || true

if python3 - "$E2E_MOCK_LLM_LOG" <<'PY'
import json, sys
for line in open(sys.argv[1], encoding="utf-8"):
    try:
        body = json.loads(line)
    except ValueError:
        continue
    msgs = body.get("messages", [])
    if not msgs or "CHATAUTOE2E hello" != (msgs[-1].get("content") or ""):
        continue
    names = {t.get("function", {}).get("name") for t in body.get("tools", [])}
    sys.exit(0 if {"read_file", "web_search"} <= names else 1)
sys.exit(1)
PY
then
    test_passed "Workflow chat auto - tools.list: auto offers the built-in tools"
else
    test_failed "Workflow chat auto - tools.list: auto offers the built-in tools" "no request carried read_file and web_search"
fi

if grep -qF "You generate a panel of reviewer personas" "$E2E_MOCK_LLM_LOG" \
    && grep -qF "CHATAUTOE2E hello" "$E2E_MOCK_LLM_LOG"; then
    test_passed "Workflow chat auto - guardrails.judges: auto makes a roster call"
else
    test_failed "Workflow chat auto - guardrails.judges: auto makes a roster call" "no roster request in the mock log"
fi

rm -rf "$CA_DIR"
trap - EXIT
