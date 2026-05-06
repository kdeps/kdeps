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

# E2E tests for `kdeps doctor` command.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo ""
echo "Testing kdeps doctor..."

KDEPS_BIN="${KDEPS_BIN:-kdeps}"

# ── doctor --help ──────────────────────────────────────────────────────────

OUTPUT=$("$KDEPS_BIN" doctor --help 2>&1) || true
if echo "$OUTPUT" | grep -q "Run diagnostic health checks"; then
    test_passed "[PASS] doctor --help shows description"
else
    test_failed "[FAIL] doctor --help shows description" "got: $OUTPUT"
fi

# ── doctor runs successfully ───────────────────────────────────────────────

CONFIG_DIR=$(mktemp -d)
CONFIG_PATH="$CONFIG_DIR/config.yaml"

cat > "$CONFIG_PATH" << 'EOF'
llm:
  ollama_host: http://localhost:11434
  backend: ollama
  models:
    - llama3.2
defaults:
  timezone: UTC
resource_defaults:
  chat:
    timeout: "60s"
EOF

OUTPUT=$(KDEPS_CONFIG_PATH="$CONFIG_PATH" KDEPS_SKIP_BOOTSTRAP=1 "$KDEPS_BIN" doctor 2>&1) || true
if echo "$OUTPUT" | grep -q "kdeps doctor"; then
    test_passed "[PASS] doctor produces report header"
else
    test_failed "[FAIL] doctor produces report header" "got: $OUTPUT"
fi

if echo "$OUTPUT" | grep -q "Config file"; then
    test_passed "[PASS] doctor reports config file status"
else
    test_failed "[FAIL] doctor reports config file status" "got: $OUTPUT"
fi

if echo "$OUTPUT" | grep -q "Python"; then
    test_passed "[PASS] doctor reports Python status"
else
    test_failed "[FAIL] doctor reports Python status" "got: $OUTPUT"
fi

if echo "$OUTPUT" | grep -q "Overall"; then
    test_passed "[PASS] doctor reports overall status"
else
    test_failed "[FAIL] doctor reports overall status" "got: $OUTPUT"
fi

# ── doctor with typo in config ─────────────────────────────────────────────

cat > "$CONFIG_PATH" << 'EOF'
llm:
  openai_apikey: sk-typo
EOF

OUTPUT=$(KDEPS_CONFIG_PATH="$CONFIG_PATH" KDEPS_SKIP_BOOTSTRAP=1 "$KDEPS_BIN" doctor 2>&1) || true
if echo "$OUTPUT" | grep -q "openai_apikey"; then
    test_passed "[PASS] doctor surfaces config validation warnings"
else
    test_failed "[FAIL] doctor surfaces config validation warnings" "got: $OUTPUT"
fi

# ── doctor with missing config ─────────────────────────────────────────────

MISSING_CONFIG=$(mktemp -d)/nonexistent.yaml
OUTPUT=$(KDEPS_CONFIG_PATH="$MISSING_CONFIG" KDEPS_SKIP_BOOTSTRAP=1 "$KDEPS_BIN" doctor 2>&1) || true
if echo "$OUTPUT" | grep -q "not found"; then
    test_passed "[PASS] doctor warns when config file is missing"
else
    test_failed "[FAIL] doctor warns when config file is missing" "got: $OUTPUT"
fi

rm -rf "$CONFIG_DIR"

echo ""
echo "kdeps doctor E2E: done"
