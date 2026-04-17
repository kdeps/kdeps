#!/usr/bin/env python3
"""Simple Flask backend for testing KDeps WebServer proxy mode."""

from flask import Flask, jsonify
import datetime

app = Flask(__name__)

@app.route('/')
def home():
    return jsonify({
        "message": "Hello from Flask backend!",
        "timestamp": datetime.datetime.now().isoformat(),
        "status": "running"
    })

@app.route('/api/data')
def get_data():
    return jsonify({
        "items": [
            {"id": 1, "name": "Item 1"},
            {"id": 2, "name": "Item 2"},
            {"id": 3, "name": "Item 3"}
        ]
    })

@app.route('/health')
def health():
    return jsonify({"status": "healthy"})

if __name__ == '__main__':
    app.run(host='127.0.0.1', port=8501, debug=False)
