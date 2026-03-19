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

package transcriber

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// ----- mock helpers -------------------------------------------------------

// queuedTransport serves responses from a pre-built slice in order.
type queuedTransport struct {
	responses []*http.Response
	idx       int
}

func (q *queuedTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	if q.idx < len(q.responses) {
		r := q.responses[q.idx]
		q.idx++
		return r, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

// alwaysErrTransport always returns a transport-level error.
type alwaysErrTransport struct{ err error }

func (e *alwaysErrTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, e.err
}

func jsonResp(v interface{}) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(http.Header),
	}
}

func rawResp(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func newTestOnlineTranscriber(provider, language, outputMode string, transport http.RoundTripper) *onlineTranscriber {
	return &onlineTranscriber{
		cfg: &domain.TranscriberConfig{
			Language: language,
			Online: &domain.OnlineTranscriberConfig{
				Provider: provider,
				APIKey:   "test-api-key",
			},
		},
		outputMode: outputMode,
		logger:     slog.Default(),
		client:     &http.Client{Transport: transport},
	}
}

func tempAudioFile(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(f, []byte("fake audio data"), 0o600))
	return f
}

// ----- openAIWhisper ------------------------------------------------------

func TestOpenAIWhisper_Success(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderOpenAIWhisper, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			jsonResp(openAIWhisperResponse{Text: "hello world"}),
		}},
	)
	r, err := tr.openAIWhisper(audio)
	require.NoError(t, err)
	assert.Equal(t, "hello world", r.Text)
}

func TestOpenAIWhisper_WithLanguage(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderOpenAIWhisper, "fr", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			jsonResp(openAIWhisperResponse{Text: "bonjour"}),
		}},
	)
	r, err := tr.openAIWhisper(audio)
	require.NoError(t, err)
	assert.Equal(t, "bonjour", r.Text)
}

func TestOpenAIWhisper_HTTPError(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderOpenAIWhisper, "", domain.TranscriberOutputText,
		&alwaysErrTransport{err: errors.New("connection refused")},
	)
	_, err := tr.openAIWhisper(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "openai-whisper: request")
}

func TestOpenAIWhisper_Non200Status(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderOpenAIWhisper, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusUnauthorized, `{"error":"invalid api key"}`),
		}},
	)
	_, err := tr.openAIWhisper(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 401")
}

func TestOpenAIWhisper_InvalidJSON(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderOpenAIWhisper, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusOK, `not valid json`),
		}},
	)
	_, err := tr.openAIWhisper(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestOpenAIWhisper_MediaOutputMode(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderOpenAIWhisper, "", domain.TranscriberOutputMedia,
		&queuedTransport{responses: []*http.Response{
			jsonResp(openAIWhisperResponse{Text: "transcript"}),
		}},
	)
	r, err := tr.openAIWhisper(audio)
	require.NoError(t, err)
	assert.Equal(t, "transcript", r.Text)
	assert.NotEmpty(t, r.MediaFile)
}

// ----- deepgram -----------------------------------------------------------

func TestDeepgram_Success(t *testing.T) {
	audio := tempAudioFile(t)
	resp := deepgramResponse{}
	resp.Results.Channels = []struct {
		Alternatives []struct {
			Transcript string `json:"transcript"`
		} `json:"alternatives"`
	}{
		{Alternatives: []struct {
			Transcript string `json:"transcript"`
		}{{Transcript: "deepgram transcript"}}},
	}
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderDeepgram, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{jsonResp(resp)}},
	)
	r, err := tr.deepgram(audio)
	require.NoError(t, err)
	assert.Equal(t, "deepgram transcript", r.Text)
}

func TestDeepgram_WithLanguage(t *testing.T) {
	audio := tempAudioFile(t)
	resp := deepgramResponse{}
	resp.Results.Channels = []struct {
		Alternatives []struct {
			Transcript string `json:"transcript"`
		} `json:"alternatives"`
	}{
		{Alternatives: []struct {
			Transcript string `json:"transcript"`
		}{{Transcript: "bonjour"}}},
	}
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderDeepgram, "fr", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{jsonResp(resp)}},
	)
	r, err := tr.deepgram(audio)
	require.NoError(t, err)
	assert.Equal(t, "bonjour", r.Text)
}

func TestDeepgram_EmptyChannels(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderDeepgram, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			jsonResp(deepgramResponse{}),
		}},
	)
	r, err := tr.deepgram(audio)
	require.NoError(t, err)
	assert.Empty(t, r.Text)
}

func TestDeepgram_Non200Status(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderDeepgram, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusForbidden, `{"error":"forbidden"}`),
		}},
	)
	_, err := tr.deepgram(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 403")
}

func TestDeepgram_HTTPError(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderDeepgram, "", domain.TranscriberOutputText,
		&alwaysErrTransport{err: errors.New("dial failed")},
	)
	_, err := tr.deepgram(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deepgram: request")
}

func TestDeepgram_InvalidJSON(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderDeepgram, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusOK, `{bad json}`),
		}},
	)
	_, err := tr.deepgram(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

// ----- assemblyAI (full flow) ---------------------------------------------

func TestAssemblyAI_SuccessFlow(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			// 1. upload
			jsonResp(assemblyAIUploadResponse{UploadURL: "https://cdn.assemblyai.com/audio/abc"}),
			// 2. submit transcript
			jsonResp(assemblyAITranscriptResponse{ID: "t1", Status: "completed", Text: "assembly result"}),
		}},
	)
	r, err := tr.assemblyAI(audio)
	require.NoError(t, err)
	assert.Equal(t, "assembly result", r.Text)
}

func TestAssemblyAI_UploadFails(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusInternalServerError, `{"error":"server error"}`),
		}},
	)
	_, err := tr.assemblyAI(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upload status 500")
}

func TestAssemblyAI_SubmitFails(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			// upload succeeds
			jsonResp(assemblyAIUploadResponse{UploadURL: "https://cdn.assemblyai.com/audio/abc"}),
			// submit fails
			rawResp(http.StatusBadRequest, `{"error":"bad request"}`),
		}},
	)
	_, err := tr.assemblyAI(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcript submit status 400")
}

// ----- assemblyAIUpload ---------------------------------------------------

func TestAssemblyAIUpload_Success(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			jsonResp(assemblyAIUploadResponse{UploadURL: "https://cdn.assemblyai.com/upload/xyz"}),
		}},
	)
	url, err := tr.assemblyAIUpload([]byte("audio bytes"))
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.assemblyai.com/upload/xyz", url)
}

func TestAssemblyAIUpload_HTTPError(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&alwaysErrTransport{err: errors.New("network failure")},
	)
	_, err := tr.assemblyAIUpload([]byte("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assemblyai: upload")
}

func TestAssemblyAIUpload_Non200(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusPaymentRequired, `{"error":"payment required"}`),
		}},
	)
	_, err := tr.assemblyAIUpload([]byte("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upload status 402")
}

func TestAssemblyAIUpload_InvalidJSON(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusOK, `not json at all`),
		}},
	)
	_, err := tr.assemblyAIUpload([]byte("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode upload response")
}

// ----- assemblyAISubmit ---------------------------------------------------

func TestAssemblyAISubmit_Success(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "en", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			jsonResp(assemblyAITranscriptResponse{ID: "job1", Status: "processing"}),
		}},
	)
	resp, err := tr.assemblyAISubmit("https://cdn.assemblyai.com/audio/abc")
	require.NoError(t, err)
	assert.Equal(t, "job1", resp.ID)
	assert.Equal(t, "processing", resp.Status)
}

func TestAssemblyAISubmit_HTTPError(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&alwaysErrTransport{err: errors.New("connection refused")},
	)
	_, err := tr.assemblyAISubmit("https://cdn.assemblyai.com/audio/abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assemblyai: transcript submit")
}

func TestAssemblyAISubmit_Non200(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusUnprocessableEntity, `{"error":"unprocessable"}`),
		}},
	)
	_, err := tr.assemblyAISubmit("https://cdn.assemblyai.com/audio/abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcript submit status 422")
}

func TestAssemblyAISubmit_InvalidJSON(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusOK, `{not valid json}`),
		}},
	)
	_, err := tr.assemblyAISubmit("https://cdn.assemblyai.com/audio/abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode transcript response")
}

// ----- assemblyAIPoll -----------------------------------------------------

func TestAssemblyAIPoll_AlreadyCompleted(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{},
	)
	initial := &assemblyAITranscriptResponse{ID: "j1", Status: "completed", Text: "done text"}
	resp, err := tr.assemblyAIPoll(initial)
	require.NoError(t, err)
	assert.Equal(t, "done text", resp.Text)
}

func TestAssemblyAIPoll_ErrorStatus(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{},
	)
	initial := &assemblyAITranscriptResponse{ID: "j1", Status: "error", Error: "audio too short"}
	_, err := tr.assemblyAIPoll(initial)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcription failed")
	assert.Contains(t, err.Error(), "audio too short")
}

func TestAssemblyAIPoll_HTTPError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping poll loop test in short mode (requires 3 s sleep)")
	}
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&alwaysErrTransport{err: errors.New("poll network error")},
	)
	initial := &assemblyAITranscriptResponse{ID: "j1", Status: "processing"}
	_, err := tr.assemblyAIPoll(initial)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assemblyai: poll")
}

func TestAssemblyAIPoll_PollUntilComplete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping poll loop test in short mode (requires 3 s sleep)")
	}
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			jsonResp(assemblyAITranscriptResponse{ID: "j1", Status: "completed", Text: "polled text"}),
		}},
	)
	initial := &assemblyAITranscriptResponse{ID: "j1", Status: "processing"}
	resp, err := tr.assemblyAIPoll(initial)
	require.NoError(t, err)
	assert.Equal(t, "polled text", resp.Text)
}

func TestAssemblyAIPoll_PollInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping poll loop test in short mode (requires 3 s sleep)")
	}
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderAssemblyAI, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusOK, `{not valid}`),
		}},
	)
	initial := &assemblyAITranscriptResponse{ID: "j1", Status: "processing"}
	_, err := tr.assemblyAIPoll(initial)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode poll response")
}

// ----- googleSTT ----------------------------------------------------------

func TestGoogleSTT_Success(t *testing.T) {
	audio := tempAudioFile(t)
	resp := googleSTTResponse{
		Results: []struct {
			Alternatives []struct {
				Transcript string `json:"transcript"`
			} `json:"alternatives"`
		}{
			{Alternatives: []struct {
				Transcript string `json:"transcript"`
			}{{Transcript: "google transcript"}}},
		},
	}
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderGoogleSTT, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{jsonResp(resp)}},
	)
	r, err := tr.googleSTT(audio)
	require.NoError(t, err)
	assert.Contains(t, r.Text, "google transcript")
}

func TestGoogleSTT_WithLanguage(t *testing.T) {
	audio := tempAudioFile(t)
	resp := googleSTTResponse{
		Results: []struct {
			Alternatives []struct {
				Transcript string `json:"transcript"`
			} `json:"alternatives"`
		}{
			{Alternatives: []struct {
				Transcript string `json:"transcript"`
			}{{Transcript: "bonjour"}}},
		},
	}
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderGoogleSTT, "fr-FR", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{jsonResp(resp)}},
	)
	r, err := tr.googleSTT(audio)
	require.NoError(t, err)
	assert.Contains(t, r.Text, "bonjour")
}

func TestGoogleSTT_MultipleResults(t *testing.T) {
	audio := tempAudioFile(t)
	resp := googleSTTResponse{
		Results: []struct {
			Alternatives []struct {
				Transcript string `json:"transcript"`
			} `json:"alternatives"`
		}{
			{Alternatives: []struct {
				Transcript string `json:"transcript"`
			}{{Transcript: "hello"}}},
			{Alternatives: []struct {
				Transcript string `json:"transcript"`
			}{{Transcript: "world"}}},
			// empty alternatives slice — should be skipped
			{},
		},
	}
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderGoogleSTT, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{jsonResp(resp)}},
	)
	r, err := tr.googleSTT(audio)
	require.NoError(t, err)
	assert.Contains(t, r.Text, "hello")
	assert.Contains(t, r.Text, "world")
}

func TestGoogleSTT_HTTPError(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderGoogleSTT, "", domain.TranscriberOutputText,
		&alwaysErrTransport{err: errors.New("timeout")},
	)
	_, err := tr.googleSTT(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "google-stt: request")
}

func TestGoogleSTT_Non200Status(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderGoogleSTT, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusUnauthorized, `{"error":"invalid key"}`),
		}},
	)
	_, err := tr.googleSTT(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 401")
}

func TestGoogleSTT_InvalidJSON(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderGoogleSTT, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			rawResp(http.StatusOK, `bad json here`),
		}},
	)
	_, err := tr.googleSTT(audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestGoogleSTT_EmptyResults(t *testing.T) {
	audio := tempAudioFile(t)
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderGoogleSTT, "", domain.TranscriberOutputText,
		&queuedTransport{responses: []*http.Response{
			jsonResp(googleSTTResponse{}),
		}},
	)
	r, err := tr.googleSTT(audio)
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(r.Text))
}

// ----- buildResult error path -------------------------------------------

func TestBuildResult_MediaMode_MissingFile(t *testing.T) {
	tr := newTestOnlineTranscriber(
		domain.TranscriberProviderOpenAIWhisper, "", domain.TranscriberOutputMedia,
		&queuedTransport{},
	)
	_, err := tr.buildResult("transcript text", "/nonexistent/path/audio.wav")
	require.Error(t, err)
}
