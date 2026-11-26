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

package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/validator"
)

func TestValidateInputConfig_Listen(t *testing.T) {
	v := validator.NewWorkflowValidator(nil)
	file := &domain.ListenFileSpec{Path: "in.txt"}
	poll := &domain.ListenPollSpec{Tool: "github_get", Every: "1m"}

	ok := []*domain.InputConfig{
		{Listen: []domain.ListenSpec{{File: file}}},
		{Sources: []string{"api"}, Listen: []domain.ListenSpec{{ID: "a", Poll: poll}, {ID: "b", File: file}}},
		{Listen: []domain.ListenSpec{{Fifo: &domain.ListenFifoSpec{Path: "p", Framing: "json"}}}},
		{Listen: []domain.ListenSpec{{Poll: &domain.ListenPollSpec{Tool: "t"}}}},
		{Listen: []domain.ListenSpec{{Webhook: &domain.ListenWebhookSpec{Path: "/h"}}}},
		{Listen: []domain.ListenSpec{{Webhook: &domain.ListenWebhookSpec{Path: "/h", Verify: &domain.ListenVerifySpec{
			Secret: []string{"S"}, Signature: "x", Algorithm: "sha1", Encoding: "base64",
		}}}}},
	}
	for _, c := range ok {
		require.NoError(t, v.ValidateInputConfig(c))
	}

	bad := map[string]*domain.InputConfig{
		"input needs sources or listen": {},
		"cannot be used with the bot or file source": {
			Sources: []string{"file"}, Listen: []domain.ListenSpec{{File: file}},
		},
		`id "a b"`: {Listen: []domain.ListenSpec{{ID: "a b", File: file}}},
		`duplicate id "listen-0"`: {
			Listen: []domain.ListenSpec{{File: file}, {ID: "listen-0", File: file}},
		},
		"set exactly one of fifo, file, poll and webhook": {Listen: []domain.ListenSpec{{File: file, Poll: poll}}},
		"listen listen-0: set exactly one":                {Listen: []domain.ListenSpec{{}}},
		"fifo.path is required": {
			Listen: []domain.ListenSpec{{Fifo: &domain.ListenFifoSpec{}}},
		},
		`file.framing "xml"`: {
			Listen: []domain.ListenSpec{{File: &domain.ListenFileSpec{Path: "x", Framing: "xml"}}},
		},
		"poll.tool is required": {Listen: []domain.ListenSpec{{Poll: &domain.ListenPollSpec{}}}},
		`poll.every "100ms"`: {
			Listen: []domain.ListenSpec{{Poll: &domain.ListenPollSpec{Tool: "t", Every: "100ms"}}},
		},
		`poll.every "often"`: {
			Listen: []domain.ListenSpec{{Poll: &domain.ListenPollSpec{Tool: "t", Every: "often"}}},
		},
	}
	hook := func(v *domain.ListenVerifySpec) *domain.InputConfig {
		return &domain.InputConfig{
			Listen: []domain.ListenSpec{{Webhook: &domain.ListenWebhookSpec{Path: "/h", Verify: v}}},
		}
	}
	bad["webhook.path must start with /"] = &domain.InputConfig{
		Listen: []domain.ListenSpec{{Webhook: &domain.ListenWebhookSpec{Path: "h"}}},
	}
	bad["webhook.verify.secret is required"] = hook(&domain.ListenVerifySpec{Signature: "x"})
	bad["webhook.verify.signature is required"] = hook(&domain.ListenVerifySpec{Secret: []string{"S"}})
	bad[`algorithm "md5"`] = hook(&domain.ListenVerifySpec{Secret: []string{"S"}, Signature: "x", Algorithm: "md5"})
	bad[`encoding "b32"`] = hook(&domain.ListenVerifySpec{Secret: []string{"S"}, Signature: "x", Encoding: "b32"})
	for want, c := range bad {
		err := v.ValidateInputConfig(c)
		require.Error(t, err, want)
		assert.Contains(t, err.Error(), want)
	}
}
