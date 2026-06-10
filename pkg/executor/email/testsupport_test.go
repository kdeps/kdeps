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

package email_test

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	executorEmail "github.com/kdeps/kdeps/v2/pkg/executor/email"
)

type smtpCapture struct {
	from    string
	to      []string
	message []byte
}

func startFakeSMTP(t *testing.T) (string, int, func() *smtpCapture) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	smtpHost := "127.0.0.1"
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	smtpPort, _ := strconv.Atoi(portStr)

	var mu sync.Mutex
	var captured *smtpCapture

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			go handleSMTPConn(conn, &mu, &captured)
		}
	}()

	return smtpHost, smtpPort, func() *smtpCapture {
		mu.Lock()
		defer mu.Unlock()
		return captured
	}
}

func handleSMTPConn(conn net.Conn, mu *sync.Mutex, recv **smtpCapture) {
	defer conn.Close()
	r := bufio.NewReader(conn)

	_, _ = fmt.Fprint(conn, "220 test.local ESMTP\r\n")

	var from string
	var to []string
	var msgBuf []byte
	inData := false

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)

		if inData {
			if line == "." {
				inData = false
				mu.Lock()
				*recv = &smtpCapture{from: from, to: to, message: msgBuf}
				mu.Unlock()
				_, _ = fmt.Fprint(conn, "250 OK\r\n")
			} else {
				line = strings.TrimPrefix(line, ".")
				msgBuf = append(msgBuf, []byte(line+"\r\n")...)
			}
			continue
		}

		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			_, _ = fmt.Fprint(conn, "250 OK\r\n")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			from = strings.Trim(line[10:], "< >")
			_, _ = fmt.Fprint(conn, "250 OK\r\n")
		case strings.HasPrefix(upper, "RCPT TO:"):
			to = append(to, strings.Trim(line[8:], "< >"))
			_, _ = fmt.Fprint(conn, "250 OK\r\n")
		case upper == "DATA":
			inData = true
			msgBuf = nil
			_, _ = fmt.Fprint(conn, "354 Start mail input; end with <CRLF>.<CRLF>\r\n")
		case strings.HasPrefix(upper, "QUIT"):
			_, _ = fmt.Fprint(conn, "221 Bye\r\n")
			return
		default:
			_, _ = fmt.Fprint(conn, "500 Unknown command\r\n")
		}
	}
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

func newAdapter() executor.ResourceExecutor {
	return executorEmail.NewAdapter(newLogger())
}

func newExecCtx(t *testing.T) *executor.ExecutionContext {
	t.Helper()
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test-wf", TargetActionID: "r"},
		Resources: []*domain.Resource{
			{
				ActionID: "r",
				Name:     "R",
				Email:    &domain.EmailConfig{},
			},
		},
	}
	ctx, err := executor.NewExecutionContext(wf)
	require.NoError(t, err)
	return ctx
}

func newExecCtxWithIMAP(t *testing.T, imapCfg kdepsconfig.IMAPConnectionConfig) *executor.ExecutionContext {
	t.Helper()
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test-wf", TargetActionID: "r"},
		Resources: []*domain.Resource{
			{ActionID: "r", Name: "R", Email: &domain.EmailConfig{}},
		},
	}
	ctx, err := executor.NewExecutionContext(wf)
	require.NoError(t, err)
	ctx.Config = &kdepsconfig.Config{
		IMAPConnections: map[string]kdepsconfig.IMAPConnectionConfig{
			"test": imapCfg,
		},
	}
	return ctx
}

func newExecCtxWithSMTP(t *testing.T, smtpCfg kdepsconfig.SMTPConnectionConfig) *executor.ExecutionContext {
	t.Helper()
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test-wf", TargetActionID: "r"},
		Resources: []*domain.Resource{
			{ActionID: "r", Name: "R", Email: &domain.EmailConfig{}},
		},
	}
	ctx, err := executor.NewExecutionContext(wf)
	require.NoError(t, err)
	ctx.Config = &kdepsconfig.Config{
		SMTPConnections: map[string]kdepsconfig.SMTPConnectionConfig{
			"test": smtpCfg,
		},
	}
	return ctx
}

func closedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	p, _ := strconv.Atoi(portStr)
	ln.Close()
	return p
}

type fakeIMAPMsg struct {
	subject string
	from    string
	to      string
	body    string
	seen    bool
}

type memIMAPServer struct {
	server *imapserver.Server
	ln     net.Listener
	user   *imapmemserver.User
}

func startFakeIMAP(t *testing.T, msgs []fakeIMAPMsg) *memIMAPServer {
	t.Helper()

	const (
		imapUser = "testuser"
		imapPass = "testpass"
	)

	memSrv := imapmemserver.New()
	u := imapmemserver.NewUser(imapUser, imapPass)
	require.NoError(t, u.Create("INBOX", nil))
	require.NoError(t, u.Create("Archive", nil))
	memSrv.AddUser(u)

	srv := imapserver.New(&imapserver.Options{
		NewSession: func(_ *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memSrv.NewSession(), nil, nil
		},
		Caps: imap.CapSet{ //nolint:exhaustive // test server only advertises IMAP4rev1.
			imap.CapIMAP4rev1: {},
		},
		InsecureAuth: true,
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = srv.Close()
		_ = ln.Close()
	})

	go srv.Serve(ln) //nolint:errcheck

	fi := &memIMAPServer{server: srv, ln: ln, user: u}

	for _, msg := range msgs {
		raw := buildRawMessage(msg)
		flags := []imap.Flag{}
		if msg.seen {
			flags = append(flags, imap.FlagSeen)
		}
		r := bytes.NewReader(raw)
		_, appendErr := u.Append("INBOX", r, &imap.AppendOptions{Flags: flags})
		require.NoError(t, appendErr)
	}

	return fi
}

func (fi *memIMAPServer) host() string {
	h, _, _ := net.SplitHostPort(fi.ln.Addr().String())
	return h
}

func (fi *memIMAPServer) port() int {
	_, p, _ := net.SplitHostPort(fi.ln.Addr().String())
	n, _ := strconv.Atoi(p)
	return n
}

func (fi *memIMAPServer) imapConfig() kdepsconfig.IMAPConnectionConfig {
	return kdepsconfig.IMAPConnectionConfig{
		Host:     fi.host(),
		Port:     fi.port(),
		Username: "testuser",
		Password: "testpass",
	}
}

func buildRawMessage(msg fakeIMAPMsg) []byte {
	from := msg.from
	if from == "" {
		from = "sender@example.com"
	}
	to := msg.to
	if to == "" {
		to = "recipient@example.com"
	}
	subject := msg.subject
	if subject == "" {
		subject = "(no subject)"
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "Date: Mon, 01 Jan 2024 12:00:00 +0000\r\n")
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprintf(&buf, "\r\n")
	fmt.Fprintf(&buf, "%s", msg.body)
	return buf.Bytes()
}
