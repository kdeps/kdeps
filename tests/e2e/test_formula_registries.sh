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

# E2E: formula registries. kdeps reads formulas from its local copy of each
# registry (~/.kdeps/registries/<owner>/<repo>), so search, info and install
# need no server. The copy here is written by hand, as kdeps update leaves it;
# the package archive comes from a local HTTP server.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing formula registries..."

if ! command -v python3 > /dev/null 2>&1; then
    test_skipped "formula registries - python3 not on PATH"
    return 0 2>/dev/null || exit 0
fi

FR_DIR=$(mktemp -d)
FR_HOME="$FR_DIR/home"
mkdir -p "$FR_HOME" "$FR_DIR/www" "$FR_DIR/pkg"
trap 'kill "$FR_PID" 2>/dev/null; rm -rf "$FR_DIR"' EXIT

# A workflow package archive.
cat > "$FR_DIR/pkg/kdeps.pkg.yaml" <<'YEOF'
name: invoice-bot
version: 1.0.0
type: workflow
description: Reads invoices
YEOF
printf 'apiVersion: kdeps.io/v1\nkind: Workflow\nmetadata:\n  name: invoice-bot\n' > "$FR_DIR/pkg/workflow.yaml"
(cd "$FR_DIR/pkg" && tar -czf "$FR_DIR/www/invoice-bot-1.0.0.kdeps" kdeps.pkg.yaml workflow.yaml)

FR_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
(cd "$FR_DIR/www" && python3 -m http.server "$FR_PORT" --bind 127.0.0.1 >/dev/null 2>&1) &
FR_PID=$!
for _ in 1 2 3 4 5 6 7 8 9 10; do
    curl -s -o /dev/null "http://127.0.0.1:$FR_PORT/" && break
    sleep 0.3
done

# The local copy of two registries.
write_formula() { # registry name type tarball description
    local dir="$FR_HOME/.kdeps/registries/$1"
    mkdir -p "$dir/formulas"
    echo "now" > "$dir/.fetched"
    printf 'name: %s\nversion: 1.0.0\ntype: %s\ntarball: %s\ndescription: %s\ntags: [e2e]\n' \
        "$2" "$3" "$4" "$5" > "$dir/formulas/$2.yaml"
}
write_formula kdeps/registry invoice-bot workflow "http://127.0.0.1:$FR_PORT/invoice-bot-1.0.0.kdeps" "Reads invoices"
write_formula acme/tools scraper component "http://127.0.0.1:$FR_PORT/missing.komponent" "Scrapes pages"
printf 'registries:\n  - acme/tools\n' > "$FR_HOME/config.yaml"

fr() { (cd "$FR_DIR" && HOME="$FR_HOME" USERPROFILE="$FR_HOME" KDEPS_CONFIG_PATH="$FR_HOME/config.yaml" \
    KDEPS_AGENTS_DIR="$FR_HOME/agents" "$KDEPS_BIN" "$@" 2>&1); }

OUT=$(fr search invoice)
if output_grep "invoice-bot +workflow +1.0.0" "$OUT" && ! output_grep "scraper" "$OUT"; then
    test_passed "formula registries - search matches a description, not other formulas"
else
    test_failed "formula registries - search" "$OUT"
fi

# Git Bash on Windows rewrites an argument that starts with / as a path; keep /^...
OUT=$(MSYS2_ARG_CONV_EXCL='/^' fr search '/^scr/')
if output_grep "acme/tools/scraper" "$OUT"; then
    test_passed "formula registries - /regex/ search finds a formula in another registry, named with it"
else
    test_failed "formula registries - regex search" "$OUT"
fi

OUT=$(fr info acme/tools/scraper)
if output_grep "Registry: +acme/tools" "$OUT" && output_grep "formulas/scraper.yaml" "$OUT"; then
    test_passed "formula registries - info owner/repo/name shows the formula and where it comes from"
else
    test_failed "formula registries - info" "$OUT"
fi

OUT=$(fr install invoice-bot)
if output_grep "Downloading invoice-bot@1.0.0" "$OUT" && [ -f "$FR_HOME/agents/invoice-bot/workflow.yaml" ]; then
    test_passed "formula registries - install downloads the formula's archive and installs the agent"
else
    test_failed "formula registries - install" "$OUT"
fi

OUT=$(fr install invoice-bot@9.9.9)
if output_grep "the registry has version 1.0.0, not 9.9.9" "$OUT"; then
    test_passed "formula registries - a version the registry does not have is refused"
else
    test_failed "formula registries - pinned version" "$OUT"
fi

OUT=$(fr config)
if output_grep "Backend: " "$OUT" && output_grep "acme/tools +downloaded now" "$OUT" && output_grep "kdeps/registry +downloaded now" "$OUT"; then
    test_passed "formula registries - kdeps config lists each registry and its download time"
else
    test_failed "formula registries - kdeps config" "$OUT"
fi

echo ""
echo "Formula registry E2E tests complete."
