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
	"net/url"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestWebServer_HandleAppRequest_ParseURLError(t *testing.T) {
	orig := parseProxyURL
	t.Cleanup(func() { parseProxyURL = orig })
	parseProxyURL = func(string) (*url.URL, error) {
		return nil, errors.New("parse failed")
	}

	workflow := &domain.Workflow{Settings: domain.WorkflowSettings{WebServer: &domain.WebServerConfig{}}}
	webServer, err := NewWebServer(workflow, slog.Default())
	require.NoError(t, err)

	route := &domain.WebRoute{AppPort: 8080, Path: "/"}
	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	webServer.HandleAppRequest(rec, req, route)
	assert.Equal(t, stdhttp.StatusInternalServerError, rec.Code)
}

func TestWebServer_HandleWebSocketProxy_NonSwitchingResponse(t *testing.T) {
	echoSrv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*stdhttp.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
	}))
	defer echoSrv.Close()

	wsURL := "ws://" + echoSrv.Listener.Addr().String()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	orig := dialTargetWebSocketHook
	t.Cleanup(func() { dialTargetWebSocketHook = orig })
	dialTargetWebSocketHook = func(_ url.URL, _ stdhttp.Header) (*websocket.Conn, *stdhttp.Response, error) {
		return conn, &stdhttp.Response{StatusCode: stdhttp.StatusBadRequest, Body: stdhttp.NoBody}, nil
	}

	workflow := &domain.Workflow{Settings: domain.WorkflowSettings{WebServer: &domain.WebServerConfig{}}}
	webServer, err := NewWebServer(workflow, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	rec := httptest.NewRecorder()

	targetURL, err := url.Parse(wsURL)
	require.NoError(t, err)
	webServer.HandleWebSocketProxy(rec, req, targetURL, &domain.WebRoute{Path: "/"})
	assert.Equal(t, stdhttp.StatusBadGateway, rec.Code)
}

func TestWebServer_HandleWebSocketProxy_HandshakeFailure(t *testing.T) {
	targetSrv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusBadRequest)
	}))
	defer targetSrv.Close()

	targetURL, err := url.Parse("ws://" + targetSrv.Listener.Addr().String())
	require.NoError(t, err)

	workflow := &domain.Workflow{Settings: domain.WorkflowSettings{WebServer: &domain.WebServerConfig{}}}
	webServer, err := NewWebServer(workflow, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	rec := httptest.NewRecorder()

	webServer.HandleWebSocketProxy(rec, req, targetURL, &domain.WebRoute{Path: "/"})
	assert.Equal(t, stdhttp.StatusBadGateway, rec.Code)
}
