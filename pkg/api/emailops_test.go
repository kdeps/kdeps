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

package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

// stubEmailConfig replaces the config.yaml read for one test.
func stubEmailConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	previous := emailConfig
	emailConfig = func() (*config.Config, error) { return cfg, nil }
	t.Cleanup(func() { emailConfig = previous })
}

// emailTestSMTP accepts plain SMTP and records every message's envelope and data.
func emailTestSMTP(t *testing.T) (string, int, func() (string, []string, string)) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	var mu sync.Mutex
	var from, data string
	var recipients []string
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				_, _ = fmt.Fprint(c, "220 test ESMTP\r\n")
				var buf strings.Builder
				inData := false
				for {
					line, readErr := reader.ReadString('\n')
					if readErr != nil {
						return
					}
					line = strings.TrimRight(line, "\r\n")
					upper := strings.ToUpper(line)
					switch {
					case inData && line == ".":
						inData = false
						mu.Lock()
						data = buf.String()
						mu.Unlock()
						_, _ = fmt.Fprint(c, "250 OK\r\n")
					case inData:
						buf.WriteString(strings.TrimPrefix(line, ".") + "\n")
					case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
						_, _ = fmt.Fprint(c, "250 test\r\n")
					case strings.HasPrefix(upper, "MAIL FROM:"):
						mu.Lock()
						from, recipients = strings.Trim(line[10:], "<> "), nil
						mu.Unlock()
						_, _ = fmt.Fprint(c, "250 OK\r\n")
					case strings.HasPrefix(upper, "RCPT TO:"):
						mu.Lock()
						recipients = append(recipients, strings.Trim(line[8:], "<> "))
						mu.Unlock()
						_, _ = fmt.Fprint(c, "250 OK\r\n")
					case upper == "DATA":
						inData = true
						_, _ = fmt.Fprint(c, "354 go\r\n")
					case strings.HasPrefix(upper, "QUIT"):
						_, _ = fmt.Fprint(c, "221 bye\r\n")
						return
					default:
						_, _ = fmt.Fprint(c, "250 OK\r\n")
					}
				}
			}(conn)
		}
	}()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	return host, port, func() (string, []string, string) {
		mu.Lock()
		defer mu.Unlock()
		return from, recipients, data
	}
}

// emailTestIMAP serves an in-memory IMAP4rev1 mailbox holding messages.
func emailTestIMAP(t *testing.T, messages ...string) (string, int) {
	t.Helper()
	memory := imapmemserver.New()
	user := imapmemserver.NewUser("u", "p")
	require.NoError(t, user.Create("INBOX", nil))
	require.NoError(t, user.Create("Archive", nil))
	memory.AddUser(user)
	server := imapserver.New(&imapserver.Options{
		NewSession: func(_ *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memory.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}}, //nolint:exhaustive // IMAP4rev1 only
		InsecureAuth: true,
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close(); _ = listener.Close() })
	go server.Serve(listener) //nolint:errcheck // stops on Close
	for _, message := range messages {
		_, appendErr := user.Append("INBOX", bytes.NewReader([]byte(message)), &imap.AppendOptions{})
		require.NoError(t, appendErr)
	}
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	return host, port
}

func emailTestRaw(subject, body, extra string) string {
	return "From: Ann <ann@example.com>\r\nTo: bob@example.com\r\nSubject: " + subject +
		"\r\nDate: Mon, 01 Jan 2024 12:00:00 +0000\r\n" + extra + "\r\n" + body
}

func emailCall(t *testing.T, args map[string]any) (map[string]any, error) {
	t.Helper()
	out, err := EmailOps(t.Context(), args)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	return result, nil
}

func TestEmailOps_Send(t *testing.T) {
	host, port, received := emailTestSMTP(t)
	stubEmailConfig(t, &config.Config{
		Identity: &config.IdentityConfig{Email: "me@example.com"},
		SMTPConnections: map[string]config.SMTPConnectionConfig{
			"out": {Host: host, Port: port},
		},
	})
	attachment := filepath.Join(t.TempDir(), "note.txt")
	require.NoError(t, os.WriteFile(attachment, []byte("attached"), 0o600))

	result, err := emailCall(t, map[string]any{
		"action":         "send",
		"smtpConnection": "out",
		"to":             []any{"bob@example.com"},
		"bcc":            "hidden@example.com",
		"subject":        "Report",
		"body":           "All good",
		"attachments":    []any{attachment},
	})
	require.NoError(t, err)
	assert.Equal(t, "send", result["action"])
	assert.Equal(t, "me@example.com", result["from"])
	assert.EqualValues(t, 1, result["attachments"])

	from, recipients, data := received()
	assert.Equal(t, "me@example.com", from)
	assert.Equal(t, []string{"bob@example.com", "hidden@example.com"}, recipients)
	assert.Contains(t, data, "Subject: Report")
	assert.Contains(t, data, "All good")
	assert.NotContains(t, data, "hidden@example.com", "bcc stays out of the headers")
	assert.Contains(t, data, "filename=note.txt")
}

func TestEmailOps_SendHTMLEscapes(t *testing.T) {
	host, port, received := emailTestSMTP(t)
	stubEmailConfig(t, &config.Config{SMTPConnections: map[string]config.SMTPConnectionConfig{
		"out": {Host: host, Port: port},
	}})
	_, err := emailCall(t, map[string]any{
		"smtpConnection": "out", "from": "bot@example.com", "to": "one@example.com, two@example.com",
		"subject": "Hi", "body": "<b>bold</b>", "html": true,
	})
	require.NoError(t, err)
	_, recipients, data := received()
	assert.Equal(t, []string{"one@example.com", "two@example.com"}, recipients)
	assert.Contains(t, data, "Content-Type: text/html; charset=utf-8")
	assert.Contains(t, data, "<b>bold</b>")
}

func TestEmailOps_SendErrors(t *testing.T) {
	stubEmailConfig(t, &config.Config{
		SMTPConnections: map[string]config.SMTPConnectionConfig{
			"out": {Host: "127.0.0.1", Port: 1},
		},
	})
	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{"no connection name", map[string]any{"to": []string{"b@x"}, "subject": "s"},
			"smtpConnection is required for send"},
		{"unknown connection", map[string]any{"smtpConnection": "nope", "to": []string{"b@x"}, "subject": "s"},
			"smtpConnection 'nope' not found"},
		{"no from", map[string]any{"smtpConnection": "out", "to": []string{"b@x"}, "subject": "s"},
			"from is required for send"},
		{"no recipients", map[string]any{"smtpConnection": "out", "from": "a@x", "subject": "s"},
			"at least one recipient"},
		{"no subject", map[string]any{"smtpConnection": "out", "from": "a@x", "to": []string{"b@x"}},
			"subject is required for send"},
		{"header injection", map[string]any{
			"smtpConnection": "out", "from": "a@x", "to": []string{"b@x"}, "subject": "a\r\nBcc: evil@x",
		}, "header injection"},
		{"unknown action", map[string]any{"action": "fly"}, "unknown action 'fly'"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := emailCall(t, testCase.args)
			require.ErrorContains(t, err, testCase.want)
		})
	}
}

func TestEmailOps_NilContext(t *testing.T) {
	var nilCtx context.Context
	_, err := EmailOps(nilCtx, nil)
	require.EqualError(t, err, "email_ops: nil context")
}

func TestEmailOps_ReadSearch(t *testing.T) {
	host, port := emailTestIMAP(t,
		emailTestRaw("Invoice 1", "pay me", "List-Unsubscribe: <https://u.example/x>, <mailto:u@example.com>\r\n"+
			"List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n"),
		emailTestRaw("Hello", "hi there", ""),
		emailTestRaw("Invoice 2", "pay again", ""),
	)
	stubEmailConfig(t, &config.Config{IMAPConnections: map[string]config.IMAPConnectionConfig{
		"in": {Host: host, Port: port, Username: "u", Password: "p"},
	}})

	result, err := emailCall(t, map[string]any{"action": "read", "imapConnection": "in", "limit": 2})
	require.NoError(t, err)
	assert.EqualValues(t, 2, result["count"])
	messages, ok := result["messages"].([]any)
	require.True(t, ok)
	last := messages[1].(map[string]any)
	assert.Equal(t, "Invoice 2", last["subject"])
	assert.Equal(t, "pay again", last["body"])
	assert.Equal(t, "Ann <ann@example.com>", last["from"])
	assert.Equal(t, "2024-01-01T12:00:00Z", last["date"])
	assert.Equal(t, false, last["seen"])

	found, err := emailCall(t, map[string]any{
		"action": "search", "imapConnection": "in", "search": map[string]any{"subject": "Invoice"},
		"markRead": true,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, found["count"])
	first := found["messages"].([]any)[0].(map[string]any)
	assert.Equal(t, "https://u.example/x", first["unsubscribeUrl"])
	assert.Equal(t, "mailto:u@example.com", first["unsubscribeMailto"])
	assert.Equal(t, true, first["unsubscribeOneClick"])

	unseen, err := emailCall(t, map[string]any{
		"action": "search", "imapConnection": "in", "search": map[string]any{"unseen": true},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, unseen["count"], "markRead marked both invoices seen")
}

func TestEmailOps_ListDeleteAndModify(t *testing.T) {
	host, port := emailTestIMAP(t,
		emailTestRaw("Hello", "hi there", ""),
		emailTestRaw("Invoice 2", "pay again", ""),
	)
	stubEmailConfig(t, &config.Config{IMAPConnections: map[string]config.IMAPConnectionConfig{
		"in": {Host: host, Port: port, Username: "u", Password: "p"},
	}})

	list, err := emailCall(t, map[string]any{"action": "list", "imapConnection": "in"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []any{"INBOX", "Archive"}, list["folders"])

	modified, err := emailCall(t, map[string]any{
		"action": "modify", "imapConnection": "in",
		"search": map[string]any{"subject": "Hello"}, "modify": map[string]any{"moveTo": "archive"},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, modified["count"])

	archive, err := emailCall(t, map[string]any{"action": "read", "imapConnection": "in", "mailbox": "Archive"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, archive["count"], "moveTo matches Archive case-insensitively")

	deleted, err := emailCall(t, map[string]any{
		"action": "delete", "imapConnection": "in", "mailbox": "Archive",
	})
	require.NoError(t, err)
	assert.Equal(t, true, deleted["success"])

	_, err = emailCall(t, map[string]any{"action": "delete", "imapConnection": "in", "mailbox": "INBOX"})
	require.ErrorContains(t, err, "protected mailbox")

	_, err = emailCall(t, map[string]any{
		"action": "modify", "imapConnection": "in", "uids": []any{"not-a-uid"},
	})
	require.ErrorContains(t, err, "no valid UIDs resolved")
}

func TestEmailOps_IMAPErrors(t *testing.T) {
	stubEmailConfig(t, &config.Config{IMAPConnections: map[string]config.IMAPConnectionConfig{
		"in": {Host: "127.0.0.1", Port: 1},
	}})
	_, err := emailCall(t, map[string]any{"action": "read"})
	require.ErrorContains(t, err, "imapConnection is required")
	_, err = emailCall(t, map[string]any{"action": "read", "imapConnection": "nope"})
	require.ErrorContains(t, err, "imapConnection 'nope' not found")
	_, err = emailCall(t, map[string]any{"action": "read", "imapConnection": "in", "timeout": "1ms"})
	require.ErrorContains(t, err, "imap connect")
}

func TestEmailOps_ReadNonexistentMailbox(t *testing.T) {
	host, port := emailTestIMAP(t, emailTestRaw("Hello", "hi", ""))
	stubEmailConfig(t, &config.Config{IMAPConnections: map[string]config.IMAPConnectionConfig{
		"in": {Host: host, Port: port, Username: "u", Password: "p"},
	}})
	_, err := emailCall(t, map[string]any{"action": "read", "imapConnection": "in", "mailbox": "Missing"})
	require.ErrorContains(t, err, "select 'Missing'")
}

func TestEmailOps_Helpers(t *testing.T) {
	assert.Equal(t, 7*time.Second, emailWait("7s"))
	assert.Equal(t, 5*time.Second, emailWait("5"))
	assert.Equal(t, emailDefaultWait, emailWait("nonsense"))
	assert.Equal(t, emailDefaultWait, emailWait("-1s"))
	assert.Equal(t, 12, emailLimit(float64(12)))
	assert.Equal(t, emailDefaultLimit, emailLimit(-1))
	assert.Equal(t, []string{"a", "b"}, emailValues("a, b"))
	assert.Equal(t, []string{"a", "b"}, emailValues([]any{"a", "b", " "}))
	assert.True(t, emailBool("TRUE"))
	assert.False(t, emailBool("no"))
	parsed, ok := emailDate("2024-03-15")
	require.True(t, ok)
	assert.Equal(t, 2024, parsed.Year())
	_, ok = emailDate("someday")
	assert.False(t, ok)
}
