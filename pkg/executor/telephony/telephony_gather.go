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

package telephony

import (
	"strconv"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

const defaultGatherTimeoutSeconds = 5

func gatherTimeoutSeconds(timeout string) int {
	if seconds := parseDurationSeconds(timeout); seconds > 0 {
		return seconds
	}
	return defaultGatherTimeoutSeconds
}

func buildSpeechGatherOptions(g *GatherOptions, cfg *domain.TelephonyActionConfig, inputAttr string) {
	if !strings.Contains(inputAttr, "speech") {
		return
	}
	if cfg.Timeout != "" {
		g.SpeechTimeout = cfg.Timeout
	} else {
		g.SpeechTimeout = "auto"
	}
}

func buildGatherOptions(cfg *domain.TelephonyActionConfig, numDigits int, finishOnKey string) GatherOptions {
	inputAttr := inputAttrFromMode(cfg.Mode)
	g := GatherOptions{
		Input:       inputAttr,
		NumDigits:   numDigits,
		Timeout:     gatherTimeoutSeconds(cfg.Timeout),
		FinishOnKey: finishOnKey,
		Say:         cfg.Say,
		Voice:       cfg.Voice,
		Audio:       cfg.Audio,
	}
	buildSpeechGatherOptions(&g, cfg, inputAttr)
	return g
}

// parseDurationSeconds parses a duration string like "5s", "30s", "2m" into
// whole seconds. Returns 0 on empty input or parse error.
func parseDurationSeconds(s string) int {
	if s == "" {
		return 0
	}
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "s") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "s"))
		if err == nil {
			return n
		}
	}
	if strings.HasSuffix(s, "m") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err == nil {
			const secsPerMinute = 60
			return n * secsPerMinute
		}
	}
	n, err := strconv.Atoi(s)
	if err == nil {
		return n
	}
	return 0
}
