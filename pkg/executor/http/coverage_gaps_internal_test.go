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

package http

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

func TestBuildTLSTransport_WithClientCert(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, os.WriteFile(certFile, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0o600))

	transport, err := buildTLSTransport(&domain.HTTPTLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
	})
	require.NoError(t, err)
	assert.Len(t, transport.TLSClientConfig.Certificates, 1)
}

func TestEvaluateData_MapFieldError(t *testing.T) {
	e := NewExecutor()
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	eval := expression.NewEvaluator(ctx.API)

	_, err = e.evaluateData(eval, ctx, map[string]interface{}{
		"bad": "{{{",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to evaluate data field bad")
}

func TestPrepareRequest_DefaultMethod(t *testing.T) {
	e := NewExecutor()
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	eval := expression.NewEvaluator(ctx.API)

	_, method, _, err := e.prepareRequest(eval, ctx, &domain.HTTPClientConfig{URL: "http://example.com"}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, method)
}

func TestExecuteRequestWithRetry_PostLoopError(t *testing.T) {
	orig := forceRetryLoopExit
	t.Cleanup(func() { forceRetryLoopExit = orig })
	forceRetryLoopExit = true

	e := NewExecutor()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	_, err = e.executeRequestWithRetry(http.DefaultClient, req, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after all retries")
}

func TestReadLimitedResponseBody_ReadError(t *testing.T) {
	resp := &http.Response{Body: io.NopCloser(&failReader{})}
	_, err := readLimitedResponseBody(resp, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read response")
}

type failReader struct{}

func (failReader) Read(_ []byte) (int, error) { return 0, errors.New("read failed") }
