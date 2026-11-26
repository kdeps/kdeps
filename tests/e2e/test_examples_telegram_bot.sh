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

# E2E test for telegram-bot example (validation only — requires external services)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing Telegram Bot Example..."

WORKFLOW_PATH="$PROJECT_ROOT/examples/telegram-bot/workflow.yaml"
[ ! -f "$WORKFLOW_PATH" ] && { test_skipped "Telegram Bot (workflow not found)"; return 0; }

if "$KDEPS_BIN" validate "$WORKFLOW_PATH" &> /dev/null; then
    test_passed "Telegram Bot - Workflow validation"
else
    test_failed "Telegram Bot - Workflow validation" "Validation failed"
fi

RESOURCE_COUNT=0
for f in "$PROJECT_ROOT/examples/telegram-bot/resources/"*.yaml; do
    [ -f "$f" ] && RESOURCE_COUNT=$((RESOURCE_COUNT + 1))
done
[ $RESOURCE_COUNT -gt 0 ] && test_passed "Telegram Bot - Resource files exist ($RESOURCE_COUNT found)"

# Messages come from the telegram_updates listener; the reply goes out with telegram_post
if grep -q "listener: telegram_updates" "$WORKFLOW_PATH" &&
    grep -q "name: telegram_post" "$PROJECT_ROOT/examples/telegram-bot/resources/reply.yaml"; then
    test_passed "Telegram Bot - telegram_updates listener in, telegram_post reply out"
else
    test_failed "Telegram Bot - listener and reply component" "listener: telegram_updates or telegram_post missing"
fi

test_skipped "Telegram Bot - Server test (requires external service/LLM)"
echo ""
