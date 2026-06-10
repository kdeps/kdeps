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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestTelegramRunner_NewRunner(t *testing.T) {
	creds := &kdepsconfig.TelegramConnectionConfig{BotToken: "12345:test-token"}
	runner := newTelegramRunner(&domain.TelegramConfig{}, creds, nil)
	require.NotNil(t, runner)
	assert.Equal(t, "12345:test-token", runner.botToken)
	assert.Nil(t, runner.bot)
}

func TestTelegramRunner_Reply_BotNil(t *testing.T) {
	runner := newTelegramRunner(
		&domain.TelegramConfig{},
		&kdepsconfig.TelegramConnectionConfig{BotToken: "12345:test-token"},
		nil,
	)
	// bot is nil before Start() is called — should return error.
	err := runner.Reply(context.Background(), "12345", "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bot not started")
}
