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

package bot

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestCloseDiscordSession_DefaultClose(t *testing.T) {
	s, err := discordgo.New("Bot test-token")
	require.NoError(t, err)

	r := &discordRunner{logger: slog.Default()}
	r.closeDiscordSession(s)
}

func TestDiscordRunner_Start_SuccessAndClose(t *testing.T) {
	origNew := discordNewSession
	origOpen := discordSessionOpen
	origClose := discordSessionClose
	t.Cleanup(func() {
		discordNewSession = origNew
		discordSessionOpen = origOpen
		discordSessionClose = origClose
	})

	discordNewSession = func(_ string) (*discordgo.Session, error) {
		return &discordgo.Session{}, nil
	}
	discordSessionOpen = func(_ *discordgo.Session) error { return nil }
	discordSessionClose = func(_ *discordgo.Session) error {
		return errors.New("close failed")
	}

	r := &discordRunner{botToken: "token", logger: slog.Default()}
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Start(ctx, make(chan Message, 1))
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-errCh
	require.NoError(t, err)
}

func TestDiscordReply_Success(t *testing.T) {
	var capturedMethod, capturedPath string
	mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		capturedMethod = req.Method
		capturedPath = req.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(
				`{"id":"123","channel_id":"ch-1","content":"hello","timestamp":"2024-01-01T00:00:00Z","type":0}`,
			))),
		}, nil
	})

	s, err := discordgo.New("Bot test-token")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: mockTransport}

	r := &discordRunner{
		botToken: "test-token",
		session:  s,
		logger:   slog.Default(),
	}

	err = r.Reply(context.Background(), "ch-1", "hello")
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Contains(t, capturedPath, "/channels/ch-1/messages")
}

func TestDiscordRunner_Start_OpenError(t *testing.T) {
	oldTransport := http.DefaultTransport
	http.DefaultTransport = &failTransport{}
	defer func() { http.DefaultTransport = oldTransport }()

	r := &discordRunner{
		botToken: "test-token",
		logger:   slog.Default(),
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	err := r.Start(ctx, make(chan Message, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "discord: open session")
}

func TestDiscordRunner_NewRunner(t *testing.T) {
	cfg := &domain.DiscordConfig{}
	creds := &kdepsconfig.DiscordConnectionConfig{BotToken: "Bot testtoken"}
	runner := newDiscordRunner(cfg, creds, nil)
	require.NotNil(t, runner)
	assert.Equal(t, "Bot testtoken", runner.botToken)
	assert.Nil(t, runner.session)
}

func TestDiscordRunner_Reply_SessionNil(t *testing.T) {
	runner := newDiscordRunner(
		&domain.DiscordConfig{},
		&kdepsconfig.DiscordConnectionConfig{BotToken: "Bot testtoken"},
		nil,
	)
	// Session is nil before Start() is called — should return error.
	err := runner.Reply(context.Background(), "channel-1", "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not started")
}
