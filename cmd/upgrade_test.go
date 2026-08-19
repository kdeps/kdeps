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

//go:build !js

package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/upgrade"
)

func stubCmdUpgradeHooks(
	t *testing.T,
	fresh func(context.Context) (upgrade.CheckResult, error),
	detect func() upgrade.Method,
	perform func(context.Context, io.Writer, string) error,
) {
	t.Helper()
	origFresh, origDetect, origPerform := upgradeFreshFunc, upgradeDetectFunc, upgradePerformFunc
	if fresh != nil {
		upgradeFreshFunc = fresh
	}
	if detect != nil {
		upgradeDetectFunc = detect
	}
	if perform != nil {
		upgradePerformFunc = perform
	}
	t.Cleanup(func() {
		upgradeFreshFunc, upgradeDetectFunc, upgradePerformFunc = origFresh, origDetect, origPerform
	})
}

func TestRunUpgradeCmd_CheckFails(t *testing.T) {
	stubCmdUpgradeHooks(t, func(context.Context) (upgrade.CheckResult, error) {
		return upgrade.CheckResult{}, errors.New("network down")
	}, nil, nil)

	var out bytes.Buffer
	err := runUpgradeCmd(&out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check failed")
}

func TestRunUpgradeCmd_UpToDate(t *testing.T) {
	stubCmdUpgradeHooks(t, func(context.Context) (upgrade.CheckResult, error) {
		return upgrade.CheckResult{Current: "2.9.0", Latest: "2.9.0", Available: false}, nil
	}, nil, nil)

	var out bytes.Buffer
	require.NoError(t, runUpgradeCmd(&out))
	assert.Contains(t, out.String(), "up to date")
}

func TestRunUpgradeCmd_NonStandaloneInstructionsOnly(t *testing.T) {
	performCalled := false
	stubCmdUpgradeHooks(t,
		func(context.Context) (upgrade.CheckResult, error) {
			return upgrade.CheckResult{Current: "2.8.0", Latest: "2.9.0", Available: true}, nil
		},
		func() upgrade.Method { return upgrade.MethodDebPkg },
		func(context.Context, io.Writer, string) error { performCalled = true; return nil },
	)

	var out bytes.Buffer
	require.NoError(t, runUpgradeCmd(&out))
	assert.False(t, performCalled)
	assert.Contains(t, out.String(), "apt")
}

func TestRunUpgradeCmd_StandaloneDeclinedWithoutYes(t *testing.T) {
	t.Setenv("KDEPS_YES", "")
	t.Setenv("KDEPS_ASSUME_YES", "")
	performCalled := false
	stubCmdUpgradeHooks(t,
		func(context.Context) (upgrade.CheckResult, error) {
			return upgrade.CheckResult{Current: "2.8.0", Latest: "2.9.0", Available: true}, nil
		},
		func() upgrade.Method { return upgrade.MethodStandalone },
		func(context.Context, io.Writer, string) error { performCalled = true; return nil },
	)

	var out bytes.Buffer
	require.NoError(t, runUpgradeCmd(&out))
	assert.False(t, performCalled)
	assert.Contains(t, out.String(), "skipped")
}

func TestRunUpgradeCmd_StandaloneConfirmedViaEnv(t *testing.T) {
	t.Setenv("KDEPS_YES", "1")
	var gotTag string
	stubCmdUpgradeHooks(t,
		func(context.Context) (upgrade.CheckResult, error) {
			return upgrade.CheckResult{Current: "2.8.0", Latest: "2.9.0", Available: true}, nil
		},
		func() upgrade.Method { return upgrade.MethodStandalone },
		func(_ context.Context, _ io.Writer, tag string) error { gotTag = tag; return nil },
	)

	var out bytes.Buffer
	require.NoError(t, runUpgradeCmd(&out))
	assert.Equal(t, "2.9.0", gotTag)
	assert.Contains(t, out.String(), "Updated to v2.9.0")
}

func TestRunUpgradeCmd_PerformFails(t *testing.T) {
	t.Setenv("KDEPS_YES", "1")
	stubCmdUpgradeHooks(t,
		func(context.Context) (upgrade.CheckResult, error) {
			return upgrade.CheckResult{Current: "2.8.0", Latest: "2.9.0", Available: true}, nil
		},
		func() upgrade.Method { return upgrade.MethodStandalone },
		func(context.Context, io.Writer, string) error { return errors.New("checksum mismatch") },
	)

	var out bytes.Buffer
	err := runUpgradeCmd(&out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestRootCmd_UpgradeFlagShortCircuits(t *testing.T) {
	t.Setenv("KDEPS_YES", "1")
	stubCmdUpgradeHooks(t,
		func(context.Context) (upgrade.CheckResult, error) {
			return upgrade.CheckResult{Current: "2.9.0", Latest: "2.9.0", Available: false}, nil
		},
		func() upgrade.Method { return upgrade.MethodStandalone },
		nil,
	)

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--upgrade"})
	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "up to date")
}
