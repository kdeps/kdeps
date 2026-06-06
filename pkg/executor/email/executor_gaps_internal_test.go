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

package email

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestApplySMTPDeadline_NoTimeout(t *testing.T) {
	server, client := net.Pipe()
	t.Cleanup(func() {
		_ = server.Close()
		_ = client.Close()
	})
	require.NoError(t, applySMTPDeadline(client, 0))
}

func TestSendSTARTTLS_SetDeadlineError(t *testing.T) {
	orig := connSetDeadline
	t.Cleanup(func() { connSetDeadline = orig })
	connSetDeadline = func(_ net.Conn, _ time.Time) error {
		return errors.New("set deadline failed")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)

	err = sendSTARTTLS(
		"127.0.0.1:"+portStr, "localhost", "", "",
		"from@x.com", []string{"to@x.com"}, []byte("msg"),
		false, time.Second,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "set deadline")
}

func TestSendImplicitTLS_SetDeadlineErrorAfterDial(t *testing.T) {
	origDeadline := connSetDeadline
	t.Cleanup(func() { connSetDeadline = origDeadline })
	connSetDeadline = func(_ net.Conn, _ time.Time) error {
		return errors.New("set deadline failed")
	}

	origDial := implicitTLSDial
	t.Cleanup(func() { implicitTLSDial = origDial })
	server, client := net.Pipe()
	t.Cleanup(func() { _ = server.Close() })
	implicitTLSDial = func(_ string, _ *tls.Config) (net.Conn, error) {
		return client, nil
	}

	err := sendImplicitTLS(
		"127.0.0.1:0", "localhost", "", "",
		"from@x.com", []string{"to@x.com"}, []byte("msg"),
		true, time.Second,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "set deadline")
}

func TestOpenIMAPClient_DialTLSSuccess(t *testing.T) {
	orig := imapDialTLS
	t.Cleanup(func() { imapDialTLS = orig })
	imapDialTLS = func(_ string, _ *imapclient.Options) (*imapclient.Client, error) {
		return &imapclient.Client{}, nil
	}

	c, err := openIMAPClient(&imapDialParams{
		addr: "imap.example.com:993", host: "imap.example.com", useTLS: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestDoSend_WriteBodyError(t *testing.T) {
	orig := smtpDataWrite
	t.Cleanup(func() { smtpDataWrite = orig })
	smtpDataWrite = func(_ io.Writer, _ []byte) (int, error) {
		return 0, errors.New("write body failed")
	}

	conn, ready, done := withSMTPServer(func(srvConn net.Conn, br *bufio.Reader) {
		_, _ = br.ReadString('\n')
		_, _ = srvConn.Write([]byte("250 OK\r\n"))
		_, _ = br.ReadString('\n')
		_, _ = srvConn.Write([]byte("250 OK\r\n"))
		_, _ = br.ReadString('\n')
		_, _ = srvConn.Write([]byte("250 OK\r\n"))
		_, _ = br.ReadString('\n')
		_, _ = srvConn.Write([]byte("354 Go ahead\r\n"))
	})
	defer conn.Close()
	<-ready

	client, err := smtp.NewClient(conn, "localhost")
	require.NoError(t, err)

	err = doSend(client, "from@x.com", []string{"to@x.com"}, []byte("body"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write body")
	_ = client.Close()
	<-done
}

func TestWriteMultipartBody_CreatePartError(t *testing.T) {
	orig := multipartCreatePart
	t.Cleanup(func() { multipartCreatePart = orig })
	multipartCreatePart = func(_ *multipart.Writer, _ textproto.MIMEHeader) (io.Writer, error) {
		return nil, errors.New("create part failed")
	}

	var buf bytes.Buffer
	err := writeMultipartBody(&buf, "body", false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create body part")
}

func TestWriteMultipartBody_WriteBodyPartError(t *testing.T) {
	orig := multipartCreatePart
	t.Cleanup(func() { multipartCreatePart = orig })
	multipartCreatePart = func(_ *multipart.Writer, _ textproto.MIMEHeader) (io.Writer, error) {
		return &failWriter{}, nil
	}

	var buf bytes.Buffer
	err := writeMultipartBody(&buf, "body", false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write body part")
}

func TestWriteAttachmentPart_CreatePartError(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	AppFS = afero.NewMemMapFs()
	_ = afero.WriteFile(AppFS, "/att.txt", []byte("data"), 0o644)

	orig := multipartCreatePart
	t.Cleanup(func() { multipartCreatePart = orig })
	multipartCreatePart = func(_ *multipart.Writer, _ textproto.MIMEHeader) (io.Writer, error) {
		return nil, errors.New("create part failed")
	}

	mw := multipart.NewWriter(&bytes.Buffer{})
	err := writeAttachmentPart(mw, "/att.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create attachment part")
}

func TestWriteAttachmentPart_EncodeError(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	AppFS = afero.NewMemMapFs()
	_ = afero.WriteFile(AppFS, "/att.txt", []byte("data"), 0o644)

	orig := multipartCreatePart
	t.Cleanup(func() { multipartCreatePart = orig })
	multipartCreatePart = func(_ *multipart.Writer, _ textproto.MIMEHeader) (io.Writer, error) {
		return &failWriter{}, nil
	}

	mw := multipart.NewWriter(&bytes.Buffer{})
	err := writeAttachmentPart(mw, "/att.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encode attachment")
}

func TestWriteMultipartBody_CloseError(t *testing.T) {
	orig := multipartWriterClose
	t.Cleanup(func() { multipartWriterClose = orig })
	multipartWriterClose = func(_ *multipart.Writer) error {
		return errors.New("close failed")
	}

	var buf bytes.Buffer
	err := writeMultipartBody(&buf, "body", false, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close multipart writer")
}

func TestApplyModifyOperations_ExpungeError(t *testing.T) {
	orig := imapExpungeClose
	t.Cleanup(func() { imapExpungeClose = orig })
	imapExpungeClose = func(_ *imapclient.Client) error {
		return errors.New("expunge failed")
	}

	err := applyModifyOperations(&imapclient.Client{}, domain.EmailModifyConfig{Expunge: true}, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expunge")
}

type failWriter struct{}

func (failWriter) Write(_ []byte) (int, error) { return 0, errors.New("write failed") }
