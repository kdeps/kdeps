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

# E2E: bare kdeps runs ./workflow.yaml. The assistant template is that
# workflow: an interactive prompt whose chat loop: calls built-in tools.
# With no workflow.yaml, bare kdeps prints help; an agent-REPL flag without
# --agent-loop is an error.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing kdeps entry points..."

EP_DIR=$(mktemp -d)
trap 'rm -rf "$EP_DIR"' EXIT

# --- the assistant template scaffolds a workflow that validates -------------
if (cd "$EP_DIR" && "$KDEPS_BIN" new asst --template assistant > /dev/null 2>&1) \
    && [ -f "$EP_DIR/asst/workflow.yaml" ] && [ -f "$EP_DIR/asst/resources/assistant.yaml" ] \
    && "$KDEPS_BIN" validate "$EP_DIR/asst/workflow.yaml" > /dev/null 2>&1; then
    test_passed "entry points - the assistant template scaffolds a valid workflow"
else
    test_failed "entry points - the assistant template scaffolds a valid workflow" "$(ls -R "$EP_DIR" 2>&1 | head -20)"
fi

# --- bare kdeps in the project runs it: the prompt drives the chat loop -----
if [ -n "${E2E_MOCK_LLM_LOG:-}" ] && [ -f "$EP_DIR/asst/workflow.yaml" ]; then
    echo "ENTRYSECRET 9" > "$EP_DIR/asst/note.txt"
    LINE=$(python3 -c 'import json,sys; print("CALLJSON read_file " + json.dumps({"file_path": sys.argv[1]}))' "$EP_DIR/asst/note.txt")
    OUT=$(cd "$EP_DIR/asst" && printf '%s\n/quit\n' "$LINE" | timeout 120 "$KDEPS_BIN" 2>&1 || true)
    if output_grep_fixed "TOOL RESULT" "$OUT" && output_grep_fixed "ENTRYSECRET 9" "$OUT"; then
        test_passed "entry points - bare kdeps runs workflow.yaml and the assistant reads a file"
    else
        test_failed "entry points - bare kdeps runs workflow.yaml and the assistant reads a file" "$(printf '%s' "$OUT" | tail -c 800)"
    fi
else
    test_skipped "entry points - bare kdeps runs the assistant (needs the mock LLM)"
fi

# --- the workflow's prompt is the agent REPL: slash commands work ---------------
if [ -f "$EP_DIR/asst/workflow.yaml" ]; then
    OUT=$(cd "$EP_DIR/asst" && printf '/help\n/quit\n' | timeout 120 "$KDEPS_BIN" run workflow.yaml 2>&1 || true)
    if output_grep_fixed "/model" "$OUT" && output_grep_fixed "/session" "$OUT"; then
        test_passed "entry points - kdeps run on the assistant opens the agent REPL (slash commands)"
    else
        test_failed "entry points - kdeps run on the assistant opens the agent REPL (slash commands)" "$(printf '%s' "$OUT" | tail -c 600)"
    fi
fi

# --- kdeps --agent-loop in the project: the workflow's loop.tools are the tools --
if [ -n "${E2E_MOCK_LLM_LOG:-}" ] && [ -f "$EP_DIR/asst/workflow.yaml" ]; then
    (cd "$EP_DIR/asst" && printf 'ENTRYLOOPTOOLS hi\n/quit\n' \
        | timeout 180 "$KDEPS_BIN" --agent-loop --new --backend openai --model fake > /dev/null 2>&1) || true
    if python3 - "$E2E_MOCK_LLM_LOG" <<'PY'
import json, sys
for line in open(sys.argv[1], encoding="utf-8"):
    try:
        body = json.loads(line)
    except ValueError:
        continue
    msgs = body.get("messages", [])
    last = msgs[-1].get("content") if msgs else ""
    if not isinstance(last, str) or "ENTRYLOOPTOOLS" not in last:
        continue
    names = {t.get("function", {}).get("name") for t in body.get("tools", [])}
    ok = {"read_file", "bash_exec"} <= names and not names & {"web_search", "load_tools"}
    sys.exit(0 if ok else 1)
sys.exit(1)
PY
    then
        test_passed "entry points - kdeps --agent-loop offers the workflow's loop.tools"
    else
        test_failed "entry points - kdeps --agent-loop offers the workflow's loop.tools" "no request with the allowlisted tools"
    fi
fi

# --- no workflow.yaml: help, naming --agent-loop ------------------------------
mkdir -p "$EP_DIR/empty"
OUT=$(cd "$EP_DIR/empty" && "$KDEPS_BIN" < /dev/null 2>&1)
if output_grep_fixed "kdeps --agent-loop [path] opens the agent REPL" "$OUT"; then
    test_passed "entry points - bare kdeps with no workflow.yaml prints help"
else
    test_failed "entry points - bare kdeps with no workflow.yaml prints help" "$(printf '%s' "$OUT" | tail -c 400)"
fi

# --- an agent-REPL flag needs --agent-loop ------------------------------------
OUT=$(cd "$EP_DIR/empty" && "$KDEPS_BIN" --model fake < /dev/null 2>&1)
if output_grep_fixed "--model applies to the agent REPL: use kdeps --agent-loop --model" "$OUT"; then
    test_passed "entry points - --model without --agent-loop is an error"
else
    test_failed "entry points - --model without --agent-loop is an error" "$(printf '%s' "$OUT" | tail -c 400)"
fi

rm -rf "$EP_DIR"
trap - EXIT
