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

# E2E tests for `kdeps token`: grant, list and revoke a capability token.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo ""
echo "Testing kdeps token..."

KDEPS_BIN="${KDEPS_BIN:-kdeps}"
TOKEN_HOME=$(mktemp -d)
trap 'rm -rf "$TOKEN_HOME"' EXIT
tok() { env HOME="$TOKEN_HOME" USERPROFILE="$TOKEN_HOME" "$KDEPS_BIN" token "$@" 2>&1; }

OUTPUT=$(tok grant stage:exec --ttl 1h --actor e2e) || true
TOKEN_ID=$(printf '%s\n' "$OUTPUT" | sed -n 's/^token \([0-9a-f]*\) for stage:exec.*/\1/p')
if [ -n "$TOKEN_ID" ] && output_grep_fixed "for stage:exec" "$OUTPUT"; then
    test_passed "[PASS] token grant issues a token for a scope"
else
    test_failed "[FAIL] token grant issues a token for a scope" "got: $OUTPUT"
fi

OUTPUT=$(tok list) || true
if output_grep_fixed "$TOKEN_ID" "$OUTPUT" && output_grep_fixed "valid" "$OUTPUT"; then
    test_passed "[PASS] token list shows the token as valid"
else
    test_failed "[FAIL] token list shows the token as valid" "got: $OUTPUT"
fi

OUTPUT=$(tok revoke "$TOKEN_ID") || true
if output_grep_fixed "revoked $TOKEN_ID" "$OUTPUT"; then
    test_passed "[PASS] token revoke deletes the token"
else
    test_failed "[FAIL] token revoke deletes the token" "got: $OUTPUT"
fi

OUTPUT=$(tok list) || true
if ! output_grep_fixed "$TOKEN_ID" "$OUTPUT"; then
    test_passed "[PASS] a revoked token is gone from the list"
else
    test_failed "[FAIL] a revoked token is gone from the list" "got: $OUTPUT"
fi

echo ""
echo "kdeps token E2E: done"
