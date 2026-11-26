#!/usr/bin/env python3
"""A fake websocket gateway for the listen E2E (Python stdlib only).

Usage: ws_gateway.py PORT LOG

Each connection gets {"op":10,"d":{"heartbeat_interval":200}}. After the
client sends an identify ({"op":2,...}) the gateway sends one
MESSAGE_CREATE event. Every text frame the client sends is appended to LOG.
"""
import base64
import hashlib
import json
import socket
import struct
import sys
import threading

GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"


def send_text(conn, text):
    data = text.encode()
    head = bytes([0x81])
    if len(data) < 126:
        head += bytes([len(data)])
    elif len(data) < 65536:
        head += bytes([126]) + struct.pack(">H", len(data))
    else:
        head += bytes([127]) + struct.pack(">Q", len(data))
    conn.sendall(head + data)


def recv_exact(conn, n):
    buf = b""
    while len(buf) < n:
        chunk = conn.recv(n - len(buf))
        if not chunk:
            raise ConnectionError("closed")
        buf += chunk
    return buf


def recv_frame(conn):
    b1, b2 = recv_exact(conn, 2)
    opcode = b1 & 0x0F
    length = b2 & 0x7F
    if length == 126:
        length = struct.unpack(">H", recv_exact(conn, 2))[0]
    elif length == 127:
        length = struct.unpack(">Q", recv_exact(conn, 8))[0]
    mask = recv_exact(conn, 4) if b2 & 0x80 else b"\0\0\0\0"
    payload = bytes(c ^ mask[i % 4] for i, c in enumerate(recv_exact(conn, length)))
    return opcode, payload


def handle(conn, log):
    try:
        req = b""
        while b"\r\n\r\n" not in req:
            req += conn.recv(4096)
        key = ""
        for line in req.decode().split("\r\n"):
            if line.lower().startswith("sec-websocket-key:"):
                key = line.split(":", 1)[1].strip()
        accept = base64.b64encode(hashlib.sha1((key + GUID).encode()).digest()).decode()
        conn.sendall(
            ("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
             "Sec-WebSocket-Accept: " + accept + "\r\n\r\n").encode()
        )
        send_text(conn, json.dumps({"op": 10, "d": {"heartbeat_interval": 200}}))
        while True:
            opcode, payload = recv_frame(conn)
            if opcode == 8:
                return
            if opcode != 1:
                continue
            text = payload.decode()
            with open(log, "a", encoding="utf-8") as f:
                f.write(text + "\n")
            if '"op":2' in text:
                send_text(conn, json.dumps(
                    {"op": 0, "s": 1, "t": "MESSAGE_CREATE", "d": {"content": "hello from the gateway"}}))
    except (ConnectionError, OSError, ValueError):
        return
    finally:
        conn.close()


def main():
    port, log = int(sys.argv[1]), sys.argv[2]
    srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    srv.bind(("127.0.0.1", port))
    srv.listen(8)
    while True:
        conn, _ = srv.accept()
        threading.Thread(target=handle, args=(conn, log), daemon=True).start()


if __name__ == "__main__":
    main()
