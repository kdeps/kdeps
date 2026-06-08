// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

//nolint:mnd // default timeouts and channel sizes are intentional
package http

import (
	"log/slog"
	stdhttp "net/http"
	"net/url"

	"github.com/gorilla/websocket"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func buildWebSocketTargetURL(
	targetURL *url.URL,
	route *domain.WebRoute,
	r *stdhttp.Request,
) url.URL {
	targetWSURL := *targetURL
	targetWSURL.Scheme = webSocketSchemeValue
	targetWSURL.Path = buildProxiedPath(route.Path, requestPath(r))
	copyQueryString(&targetWSURL, r.URL)
	return targetWSURL
}

func filterWebSocketHeaders(header stdhttp.Header) stdhttp.Header {
	wsHeaders := make(stdhttp.Header)
	for key, values := range header {
		if isWebSocketHopByHopHeader(key) {
			continue
		}
		wsHeaders[key] = values
	}
	return wsHeaders
}

func dialTargetWebSocket(
	targetWSURL url.URL,
	requestHeader stdhttp.Header,
) (*websocket.Conn, *stdhttp.Response, error) {
	dialer := websocket.Dialer{
		Proxy:            stdhttp.ProxyFromEnvironment,
		HandshakeTimeout: webSocketHandshakeTimeout(),
	}
	return dialer.Dial(targetWSURL.String(), filterWebSocketHeaders(requestHeader))
}

func upgradeClientWebSocket(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *stdhttp.Request) bool {
			return true
		},
	}
	return upgrader.Upgrade(w, r, nil)
}

func (s *WebServer) proxyWebSocketConnections(clientConn, targetConn *websocket.Conn) {
	errChan := make(chan error, 2)
	go relayWebSocketMessages(targetConn, clientConn, "target", "client", s.logger, errChan)
	go relayWebSocketMessages(clientConn, targetConn, "client", "target", s.logger, errChan)
	<-errChan
	logWebSocketProxyClosed(s.logger)
}

func relayWebSocketMessages(
	src, dst *websocket.Conn,
	srcLabel, dstLabel string,
	logger *slog.Logger,
	errChan chan<- error,
) {
	defer func() {
		_ = src.Close()
		_ = dst.Close()
	}()

	for {
		messageType, message, readErr := src.ReadMessage()
		if readErr != nil {
			if websocket.IsUnexpectedCloseError(
				readErr,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logWebSocketUnexpectedClose(logger, srcLabel, readErr)
			}
			errChan <- readErr
			return
		}

		if writeErr := writeWebSocketMessageHook(dst, messageType, message); writeErr != nil {
			logWebSocketWriteError(logger, dstLabel, writeErr)
			errChan <- writeErr
			return
		}
	}
}
