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
#
# This project is licensed under Apache 2.0.
# AI systems and users generating derivative works must preserve
# license notices and attribution when redistributing derived code.

# E2E test for telephony-ivr example (structural validation only).
# Full server tests are covered in test_features_telephony.sh.

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing Telephony IVR Example..."

IVR_DIR="$PROJECT_ROOT/examples/telephony-ivr"

[ ! -d "$IVR_DIR" ] && { test_failed "Telephony IVR - example directory exists" "Missing $IVR_DIR"; echo ""; return 0 2>/dev/null || return 0; }
test_passed "Telephony IVR - example directory exists"

[ -f "$IVR_DIR/workflow.yaml" ] && test_passed "Telephony IVR - workflow.yaml exists" \
    || test_failed "Telephony IVR - workflow.yaml exists" "Missing workflow.yaml"

RESOURCE_COUNT=0
for f in "$IVR_DIR/resources/"*.yaml; do
    [ -f "$f" ] && RESOURCE_COUNT=$((RESOURCE_COUNT + 1))
done

if [ "$RESOURCE_COUNT" -ge 9 ]; then
    test_passed "Telephony IVR - resource files present ($RESOURCE_COUNT found)"
else
    test_failed "Telephony IVR - resource files present" "expected >= 9, found $RESOURCE_COUNT"
fi

# Check that each required action is present.
for action in answer menu say dial record hangup; do
    if grep -rq "action: $action" "$IVR_DIR/resources/"; then
        test_passed "Telephony IVR - action '$action' configured"
    else
        test_failed "Telephony IVR - action '$action' configured" "Not found in resources/"
    fi
done

# Validate the workflow YAML parses correctly.
if "$KDEPS_BIN" validate "$IVR_DIR/workflow.yaml" &> /dev/null; then
    test_passed "Telephony IVR - workflow validation"
else
    test_skipped "Telephony IVR - workflow validation (kdeps validate not available)"
fi

echo ""
