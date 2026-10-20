// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law and agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package agent

// MINE-07/08/09: Task, Team, Cron, and Approval Token tools.
// Wires the real in-memory registries (GlobalTaskRegistry, GlobalTeamRegistry,
// GlobalCronRegistry, GlobalApprovalTokenRegistry) as callable LLM tools.

import (
	"context"
	"fmt"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/api"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

// The call route lives in pkg/api. This package owns the registries, so the
// handler reaches them through the setters.
//
//nolint:gochecknoinits // wires task and team tools before any call
func init() {
	api.SetTaskBackend(taskAPIBackend{})
	api.SetTeamBackend(teamAPIBackend{})
}

// Approval token default duration.
const approvalTokenDuration = 5 * time.Minute

// registerTaskTeamTools registers task/team/cron/approval-token management tools.
// These are always available in the agent loop (not filtered by lean mode).
func registerTaskTeamTools(ctx context.Context, reg *kdepstools.Registry) {
	registerTaskTools(ctx, reg)
	registerTeamTools(ctx, reg)
	registerCronTools(reg)
	registerApprovalTokenTools(reg)
}

// --- Task Tools ---

func registerTaskTools(ctx context.Context, reg *kdepstools.Registry) {
	registerAPITool(ctx, reg, "task_create", api.TaskCreate)
	registerAPITool(ctx, reg, "task_get", api.TaskGet)
	registerAPITool(ctx, reg, "task_list", api.TaskList)
	registerAPITool(ctx, reg, "task_stop", api.TaskStop)
	registerAPITool(ctx, reg, "task_complete", api.TaskComplete)
	registerAPITool(ctx, reg, "task_append_output", api.TaskAppendOutput)
	registerAPITool(ctx, reg, "task_assign_team", api.TaskAssignTeam)
}

// --- Team Tools ---

func registerTeamTools(ctx context.Context, reg *kdepstools.Registry) {
	registerAPITool(ctx, reg, "team_create", api.TeamCreate)
	registerAPITool(ctx, reg, "team_get", api.TeamGet)
	registerAPITool(ctx, reg, "team_list", api.TeamList)
	registerAPITool(ctx, reg, "team_add_task", api.TeamAddTask)
	registerAPITool(ctx, reg, "team_delete", api.TeamDelete)
}

type taskAPIBackend struct{}

func (taskAPIBackend) CreateTask(prompt, description string) api.TaskView {
	return taskView(GlobalTaskRegistry.Create(prompt, description))
}

func (taskAPIBackend) GetTask(id string) (api.TaskView, bool) {
	task := GlobalTaskRegistry.Get(id)
	if task == nil {
		return api.TaskView{}, false
	}
	return taskView(task), true
}

func (taskAPIBackend) ListTasks() []api.TaskView {
	tasks := GlobalTaskRegistry.List()
	out := make([]api.TaskView, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, taskView(task))
	}
	return out
}

func (taskAPIBackend) StopTask(id string) bool {
	return GlobalTaskRegistry.Stop(id)
}

func (taskAPIBackend) CompleteTask(id string) bool {
	return GlobalTaskRegistry.SetStatus(id, TaskCompleted)
}

func (taskAPIBackend) AppendTaskOutput(id, text string) bool {
	return GlobalTaskRegistry.AppendOutput(id, text)
}

func (taskAPIBackend) AssignTaskTeam(taskID, teamID string) bool {
	return GlobalTaskRegistry.AssignTeam(taskID, teamID)
}

func taskView(task *Task) api.TaskView {
	return api.TaskView{
		ID:          task.TaskID,
		Status:      task.Status.String(),
		Description: task.Description,
		Output:      task.Output,
		TeamID:      task.TeamID,
		CreatedAt:   task.CreatedAt,
	}
}

type teamAPIBackend struct{}

func (teamAPIBackend) CreateTeam(name string) api.TeamView {
	return teamView(GlobalTeamRegistry.Create(name))
}

func (teamAPIBackend) GetTeam(id string) (api.TeamView, bool) {
	team := GlobalTeamRegistry.Get(id)
	if team == nil {
		return api.TeamView{}, false
	}
	return teamView(team), true
}

func (teamAPIBackend) ListTeams() []api.TeamView {
	teams := GlobalTeamRegistry.List()
	out := make([]api.TeamView, 0, len(teams))
	for _, team := range teams {
		out = append(out, teamView(team))
	}
	return out
}

func (teamAPIBackend) AddTeamTask(teamID, taskID string) bool {
	return GlobalTeamRegistry.AddTask(teamID, taskID)
}

func (teamAPIBackend) DeleteTeam(id string) bool {
	return GlobalTeamRegistry.Delete(id)
}

func teamView(team *Team) api.TeamView {
	return api.TeamView{
		ID:        team.TeamID,
		Name:      team.Name,
		Status:    team.Status.String(),
		TaskCount: len(team.TaskIDs),
	}
}

// --- Cron Tools ---

func registerCronTools(reg *kdepstools.Registry) {
	reg.Register(defined(&kdepstools.Tool{
		Name: "cron_create",
		Execute: func(args map[string]any) (string, error) {
			name, _ := args["name"].(string)
			expr, _ := args["expression"].(string)
			prompt, _ := args["task_prompt"].(string)
			desc, _ := args["task_description"].(string)
			cron := GlobalCronRegistry.Create(name, expr, prompt, desc)
			return fmt.Sprintf("Created cron %s: %s (%s)", cron.CronID, cron.Name, cron.Expression), nil
		},
	}))

	reg.Register(defined(&kdepstools.Tool{
		Name: "cron_list",
		Execute: func(_ map[string]any) (string, error) {
			return GlobalCronRegistry.CronSummary(), nil
		},
	}))

	reg.Register(defined(&kdepstools.Tool{
		Name: "cron_pause",
		Execute: func(args map[string]any) (string, error) {
			cronID, _ := args["cron_id"].(string)
			if GlobalCronRegistry.Pause(cronID) {
				return fmt.Sprintf("Paused cron %s", cronID), nil
			}
			return "", fmt.Errorf("cron %q not found", cronID)
		},
	}))

	reg.Register(defined(&kdepstools.Tool{
		Name: "cron_resume",
		Execute: func(args map[string]any) (string, error) {
			cronID, _ := args["cron_id"].(string)
			if GlobalCronRegistry.Resume(cronID) {
				return fmt.Sprintf("Resumed cron %s", cronID), nil
			}
			return "", fmt.Errorf("cron %q not found", cronID)
		},
	}))

	reg.Register(defined(&kdepstools.Tool{
		Name: "cron_delete",
		Execute: func(args map[string]any) (string, error) {
			cronID, _ := args["cron_id"].(string)
			if GlobalCronRegistry.Delete(cronID) {
				return fmt.Sprintf("Deleted cron %s", cronID), nil
			}
			return "", fmt.Errorf("cron %q not found", cronID)
		},
	}))
}

// --- Approval Token Tools ---

func registerApprovalTokenTools(reg *kdepstools.Registry) {
	reg.Register(defined(&kdepstools.Tool{
		Name: "approval_request",
		Execute: func(args map[string]any) (string, error) {
			toolName, _ := args["tool_name"].(string)
			action, _ := args["action"].(string)
			scope := ApprovalScope{
				ToolName: toolName,
				Action:   action,
			}
			token := GlobalApprovalTokenRegistry.Request(scope, approvalTokenDuration)
			return fmt.Sprintf("Requested approval token %s for tool=%q action=%q\nAsk the user to grant it.",
				token.TokenID, toolName, action), nil
		},
	}))

	reg.Register(defined(&kdepstools.Tool{
		Name: "approval_grant",
		Execute: func(args map[string]any) (string, error) {
			tokenID, _ := args["token_id"].(string)
			if GlobalApprovalTokenRegistry.Grant(tokenID, "agent", "", "user approved") {
				return fmt.Sprintf("Granted approval token %s", tokenID), nil
			}
			return "", fmt.Errorf("token %q not found or not in pending state", tokenID)
		},
	}))

	reg.Register(defined(&kdepstools.Tool{
		Name: "approval_list",
		Execute: func(_ map[string]any) (string, error) {
			return GlobalApprovalTokenRegistry.TokenSummary(), nil
		},
	}))

	reg.Register(defined(&kdepstools.Tool{
		Name: "approval_revoke",
		Execute: func(args map[string]any) (string, error) {
			tokenID, _ := args["token_id"].(string)
			if GlobalApprovalTokenRegistry.Revoke(tokenID) {
				return fmt.Sprintf("Revoked approval token %s", tokenID), nil
			}
			return "", fmt.Errorf("token %q not found or already consumed", tokenID)
		},
	}))
}
