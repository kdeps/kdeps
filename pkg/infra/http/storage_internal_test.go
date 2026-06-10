// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"errors"
	"log/slog"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemporaryFileStore_CleanupLoop(t *testing.T) {
	orig := cleanupLoopInterval
	cleanupLoopInterval = 5 * time.Millisecond
	t.Cleanup(func() { cleanupLoopInterval = orig })

	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	file, err := store.Store("old.txt", []byte("data"), "text/plain")
	require.NoError(t, err)
	store.files[file.ID].UploadedAt = time.Now().Add(-2 * time.Hour)

	time.Sleep(20 * time.Millisecond)
	_ = store.Close()
}

func TestRelayWebSocketMessages_UnexpectedClose(t *testing.T) {
	serverA := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*stdhttp.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				return
			}
		}
	}))
	defer serverA.Close()

	dialer := websocket.Dialer{}
	wsURL := "ws://" + serverA.Listener.Addr().String()
	srcConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)

	dstConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go relayWebSocketMessages(srcConn, dstConn, "src", "dst", slog.Default(), errCh)

	require.NoError(t, srcConn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "boom"),
	))

	select {
	case relayErr := <-errCh:
		require.Error(t, relayErr)
	case <-time.After(2 * time.Second):
		t.Fatal("expected unexpected close error")
	}
}

func TestRelayWebSocketMessages_WriteError(t *testing.T) {
	serverA := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*stdhttp.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, msg, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}
			_ = conn.WriteMessage(mt, msg)
		}
	}))
	defer serverA.Close()

	dialer := websocket.Dialer{}
	wsURL := "ws://" + serverA.Listener.Addr().String()
	srcConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer srcConn.Close()

	dstConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer dstConn.Close()

	orig := writeWebSocketMessageHook
	t.Cleanup(func() { writeWebSocketMessageHook = orig })
	writeWebSocketMessageHook = func(*websocket.Conn, int, []byte) error {
		return errors.New("write failed")
	}

	errCh := make(chan error, 1)
	go relayWebSocketMessages(srcConn, dstConn, "src", "dst", slog.Default(), errCh)

	require.NoError(t, srcConn.WriteMessage(websocket.TextMessage, []byte("ping")))
	select {
	case relayErr := <-errCh:
		require.Error(t, relayErr)
		assert.Contains(t, relayErr.Error(), "write failed")
	case <-time.After(2 * time.Second):
		t.Fatal("expected write error")
	}
}
