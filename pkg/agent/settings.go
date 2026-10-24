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
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/template"
)

// settingSpec is a command asset's setting: block. The command reads or sets
// one session setting; what it prints is the asset's templates, over the
// setting's state (value plus the setting's own fields) and the typed
// argument (.arg).
type settingSpec struct {
	Key string `yaml:"key"` // a sessionSettings entry
	// Values are the arguments the command accepts (matched without case);
	// each is stored as the setting's new value.
	Values  []string `yaml:"values"`
	Show    string   `yaml:"show"`    // printed with no argument
	Set     string   `yaml:"set"`     // printed after a value is stored
	Invalid string   `yaml:"invalid"` // printed for an argument not in values
}

// sessionSetting reads and changes one piece of live REPL state. get returns
// the state its command templates see: "value", plus any field the setting
// adds. set stores one of the command's values.
type sessionSetting struct {
	get func(r *REPL) map[string]any
	set func(r *REPL, value string)
}

// sessionSettings are the settings a command asset's setting: block may name.
// They are kernel state; which values exist and what is printed is the asset's.
//
//nolint:gochecknoglobals // fixed table of live REPL state
var sessionSettings = map[string]sessionSetting{
	"refine": {
		get: func(r *REPL) map[string]any { return onOffState(r.loop.PromptRefineEnabled()) },
		set: func(r *REPL, v string) { r.loop.SetPromptRefine(v == toggleOn) },
	},
	"handshake": {
		get: func(r *REPL) map[string]any { return onOffState(r.loop.HandshakeEnabled()) },
		set: func(r *REPL, v string) { r.loop.SetHandshakeEnabled(v == toggleOn) },
	},
	"autocontext": {
		get: func(r *REPL) map[string]any { return onOffState(r.autoContextDetect) },
		set: func(r *REPL, v string) { r.autoContextDetect = v == toggleOn },
	},
	"permission": {
		get: func(r *REPL) map[string]any {
			mode := r.loop.config.PermissionMode
			if mode == "" {
				mode = resolvePermissionMode()
			}
			return map[string]any{"value": string(mode)}
		},
		set: func(r *REPL, v string) { r.loop.config.PermissionMode = PermissionMode(v) },
	},
	"tools": {
		get: func(r *REPL) map[string]any {
			value := "lean"
			if r.toolsFullMode {
				value = "full"
			}
			// No filter: lean mode excluded nothing at startup, so there is
			// nothing to switch.
			return map[string]any{"value": value, "count": r.toolsCount, "available": r.toolsFilterFn != nil}
		},
		set: func(r *REPL, v string) {
			if r.toolsFilterFn == nil {
				return
			}
			r.toolsFullMode = v == "full"
			r.toolsCount = r.toolsFilterFn(r.toolsFullMode)
		},
	},
}

func onOffState(on bool) map[string]any {
	if on {
		return map[string]any{"value": toggleOn}
	}
	return map[string]any{"value": toggleOff}
}

// validate checks the block names a setting, lists values and has templates
// that parse.
func (s *settingSpec) validate() error {
	if _, ok := sessionSettings[s.Key]; !ok {
		return fmt.Errorf("setting: unknown key %q", s.Key)
	}
	if len(s.Values) == 0 {
		return errors.New("setting: values is required")
	}
	for name, src := range map[string]string{"show": s.Show, "set": s.Set, "invalid": s.Invalid} {
		if strings.TrimSpace(src) == "" {
			return fmt.Errorf("setting: %s is required", name)
		}
		if _, err := template.New(name).Parse(src); err != nil {
			return fmt.Errorf("setting: %s: %w", name, err)
		}
	}
	return nil
}

// runSettingCommand shows the setting with no argument, stores a listed
// value and persists it, or reports an unlisted one.
func (r *REPL) runSettingCommand(c *slashCommand, args []string) error {
	s := c.Setting
	setting := sessionSettings[s.Key]
	if len(args) == 0 {
		return printSetting(s.Show, setting.get(r), "", styleReplMeta.Render)
	}
	arg := strings.ToLower(strings.TrimSpace(args[0]))
	if !slices.Contains(s.Values, arg) {
		return printSetting(s.Invalid, setting.get(r), args[0], styleReplError.Render)
	}
	setting.set(r, arg)
	r.persistTuning() // survive across sessions
	return printSetting(s.Set, setting.get(r), arg, styleReplSuccess.Render)
}

func printSetting(src string, state map[string]any, arg string, style func(...string) string) error {
	tmpl, err := template.New("setting").Parse(src)
	if err != nil {
		return err
	}
	data := map[string]any{"arg": arg}
	for k, v := range state {
		data[k] = v
	}
	var sb strings.Builder
	if err = tmpl.Execute(&sb, data); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, style(strings.TrimSpace(sb.String())))
	return nil
}
