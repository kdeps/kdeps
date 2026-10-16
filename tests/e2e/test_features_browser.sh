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

# E2E tests for the browser component (kdeps components asset set): Playwright
# steps called with component: {name: browser}; the browser: key is gone.
# The live run needs uv and Playwright (CI installs both); otherwise skipped.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the browser component..."

BR_DIR=$(mktemp -d)
BR_PID=""
trap '[ -n "$BR_PID" ] && kill "$BR_PID" 2>/dev/null; rm -rf "$BR_DIR"' EXIT
mkdir -p "$BR_DIR/wf/resources"
printf 'apiVersion: kdeps.io/v1\nkind: Workflow\nmetadata:\n  name: b\n  version: "1.0.0"\n  targetActionId: b\nsettings: {}\n' > "$BR_DIR/wf/workflow.yaml"

# br_resource writes the component call for url $1.
br_resource() {
    cat > "$BR_DIR/wf/resources/b.yaml" <<EOF
actionId: b
name: Browser
component:
  name: browser
  with:
    url: "$1"
    actions:
      - {action: fill, selector: "#name", value: World}
      - {action: click, selector: "#btn"}
      - {action: wait, wait: "#result"}
      - {action: evaluate, script: "document.getElementById('result').textContent"}
EOF
}

# A registry component named browser would win over the shipped one.
export KDEPS_COMPONENT_DIR="$BR_DIR/components"

br_resource "https://example.com"
if "$KDEPS_BIN" validate "$BR_DIR/wf/workflow.yaml" >/dev/null 2>&1; then
    test_passed "browser component - a component: browser call validates"
else
    test_failed "browser component - validate" "$("$KDEPS_BIN" validate "$BR_DIR/wf/workflow.yaml" 2>&1 | tail -3)"
fi

if command -v uv >/dev/null 2>&1 && command -v playwright >/dev/null 2>&1 && command -v python3 >/dev/null 2>&1; then
    printf '%s' '<html><head><title>E2E Page</title></head><body><input id="name"><button id="btn" onclick="document.getElementById(&quot;result&quot;).textContent=&quot;Hello, &quot;+document.getElementById(&quot;name&quot;).value">Go</button><div id="result"></div></body></html>' > "$BR_DIR/index.html"
    PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1])')
    (cd "$BR_DIR" && python3 -m http.server "$PORT" --bind 127.0.0.1 >/dev/null 2>&1) & BR_PID=$!
    for _ in $(seq 1 50); do command curl -s -o /dev/null "http://127.0.0.1:$PORT/" && break; sleep 0.1; done
    br_resource "http://127.0.0.1:$PORT/index.html"
    OUT=$(timeout 600 "$KDEPS_BIN" run "$BR_DIR/wf" 2>&1)
    if output_grep "result:Hello, World" "$OUT" && output_grep "title:E2E Page" "$OUT"; then
        test_passed "browser component - fills, clicks and reads the page with Playwright"
    else
        test_failed "browser component - live run" "$(echo "$OUT" | tail -15)"
    fi
else
    test_skipped "browser component - live run needs uv and Playwright"
fi

printf 'actionId: b\nname: B\nbrowser:\n  url: https://example.com\n' > "$BR_DIR/wf/resources/b.yaml"
if OUT=$("$KDEPS_BIN" validate "$BR_DIR/wf/workflow.yaml" 2>&1); then
    test_failed "browser component - the browser: key is gone" "validate accepted it: $OUT"
else
    test_passed "browser component - the browser: resource key is gone (use component: browser)"
fi

[ -n "$BR_PID" ] && kill "$BR_PID" 2>/dev/null
rm -rf "$BR_DIR"
trap - EXIT
echo ""
