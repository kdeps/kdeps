package resolver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alexellis/go-execute/v2"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// Test that activate/deactivate use the injected ExecTaskRunnerFn and succeed.
func TestCondaEnvironmentExecutionInjectedSuccess(t *testing.T) {
	var activateCalled, deactivateCalled bool

	dr := &resolver.DependencyResolver{
		Fs:      afero.NewMemMapFs(),
		Logger:  logging.GetLogger(),
		Context: context.Background(),
		ExecTaskRunnerFn: func(ctx context.Context, task execute.ExecTask) (string, string, error) {
			if task.Command == "conda" && len(task.Args) >= 1 {
				switch task.Args[0] {
				case "activate":
					activateCalled = true
				case "deactivate":
					deactivateCalled = true
				}
			}
			return "", "", nil
		},
	}

	assert.NoError(t, dr.ActivateCondaEnvironment("myenv"))
	assert.NoError(t, dr.DeactivateCondaEnvironment())
	assert.True(t, activateCalled, "activate runner was not called")
	assert.True(t, deactivateCalled, "deactivate runner was not called")
}

// Test that errors from injected runner are propagated.
func TestCondaEnvironmentExecutionInjectedFailure(t *testing.T) {
	expectedErr := errors.New("conda failure")
	dr := &resolver.DependencyResolver{
		Fs:      afero.NewMemMapFs(),
		Logger:  logging.GetLogger(),
		Context: context.Background(),
		ExecTaskRunnerFn: func(ctx context.Context, task execute.ExecTask) (string, string, error) {
			return "", "", expectedErr
		},
	}

	err := dr.ActivateCondaEnvironment("myenv")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expectedErr.Error())
}

// Test that handleFileImports uses injected import helpers.
func TestHandleFileImportsUsesInjection(t *testing.T) {
	var prependCalled, placeholderCalled bool

	dr := &resolver.DependencyResolver{
		Fs:     afero.NewMemMapFs(),
		Logger: logging.GetLogger(),
		PrependDynamicImportsFn: func(path string) error {
			prependCalled = true
			return nil
		},
		AddPlaceholderImportsFn: func(path string) error {
			placeholderCalled = true
			return nil
		},
	}

	err := dr.HandleFileImports("dummy.pkl")
	assert.NoError(t, err)
	assert.True(t, prependCalled, "PrependDynamicImportsFn was not called")
	assert.True(t, placeholderCalled, "AddPlaceholderImportsFn was not called")
}
