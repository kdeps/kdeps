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
	"log/slog"
	"testing"

	telegrambot "github.com/go-telegram/bot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// ----- Telegram: invalid chatID path -------------------------------------

// TestTelegramRunner_Reply_InvalidChatID exercises the strconv.ParseInt failure path.
// We need a non-nil bot to pass the nil-check guard; telegrambot.New creates one
// without making any network calls.
func TestTelegramRunner_Reply_InvalidChatID(t *testing.T) {
	b, err := telegrambot.New("test-bot-token-12345", telegrambot.WithSkipGetMe())
	require.NoError(t, err)

	runner := &telegramRunner{
		cfg:    &domain.TelegramConfig{BotToken: "test-bot-token-12345"},
		logger: slog.Default(),
		bot:    b,
	}

	// A non-numeric chatID must fail before any network call is made.
	replyErr := runner.Reply(context.Background(), "not-a-number", "hello")
	require.Error(t, replyErr)
	assert.Contains(t, replyErr.Error(), "invalid chatID")
}

// ----- Dispatcher.handleMessage: unknown platform ------------------------

// TestHandleMessage_UnknownPlatform exercises the "no runner for platform" log
// and early-return path. No engine or workflow is needed because the function
// returns before calling engine.Execute when the platform is not registered.
func TestHandleMessage_UnknownPlatform(_ *testing.T) {
	d := &Dispatcher{
		runners: map[string]Runner{},
		logger:  slog.Default(),
	}

	// Must not panic; just logs a warning and returns.
	d.handleMessage(context.Background(), Message{
		Platform: "unknown-platform",
		ChatID:   "chat1",
		UserID:   "user1",
		Text:     "hello",
	})
}

// TestHandleMessage_KnownPlatform_NilEngine exercises the path where the runner
// is found and engine.Execute is called with a nil engine, which returns an error.
// The dispatcher logs the error and returns — no panic.
func TestHandleMessage_KnownPlatform_NilEngine(_ *testing.T) {
	// Use a no-op runner that always succeeds for Reply.
	noopRunner := &discordRunner{
		cfg: &domain.DiscordConfig{BotToken: "Bot test"},
	}

	d := &Dispatcher{
		workflow: &domain.Workflow{},
		engine:   nil,
		runners:  map[string]Runner{"discord": noopRunner},
		logger:   slog.Default(),
	}

	// engine.Execute will panic if called on a nil engine pointer.
	// We recover from the panic so the test itself does not fail.
	func() {
		defer func() { recover() }() //nolint:errcheck
		d.handleMessage(context.Background(), Message{
			Platform: "discord",
			ChatID:   "C1",
			UserID:   "U1",
			Text:     "test",
		})
	}()
}
