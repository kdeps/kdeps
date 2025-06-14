package resolver

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklData "github.com/kdeps/schema/gen/data"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestAppendDataEntry_ContextNil(t *testing.T) {
	dr := &DependencyResolver{
		Fs:        afero.NewMemMapFs(),
		Logger:    logging.NewTestLogger(),
		ActionDir: "/tmp",
		RequestID: "req",
		// Context is nil
	}
	err := dr.AppendDataEntry("id", &pklData.DataImpl{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "context is nil")
}
