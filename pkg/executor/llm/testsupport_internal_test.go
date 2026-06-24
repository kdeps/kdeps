// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
package llm

import (
	"errors"
	"log/slog"
	"net"
)

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

type failingListener struct{}

func (f failingListener) Accept() (net.Conn, error) { return nil, nil }

func (f failingListener) Close() error { return errors.New("close failed") }

func (f failingListener) Addr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999} }
