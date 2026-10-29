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

package agent

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/assets"
)

// PermissionMode controls which tools the agent is allowed to call.
// Higher modes are strict supersets: read-only < workspace-write < danger-full-access.
type PermissionMode string

const (
	// PermissionReadOnly allows only read operations (file reads, searches,
	// web lookups). Writes and command execution are blocked.
	PermissionReadOnly PermissionMode = "read-only"

	// PermissionWorkspaceWrite allows read operations plus file writes and
	// command execution within the workspace.
	PermissionWorkspaceWrite PermissionMode = "workspace-write"

	// PermissionDangerFullAccess allows all operations with no restrictions.
	// Equivalent to running without the permission system. This is the default.
	PermissionDangerFullAccess PermissionMode = "danger-full-access"

	// PermissionAsk allows read-only tools freely, but prompts the user before
	// each mutating tool call (allow once / allow always / deny). "Allow always"
	// mints a session-scoped approval token (see approval_tokens.go) so the same
	// tool is not re-prompted. It requires an interactive terminal; the loop's
	// streaming gate handles the prompting. On non-interactive paths there is no
	// way to prompt, so Allow treats ask as read-only-only (mutating denied), and
	// the loop falls back to the static env mode instead.
	PermissionAsk PermissionMode = "ask"
)

// Numeric ranks for mode comparison. Unknown modes rank below read-only,
// so a typo denies rather than allows.
const (
	levelUnknownMode    = -1
	levelReadOnly       = 0
	levelWorkspaceWrite = 1
	levelFullAccess     = 2
)

// permissionLevel assigns a numeric rank to each mode for comparison.
func permissionLevel(m PermissionMode) int {
	switch m {
	case PermissionReadOnly:
		return levelReadOnly
	case PermissionWorkspaceWrite:
		return levelWorkspaceWrite
	case PermissionDangerFullAccess:
		return levelFullAccess
	case PermissionAsk:
		// ask is an interactive policy, not a static rank; it never reaches the
		// numeric ladder (the loop gate intercepts it). Rank it as unknown so any
		// accidental comparison denies rather than allows.
		return levelUnknownMode
	default:
		return levelUnknownMode
	}
}

// permissionPolicy is the assets/policy/permission.yaml file: the one tool
// policy. Tiers name the mode each tier needs; tools name their tier.
type permissionPolicy struct {
	Default string            `yaml:"default"`
	Tiers   map[string]string `yaml:"tiers"`
	Tools   map[string]string `yaml:"tools"`
}

// parsePermissionPolicy reads the policy file. An unknown default is an error,
// so a typo cannot silently change what runs.
func parsePermissionPolicy(data []byte) (permissionPolicy, error) {
	var p permissionPolicy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return permissionPolicy{}, err
	}
	if p.Default != "allow" && p.Default != "deny" {
		return permissionPolicy{}, fmt.Errorf("permission policy: default must be allow or deny, got %q", p.Default)
	}
	for tier, mode := range p.Tiers {
		switch PermissionMode(mode) {
		case PermissionReadOnly, PermissionWorkspaceWrite, PermissionDangerFullAccess:
		case PermissionAsk:
			return permissionPolicy{}, fmt.Errorf("permission policy: tier %s cannot name ask", tier)
		default:
			return permissionPolicy{}, fmt.Errorf("permission policy: tier %s names unknown mode %q", tier, mode)
		}
	}
	return p, nil
}

// permissionPolicyCache holds the loaded policy until the assets are reloaded.
//
//nolint:gochecknoglobals // one parsed policy per process, reset by ReloadAssets
var permissionPolicyCache struct {
	sync.Mutex
	policy *permissionPolicy
}

// initPermissionPolicy drops the cached policy so the next call reads the
// assets again. ReloadAssets calls it.
func initPermissionPolicy() {
	permissionPolicyCache.Lock()
	defer permissionPolicyCache.Unlock()
	permissionPolicyCache.policy = nil
}

// loadPermissionPolicy returns the policy from the policy asset set, read once
// and cached. A missing or invalid file falls back to allow with
// workspace-write, the behaviour before the file existed.
//
//kdeps:flow policy: a tool call is about to run > the agent loop reads the permission policy > [its tool tier names a mode, or the default applies] > the call runs or is refused
func loadPermissionPolicy() permissionPolicy {
	permissionPolicyCache.Lock()
	defer permissionPolicyCache.Unlock()
	if permissionPolicyCache.policy != nil {
		return *permissionPolicyCache.policy
	}
	p := readPermissionPolicy()
	permissionPolicyCache.policy = &p
	return p
}

// userPermissionPolicyPath is ~/.kdeps/policy/permission.yaml, the user's
// override of the built-in policy.
func userPermissionPolicyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kdeps", "policy", "permission.yaml"), nil
}

// readPermissionPolicy reads the user's override when it exists, else the
// built-in policy asset. A user file that does not parse fails closed to deny:
// a broken override must not widen what runs.
func readPermissionPolicy() permissionPolicy {
	if path, err := userPermissionPolicyPath(); err == nil {
		if data, readErr := os.ReadFile(path); readErr == nil {
			p, parseErr := parsePermissionPolicy(data)
			if parseErr != nil {
				warnAsset(parseErr)
				return permissionPolicy{Default: "deny"}
			}
			return p
		}
	}
	data, err := fs.ReadFile(assets.Open("policy"), "policy/permission.yaml")
	if err != nil {
		warnAsset(err)
		return permissionPolicy{Default: "allow"}
	}
	p, err := parsePermissionPolicy(data)
	if err != nil {
		warnAsset(err)
		return permissionPolicy{Default: "allow"}
	}
	return p
}

// requiredPermissionIn returns the minimum mode a tool needs under policy.
// A tier that names no known mode needs danger-full-access (fail closed).
func requiredPermissionIn(policy permissionPolicy, toolName string) PermissionMode {
	if tier, listed := policy.Tools[toolName]; listed {
		if mode, ok := policy.Tiers[tier]; ok {
			return PermissionMode(mode)
		}
		return PermissionDangerFullAccess
	}
	if policy.Default == "deny" {
		return PermissionDangerFullAccess
	}
	return PermissionWorkspaceWrite
}

// requiredPermission returns the minimum permission mode a tool needs under
// the loaded policy. Workflow, component and agency tools are unlisted, so
// they follow the policy default.
func requiredPermission(toolName string) PermissionMode {
	return requiredPermissionIn(loadPermissionPolicy(), toolName)
}

// PermissionEnforcer checks whether a tool call is allowed in the current
// permission mode.
type PermissionEnforcer struct {
	mode PermissionMode
}

// NewPermissionEnforcer creates an enforcer for the given mode. An empty mode
// falls back to KDEPS_PERMISSION_MODE, then to danger-full-access.
func NewPermissionEnforcer(mode PermissionMode) *PermissionEnforcer {
	if mode == "" {
		mode = resolvePermissionMode()
	}
	return &PermissionEnforcer{mode: mode}
}

// resolvePermissionMode reads KDEPS_PERMISSION_MODE from the environment,
// falling back to danger-full-access (unrestricted).
func resolvePermissionMode() PermissionMode {
	raw := PermissionMode(strings.ToLower(strings.TrimSpace(os.Getenv("KDEPS_PERMISSION_MODE"))))
	switch raw {
	case PermissionReadOnly, PermissionWorkspaceWrite, PermissionDangerFullAccess, PermissionAsk:
		return raw
	default:
		return PermissionDangerFullAccess
	}
}

// resolveStaticFallbackMode resolves the static mode to use when ask mode is
// active but no interactive terminal is available to prompt. It reads
// KDEPS_PERMISSION_MODE like resolvePermissionMode, but maps ask (and any
// empty/unknown value) to danger-full-access so headless runs never block.
func resolveStaticFallbackMode() PermissionMode {
	if m := resolvePermissionMode(); m != PermissionAsk {
		return m
	}
	return PermissionDangerFullAccess
}

// Mode returns the current permission mode.
func (e *PermissionEnforcer) Mode() PermissionMode { return e.mode }

// Allow checks whether the given tool is allowed in the current permission mode.
// Returns (true, "") if allowed, or (false, reason) if denied.
func (e *PermissionEnforcer) Allow(toolName string) (bool, string) {
	if e.mode == PermissionDangerFullAccess {
		return true, ""
	}
	// Ask mode has no way to prompt on this path (the loop's interactive gate
	// handles prompting before calling Allow). Treat it as read-only-only:
	// read-only tools pass, mutating tools are denied.
	if e.mode == PermissionAsk {
		if requiredPermission(toolName) == PermissionReadOnly {
			return true, ""
		}
		return false, fmt.Sprintf(
			"tool %q requires interactive approval in mode %q, which is unavailable here",
			toolName, e.mode,
		)
	}
	required := requiredPermission(toolName)
	if permissionLevel(e.mode) >= permissionLevel(required) {
		return true, ""
	}
	return false, fmt.Sprintf(
		"tool %q requires permission %q but current mode is %q (set KDEPS_PERMISSION_MODE or Config.PermissionMode)",
		toolName, required, e.mode,
	)
}
