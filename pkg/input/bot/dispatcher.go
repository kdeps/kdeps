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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

const msgChanBuffer = 64

// newBotRunner creates a platform runner; overridable for testing.
//
//nolint:gochecknoglobals // test-replaceable
var newBotRunner = New

// Dispatcher owns the message channel, starts all platform runners, and for each
// inbound message calls the workflow engine and dispatches the reply back.
type Dispatcher struct {
	workflow *domain.Workflow
	engine   *executor.Engine
	runners  map[string]Runner // platform name → Runner
	logger   *slog.Logger
}

func validateBotConfig(workflow *domain.Workflow) (*domain.BotConfig, error) {
	cfg := workflow.Settings.Input
	if cfg == nil {
		return nil, errors.New("bot dispatcher: workflow has no input config")
	}
	if cfg.Bot == nil {
		return nil, errors.New("bot dispatcher: workflow has no bot config")
	}
	return cfg.Bot, nil
}

func buildBotRunners(
	botCfg *domain.BotConfig,
	creds *kdepsconfig.BotConnectionConfig,
	logger *slog.Logger,
) (map[string]Runner, error) {
	runners := make(map[string]Runner)
	platforms := []struct {
		name   string
		nonNil bool
	}{
		{discordPlatform, botCfg.Discord != nil},
		{slackPlatform, botCfg.Slack != nil},
		{"telegram", botCfg.Telegram != nil},
	}

	for _, p := range platforms {
		if !p.nonNil {
			continue
		}
		r, err := newBotRunner(p.name, botCfg, creds, logger)
		if err != nil {
			return nil, fmt.Errorf("bot dispatcher: create runner for %s: %w", p.name, err)
		}
		runners[p.name] = r
	}

	if len(runners) == 0 {
		return nil, errors.New(
			"bot dispatcher: no bot platforms configured (discord, slack, or telegram sub-config required)",
		)
	}
	return runners, nil
}

// NewDispatcher creates a Dispatcher for the bot platforms configured in workflow.Settings.Input.Bot.
// creds provides bot credentials from ~/.kdeps/config.yaml bot_connections.
func NewDispatcher(
	workflow *domain.Workflow,
	engine *executor.Engine,
	creds *kdepsconfig.BotConnectionConfig,
	logger *slog.Logger,
) (*Dispatcher, error) {
	kdeps_debug.Log("enter: NewDispatcher")
	if logger == nil {
		logger = slog.Default()
	}

	botCfg, err := validateBotConfig(workflow)
	if err != nil {
		return nil, err
	}

	runners, err := buildBotRunners(botCfg, creds, logger)
	if err != nil {
		return nil, err
	}

	return &Dispatcher{
		workflow: workflow,
		engine:   engine,
		runners:  runners,
		logger:   logger,
	}, nil
}

// Run starts all runners and blocks until ctx is cancelled.
// Each inbound message is handled in its own goroutine.
func (d *Dispatcher) Run(ctx context.Context) error {
	kdeps_debug.Log("enter: Run")
	ch := make(chan Message, msgChanBuffer)

	for platform, r := range d.runners {
		go func() {
			if err := r.Start(ctx, ch); err != nil && ctx.Err() == nil {
				d.logger.ErrorContext(ctx, "bot runner stopped", "platform", platform, "err", err)
			}
		}()
		d.logger.InfoContext(ctx, "bot runner started", "platform", platform)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			go d.handleMessage(ctx, msg)
		}
	}
}

func (d *Dispatcher) buildRequestContext(msg Message) *executor.RequestContext {
	return &executor.RequestContext{
		Method: "POST",
		Path:   "/bot/" + msg.Platform,
		Body: map[string]interface{}{
			"message":  msg.Text,
			"chatId":   msg.ChatID,
			"userId":   msg.UserID,
			"platform": msg.Platform,
		},
	}
}

// handleMessage executes the workflow for one inbound message.
// The workflow's result (its target resource's output, see ReplyText) is sent
// back to the chat the message came from. After this function returns
// the dispatcher loop immediately waits for the next message (polling restart).
func (d *Dispatcher) handleMessage(ctx context.Context, msg Message) {
	kdeps_debug.Log("enter: handleMessage")
	runner, ok := d.runners[msg.Platform]
	if !ok {
		d.logger.WarnContext(ctx, "bot: no runner for platform", "platform", msg.Platform)
		return
	}

	result, err := d.engine.Execute(d.workflow, d.buildRequestContext(msg))
	if err != nil {
		d.logger.ErrorContext(ctx, "bot: workflow execution failed", "platform", msg.Platform, "err", err)
		return
	}
	text := ReplyText(result)
	if text == "" {
		d.logger.WarnContext(ctx, "bot: workflow returned no reply text", "platform", msg.Platform)
		return
	}
	if replyErr := runner.Reply(ctx, msg.ChatID, text); replyErr != nil {
		d.logger.ErrorContext(ctx, "bot: reply failed", "platform", msg.Platform, "err", replyErr)
	}
}
