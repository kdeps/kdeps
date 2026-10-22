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

package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestExecuteLoader_NilConfig(t *testing.T) {
	eng := newTestEngineInternal()
	res := &domain.Resource{ActionID: "test", Loader: nil}
	_, err := eng.executeLoader(res, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loader")
}

func TestExecuteVectorStore_NilConfig(t *testing.T) {
	eng := newTestEngineInternal()
	res := &domain.Resource{ActionID: "test", VectorStore: nil}
	_, err := eng.executeVectorStore(res, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vectorStore")
}

func TestExecuteLoader_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	cfg := &domain.LoaderConfig{}
	res := &domain.Resource{ActionID: "test", Loader: cfg}
	_, err := eng.executeLoader(res, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loader executor not available")
}

func TestExecuteVectorStore_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	cfg := &domain.VectorStoreConfig{}
	res := &domain.Resource{ActionID: "test", VectorStore: cfg}
	_, err := eng.executeVectorStore(res, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vectorStore executor not available")
}

func TestExecuteInlineLoader_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	_, err := eng.executeInlineLoader(&domain.LoaderConfig{}, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loader executor not available")
}

func TestExecuteInlineVectorStore_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	_, err := eng.executeInlineVectorStore(&domain.VectorStoreConfig{}, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vectorStore executor not available")
}
