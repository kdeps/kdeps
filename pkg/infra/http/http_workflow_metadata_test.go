// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowMetadataHelpers_NilWorkflow(t *testing.T) {
	assert.Empty(t, workflowMetadataName(nil))
	assert.Empty(t, workflowMetadataVersion(nil))
	assert.Nil(t, workflowNameVersionMap(nil))
}
