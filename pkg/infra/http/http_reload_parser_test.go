// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/validator"
)

func TestNewWorkflowParser_SchemaValidatorError(t *testing.T) {
	orig := schemaValidatorFactory
	t.Cleanup(func() { schemaValidatorFactory = orig })
	schemaValidatorFactory = func() (*validator.SchemaValidator, error) {
		return nil, errors.New("schema validator failed")
	}

	_, err := newWorkflowParser()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create schema validator")
}
