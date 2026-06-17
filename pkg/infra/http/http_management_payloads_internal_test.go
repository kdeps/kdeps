package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestManagementStatusOK(t *testing.T) {
	t.Parallel()
	m := managementStatusOK()
	assert.Equal(t, statusOKValue, m[jsonFieldStatus])
}

func TestManagementErrorPayload(t *testing.T) {
	t.Parallel()
	m := managementErrorPayload("something failed")
	assert.Equal(t, statusErrorValue, m[jsonFieldStatus])
	assert.Equal(t, "something failed", m[jsonFieldMessage])
}

func TestWorkflowStatusDetailMap_Nil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, workflowStatusDetailMap(nil))
}

func TestWorkflowStatusDetailMap_WithWorkflow(t *testing.T) {
	t.Parallel()
	wf := &domain.Workflow{}
	wf.Metadata.Name = "my-wf"
	wf.Metadata.Version = "2.0.0"
	m := workflowStatusDetailMap(wf)
	assert.Equal(t, "my-wf", m[jsonFieldName])
	assert.Equal(t, "2.0.0", m[jsonFieldVersion])
	assert.Equal(t, 0, m[jsonFieldResources])
}

func TestManagementOKStatus_Nil(t *testing.T) {
	t.Parallel()
	m := managementOKStatus(nil)
	assert.Equal(t, statusOKValue, m[jsonFieldStatus])
	_, hasWorkflow := m[jsonFieldWorkflow]
	assert.False(t, hasWorkflow)
}

func TestManagementSuccessPayload(t *testing.T) {
	t.Parallel()
	m := managementSuccessPayload("updated", nil)
	assert.Equal(t, "updated", m[jsonFieldMessage])
	assert.Equal(t, statusOKValue, m[jsonFieldStatus])
}

func TestHealthCheckPayload(t *testing.T) {
	t.Parallel()
	m := healthCheckPayload(nil)
	assert.Equal(t, statusOKValue, m[jsonFieldStatus])
}

func TestWriteWorkflowStatusJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writeWorkflowStatusJSON(w, nil, managementOKStatus)
	assert.Equal(t, stdhttp.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}
