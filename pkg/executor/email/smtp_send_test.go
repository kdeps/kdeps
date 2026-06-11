// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func smtpResolveCtx(t *testing.T) *executor.ExecutionContext {
	t.Helper()
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "smtp-test", TargetActionID: "mail"},
		Resources: []*domain.Resource{
			{ActionID: "mail", Name: "Mail", Email: &domain.EmailConfig{}},
		},
	}
	ctx, err := executor.NewExecutionContext(wf)
	require.NoError(t, err)
	return ctx
}

func TestResolveSendRequest_EvalErrors(t *testing.T) {
	t.Parallel()

	ex := &Executor{}
	ctx := smtpResolveCtx(t)
	smtpCfg := kdepsconfig.SMTPConnectionConfig{
		Host:     "smtp.example.com",
		Username: "user",
		Password: "pass",
	}

	tests := []struct {
		name string
		cfg  *domain.EmailConfig
		want string
	}{
		{
			name: "from",
			cfg:  &domain.EmailConfig{From: "{{ !bad }}"},
			want: "evaluate from",
		},
		{
			name: "subject",
			cfg:  &domain.EmailConfig{From: "a@b.com", Subject: "{{ !bad }}"},
			want: "evaluate subject",
		},
		{
			name: "body",
			cfg:  &domain.EmailConfig{From: "a@b.com", Subject: "s", Body: "{{ !bad }}"},
			want: "evaluate body",
		},
		{
			name: "to",
			cfg:  &domain.EmailConfig{From: "a@b.com", Subject: "s", Body: "b", To: []string{"{{ !bad }}"}},
			want: "evaluate to",
		},
		{
			name: "cc",
			cfg: &domain.EmailConfig{
				From: "a@b.com", Subject: "s", Body: "b", To: []string{"a@b.com"},
				CC: []string{"{{ !bad }}"},
			},
			want: "evaluate cc",
		},
		{
			name: "bcc",
			cfg: &domain.EmailConfig{
				From: "a@b.com", Subject: "s", Body: "b", To: []string{"a@b.com"},
				BCC: []string{"{{ !bad }}"},
			},
			want: "evaluate bcc",
		},
		{
			name: "attachments",
			cfg: &domain.EmailConfig{
				From: "a@b.com", Subject: "s", Body: "b", To: []string{"a@b.com"},
				Attachments: []string{"{{ !bad }}"},
			},
			want: "evaluate attachments",
		},
	}

	hostCfg := kdepsconfig.SMTPConnectionConfig{
		Host:     "{{ !bad }}",
		Username: "user",
		Password: "pass",
	}
	_, err := ex.resolveSendRequest(ctx, &domain.EmailConfig{
		From: "a@b.com", Subject: "s", Body: "b", To: []string{"a@b.com"},
	}, hostCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evaluate smtp host")

	userCfg := kdepsconfig.SMTPConnectionConfig{
		Host:     "smtp.example.com",
		Username: "{{ !bad }}",
		Password: "pass",
	}
	_, err = ex.resolveSendRequest(ctx, &domain.EmailConfig{
		From: "a@b.com", Subject: "s", Body: "b", To: []string{"a@b.com"},
	}, userCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evaluate smtp user")

	passCfg := kdepsconfig.SMTPConnectionConfig{
		Host:     "smtp.example.com",
		Username: "user",
		Password: "{{ !bad }}",
	}
	_, err = ex.resolveSendRequest(ctx, &domain.EmailConfig{
		From: "a@b.com", Subject: "s", Body: "b", To: []string{"a@b.com"},
	}, passCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evaluate smtp password")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, resolveErr := ex.resolveSendRequest(ctx, tt.cfg, smtpCfg)
			require.Error(t, resolveErr)
			assert.Contains(t, resolveErr.Error(), tt.want)
		})
	}
}
