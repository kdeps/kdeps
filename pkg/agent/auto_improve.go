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
	"path/filepath"
	"strings"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/instrument"
)

const (
	kdepsModule        = "github.com/kdeps/kdeps/v2"
	autoImproveSession = "auto-improve"
	autoImproveNotRepo = "/auto-improve runs in the kdeps repo (module " + kdepsModule + "). This folder is %s"
)

// kdepsRepoRoot walks from start to the module root. An empty start uses
// the working directory. Another module stops the walk.
func kdepsRepoRoot(start string) (string, error) {
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		start = wd
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	start = abs
	dir := start
	for {
		data, readErr := os.ReadFile(filepath.Join(dir, "go.mod"))
		if readErr == nil {
			if moduleName(data) == kdepsModule {
				return dir, nil
			}
			return "", fmt.Errorf(autoImproveNotRepo, start)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(autoImproveNotRepo, start)
		}
		dir = parent
	}
}

func moduleName(data []byte) string {
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	return ""
}

// autoImproveTasks is the fixed list. Each task uses the tools, then the
// evaluate turn reads the instrument log.
func autoImproveTasks(root string) []string {
	return []string{
		fmt.Sprintf(
			"The kdeps repo root is %s.\n"+
				"Use the read_file tool on README.md. In one sentence, "+
				"name the two modes kdeps runs. Do not edit files.",
			root,
		),
		fmt.Sprintf(
			"The kdeps repo root is %s.\n"+
				"Use the bash_exec tool to run git status --short. "+
				"Report the output. Do not edit files.",
			root,
		),
		fmt.Sprintf(
			"The kdeps repo root is %s.\n"+
				"Use the code search tools to find the definition of instrument.Start. "+
				"Quote the function signature and the file. Do not edit files.",
			root,
		),
	}
}

func autoImproveEvalPrompt(dir string, n, total int) string {
	return fmt.Sprintf(
		"Read the instrument session in %s.\n"+
			"Read session.md and events.ndjson. Each event has a kind field.\n"+
			"This is evaluate step %d of %d, after the task that just ran.\n"+
			"Report product bugs shown in those events: a loop, a refusal of work "+
			"the tools can do, a call to the wrong tool, or a tool error that is wrong.\n"+
			"A model guess is not a product bug. "+
			"A shell rejection of a command the model wrote is not a product bug.\n"+
			"For each product bug, fix the code and add a test. "+
			"If there is no product bug, say so and do not edit files.",
		dir, n, total,
	)
}

func autoImproveCommand() *slashCommand {
	for _, c := range slashCommands {
		if c.Builtin && c.API == "auto-improve" {
			return c
		}
	}
	return nil
}

func lookupAutoImproveFocus(id string) (commandFocus, error) {
	c := autoImproveCommand()
	if c == nil {
		return commandFocus{}, errors.New("/auto-improve: focus list is not loaded")
	}
	ids := make([]string, len(c.Focus))
	for i, f := range c.Focus {
		ids[i] = f.ID
		if f.ID == id {
			return f, nil
		}
	}
	return commandFocus{}, fmt.Errorf("/auto-improve: unknown focus %q (%s)", id, strings.Join(ids, ", "))
}

func focusTasks(root string, tasks []string) []string {
	out := make([]string, len(tasks))
	for i, task := range tasks {
		out[i] = fmt.Sprintf(
			"The kdeps repo root is %s.\n%s\nDo not edit files.",
			root, strings.TrimSpace(task),
		)
	}
	return out
}

func autoImproveFocusEval(dir, brief string) string {
	return fmt.Sprintf(
		"Read the instrument session in %s.\n"+
			"Read session.md and events.ndjson. Each event has a kind field.\n"+
			"Focus for this run:\n%s\n"+
			"Fix one defect in that focus and add a test when the change is code.\n"+
			"A model guess is not a defect. "+
			"A shell rejection of a command the model wrote is not a defect.\n"+
			"If there is nothing to fix, say so and do not edit files.",
		dir, strings.TrimSpace(brief),
	)
}

// ensureAutoImproveSession starts the auto-improve session, or keeps the
// session this process is already recording.
func ensureAutoImproveSession(name string, args []string) (*instrument.Session, bool, error) {
	if s := instrument.Active(); s != nil {
		return s, false, nil
	}
	s, err := instrument.Start(name, args, time.Now())
	if err != nil {
		return nil, false, fmt.Errorf("/auto-improve: %w", err)
	}
	return s, true, nil
}

func resolveAutoImprove(args []string) (*commandFocus, string, error) {
	if len(args) > 1 {
		return nil, "", errors.New("/auto-improve takes one focus id")
	}
	root, err := kdepsRepoRoot("")
	if err != nil {
		return nil, "", err
	}
	if len(args) == 0 {
		return nil, root, nil
	}
	f, err := lookupAutoImproveFocus(args[0])
	if err != nil {
		return nil, "", err
	}
	return &f, root, nil
}

func autoImproveSessionName(focus *commandFocus) (string, []string, string) {
	name := autoImproveSession
	startArgs := []string{"/auto-improve"}
	line := "/auto-improve"
	if focus == nil {
		return name, startArgs, line
	}
	name += "-" + focus.ID
	startArgs = append(startArgs, focus.ID)
	line += " " + focus.ID
	return name, startArgs, line
}

func (r *REPL) cmdAutoImprove(args []string) error {
	focus, root, err := resolveAutoImprove(args)
	if err != nil {
		return err
	}
	name, startArgs, line := autoImproveSessionName(focus)
	session, started, err := ensureAutoImproveSession(name, startArgs)
	if err != nil {
		return err
	}
	r.history = append(r.history, line)
	if started {
		fmt.Fprintf(os.Stdout, "auto-improve: recording %s\n", session.Dir)
	} else {
		fmt.Fprintf(os.Stdout, "auto-improve: recording continues at %s\n", session.Dir)
	}
	if focus != nil {
		fmt.Fprintf(os.Stdout, "auto-improve: focus %s\n", focus.ID)
	}
	tasks := autoImproveTasks(root)
	if focus != nil {
		tasks = focusTasks(root, focus.Tasks)
	}
	if err = r.runAutoImproveTasks(session.Dir, tasks, focus); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "auto-improve: done")
	return nil
}

func (r *REPL) runAutoImproveTasks(dir string, tasks []string, focus *commandFocus) error {
	for i, task := range tasks {
		fmt.Fprintf(os.Stdout, "auto-improve: task %d/%d\n", i+1, len(tasks))
		if err := r.runAutoImproveTurn(task); err != nil {
			return err
		}
		if focus != nil {
			continue
		}
		fmt.Fprintf(os.Stdout, "auto-improve: evaluate %d/%d\n", i+1, len(tasks))
		prompt := autoImproveEvalPrompt(dir, i+1, len(tasks))
		if err := r.runAutoImproveTurn(prompt); err != nil {
			return err
		}
	}
	if focus == nil {
		return nil
	}
	fmt.Fprintln(os.Stdout, "auto-improve: evaluate 1/1")
	return r.runAutoImproveTurn(autoImproveFocusEval(dir, focus.Brief))
}

func (r *REPL) runAutoImproveTurn(prompt string) error {
	resp, err := r.runWithThinking(r.ctx, prompt)
	if err != nil {
		return fmt.Errorf("/auto-improve: %w", err)
	}
	if resp != "" {
		fmt.Fprint(os.Stdout, renderREPLOutput(resp, false))
	}
	return nil
}
