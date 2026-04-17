#!/usr/bin/env python3
"""Simple HTTP server for testing KDeps WebServer proxy mode."""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import datetime

class SimpleHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {
                "message": "Hello from Python backend!",
                "timestamp": datetime.datetime.now().isoformat(),
                "status": "running",
                "proxied_by": "KDeps WebServer"
            }
            self.wfile.write(json.dumps(response).encode())
        elif self.path == '/api/data':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {
                "items": [
                    {"id": 1, "name": "Item 1"},
                    {"id": 2, "name": "Item 2"},
                    {"id": 3, "name": "Item 3"}
                ]
            }
            self.wfile.write(json.dumps(response).encode())
        elif self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "healthy"}).encode())
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, format, *args):
        # Custom log format
        print(f"[Backend] {self.address_string()} - {format % args}")

if __name__ == '__main__':
    server = HTTPServer(('127.0.0.1', 8501), SimpleHandler)
    print('[Backend] Starting HTTP server on http://127.0.0.1:8501')
    server.serve_forever()
