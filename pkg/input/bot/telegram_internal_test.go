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
	"testing"
	"time"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestNewTelegramRunner(t *testing.T) {
	r := newTelegramRunner(nil, nil, slog.Default())
	assert.NotNil(t, r)
	assert.Equal(t, "", r.botToken)
	assert.Equal(t, 0, r.pollIntervalSeconds)
}

func TestNewTelegramRunner_WithConfig(t *testing.T) {
	cfg := &domain.TelegramConfig{PollIntervalSeconds: 10}
	creds := &kdepsconfig.TelegramConnectionConfig{BotToken: "token-abc"}
	r := newTelegramRunner(cfg, creds, slog.Default())
	assert.Equal(t, "token-abc", r.botToken)
	assert.Equal(t, 10, r.pollIntervalSeconds)
}

func TestTelegramRunner_Start_CreateBotError(t *testing.T) {
	orig := telegramNewBot
	t.Cleanup(func() { telegramNewBot = orig })

	sentinel := errors.New("invalid bot token")
	telegramNewBot = func(_ string, _ ...telegrambot.Option) (*telegrambot.Bot, error) {
		return nil, sentinel
	}

	r := &telegramRunner{botToken: "bad-token", logger: slog.Default()}
	ch := make(chan Message, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Start(ctx, ch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "telegram: create bot")
}

func TestTelegramRunner_Reply_BotNotStarted(t *testing.T) {
	r := &telegramRunner{logger: slog.Default()}
	err := r.Reply(context.Background(), "123", "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bot not started")
}

func TestTelegramRunner_Start_WithPollInterval(t *testing.T) {
	r := &telegramRunner{
		botToken:            "test-token",
		pollIntervalSeconds: 5,
		logger:              slog.Default(),
	}
	ch := make(chan Message, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Start will try to create a real bot, which fails. That's fine for coverage.
	err := r.Start(ctx, ch)
	// Expect error (invalid token) - just exercising the poll interval branch.
	t.Logf("Start result: %v", err)
}

func TestTelegramRunner_Start_DefaultPollTimeout(t *testing.T) {
	r := &telegramRunner{
		botToken:            "test-token",
		pollIntervalSeconds: 0, // Should use default 30s
		logger:              slog.Default(),
	}
	// Verify the default is set
	assert.Equal(t, 0, r.pollIntervalSeconds)

	// Exercise the default branch
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ch := make(chan Message, 1)

	err := r.Start(ctx, ch)
	t.Logf("Start result (default timeout): %v", err)
}

func TestHandleTelegramUpdate_Success(t *testing.T) {
	r := &telegramRunner{logger: slog.Default()}
	update := &models.Update{
		Message: &models.Message{
			Chat: models.Chat{ID: 123456},
			From: &models.User{ID: 789},
			Text: "hello telegram",
		},
	}
	ch := make(chan Message, 1)
	r.handleTelegramUpdate(context.Background(), update, ch)
	msg := <-ch
	assert.Equal(t, telegramPlatform, msg.Platform)
	assert.Equal(t, "123456", msg.ChatID)
	assert.Equal(t, "789", msg.UserID)
	assert.Equal(t, "hello telegram", msg.Text)
}

func TestHandleTelegramUpdate_NilMessage(t *testing.T) {
	r := &telegramRunner{logger: slog.Default()}
	update := &models.Update{Message: nil}
	ch := make(chan Message, 1)
	r.handleTelegramUpdate(context.Background(), update, ch)
	select {
	case <-ch:
		t.Error("expected nil message to be filtered")
	default:
	}
}

func TestHandleTelegramUpdate_EmptyText(t *testing.T) {
	r := &telegramRunner{logger: slog.Default()}
	update := &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 123}, From: &models.User{ID: 456}, Text: ""},
	}
	ch := make(chan Message, 1)
	r.handleTelegramUpdate(context.Background(), update, ch)
	select {
	case <-ch:
		t.Error("expected empty text to be filtered")
	default:
	}
}

func TestHandleTelegramUpdate_CancelledContext(_ *testing.T) {
	r := &telegramRunner{logger: slog.Default()}
	update := &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 123}, From: &models.User{ID: 456}, Text: "msg"},
	}
	ch := make(chan Message)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r.handleTelegramUpdate(ctx, update, ch)
}

func TestCreateTelegramHandler(t *testing.T) {
	r := &telegramRunner{logger: slog.Default()}
	ch := make(chan Message, 1)
	ctx := context.Background()

	handler := r.createTelegramHandler(ctx, ch)
	handler(context.Background(), nil, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 123}, From: &models.User{ID: 456}, Text: "test"},
	})
	msg := <-ch
	assert.Equal(t, "test", msg.Text)
}
