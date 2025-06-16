package docker

import (
	"net/http"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestValidateMethodMore(t *testing.T) {

	// allowed only GET & POST
	allowed := []string{http.MethodGet, http.MethodPost}

	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	out, err := validateMethod(req, allowed)
	assert.NoError(t, err)
	assert.Equal(t, `method = "POST"`, out)

	// default empty method becomes GET and passes
	req2, _ := http.NewRequest("", "/", nil)
	out, err = validateMethod(req2, allowed)
	assert.NoError(t, err)
	assert.Equal(t, `method = "GET"`, out)

	// invalid method
	req3, _ := http.NewRequest(http.MethodPut, "/", nil)
	out, err = validateMethod(req3, allowed)
	assert.Error(t, err)
	assert.Empty(t, out)
}

func TestCleanOldFilesMore(t *testing.T) {

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// create dummy response file
	const respPath = "/tmp/response.json"
	_ = afero.WriteFile(fs, respPath, []byte("old"), 0644)

	dr := &resolver.DependencyResolver{
		Fs:                 fs,
		ResponseTargetFile: respPath,
		Logger:             logger,
	}

	// should remove existing file
	err := cleanOldFiles(dr)
	assert.NoError(t, err)
	exist, _ := afero.Exists(fs, respPath)
	assert.False(t, exist)

	// second call with file absent should still succeed
	err = cleanOldFiles(dr)
	assert.NoError(t, err)
}
