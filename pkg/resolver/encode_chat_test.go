package resolver_test

import (
	"context"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

// TestEncodeChat_AllFields removed - function no longer exists

// TestEncodeJSONResponseKeys_Nil removed - function no longer exists
// TestEncodeExecHelpers removed - functions no longer exist

func newMemResolver() *resolver.DependencyResolver {
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/files", 0o755)
	return &resolver.DependencyResolver{
		Fs:        fs,
		FilesDir:  "/files",
		ActionDir: "/action",
		RequestID: "req1",
		Context:   context.Background(),
		Logger:    logging.NewTestLogger(),
	}
}

// TestEncodeResponseHeadersAndBody removed - functions no longer exist
