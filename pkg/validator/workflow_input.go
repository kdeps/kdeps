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

package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	kdeps_debug "github.com/kdeps/kdeps/v3/pkg/debug"
	"github.com/kdeps/kdeps/v3/pkg/domain"
)

// validateSourcesList validates each source entry and per-source config requirements.
func (v *WorkflowValidator) validateSourcesList(config *domain.InputConfig) error {
	kdeps_debug.Log("enter: validateSourcesList")
	hasBot := false
	seen := make(map[string]bool)
	for _, source := range config.Sources {
		if source == "" {
			return domain.NewError(
				domain.ErrCodeInvalidWorkflow,
				"input source cannot be empty",
				nil,
			)
		}
		if !domain.IsValidWorkflowInputSource(source) {
			return domain.NewError(
				domain.ErrCodeInvalidWorkflow,
				fmt.Sprintf(
					"invalid input source: %s. Available options: [%s]",
					source,
					domain.WorkflowInputSourcesDisplay(),
				),
				nil,
			)
		}
		if seen[source] {
			return domain.NewError(
				domain.ErrCodeInvalidWorkflow,
				fmt.Sprintf("duplicate input source: %s", source),
				nil,
			)
		}
		seen[source] = true
		if source == domain.InputSourceBot {
			hasBot = true
		}
	}

	if hasBot {
		if config.Bot == nil {
			return domain.NewError(
				domain.ErrCodeInvalidWorkflow,
				"input.bot is required when sources includes bot",
				nil,
			)
		}
		if err := v.validateBotConfig(config.Bot); err != nil {
			return err
		}
	}

	return nil
}

// validateBotConfig validates the bot sub-configuration.
// For polling mode (default), at least one platform must be configured.
// For stateless mode, platform sub-configs are optional.
// Each configured platform must have the required credentials.
func (v *WorkflowValidator) validateBotConfig(cfg *domain.BotConfig) error {
	kdeps_debug.Log("enter: validateBotConfig")
	executionType := cfg.ExecutionType
	if executionType == "" {
		executionType = domain.BotExecutionTypePolling
	}
	if executionType != domain.BotExecutionTypePolling &&
		executionType != domain.BotExecutionTypeStateless {
		return domain.NewError(
			domain.ErrCodeInvalidWorkflow,
			fmt.Sprintf(
				"input.bot.executionType must be %q or %q, got %q",
				domain.BotExecutionTypePolling,
				domain.BotExecutionTypeStateless,
				cfg.ExecutionType,
			),
			nil,
		)
	}

	noPlatforms := cfg.Discord == nil && cfg.Slack == nil && cfg.Telegram == nil
	if executionType == domain.BotExecutionTypePolling && noPlatforms {
		return domain.NewError(
			domain.ErrCodeInvalidWorkflow,
			"input.bot must configure at least one platform"+
				" (discord, slack, or telegram) when executionType is polling",
			nil,
		)
	}
	return nil
}

// ValidateInputConfig validates the workflow input source configuration.
func (v *WorkflowValidator) ValidateInputConfig(config *domain.InputConfig) error {
	kdeps_debug.Log("enter: ValidateInputConfig")
	if len(config.Sources) == 0 && len(config.Listen) == 0 {
		return domain.NewError(
			domain.ErrCodeInvalidWorkflow,
			"input needs sources or listen",
			nil,
		)
	}

	if err := v.validateSourcesList(config); err != nil {
		return err
	}

	return validateListen(config)
}

var listenIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// validateListen checks the structure of input.listen. The templates are
// parsed when the listener starts.
func validateListen(config *domain.InputConfig) error {
	if len(config.Listen) == 0 {
		return nil
	}
	if config.HasBotSource() || config.HasFileSource() {
		return listenError("input.listen cannot be used with the bot or file source")
	}
	seen := map[string]bool{}
	for i, spec := range config.Listen {
		id := spec.ID
		if id == "" {
			id = fmt.Sprintf("listen-%d", i)
		} else if !listenIDPattern.MatchString(id) {
			return listenError(fmt.Sprintf("input.listen[%d].id %q: use letters, digits, _, . and -", i, id))
		}
		if seen[id] {
			return listenError(fmt.Sprintf("input.listen: duplicate id %q", id))
		}
		seen[id] = true
		if err := validateListenSpec(id, spec); err != nil {
			return err
		}
	}
	return nil
}

func validateListenSpec(id string, spec domain.ListenSpec) error {
	set := 0
	for _, ok := range []bool{spec.Fifo != nil, spec.File != nil, spec.Poll != nil, spec.Webhook != nil, spec.Websocket != nil} {
		if ok {
			set++
		}
	}
	if set != 1 {
		return listenError(fmt.Sprintf("listen %s: set exactly one of fifo, file, poll, webhook and websocket", id))
	}
	switch {
	case spec.Fifo != nil:
		return validateListenPath(id, "fifo", spec.Fifo.Path, spec.Fifo.Framing)
	case spec.File != nil:
		return validateListenPath(id, "file", spec.File.Path, spec.File.Framing)
	case spec.Webhook != nil:
		return validateWebhook(id, spec.Webhook)
	case spec.Websocket != nil:
		return validateWebsocket(id, spec.Websocket)
	}
	if spec.Poll.Tool == "" {
		return listenError(fmt.Sprintf("listen %s: poll.tool is required", id))
	}
	if spec.Poll.Every != "" {
		d, err := time.ParseDuration(spec.Poll.Every)
		if err != nil || d < time.Second {
			return listenError(
				fmt.Sprintf("listen %s: poll.every %q: use a duration of 1s or more", id, spec.Poll.Every),
			)
		}
	}
	return nil
}

func validateWebsocket(id string, w *domain.ListenWebsocketSpec) error {
	switch {
	case w.URL == "":
		return listenError(fmt.Sprintf("listen %s: websocket.url is required", id))
	case w.Open != nil && w.Open.Tool == "":
		return listenError(fmt.Sprintf("listen %s: websocket.open.tool is required", id))
	case w.Heartbeat != nil && w.Heartbeat.Send == "":
		return listenError(fmt.Sprintf("listen %s: websocket.heartbeat.send is required", id))
	}
	for i, r := range w.On {
		if r.When == "" {
			return listenError(fmt.Sprintf("listen %s: websocket.on[%d].when is required", id, i))
		}
	}
	return nil
}

func validateWebhook(id string, w *domain.ListenWebhookSpec) error {
	if !strings.HasPrefix(w.Path, "/") {
		return listenError(fmt.Sprintf("listen %s: webhook.path must start with /", id))
	}
	v := w.Verify
	if v == nil {
		return nil
	}
	switch {
	case len(v.Secret) == 0:
		return listenError(fmt.Sprintf("listen %s: webhook.verify.secret is required", id))
	case v.Signature == "":
		return listenError(fmt.Sprintf("listen %s: webhook.verify.signature is required", id))
	case v.Algorithm != "" && v.Algorithm != "sha256" && v.Algorithm != "sha1":
		return listenError(fmt.Sprintf("listen %s: webhook.verify.algorithm %q: use sha256 or sha1", id, v.Algorithm))
	case v.Encoding != "" && v.Encoding != "hex" && v.Encoding != "base64":
		return listenError(fmt.Sprintf("listen %s: webhook.verify.encoding %q: use hex or base64", id, v.Encoding))
	}
	return nil
}

func validateListenPath(id, kind, path, framing string) error {
	if path == "" {
		return listenError(fmt.Sprintf("listen %s: %s.path is required", id, kind))
	}
	if framing != "" && framing != domain.ListenFramingLine && framing != domain.ListenFramingJSON {
		return listenError(fmt.Sprintf("listen %s: %s.framing %q: use line or json", id, kind, framing))
	}
	return nil
}

func listenError(msg string) error {
	return domain.NewError(domain.ErrCodeInvalidWorkflow, msg, nil)
}
