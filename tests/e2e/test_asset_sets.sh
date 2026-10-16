# shellcheck shell=bash
# E2E: every kdeps asset set, published, installed and used.
#
# The repo's assets are copied, one item of each set is changed or added
# with a marker, and the copy is published to a local server. A fresh home
# installs the full profile from it; each set is then checked through the
# kdeps behaviour it drives, against a mock LLM where a model is involved:
#
#   harness          a new preamble section reaches the model's system prompt
#   events           a new event is listed with its trigger and action
#   actions          the event's new action is registered (no warning);
#                    uninstalled, the event warns it is unknown
#   presets          a new preset is listed and applying it retunes the event
#   themes           a new theme is listed and can be switched to
#   tools            a changed tool description reaches the model
#   commands         a new declared command is in /help and sends its prompt
#   recipes          a new LLM server recipe is listed and shown
#   models           a new GGUF registry entry is listed
#   templates        kdeps new renders the changed template
#   docker           the Dockerfile carries the changed template
#   k8s              kdeps export k8s carries the changed template
#   schemas          the published schema's stricter rule rejects a workflow
#   components       a workflow runs a new published component
#   providers        a provider that exists only as an asset serves the agent
#                    loop and workflow chat calls
#   defaults         a workflow chat call uses the changed default timeout
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing every kdeps asset set, published and in use..."

if ! command -v python3 >/dev/null 2>&1 || ! command -v go >/dev/null 2>&1; then
    test_skipped "asset sets - python3 and go are needed to publish and serve assets"
    return 0 2>/dev/null || exit 0
fi

SA_DIR=$(cd "$(mktemp -d)" && pwd -P)
SA_SRV=""
SA_LLM=""
trap '[ -n "$SA_SRV" ] && kill "$SA_SRV" 2>/dev/null; [ -n "$SA_LLM" ] && kill "$SA_LLM" 2>/dev/null; rm -rf "$SA_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
free_port() { python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])'; }
wait_url() {
    for _ in $(seq 1 50); do
        python3 -c "import urllib.request;urllib.request.urlopen('$1')" 2>/dev/null && return 0
        sleep 0.1
    done
}

# --- the mock LLM: logs every request body, answers MOCK REPLY ----------------
SA_LOG="$SA_DIR/requests.jsonl"
: > "$SA_LOG"
cat > "$SA_DIR/mock.py" <<'PY'
import http.server as h, json, sys, time
LOG = sys.argv[2]
class H(h.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        b = json.dumps({"data": [{"id": "fake"}]}).encode()
        self.send_response(200); self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}"
        with open(LOG, "a", encoding="utf-8") as f:
            f.write(raw.decode("utf-8") + "\n")
        body = json.loads(raw)
        msg = {"role": "assistant", "content": "MOCK REPLY"}
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
LLM_PORT=$(free_port)
python3 "$SA_DIR/mock.py" "$LLM_PORT" "$SA_LOG" >/dev/null 2>"$SA_DIR/mock.err" </dev/null &
SA_LLM=$!
wait_url "http://127.0.0.1:$LLM_PORT/v1/models"

# --- the published assets: the repo's, with one marked item per set -----------
SA_SRC="$SA_DIR/src"
mkdir -p "$SA_SRC"
cp -R "$PROJECT_ROOT/assets/." "$SA_SRC/"
python3 - "$SA_SRC" "$LLM_PORT" <<'PY'
import json, os, re, sys
src, port = sys.argv[1], sys.argv[2]
def p(*a): return os.path.join(src, *a)
def write(rel, text):
    os.makedirs(os.path.dirname(p(rel)), exist_ok=True)
    open(p(rel), "w").write(text)
def bump(rel, new="9.0.0"):
    s = open(p(rel)).read()
    s = re.sub(r"(?m)^version: .*$", "version: " + new, s, count=1)
    open(p(rel), "w").write(s)
    return s

write("harness/e2e-marker.yaml", "version: 1.0.0\nname: e2e-marker\nkind: preamble-section\norder: 999\n"
      "body: |-\n  <e2e>E2E-HARNESS-MARKER</e2e>\n")
write("actions/e2e_marker_action.yaml", "version: 1.0.0\nname: e2e_marker_action\nkind: rounds\n"
      "description: E2E-ACTION-MARKER\n")
write("events/e2e-marker.yaml", "version: 1.0.0\nrequires:\n  actions/e2e_marker_action: \"\"\n"
      "name: e2e-marker\non:\n  rounds: 77\nrun: e2e_marker_action\n")
write("presets/e2e-preset.yaml", "version: 1.0.0\nrequires:\n  events/e2e-marker: \"\"\n"
      "name: e2e-preset\ndescription: E2E-PRESET-MARKER\nevents:\n"
      "  - name: e2e-marker\n    on:\n      rounds: 55\n    run: e2e_marker_action\n")
theme = open(p("themes", "vim.yaml")).read().replace("name: vim", "name: e2e-theme")
write("themes/e2e-theme.yaml", re.sub(r"(?m)^prompt: .*$", 'prompt: "E2E-THEME> "', theme))
s = bump("tools/read_file.yaml")
s = re.sub(r"(?m)^description: (.*)$", lambda m: "description: " + json.dumps("E2E-TOOL-MARKER " + m.group(1).strip("'\"")), s, count=1)
open(p("tools", "read_file.yaml"), "w").write(s)
write("commands/e2e-cmd.yaml", "version: 1.0.0\nname: e2e-cmd\norder: 999\nusage:\n"
      "  - args: \"[words]\"\n    help: E2E-COMMAND-HELP\nrun:\n  - prompt: \"E2E-COMMAND-PROMPT $ARGUMENTS\"\n")
recipe = open(p("recipes", "gguf.yaml")).read()
recipe = re.sub(r"(?m)^id: .*$", "id: e2e-recipe", recipe, count=1)
recipe = re.sub(r"(?m)^name: .*$", "name: E2E Recipe", recipe, count=1)
recipe = re.sub(r"(?m)^description: .*$", "description: E2E-RECIPE-MARKER", recipe, count=1)
write("recipes/e2e-recipe.yaml", recipe)
bump("models/gguf.yaml")
with open(p("models", "gguf.yaml"), "a") as f:
    f.write('  - alias: "e2e-registry-model"\n    url: "https://example.com/e2e-registry.gguf"\n')
bump("templates/api-service/template.yaml")
with open(p("templates", "api-service", "README.md.j2"), "a") as f:
    f.write("\nE2E-TEMPLATE-MARKER\n")
bump("docker/image/template.yaml")
for t in ("alpine.Dockerfile.tmpl", "ubuntu.Dockerfile.tmpl"):
    s = open(p("docker", "image", t)).read()
    open(p("docker", "image", t), "w").write("# E2E-DOCKER-MARKER\n" + s)
bump("k8s/manifests/template.yaml")
s = open(p("k8s", "manifests", "deployment.yaml.tmpl")).read()
open(p("k8s", "manifests", "deployment.yaml.tmpl"), "w").write("# E2E-K8S-MARKER\n" + s)
bump("schemas/v1/template.yaml")
schema = json.load(open(p("schemas", "v1", "workflow.json")))
schema["properties"]["metadata"]["required"] = ["name", "targetActionId", "description"]
json.dump(schema, open(p("schemas", "v1", "workflow.json"), "w"), indent=2)
write("components/e2e-echo/template.yaml", "version: 1.0.0\nrequires:\n  schemas/v1: \"\"\n")
write("components/e2e-echo/component.yaml", "apiVersion: kdeps.io/v1\nkind: Component\nmetadata:\n"
      "  name: e2e-echo\n  version: \"1.0.0\"\n  description: E2E component\nresources:\n"
      "  - name: Echo\n    actionId: e2eEcho\n    apiResponse:\n      success: true\n"
      "      response:\n        echoed: E2E-COMPONENT-MARKER\n")
write("providers/e2eprov.yaml", "version: 1.0.0\nname: e2eprov\nlabel: E2E Provider\norder: 999\n"
      "configKey: e2eprov_api_key\nenvVar: E2EPROV_API_KEY\nurl: http://127.0.0.1:" + port + "\n"
      "chatPath: /e2e/v1/chat/completions\n")
s = bump("defaults/executors.yaml")
open(p("defaults", "executors.yaml"), "w").write(s.replace('timeout: "60s"', 'timeout: "77s"', 1))
PY

SA_PUB="$SA_DIR/pub"
if ! (cd "$PROJECT_ROOT" && go run ./tools/publish-assets -src "$SA_SRC" -out "$SA_PUB" \
    -formulas "$SA_DIR/formulas" >"$SA_DIR/publish.log" 2>&1); then
    test_failed "asset sets - publish the marked assets" "$(tail -20 "$SA_DIR/publish.log")"
    return 0 2>/dev/null || exit 0
fi
SRV_PORT=$(free_port)
(cd "$SA_PUB" && exec python3 -c 'import functools,http.server as h,sys
class H(h.SimpleHTTPRequestHandler):
    def log_message(self, *a): pass
h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), functools.partial(H, directory=".")).serve_forever()' \
    "$SRV_PORT" >/dev/null 2>&1) &
SA_SRV=$!
wait_url "http://127.0.0.1:$SRV_PORT/index.json"

# --- a fresh home installs the full profile ------------------------------------
SA_HOME="$SA_DIR/home"
SA_WORK="$SA_DIR/work"
mkdir -p "$SA_HOME/.kdeps" "$SA_WORK"
printf 'llm:\n  e2eprov_api_key: sk-e2e\n' > "$SA_HOME/.kdeps/config.yaml"
# sa runs kdeps as a release binary: no source dir, the local servers.
sa() {
    (cd "$SA_WORK" && env -u E2EPROV_API_KEY HOME="$SA_HOME" USERPROFILE="$(native "$SA_HOME")" KDEPS_ASSETS_SRC='' \
        KDEPS_ASSETS_DIR="$SA_HOME/store" KDEPS_ASSETS_URL="http://127.0.0.1:$SRV_PORT" \
        KDEPS_CONFIG_PATH="$SA_HOME/.kdeps/config.yaml" KDEPS_DATA_DIR="$SA_HOME/data" KDEPS_PROFILE=full \
        "$@" 2>&1 </dev/null)
}
# sa_loop runs one piped agent session against the asset-only provider.
sa_loop() {
    (cd "$SA_WORK" && printf '%b' "$1" | env -u E2EPROV_API_KEY HOME="$SA_HOME" USERPROFILE="$(native "$SA_HOME")" \
        KDEPS_ASSETS_SRC='' KDEPS_ASSETS_DIR="$SA_HOME/store" KDEPS_ASSETS_URL="http://127.0.0.1:$SRV_PORT" \
        KDEPS_CONFIG_PATH="$SA_HOME/.kdeps/config.yaml" KDEPS_DATA_DIR="$SA_HOME/data" \
        timeout 180 "$KDEPS_BIN" --new --backend e2eprov --model fake 2>&1) || true
}
# requests_with prints the logged request bodies containing $1.
requests_with() { grep -F -- "$1" "$SA_LOG" || true; }

# A workflow with a description (the published schema requires one) whose
# chat call goes to the asset-only provider and whose component call runs
# the published component.
SA_WF="$SA_DIR/wf"
mkdir -p "$SA_WF/resources"
cat > "$SA_WF/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: sa-wf
  description: every asset set in use
  version: "1.0.0"
  targetActionId: response
settings:
  input:
    sources: [file]
  agentSettings:
    versions:
      kdeps: v2.0.0
      ollama: 0.5.4
      uv: 0.6.3
YAML
cat > "$SA_WF/resources/answer.yaml" <<'YAML'
actionId: answer
name: Answer
chat:
  backend: e2eprov
  model: fake
  prompt: "E2E-WORKFLOW-PROMPT {{ input('fileContent') }}"
  knowledge: false
YAML
cat > "$SA_WF/resources/echo.yaml" <<'YAML'
actionId: echo
name: Echo
requires: [answer]
component:
  name: e2e-echo
YAML
cat > "$SA_WF/resources/response.yaml" <<'YAML'
actionId: response
name: Response
requires: [echo]
apiResponse:
  success: true
  response:
    answer: "{{ get('answer') }}"
    echo: "{{ get('echo') }}"
YAML
printf 'hello' > "$SA_DIR/input.txt"

OUT=$(sa "$KDEPS_BIN" validate "$SA_WF/workflow.yaml")
if output_grep "Installing the full profile" "$OUT" && [ -f "$SA_HOME/store/lock.json" ]; then
    test_passed "asset sets - a fresh home installs the full profile from the published assets"
else
    test_failed "asset sets - full install" "$OUT"
fi

LIST=$(sa "$KDEPS_BIN" list --assets)
missing=""
for id in harness/e2e-marker events/e2e-marker actions/e2e_marker_action presets/e2e-preset themes/e2e-theme \
    tools/read_file commands/e2e-cmd recipes/e2e-recipe models/gguf templates/api-service docker/image \
    k8s/manifests schemas/v1 components/e2e-echo providers/e2eprov defaults/executors; do
    printf '%s\n' "$LIST" | grep -qE "^ *$id +[0-9]" || missing="$missing $id"
done
if [ -z "$missing" ]; then
    test_passed "asset sets - kdeps list --assets shows the published item of all 16 sets"
else
    test_failed "asset sets - list --assets" "missing:$missing $LIST"
fi

# --- schemas -----------------------------------------------------------------
if output_grep "Schema validation passed" "$OUT"; then
    test_passed "asset sets - schemas: a workflow meeting the published schema validates"
else
    test_failed "asset sets - schemas: valid workflow" "$OUT"
fi
sed '/^  description:/d' "$SA_WF/workflow.yaml" > "$SA_DIR/nodesc.yaml"
mkdir -p "$SA_DIR/nodesc/resources" && cp "$SA_WF/resources/"*.yaml "$SA_DIR/nodesc/resources/"
mv "$SA_DIR/nodesc.yaml" "$SA_DIR/nodesc/workflow.yaml"
if OUT=$(sa "$KDEPS_BIN" validate "$SA_DIR/nodesc/workflow.yaml"); then
    test_failed "asset sets - schemas: the published rule is enforced" "$OUT"
elif output_grep "description" "$OUT"; then
    test_passed "asset sets - schemas: the published schema's stricter rule rejects a workflow"
else
    test_failed "asset sets - schemas: rejection reason" "$OUT"
fi

# --- kdeps info: purpose and where each asset runs ------------------------------
OUT=$(sa "$KDEPS_BIN" info tools/read_file)
if output_grep_fixed "E2E-TOOL-MARKER" "$OUT" && output_grep "Where it runs" "$OUT" &&
    output_grep "tool definitions are sent to the model \] +<- tools/read_file" "$OUT" && output_grep "Installed: 9\.0\.0" "$OUT"; then
    test_passed "asset sets - kdeps info shows an asset's own purpose, where it runs and its installed version"
else
    test_failed "asset sets - kdeps info item" "$OUT"
fi
OUT=$(sa "$KDEPS_BIN" info events)
if output_grep "<- events" "$OUT" && output_grep "events/e2e-marker +1\.0\.0" "$OUT"; then
    test_passed "asset sets - kdeps info <set> shows the set's flow and its items"
else
    test_failed "asset sets - kdeps info set" "$OUT"
fi
if [ "$(python3 -c "import json,sys;d=json.load(open(sys.argv[1]));print(len(d.get('sets',{})), d['items']['tools/read_file'].get('description','')[:15])" "$SA_PUB/index.json")" = "16 E2E-TOOL-MARKER" ]; then
    test_passed "asset sets - the published index says where all 16 sets run, and each item's purpose"
else
    test_failed "asset sets - index sets" "$(head -c 400 "$SA_PUB/index.json")"
fi

# --- agent loop: harness, tools, commands, providers, events, actions, presets, themes
: > "$SA_LOG"
OUT=$(sa_loop '/help\nsay hi\n/e2e-cmd from the e2e\n/harness\n/harness events\n/harness preset list\n/harness preset e2e-preset\n/harness events\n/theme list\n/theme e2e-theme\n/quit\n')
if output_grep_fixed "MOCK REPLY" "$OUT" && [ -n "$(requests_with "say hi")" ]; then
    test_passed "asset sets - providers: the agent loop talks to a provider that exists only as an asset"
else
    test_failed "asset sets - providers: agent loop" "$OUT $(cat "$SA_DIR/mock.err")"
fi
if [ -n "$(requests_with "say hi" | grep -F "E2E-HARNESS-MARKER")" ] && output_grep "e2e-marker +preamble-section, order 999" "$OUT"; then
    test_passed "asset sets - harness: a published preamble section reaches the model's system prompt"
else
    test_failed "asset sets - harness" "$OUT"
fi
if [ -n "$(requests_with "say hi" | grep -F "E2E-TOOL-MARKER")" ]; then
    test_passed "asset sets - tools: a published tool description reaches the model"
else
    test_failed "asset sets - tools" "$(requests_with "say hi" | head -c 600)"
fi
if output_grep "/e2e-cmd \[words\] +E2E-COMMAND-HELP" "$OUT" && [ -n "$(requests_with "E2E-COMMAND-PROMPT from the e2e")" ]; then
    test_passed "asset sets - commands: a published command is in /help and sends its prompt with the arguments"
else
    test_failed "asset sets - commands" "$OUT"
fi
if output_grep "e2e-marker +on: rounds=77 +run: e2e_marker_action" "$OUT"; then
    test_passed "asset sets - events: a published event is listed with its trigger and action"
else
    test_failed "asset sets - events" "$OUT"
fi
if ! output_grep "e2e-marker.*not a registered action" "$OUT"; then
    test_passed "asset sets - actions: the event's published action is registered"
else
    test_failed "asset sets - actions" "$OUT"
fi
if output_grep "e2e-preset +E2E-PRESET-MARKER" "$OUT" && output_grep "Applied preset \"e2e-preset\"" "$OUT" &&
    output_grep "e2e-marker +on: rounds=55 +run: e2e_marker_action" "$OUT"; then
    test_passed "asset sets - presets: a published preset is listed and applying it retunes its event"
else
    test_failed "asset sets - presets" "$OUT"
fi
if output_grep "Built-in: .*e2e-theme" "$OUT" && ! output_grep -i "unknown theme" "$OUT"; then
    test_passed "asset sets - themes: a published theme is listed and can be switched to"
else
    test_failed "asset sets - themes" "$OUT"
fi
OUT=$(sa "$KDEPS_BIN" uninstall actions/e2e_marker_action)
OUT2=$(sa_loop '/harness events\n/quit\n')
if output_grep "e2e-marker.*e2e_marker_action.*not a registered action|e2e_marker_action" "$OUT2" &&
    output_grep "not a registered action" "$OUT2"; then
    test_passed "asset sets - actions: without the action installed, the event warns it is unknown"
else
    test_failed "asset sets - actions removed" "$OUT $OUT2"
fi
sa "$KDEPS_BIN" install actions/e2e_marker_action >/dev/null

# --- workflow mode: providers, defaults, components ---------------------------
: > "$SA_LOG"
OUT=$(sa "$KDEPS_BIN" run "$SA_WF/workflow.yaml" --file "$(native "$SA_DIR/input.txt")")
if [ -n "$(requests_with "E2E-WORKFLOW-PROMPT hello")" ] && grep -q "/e2e/v1/chat/completions" "$SA_DIR/mock.err" 2>/dev/null ||
    [ -n "$(requests_with "E2E-WORKFLOW-PROMPT hello")" ]; then
    test_passed "asset sets - providers: a workflow chat call goes to the asset-only provider"
else
    test_failed "asset sets - providers: workflow" "$OUT"
fi
if output_grep 'timeout: "(77s|1m17s)"' "$OUT"; then
    test_passed "asset sets - defaults: a chat call without a timeout uses the published default"
else
    test_failed "asset sets - defaults" "$OUT"
fi
if output_grep_fixed "E2E-COMPONENT-MARKER" "$OUT"; then
    test_passed "asset sets - components: a workflow runs a published component"
else
    test_failed "asset sets - components" "$OUT"
fi

# --- recipes, models, templates, docker, k8s ----------------------------------
OUT=$(sa "$KDEPS_BIN" llm list)
OUT2=$(sa "$KDEPS_BIN" llm show e2e-recipe)
if output_grep "e2e-recipe" "$OUT" && output_grep_fixed "E2E-RECIPE-MARKER" "$OUT2"; then
    test_passed "asset sets - recipes: a published LLM server recipe is listed and shown"
else
    test_failed "asset sets - recipes" "$OUT $OUT2"
fi
OUT=$(sa "$KDEPS_BIN" llamafile list)
if output_grep "GGUF +e2e-registry-model" "$OUT"; then
    test_passed "asset sets - models: a published model registry entry is listed"
else
    test_failed "asset sets - models" "$OUT"
fi
OUT=$(cd "$SA_WORK" && sa "$KDEPS_BIN" new e2e-agent --template api-service)
if grep -rqF "E2E-TEMPLATE-MARKER" "$SA_WORK/e2e-agent" 2>/dev/null; then
    test_passed "asset sets - templates: kdeps new renders the published template"
else
    test_failed "asset sets - templates" "$OUT $(ls -R "$SA_WORK/e2e-agent" 2>&1 | head -20)"
fi
OUT=$(sa "$KDEPS_BIN" bundle build "$SA_WF" --show-dockerfile)
if output_grep_fixed "E2E-DOCKER-MARKER" "$OUT"; then
    test_passed "asset sets - docker: the Dockerfile is rendered from the published template"
else
    test_failed "asset sets - docker" "$(printf '%s' "$OUT" | head -30)"
fi
OUT=$(sa "$KDEPS_BIN" export k8s "$SA_WF")
if output_grep_fixed "E2E-K8S-MARKER" "$OUT" && output_grep "kind: Deployment" "$OUT"; then
    test_passed "asset sets - k8s: kdeps export k8s renders the published template"
else
    test_failed "asset sets - k8s" "$(printf '%s' "$OUT" | head -30)"
fi

kill "$SA_SRV" "$SA_LLM" 2>/dev/null
wait "$SA_SRV" "$SA_LLM" 2>/dev/null
SA_SRV=""
SA_LLM=""
[ -n "${SA_KEEP:-}" ] || rm -rf "$SA_DIR"
trap - EXIT

echo ""
echo "Asset sets E2E: done"
