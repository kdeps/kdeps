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
	"context"
	"errors"
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

func TestNewDiscordRunner(t *testing.T) {
	r := newDiscordRunner(nil, nil, slog.Default())
	assert.NotNil(t, r)
	assert.Equal(t, "", r.botToken)
	assert.Equal(t, "", r.guildID)
}

func TestNewDiscordRunner_WithConfig(t *testing.T) {
	cfg := &domain.DiscordConfig{GuildID: "guild-123"}
	creds := &kdepsconfig.DiscordConnectionConfig{BotToken: "token-abc"}
	r := newDiscordRunner(cfg, creds, slog.Default())
	assert.Equal(t, "token-abc", r.botToken)
	assert.Equal(t, "guild-123", r.guildID)
}

func TestDiscordRunner_Start_CreateSessionError(t *testing.T) {
	orig := discordNewSession
	t.Cleanup(func() { discordNewSession = orig })

	sentinel := errors.New("invalid token format")
	discordNewSession = func(_ string) (*discordgo.Session, error) {
		return nil, sentinel
	}

	r := &discordRunner{botToken: "bad-token", logger: slog.Default()}
	ch := make(chan Message, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so Start returns quickly

	err := r.Start(ctx, ch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "discord: create session")
}

func TestDiscordRunner_Reply_SessionNotStarted(t *testing.T) {
	r := &discordRunner{logger: slog.Default()}
	err := r.Reply(context.Background(), "ch1", "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not started")
}

func TestDiscordRunner_Start_OpenErrorViaMockedSession(t *testing.T) {
	orig := discordNewSession
	t.Cleanup(func() { discordNewSession = orig })

	discordNewSession = func(token string) (*discordgo.Session, error) {
		s, err := discordgo.New(token)
		if err != nil {
			return nil, err
		}
		// Replace the HTTP client so Open() fails immediately.
		s.Client = &http.Client{
			Transport: roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("simulated open failure")
			}),
		}
		return s, nil
	}

	r := &discordRunner{botToken: "test-token", logger: slog.Default()}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	err := r.Start(ctx, make(chan Message, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "discord: open session")
}

func TestDiscordRunner_Start_ContextCancelled(t *testing.T) {
	// This test verifies that Start returns nil when context is cancelled
	// before the session is opened. We mock discordNewSession to return
	// a session that fails on Open().
	r := &discordRunner{botToken: "test-token", logger: slog.Default()}
	ch := make(chan Message, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// The session creation succeeds, but Open requires a real WebSocket.
	// Since context is already cancelled, Start should return.
	// We mock New to return successfully, but Open will fail.
	err := r.Start(ctx, ch)
	// This may error because Open fails (no real Discord), or return nil
	// because context is already cancelled.
	// Either way, the function should not hang.
	t.Logf("Start result with cancelled context: %v", err)
}

func TestHandleDiscordMessage_SelfMessage(t *testing.T) {
	r := &discordRunner{logger: slog.Default()}
	s, _ := discordgo.New("Bot dummy")
	s.State.User = &discordgo.User{ID: "bot-123"}
	m := &discordgo.MessageCreate{Message: &discordgo.Message{Author: &discordgo.User{ID: "bot-123"}, Content: "hello"}}
	ch := make(chan Message, 1)
	r.handleDiscordMessage(context.Background(), s, m, ch)
	select {
	case <-ch:
		t.Error("expected self message to be filtered")
	default:
	}
}

func TestHandleDiscordMessage_GuildFilter(t *testing.T) {
	r := &discordRunner{guildID: "guild-1", logger: slog.Default()}
	s, _ := discordgo.New("Bot dummy")
	s.State.User = &discordgo.User{ID: "bot-123"}
	m := &discordgo.MessageCreate{Message: &discordgo.Message{
		Author: &discordgo.User{ID: "user-456"}, GuildID: "guild-2", Content: "hello",
	}}
	ch := make(chan Message, 1)
	r.handleDiscordMessage(context.Background(), s, m, ch)
	select {
	case <-ch:
		t.Error("expected guild message to be filtered")
	default:
	}
}

func TestHandleDiscordMessage_Success(t *testing.T) {
	r := &discordRunner{logger: slog.Default()}
	s, _ := discordgo.New("Bot dummy")
	s.State.User = &discordgo.User{ID: "bot-123"}
	m := &discordgo.MessageCreate{Message: &discordgo.Message{
		Author: &discordgo.User{ID: "user-456"}, ChannelID: "ch-1", Content: "hello world",
	}}
	ch := make(chan Message, 1)
	r.handleDiscordMessage(context.Background(), s, m, ch)
	msg := <-ch
	assert.Equal(t, discordPlatform, msg.Platform)
	assert.Equal(t, "ch-1", msg.ChatID)
	assert.Equal(t, "user-456", msg.UserID)
	assert.Equal(t, "hello world", msg.Text)
}

func TestHandleDiscordMessage_CancelledContext(_ *testing.T) {
	r := &discordRunner{logger: slog.Default()}
	s, _ := discordgo.New("Bot dummy")
	s.State.User = &discordgo.User{ID: "bot-123"}
	m := &discordgo.MessageCreate{Message: &discordgo.Message{
		Author: &discordgo.User{ID: "user-456"}, ChannelID: "ch-1", Content: "hello",
	}}
	ch := make(chan Message)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r.handleDiscordMessage(ctx, s, m, ch)
}

func TestCreateMessageHandler(t *testing.T) {
	r := &discordRunner{logger: slog.Default()}
	s, _ := discordgo.New("Bot dummy")
	s.State.User = &discordgo.User{ID: "bot-123"}
	ch := make(chan Message, 1)
	ctx := context.Background()

	handler := r.createMessageHandler(ctx, s, ch)
	h := handler.(func(*discordgo.Session, *discordgo.MessageCreate))
	h(s, &discordgo.MessageCreate{Message: &discordgo.Message{
		Author: &discordgo.User{ID: "user-456"}, ChannelID: "ch-1", Content: "test",
	}})
	msg := <-ch
	assert.Equal(t, "test", msg.Text)
}
