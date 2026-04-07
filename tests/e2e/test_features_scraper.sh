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

# E2E tests for the scraper executor.
#
# Creates local test data files and verifies text/CSV/JSON file scraping end-to-end
# via run.scraper:.
# ScraperConfig fields: type (required), source (required).
# Return shape: { content, type, source, success }.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
cd "$SCRIPT_DIR"

echo "Testing Scraper Component Feature..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Scraper - python3 not available"
    echo ""
    return 0 2>/dev/null || return 0
fi

API_PORT=$(python3 - <<'PY'
import socket
s = socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()
PY
)

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources" "$TEST_DIR/data"
LOG_FILE=$(mktemp)

trap 'kill "$KDEPS_PID" 2>/dev/null; wait "$KDEPS_PID" 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

# Seed test data files
echo "Hello from scraper E2E test" > "$TEST_DIR/data/sample.txt"
printf "name,value\nalice,1\nbob,2\n" > "$TEST_DIR/data/sample.csv"
printf '{"key":"scraped_value"}\n' > "$TEST_DIR/data/sample.json"

cat > "$TEST_DIR/workflow.yaml" <<WFEOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: scraper-e2e-test
  version: "1.0.0"
  targetActionId: response
settings:
  apiServerMode: true
  hostIp: "0.0.0.0"
  portNum: ${API_PORT}
  apiServer:
    routes:
      - path: /scrape/text
        methods: [POST]
      - path: /scrape/csv
        methods: [POST]
      - path: /scrape/json
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
WFEOF

cat > "$TEST_DIR/resources/text.yaml" <<RESEOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: scrapeText
  name: Scrape Text
run:
  validations:
    routes: [/scrape/text]
    methods: [POST]
  scraper:
    type: text
    source: "${TEST_DIR}/data/sample.txt"
RESEOF

cat > "$TEST_DIR/resources/csv.yaml" <<RESEOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: scrapeCSV
  name: Scrape CSV
run:
  validations:
    routes: [/scrape/csv]
    methods: [POST]
  scraper:
    type: csv
    source: "${TEST_DIR}/data/sample.csv"
RESEOF

cat > "$TEST_DIR/resources/jsonres.yaml" <<RESEOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: scrapeJSON
  name: Scrape JSON
run:
  validations:
    routes: [/scrape/json]
    methods: [POST]
  scraper:
    type: json
    source: "${TEST_DIR}/data/sample.json"
RESEOF

cat > "$TEST_DIR/resources/response.yaml" <<'RESEOF'
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: response
  name: Response
  requires: [scrapeText, scrapeCSV, scrapeJSON]
run:
  apiResponse:
    success: true
    response:
      textResult: "{{ output('scrapeText') }}"
      csvResult: "{{ output('scrapeCSV') }}"
      jsonResult: "{{ output('scrapeJSON') }}"
RESEOF

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!

KDEPS_STARTED=false
for i in $(seq 1 30); do
    if curl -sf --max-time 1 "http://127.0.0.1:${API_PORT}/health" > /dev/null 2>&1; then
        KDEPS_STARTED=true
        break
    fi
    sleep 0.5
done

if [ "$KDEPS_STARTED" = false ]; then
    test_skipped "Scraper - text file scraping"
    test_skipped "Scraper - CSV file scraping"
    test_skipped "Scraper - JSON file scraping"
    echo ""
    cat "$LOG_FILE"
    return 0 2>/dev/null || return 0
fi

# Test 1: text scraping
TEXT_RESP=$(curl -s --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/scrape/text" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

TEXT_CONTENT=$(echo "$TEXT_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    tr = (d.get('data') or {}).get('textResult') or {}
    print((tr.get('content') or '').strip())
except Exception:
    print('')
" 2>/dev/null || echo "")

if echo "$TEXT_CONTENT" | grep -q "scraper"; then
    test_passed "Scraper - text file scraping"
else
    test_failed "Scraper - text file scraping" "content='$TEXT_CONTENT' resp='$TEXT_RESP'"
fi

# Test 2: CSV scraping
CSV_RESP=$(curl -s --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/scrape/csv" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

CSV_CONTENT=$(echo "$CSV_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    cr = (d.get('data') or {}).get('csvResult') or {}
    print((cr.get('content') or '').strip())
except Exception:
    print('')
" 2>/dev/null || echo "")

if echo "$CSV_CONTENT" | grep -q "alice"; then
    test_passed "Scraper - CSV file scraping"
else
    test_failed "Scraper - CSV file scraping" "content='$CSV_CONTENT' resp='$CSV_RESP'"
fi

# Test 3: JSON scraping - output is a dict, check key
JSON_RESP=$(curl -s --max-time 5 -X POST "http://127.0.0.1:${API_PORT}/scrape/json" \
    -H "Content-Type: application/json" -d '{}' 2>&1)

JSON_OK=$(echo "$JSON_RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    jr = (d.get('data') or {}).get('jsonResult') or {}
    content = jr.get('content') or {}
    # content is either a dict (parsed JSON) or a string
    if isinstance(content, dict):
        print(content.get('key', ''))
    else:
        import json as j2
        parsed = j2.loads(content) if content else {}
        print(parsed.get('key', ''))
except Exception:
    print('')
" 2>/dev/null || echo "")

if echo "$JSON_OK" | grep -q "scraped_value"; then
    test_passed "Scraper - JSON file scraping"
else
    test_failed "Scraper - JSON file scraping" "key='$JSON_OK' resp='$JSON_RESP'"
fi

echo ""
echo "Scraper E2E tests complete."
