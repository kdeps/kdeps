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

# E2E test for cv-matcher-deepseek example (validation only — requires LLM API keys)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing Cv Matcher Deepseek Example..."

WORKFLOW_PATH="$PROJECT_ROOT/examples/cv-matcher-deepseek/workflow.yaml"
[ ! -f "$WORKFLOW_PATH" ] && { test_skipped "Cv Matcher Deepseek (workflow not found)"; return 0; }

if "$KDEPS_BIN" validate "$WORKFLOW_PATH" &> /dev/null; then
    test_passed "Cv Matcher Deepseek - Workflow validation"
else
    test_failed "Cv Matcher Deepseek - Workflow validation" "Validation failed"
fi

RESOURCE_COUNT=0
for f in "$PROJECT_ROOT/examples/cv-matcher-deepseek/resources/"*.yaml; do
    [ -f "$f" ] && RESOURCE_COUNT=$((RESOURCE_COUNT + 1))
done
[ $RESOURCE_COUNT -gt 0 ] && test_passed "Cv Matcher Deepseek - Resource files exist ($RESOURCE_COUNT found)"

test_skipped "Cv Matcher Deepseek - Server test (requires LLM API keys and embedding DB)"
echo ""
