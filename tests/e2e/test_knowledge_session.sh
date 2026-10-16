# shellcheck shell=bash
# E2E: the knowledge vault inside real agent sessions and workflow runs,
# against a mock LLM that records every request it receives.
#
#   - a session sets up the vault, builds its graph in <vault>/.index
#   - a place no card covers gets projects/<name>/ and its card on the first
#     turn (named after the git repository root); archiving moves it to
#     archive/, working there again moves it back
#   - each turn's request carries <knowledge-recall> (the related notes),
#     <knowledge-status> (the cards) and the routing table; the terminal
#     shows "recall: N notes"; /recall and /knowledge report it
#   - memory saved in a turn is a note in projects/<name>/memory/ and
#     reaches the very next turn's request
#   - the session note is written after every turn (one note, all prompts)
#   - a piped session exits without a summary call
#   - /compact writes the model's summary into the note; its next step
#     goes to the card, each Key Decision to decisions/
#   - /knowledge off stops recall and is remembered by the next session;
#     knowledge.enabled: false turns everything off
#   - an old ~/.kdeps/memory database moves into the vault once
#   - a workflow chat call gets recall; chat.knowledge: false does not;
#     knowledge.workflow_notes: true logs each call to the workflow's project
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the knowledge vault in agent sessions and workflows (mock LLM)..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "knowledge session - python3 not available for the mock LLM"
    return 0 2>/dev/null || exit 0
fi

KS_DIR=$(cd "$(mktemp -d)" && pwd -P)
KS_SRV=""
trap '[ -n "$KS_SRV" ] && kill "$KS_SRV" 2>/dev/null; [ -n "${KS_KEEP:-}" ] || rm -rf "$KS_DIR"' EXIT
KS_LOG="$KS_DIR/requests.jsonl"
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }

# The mock answers prompt refinement with the request itself, compaction with
# a fixed summary, "remember ..." with a [MEMORY:] marker, anything else with
# MOCK REPLY. Every request body is appended to requests.jsonl.
cat > "$KS_DIR/mock.py" <<'PY'
import http.server as h, json, sys, time
LOG = sys.argv[2]
SUMMARY = ("## Goal\nShip monthly billing\n\n## Constraints & Preferences\n(none)\n\n## Progress\n### Done\n"
           "- [x] switched billing\n\n## Key Decisions\n- **Monthly**: per-call doubled churn\n\n## Mistakes\n"
           "- forgot webhook retries\n\n## Wins\n- tests green\n\n## Next Steps\n1. write the invoice job\n\n"
           "## Critical Context\n(none)")
def text(m):
    c = m.get("content", "")
    if isinstance(c, list):
        return " ".join(p.get("text", "") for p in c if isinstance(p, dict))
    return c or ""
class H(h.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        b = json.dumps({"data": [{"id": "fake"}]}).encode()
        self.send_response(200); self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}")
        with open(LOG, "a") as f:
            f.write(json.dumps(body) + "\n")
        msgs = body.get("messages", [])
        everything = "\n".join(text(m) for m in msgs)
        last = text(msgs[-1]) if msgs else ""
        if "You rewrite a user's request" in everything:
            reply = last.split("Request:\n", 1)[-1]
        elif "<conversation>" in everything:
            reply = SUMMARY
        elif "Saved note" in last or "Updated card" in last or "already exists" in last:
            reply = "TOOL DONE"
        elif "save a gotcha" in last.lower():
            reply = ('<invoke name="knowledge_note_save">\n<parameter name="title">Invoice timezone</parameter>\n'
                     '<parameter name="summary">Invoices use UTC dates.</parameter>\n'
                     '<parameter name="body">All invoice dates are UTC. See [[Billing decision]].</parameter>\n'
                     '<parameter name="type">gotcha</parameter>\n</invoke>')
        elif "archive the project" in last.lower():
            reply = ('<invoke name="knowledge_card_update">\n<parameter name="status">archived</parameter>\n</invoke>')
        elif "pause the project" in last.lower():
            reply = ('<invoke name="knowledge_card_update">\n<parameter name="status">paused</parameter>\n'
                     '<parameter name="next">wait for review</parameter>\n</invoke>')
        elif "remember" in last.lower():
            reply = "Noted. [MEMORY: fact:invoices] invoices go out on the 1st"
        else:
            reply = "MOCK REPLY"
        msg = {"role": "assistant", "content": reply}
        if body.get("stream"):
            self.send_response(200); self.send_header("Content-Type", "text/event-stream"); self.end_headers()
            ch = {"id": "x", "object": "chat.completion.chunk", "created": int(time.time()), "model": "fake",
                  "choices": [{"index": 0, "delta": msg, "finish_reason": None}]}
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\n")
            ch["choices"] = [{"index": 0, "delta": {}, "finish_reason": "stop"}]
            ch["usage"] = {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\ndata: [DONE]\n\n")
            return
        b = json.dumps({"id": "x", "object": "chat.completion", "created": int(time.time()), "model": "fake",
                        "choices": [{"index": 0, "message": msg, "finish_reason": "stop"}],
                        "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}}).encode()
        self.send_response(200); self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
PY

KS_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
python3 "$KS_DIR/mock.py" "$KS_PORT" "$KS_LOG" >/dev/null 2>"$KS_DIR/mock.err" </dev/null &
KS_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:$KS_PORT/v1/models')" 2>/dev/null && break
    sleep 0.1
done

# Assets with a provider that points at the mock.
KS_SRC="$KS_DIR/src"
mkdir -p "$KS_SRC"
cp -R "$PROJECT_ROOT/assets/." "$KS_SRC/"
cat > "$KS_SRC/providers/mockllm.yaml" <<YAML
version: 1.0.0
name: mockllm
label: Mock LLM
order: 999
configKey: mockllm_api_key
envVar: MOCKLLM_API_KEY
url: http://127.0.0.1:$KS_PORT
chatPath: /v1/chat/completions
YAML

# A vault with a card covering the project folder and two linked notes.
KS_HOME="$KS_DIR/home"
KS_VAULT="$KS_DIR/vault"
KS_PROJ="$KS_DIR/app"
mkdir -p "$KS_HOME/.kdeps" "$KS_PROJ" "$KS_VAULT/projects/app/decisions" "$KS_VAULT/shared"
cat > "$KS_VAULT/projects/app/app.md" <<CARD
---
type: project
status: active
next: ship billing
updated: 2026-10-01
---
| Source | Path |
|---|---|
| folder | $(native "$KS_PROJ") |
CARD
cat > "$KS_VAULT/projects/app/decisions/Billing decision.md" <<'NOTE'
---
type: decision
summary: Bill monthly, not per call - per-call billing doubled churn.
---
See [[Stripe gotcha]].
NOTE
cat > "$KS_VAULT/shared/Stripe gotcha.md" <<'NOTE'
---
type: gotcha
summary: Stripe retries webhooks for 3 days; handlers must be idempotent.
---
NOTE
printf '# Workspace routing\n\nE2E ROUTING TABLE\n' > "$KS_VAULT/KDEPS.md"

ks_config() { # $1: extra knowledge lines
    printf 'llm:\n  mockllm_api_key: sk-mock\nknowledge:\n  vault: %s\n%s' "$(native "$KS_VAULT")" "${1:-}" \
        > "$KS_HOME/.kdeps/config.yaml"
}
ks_config ""

# ks_session runs one piped agent session in the project folder;
# ks_session_in runs one in folder $1.
ks_session() { ks_session_in "$KS_PROJ" "$1"; }
ks_session_in() {
    (cd "$1" && printf '%b' "$2" | HOME="$KS_HOME" USERPROFILE="$(native "$KS_HOME")" \
        KDEPS_CONFIG_PATH="$KS_HOME/.kdeps/config.yaml" KDEPS_ASSETS_SRC="$KS_SRC" \
        KDEPS_ASSETS_DIR="$KS_HOME/store" KDEPS_ASSETS_URL=off \
        timeout 180 "$KDEPS_BIN" --new --backend mockllm --model fake 2>&1) || true
}

# ks_req prints the Nth logged request whose last user message contains $1.
ks_req() {
    # UTF-8 output: Windows defaults Python's stdout to a code page that
    # cannot print the arrows and dashes in the system prompt.
    PYTHONIOENCODING=utf-8 python3 - "$KS_LOG" "$1" <<'PY'
import json, sys
def text(m):
    c = m.get("content", "")
    return " ".join(p.get("text", "") for p in c if isinstance(p, dict)) if isinstance(c, list) else (c or "")
for line in open(sys.argv[1], encoding="utf-8"):
    body = json.loads(line)
    msgs = body.get("messages", [])
    if not msgs or sys.argv[2] not in text(msgs[-1]):
        continue
    if any("You rewrite a user's request" in text(m) for m in msgs):
        continue
    print("\n".join(text(m) for m in msgs))
PY
}

# An old-format memory database for this folder, to be moved in once.
OLD_DB=""
if command -v go >/dev/null 2>&1; then
    OLD_DB=$(cd "$PROJECT_ROOT" && go run ./tests/e2e/helpers/oldmemory "$(native "$KS_HOME")" "$(native "$KS_PROJ")" \
        "fact:legacy" "migrated from the old store" 2>/dev/null) || OLD_DB=""
fi

# --- session 1: recall, status, memory, live session note ----------------------
: > "$KS_LOG"
OUT=$(ks_session '/knowledge\nwhy do we bill monthly\n/recall\nremember invoices\nwhen do invoices go out\n/quit\n')

if output_grep "knowledge vault is on" "$OUT" && output_grep_fixed "vault" "$OUT"; then
    test_passed "knowledge session - /knowledge reports the vault on by default"
else
    test_failed "knowledge session - /knowledge" "$OUT"
fi
if [ -f "$KS_VAULT/.index/graph.json" ]; then
    test_passed "knowledge session - the session builds the graph in <vault>/.index"
else
    test_failed "knowledge session - graph not built" "$(ls -la "$KS_VAULT" 2>&1)"
fi

REQ1=$(ks_req "why do we bill monthly")
if output_grep_fixed "<knowledge-recall>" "$REQ1" && output_grep "Billing decision.md" "$REQ1"; then
    test_passed "knowledge session - the turn's request carries the recalled note"
else
    test_failed "knowledge session - recall missing from the request" "$REQ1"
fi
if output_grep "Stripe gotcha.md" "$REQ1"; then
    test_passed "knowledge session - recall walks the link to the related note"
else
    test_failed "knowledge session - linked note not recalled" "$REQ1"
fi
if output_grep_fixed "<knowledge-status>" "$REQ1" && output_grep "\| app \| project \| active \| ship billing" "$REQ1"; then
    test_passed "knowledge session - the request carries the cards' status"
else
    test_failed "knowledge session - card status missing" "$REQ1"
fi
# (the preamble is reduced by turo, so match the file's header, not its words)
if output_grep "KDEPS.md \(scope: .*vault.KDEPS.md\)" "$REQ1"; then
    test_passed "knowledge session - the vault's routing table (KDEPS.md) reaches the model"
else
    test_failed "knowledge session - routing table missing" "$REQ1"
fi
if output_grep "recall: [0-9]+ notes \(~[0-9]+ tok\)" "$OUT"; then
    test_passed "knowledge session - the terminal shows what recall added"
else
    test_failed "knowledge session - recall line missing" "$OUT"
fi
if output_grep "Billing decision.md" "$OUT" && output_grep "matched:" "$OUT"; then
    test_passed "knowledge session - /recall lists the notes and why"
else
    test_failed "knowledge session - /recall" "$OUT"
fi

MEM_NOTE="$KS_VAULT/projects/app/memory/fact-invoices.md"
if [ -f "$MEM_NOTE" ] && grep -q "invoices go out on the 1st" "$MEM_NOTE" && grep -q "key: fact:invoices" "$MEM_NOTE"; then
    test_passed "knowledge session - memory saved in a turn is a note in projects/<name>/memory/"
else
    test_failed "knowledge session - memory note" "$(find "$KS_VAULT/projects" -type f 2>&1)"
fi
REQ3=$(ks_req "when do invoices go out")
if output_grep_fixed "invoices go out on the 1st" "$REQ3"; then
    test_passed "knowledge session - memory saved in one turn reaches the next turn's request"
else
    test_failed "knowledge session - memory not in the next request" "$REQ3"
fi
if [ -f "$KS_VAULT/projects/app/memory/turn-last.md" ]; then
    test_passed "knowledge session - the turn log is a memory note too"
else
    test_failed "knowledge session - turn:last note" "$(find "$KS_VAULT/projects" -type f 2>&1)"
fi

NOTES=$(find "$KS_VAULT/projects/app/sessions" -name '*.md' 2>/dev/null)
NOTE_COUNT=$(printf '%s\n' "$NOTES" | grep -c '\.md$')
if [ "$NOTE_COUNT" = "1" ] && grep -q "why do we bill monthly" "$NOTES" && grep -q "when do invoices go out" "$NOTES" &&
    grep -q "## Focus" "$NOTES" && grep -q '"\[\[projects/app/app\]\]"' "$NOTES"; then
    test_passed "knowledge session - one session note in projects/<name>/sessions/, updated every turn, linked to the card by full path"
else
    test_failed "knowledge session - live session note" "count=$NOTE_COUNT $(cat $NOTES 2>&1)"
fi
if ! grep -q "<conversation>" "$KS_LOG"; then
    test_passed "knowledge session - a piped session exits without a summary call"
else
    test_failed "knowledge session - summary call at exit" "$(grep -c conversation "$KS_LOG")"
fi

if [ -n "$OLD_DB" ]; then
    MOVED="$KS_VAULT/projects/app/memory/fact-legacy.md"
    if [ -f "$MOVED" ] && grep -q "migrated from the old store" "$MOVED" && [ -f "$OLD_DB.moved" ] && [ ! -e "$OLD_DB" ]; then
        test_passed "knowledge session - the old memory database moved into the vault once"
    else
        test_failed "knowledge session - old memory move" "$(ls "$(dirname "$OLD_DB")" 2>&1) $(find "$KS_VAULT/projects" -type f 2>&1)"
    fi
else
    test_skipped "knowledge session - old memory move (go not available)"
fi

# --- session 2: /compact writes the model summary; next step to the card ---------
: > "$KS_LOG"
OUT=$(ks_session 'first billing question\nsecond billing question\nthird billing question\nfourth billing question\nfifth billing question\nsixth billing question\n/compact\nlast billing question\n/quit\n')
LATEST=$(ls -t "$KS_VAULT/projects/app/sessions/"*.md 2>/dev/null | head -1)
if grep -q "<conversation>" "$KS_LOG" && grep -q "Ship monthly billing" "$LATEST" && grep -q "forgot webhook retries" "$LATEST" &&
    grep -q "tests green" "$LATEST" && grep -q "last billing question" "$LATEST"; then
    test_passed "knowledge session - /compact writes the model summary; later turns are added on top"
else
    test_failed "knowledge session - compaction note" "$OUT $(cat "$LATEST" 2>&1)"
fi
if grep -q "^next: write the invoice job" "$KS_VAULT/projects/app/app.md"; then
    test_passed "knowledge session - the summary's next step moves to the card (its only home)"
else
    test_failed "knowledge session - card next step" "$(cat "$KS_VAULT/projects/app/app.md")"
fi
DECISION="$KS_VAULT/projects/app/decisions/Monthly.md"
if [ -f "$DECISION" ] && grep -q "type: decision" "$DECISION" && grep -q "per-call doubled churn" "$DECISION" &&
    grep -q '\[\[projects/app/app\]\]' "$DECISION" && grep -q '\[\[projects/app/sessions/' "$DECISION"; then
    test_passed "knowledge session - each Key Decision of the summary becomes a note in decisions/"
else
    test_failed "knowledge session - decision note" "$(ls "$KS_VAULT/projects/app/decisions" 2>&1)"
fi

# --- session: the model uses the knowledge tools ---------------------------------
: > "$KS_LOG"
OUT=$(ks_session 'save a gotcha about invoices\npause the project\nhow is the project going\n/quit\n')
GOTCHA="$KS_VAULT/projects/app/notes/Invoice timezone.md"
if [ -f "$GOTCHA" ] && grep -q "summary: \"Invoices use UTC dates.\"" "$GOTCHA" && grep -q "type: gotcha" "$GOTCHA"; then
    test_passed "knowledge session - the model saves a note into the project's notes/ with knowledge_note_save"
else
    test_failed "knowledge session - knowledge_note_save" "$OUT $(find "$KS_VAULT/projects/app" -type f)"
fi
if grep -q "^status: paused" "$KS_VAULT/projects/app/app.md" && grep -q "^next: wait for review" "$KS_VAULT/projects/app/app.md"; then
    test_passed "knowledge session - the model updates the working directory's card"
else
    test_failed "knowledge session - knowledge_card_update" "$OUT $(cat "$KS_VAULT/projects/app/app.md")"
fi
REQ=$(ks_req "how is the project going")
if output_grep "\| app \| project \| paused \| wait for review" "$REQ"; then
    test_passed "knowledge session - the next turn sees the updated card status"
else
    test_failed "knowledge session - card status per turn" "$REQ"
fi
: > "$KS_LOG"
OUT=$(ks_session 'save a gotcha about invoices again\n/quit\n')
if grep -q "already exists" "$KS_LOG"; then
    test_passed "knowledge session - a second note with the same title is refused (one home per fact)"
else
    test_failed "knowledge session - duplicate note" "$OUT"
fi
: > "$KS_LOG"
OUT=$(ks_session 'what is the invoice timezone\n/quit\n')
REQ=$(ks_req "what is the invoice timezone")
if output_grep "Invoice timezone.md" "$REQ"; then
    test_passed "knowledge session - a note saved in one session is recalled in the next (graph rebuilt)"
else
    test_failed "knowledge session - recall of the new note" "$REQ"
fi

# --- a memory note written by hand reaches the model ----------------------------
printf 'Always deploy on Tuesdays.\n' > "$KS_VAULT/projects/app/memory/Deploy day.md"
: > "$KS_LOG"
OUT=$(ks_session 'when do we deploy\n/quit\n')
REQ=$(ks_req "when do we deploy")
if output_grep_fixed "Always deploy on Tuesdays." "$REQ"; then
    test_passed "knowledge session - a memory note written by hand reaches the model"
else
    test_failed "knowledge session - hand-written memory" "$REQ"
fi

# --- recall honours its token budget --------------------------------------------
ks_config '  recall:
    budget_tokens: 40
'
: > "$KS_LOG"
OUT=$(ks_session 'why do we bill monthly budget\n/quit\n')
REQ=$(ks_req "why do we bill monthly budget")
RECALLED=$(printf '%s\n' "$REQ" | sed -n '/<knowledge-recall>/,/<\/knowledge-recall>/p' | grep -c '^- ')
SHOWN=$(printf '%s\n' "$OUT" | sed -n 's/.*recall: \([0-9]*\) notes (~\([0-9]*\) tok).*/\1 \2/p' | head -1)
if [ -n "$SHOWN" ] && [ "${SHOWN% *}" = "$RECALLED" ] && [ "${SHOWN#* }" -le 40 ] && [ "$RECALLED" -le 2 ]; then
    test_passed "knowledge session - recall.budget_tokens caps what a turn carries"
else
    test_failed "knowledge session - recall budget" "recalled=$RECALLED shown=$SHOWN $OUT"
fi
ks_config ""

# --- the vault the sessions wrote passes the one-home check ---------------------
OUT=$(HOME="$KS_HOME" USERPROFILE="$(native "$KS_HOME")" KDEPS_CONFIG_PATH="$KS_HOME/.kdeps/config.yaml" \
    KDEPS_ASSETS_SRC="$KS_SRC" KDEPS_ASSETS_DIR="$KS_HOME/store" KDEPS_ASSETS_URL=off \
    "$KDEPS_BIN" knowledge check 2>&1 </dev/null) && CHECK_OK=1 || CHECK_OK=0
if [ "$CHECK_OK" = 1 ] && output_grep "every fact has one home" "$OUT"; then
    test_passed "knowledge session - notes, memory and cards written by sessions pass kdeps knowledge check"
else
    test_failed "knowledge session - check after sessions" "$OUT"
fi

# --- the project's search index: recalled files and a card row ----------------
printf 'package billing\n// monthly invoices go out on the first day\n' > "$KS_PROJ/invoices.go"
: > "$KS_LOG"
OUT=$(ks_session '/search index\nwhen do monthly invoices go out\n/quit\n')
REQ=$(ks_req "when do monthly invoices go out")
if output_grep_fixed "<knowledge-files>" "$REQ" && output_grep "invoices.go" "$REQ" && output_grep "recall: [0-9]+ notes .*, [0-9]+ files" "$OUT"; then
    test_passed "knowledge session - recall also names the project's files from its search index"
else
    test_failed "knowledge session - files from the search index" "$OUT | $REQ"
fi
if grep -q "^| index | " "$KS_VAULT/projects/app/app.md"; then
    test_passed "knowledge session - the card records where the project's search index lives"
else
    test_failed "knowledge session - index row on the card" "$(cat "$KS_VAULT/projects/app/app.md")"
fi

# --- a new place gets its own project; archiving moves it and back ------------
KS_SIDE="$KS_DIR/side"
mkdir -p "$KS_SIDE/.git" "$KS_SIDE/web/src"
: > "$KS_LOG"
OUT=$(ks_session_in "$KS_SIDE/web/src" 'what is in this repo\nremember invoices\n/quit\n')
SIDE_CARD="$KS_VAULT/projects/side/side.md"
if [ -f "$SIDE_CARD" ] && grep -q "^status: active" "$SIDE_CARD" && grep -q "| repo | " "$SIDE_CARD" &&
    [ -n "$(find "$KS_VAULT/projects/side/sessions" -name '*.md' 2>/dev/null)" ] &&
    [ -f "$KS_VAULT/projects/side/memory/fact-invoices.md" ]; then
    test_passed "knowledge session - a place without a card gets projects/<repo name>/ with card, sessions/ and memory/"
else
    test_failed "knowledge session - auto project" "$OUT $(find "$KS_VAULT/projects/side" 2>&1)"
fi
REQ=$(ks_req "what is in this repo")
if output_grep "\| app \| project \|" "$REQ" && output_grep "\| side \| project \| active" "$REQ"; then
    test_passed "knowledge session - the status overview lists every active project, the new one included"
else
    test_failed "knowledge session - overview with the new project" "$REQ"
fi
: > "$KS_LOG"
OUT=$(ks_session_in "$KS_SIDE" 'archive the project\n/quit\n')
if [ -f "$KS_VAULT/archive/side/side.md" ] && [ ! -d "$KS_VAULT/projects/side" ] &&
    grep -q "^status: archived" "$KS_VAULT/archive/side/side.md"; then
    test_passed "knowledge session - status archived moves the whole project folder to archive/"
else
    test_failed "knowledge session - archive" "$OUT $(find "$KS_VAULT/archive" "$KS_VAULT/projects" -maxdepth 2 2>&1)"
fi
: > "$KS_LOG"
OUT=$(ks_session_in "$KS_SIDE" 'back to work\n/quit\n')
if [ -f "$KS_VAULT/projects/side/side.md" ] && [ ! -d "$KS_VAULT/archive/side" ] &&
    grep -q "^status: active" "$KS_VAULT/projects/side/side.md" &&
    [ "$(find "$KS_VAULT/projects/side/sessions" -name '*.md' | wc -l | tr -d ' ')" -ge 3 ]; then
    test_passed "knowledge session - working in an archived project moves it back, history intact"
else
    test_failed "knowledge session - unarchive" "$OUT $(find "$KS_VAULT/archive" "$KS_VAULT/projects/side" 2>&1)"
fi

# --- sessions 3-4: /knowledge off is remembered ----------------------------------
: > "$KS_LOG"
OUT=$(ks_session '/knowledge off\nwhy do we bill monthly off\n/quit\n')
REQ=$(ks_req "why do we bill monthly off")
if output_grep "knowledge vault off" "$OUT" && [ -n "$REQ" ] && ! output_grep_fixed "<knowledge-recall>" "$REQ"; then
    test_passed "knowledge session - /knowledge off stops recall"
else
    test_failed "knowledge session - /knowledge off" "$OUT | $REQ"
fi
: > "$KS_LOG"
OUT=$(ks_session 'why do we bill monthly later\n/knowledge on\n/quit\n')
REQ=$(ks_req "why do we bill monthly later")
if [ -n "$REQ" ] && ! output_grep_fixed "<knowledge-recall>" "$REQ" && output_grep "knowledge vault on" "$OUT"; then
    test_passed "knowledge session - /knowledge off is remembered by the next session"
else
    test_failed "knowledge session - persisted off" "$OUT | $REQ"
fi

# --- session 5: knowledge.enabled: false turns everything off --------------------
ks_config '  enabled: false
'
: > "$KS_LOG"
BEFORE=$(find "$KS_VAULT/projects/app/sessions" -name '*.md' | wc -l | tr -d ' ')
OUT=$(ks_session 'why do we bill monthly disabled\n/quit\n')
REQ=$(ks_req "why do we bill monthly disabled")
AFTER=$(find "$KS_VAULT/projects/app/sessions" -name '*.md' | wc -l | tr -d ' ')
if [ -n "$REQ" ] && ! output_grep_fixed "<knowledge-recall>" "$REQ" && ! output_grep "vault.KDEPS.md" "$REQ" &&
    [ "$BEFORE" = "$AFTER" ]; then
    test_passed "knowledge session - knowledge.enabled: false: no recall, no routing table, no session note"
else
    test_failed "knowledge session - disabled" "before=$BEFORE after=$AFTER | $REQ"
fi
ks_config ""

# --- workflow mode: chat calls recall by default; knowledge: false skips -------
KS_WF="$KS_DIR/wf"
mkdir -p "$KS_WF/resources"
cat > "$KS_WF/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: ks-wf
  version: "1.0.0"
  targetActionId: response
settings:
  input:
    sources: [file]
YAML
ks_wf_resources() { # $1: knowledge line for the chat resource
    cat > "$KS_WF/resources/answer.yaml" <<YAML
actionId: answer
name: Answer
chat:
  backend: mockllm
  model: fake
  prompt: "workflow question: {{ input('fileContent') }}"
$1
YAML
    cat > "$KS_WF/resources/response.yaml" <<'YAML'
actionId: response
name: Response
requires: [answer]
apiResponse:
  success: true
  response:
    answer: "{{ get('answer') }}"
YAML
}
printf 'why do we bill monthly' > "$KS_DIR/question.txt"
ks_run() {
    HOME="$KS_HOME" USERPROFILE="$(native "$KS_HOME")" KDEPS_CONFIG_PATH="$KS_HOME/.kdeps/config.yaml" \
        KDEPS_ASSETS_SRC="$KS_SRC" KDEPS_ASSETS_DIR="$KS_HOME/store" KDEPS_ASSETS_URL=off \
        timeout 120 "$KDEPS_BIN" run "$KS_WF/workflow.yaml" --file "$(native "$KS_DIR/question.txt")" 2>&1 </dev/null || true
}
ks_wf_resources ""
: > "$KS_LOG"
OUT=$(ks_run)
REQ=$(ks_req "workflow question")
if output_grep_fixed "<knowledge-recall>" "$REQ" && output_grep "Billing decision.md" "$REQ"; then
    test_passed "knowledge workflow - a chat call recalls related notes by default"
else
    test_failed "knowledge workflow - recall in chat call" "$OUT | $REQ"
fi
ks_wf_resources "  knowledge: false"
: > "$KS_LOG"
OUT=$(ks_run)
REQ=$(ks_req "workflow question")
if [ -n "$REQ" ] && ! output_grep_fixed "<knowledge-recall>" "$REQ"; then
    test_passed "knowledge workflow - chat.knowledge: false skips recall for that call"
else
    test_failed "knowledge workflow - knowledge: false" "$OUT | $REQ"
fi
if [ ! -d "$KS_VAULT/projects/ks-wf" ]; then
    test_passed "knowledge workflow - workflow runs write nothing by default"
else
    test_failed "knowledge workflow - wrote without workflow_notes" "$(find "$KS_VAULT/projects/ks-wf")"
fi
ks_config '  workflow_notes: true
'
ks_wf_resources ""
OUT=$(ks_run)
OUT=$(ks_run)
WF_CARD="$KS_VAULT/projects/ks-wf/ks-wf.md"
WF_RUNS=$(find "$KS_VAULT/projects/ks-wf/sessions" -name '* ks-wf runs.md' 2>/dev/null)
if [ -f "$WF_CARD" ] && grep -q "| workflow | " "$WF_CARD" && [ -n "$WF_RUNS" ] &&
    [ "$(grep -c "workflow question: why do we bill monthly -> MOCK REPLY" "$WF_RUNS")" = "2" ]; then
    test_passed "knowledge workflow - workflow_notes: true logs each chat call to projects/<workflow>/sessions/"
else
    test_failed "knowledge workflow - workflow notes" "$OUT $(find "$KS_VAULT/projects/ks-wf" 2>&1) $(cat "$WF_RUNS" 2>&1)"
fi
ks_config ""

kill "$KS_SRV" 2>/dev/null
wait "$KS_SRV" 2>/dev/null
KS_SRV=""
[ -n "${KS_KEEP:-}" ] || rm -rf "$KS_DIR"
trap - EXIT

echo ""
echo "Knowledge session E2E: done"
