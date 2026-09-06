#!/bin/sh
# Serve this kdeps WASM app locally without Node.
# Sets the wasm MIME type and permissive CORS so the bookmarklet can load it.
PORT="${1:-3000}"
DIR="$(cd "$(dirname "$0")" && pwd)"
echo "kdeps WASM app on http://localhost:$PORT  (Ctrl+C to stop)"
exec python3 -c "
import http.server, functools, sys
class H(http.server.SimpleHTTPRequestHandler):
    extensions_map = {**http.server.SimpleHTTPRequestHandler.extensions_map, \".wasm\": \"application/wasm\"}
    def end_headers(self):
        self.send_header(\"Access-Control-Allow-Origin\", \"*\")
        super().end_headers()
http.server.test(HandlerClass=functools.partial(H, directory=\"$DIR\"), port=int(\"$PORT\"))
"
