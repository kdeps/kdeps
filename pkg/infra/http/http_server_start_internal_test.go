package http

import (
	stdhttp "net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestWorkflowTLSCertificates_Nil(t *testing.T) {
	t.Parallel()
	cert, key := workflowTLSCertificates(nil)
	assert.Empty(t, cert)
	assert.Empty(t, key)
}

func TestWorkflowTLSCertificates_WithValues(t *testing.T) {
	t.Parallel()
	wf := &domain.Workflow{}
	wf.Settings.CertFile = "/etc/certs/cert.pem"
	wf.Settings.KeyFile = "/etc/certs/key.pem"
	cert, key := workflowTLSCertificates(wf)
	assert.Equal(t, "/etc/certs/cert.pem", cert)
	assert.Equal(t, "/etc/certs/key.pem", key)
}

func TestNewDefaultHTTPServer(t *testing.T) {
	t.Parallel()
	noop := stdhttp.HandlerFunc(func(stdhttp.ResponseWriter, *stdhttp.Request) {})
	srv := newDefaultHTTPServer(":9999", noop)
	assert.Equal(t, ":9999", srv.Addr)
	assert.Equal(t, DefaultHTTPReadTimeout, srv.ReadTimeout)
	assert.Equal(t, DefaultHTTPWriteTimeout, srv.WriteTimeout)
	assert.True(t, srv.IdleTimeout > 0*time.Second)
}
