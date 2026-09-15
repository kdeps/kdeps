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

func TestExecuteTranscribe_NilConfig(t *testing.T) {
	eng := newTestEngineInternal()
	res := &domain.Resource{ActionID: "test", Transcribe: nil}
	_, err := eng.executeTranscribe(res, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcribe")
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

func TestExecuteTranscribe_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	cfg := &domain.TranscribeConfig{}
	res := &domain.Resource{ActionID: "test", Transcribe: cfg}
	_, err := eng.executeTranscribe(res, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcribe executor not available")
}

func TestExecuteInlineFile_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	_, err := eng.executeInlineFile(&domain.FileResourceConfig{}, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file executor not available")
}

func TestExecuteInlineGit_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	_, err := eng.executeInlineGit(&domain.GitResourceConfig{}, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git executor not available")
}

func TestExecuteInlineCodeIntelligence_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	_, err := eng.executeInlineCodeIntelligence(&domain.CodeIntelligenceConfig{}, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "codeIntelligence executor not available")
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

func TestExecuteInlineTranscribe_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	_, err := eng.executeInlineTranscribe(&domain.TranscribeConfig{}, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcribe executor not available")
}

func TestExecuteOCR_NilConfig(t *testing.T) {
	eng := newTestEngineInternal()
	res := &domain.Resource{ActionID: "test", OCR: nil}
	_, err := eng.executeOCR(res, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ocr")
}

func TestExecuteOCR_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	cfg := &domain.OCRConfig{}
	res := &domain.Resource{ActionID: "test", OCR: cfg}
	_, err := eng.executeOCR(res, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ocr executor not available")
}

func TestExecuteInlineOCR_NoExecutor(t *testing.T) {
	eng := newTestEngineInternal()
	_, err := eng.executeInlineOCR(&domain.OCRConfig{}, &ExecutionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ocr executor not available")
}

// fakeOCRExecutor is a minimal ResourceExecutor that echoes the file path it
// was asked to OCR, so the dispatch tests below can assert the real config
// reached the registered executor without depending on tesseract.
type fakeOCRExecutor struct{ lastFile string }

func (f *fakeOCRExecutor) Execute(_ *ExecutionContext, config any) (any, error) {
	cfg, ok := config.(*domain.OCRConfig)
	if !ok {
		return nil, assert.AnError
	}
	f.lastFile = cfg.File
	return map[string]any{"text": "OCR:" + cfg.File}, nil
}

func TestExecuteOCR_Dispatches(t *testing.T) {
	eng := newTestEngineInternal()
	fake := &fakeOCRExecutor{}
	eng.registry.SetOCRExecutor(fake)

	cfg := &domain.OCRConfig{File: "/tmp/scan.png"}
	res := &domain.Resource{ActionID: "test", OCR: cfg}
	out, err := eng.executeOCR(res, &ExecutionContext{})
	require.NoError(t, err)
	assert.Equal(t, "/tmp/scan.png", fake.lastFile)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "OCR:/tmp/scan.png", m["text"])
}

func TestExecuteInlineOCR_Dispatches(t *testing.T) {
	eng := newTestEngineInternal()
	fake := &fakeOCRExecutor{}
	eng.registry.SetOCRExecutor(fake)

	out, err := eng.executeInlineOCR(&domain.OCRConfig{File: "/tmp/inline.png"}, &ExecutionContext{})
	require.NoError(t, err)
	assert.Equal(t, "/tmp/inline.png", fake.lastFile)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "OCR:/tmp/inline.png", m["text"])
}
