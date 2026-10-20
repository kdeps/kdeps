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

package api

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	toolTaskAppendOutput = "task_append_output"
	toolTaskAssignTeam   = "task_assign_team"
	toolTaskComplete     = "task_complete"
	toolTaskCreate       = "task_create"
	toolTaskGet          = "task_get"
	toolTaskList         = "task_list"
	toolTaskStop         = "task_stop"
	toolTeamAddTask      = "team_add_task"
	toolTeamCreate       = "team_create"
	toolTeamDelete       = "team_delete"
	toolTeamGet          = "team_get"
	toolTeamList         = "team_list"

	argTaskID      = "task_id"
	argTeamID      = "team_id"
	argPrompt      = "prompt"
	argDescription = "description"
	argText        = "text"

	msgNoTasks      = "No tasks."
	msgNoTeams      = "No teams."
	msgTaskRegistry = "task registry is not configured"
	msgTeamRegistry = "team registry is not configured"
)

// TaskView is the task text the call route prints.
type TaskView struct {
	ID          string
	Status      string
	Description string
	Output      string
	TeamID      string
	CreatedAt   time.Time
}

// TeamView is the team text the call route prints.
type TeamView struct {
	ID        string
	Name      string
	Status    string
	TaskCount int
}

// TaskBackend is the process task registry. The agent installs it.
type TaskBackend interface {
	CreateTask(prompt, description string) TaskView
	GetTask(id string) (TaskView, bool)
	ListTasks() []TaskView
	StopTask(id string) bool
	CompleteTask(id string) bool
	AppendTaskOutput(id, text string) bool
	AssignTaskTeam(taskID, teamID string) bool
}

// TeamBackend is the process team registry. The agent installs it.
type TeamBackend interface {
	CreateTeam(name string) TeamView
	GetTeam(id string) (TeamView, bool)
	ListTeams() []TeamView
	AddTeamTask(teamID, taskID string) bool
	DeleteTeam(id string) bool
}

//nolint:gochecknoglobals // process registries installed by the agent
var (
	taskBackend TaskBackend
	teamBackend TeamBackend
)

func registerRegistryTools() {
	RegisterTool(loadToolSpec(toolTaskAppendOutput, "Append text to a task.", TaskAppendOutput))
	RegisterTool(loadToolSpec(toolTaskAssignTeam, "Assign a task to a team.", TaskAssignTeam))
	RegisterTool(loadToolSpec(toolTaskComplete, "Mark a task completed.", TaskComplete))
	RegisterTool(loadToolSpec(toolTaskCreate, "Create a task.", TaskCreate))
	RegisterTool(loadToolSpec(toolTaskGet, "Read one task.", TaskGet))
	RegisterTool(loadToolSpec(toolTaskList, "List tasks.", TaskList))
	RegisterTool(loadToolSpec(toolTaskStop, "Stop a task.", TaskStop))
	RegisterTool(loadToolSpec(toolTeamAddTask, "Add a task to a team.", TeamAddTask))
	RegisterTool(loadToolSpec(toolTeamCreate, "Create a team.", TeamCreate))
	RegisterTool(loadToolSpec(toolTeamDelete, "Delete a team.", TeamDelete))
	RegisterTool(loadToolSpec(toolTeamGet, "Read one team.", TeamGet))
	RegisterTool(loadToolSpec(toolTeamList, "List teams.", TeamList))
	registerCronTools()
	registerApprovalTools()
	registerInstructTool()
}

// SetTaskBackend installs the process task registry. A nil backend makes
// every task call return not configured.
func SetTaskBackend(backend TaskBackend) {
	taskBackend = backend
}

// SetTeamBackend installs the process team registry. A nil backend makes
// every team call return not configured.
func SetTeamBackend(backend TeamBackend) {
	teamBackend = backend
}

func taskReady(ctx context.Context, name string) error {
	if err := ctxReady(ctx, name); err != nil {
		return err
	}
	if taskBackend == nil {
		return fmt.Errorf("%s: %s", name, msgTaskRegistry)
	}
	return nil
}

func teamReady(ctx context.Context, name string) error {
	if err := ctxReady(ctx, name); err != nil {
		return err
	}
	if teamBackend == nil {
		return fmt.Errorf("%s: %s", name, msgTeamRegistry)
	}
	return nil
}

// TaskCreate adds one task and returns its id and description.
func TaskCreate(ctx context.Context, args map[string]any) (string, error) {
	if err := taskReady(ctx, toolTaskCreate); err != nil {
		return "", err
	}
	task := taskBackend.CreateTask(stringArg(args, argPrompt), stringArg(args, argDescription))
	return fmt.Sprintf("Created task %s: %s", task.ID, task.Description), nil
}

// TaskGet returns one task. An unknown id is `task "<id>" not found`.
func TaskGet(ctx context.Context, args map[string]any) (string, error) {
	if err := taskReady(ctx, toolTaskGet); err != nil {
		return "", err
	}
	id := stringArg(args, argTaskID)
	task, ok := taskBackend.GetTask(id)
	if !ok {
		return "", fmt.Errorf("task %q not found", id)
	}
	return fmt.Sprintf("Task: %s\nStatus: %s\nDescription: %s\nCreated: %s\nOutput: %s\nTeam: %s",
		task.ID, task.Status, task.Description, task.CreatedAt.Format(time.RFC3339), task.Output, task.TeamID), nil
}

// TaskList returns every task, one line each, or `No tasks.`.
func TaskList(ctx context.Context, _ map[string]any) (string, error) {
	if err := taskReady(ctx, toolTaskList); err != nil {
		return "", err
	}
	tasks := taskBackend.ListTasks()
	if len(tasks) == 0 {
		return msgNoTasks, nil
	}
	var sb strings.Builder
	for _, task := range tasks {
		fmt.Fprintf(&sb, "%s | %s | %s\n", task.ID, task.Status, task.Description)
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// TaskStop marks one task stopped.
func TaskStop(ctx context.Context, args map[string]any) (string, error) {
	if err := taskReady(ctx, toolTaskStop); err != nil {
		return "", err
	}
	id := stringArg(args, argTaskID)
	if taskBackend.StopTask(id) {
		return fmt.Sprintf("Stopped task %s", id), nil
	}
	return "", fmt.Errorf("task %q not found", id)
}

// TaskComplete marks one task completed.
func TaskComplete(ctx context.Context, args map[string]any) (string, error) {
	if err := taskReady(ctx, toolTaskComplete); err != nil {
		return "", err
	}
	id := stringArg(args, argTaskID)
	if taskBackend.CompleteTask(id) {
		return fmt.Sprintf("Completed task %s", id), nil
	}
	return "", fmt.Errorf("task %q not found", id)
}

// TaskAppendOutput appends text to one task.
func TaskAppendOutput(ctx context.Context, args map[string]any) (string, error) {
	if err := taskReady(ctx, toolTaskAppendOutput); err != nil {
		return "", err
	}
	id := stringArg(args, argTaskID)
	if taskBackend.AppendTaskOutput(id, stringArg(args, argText)) {
		return fmt.Sprintf("Appended output to task %s", id), nil
	}
	return "", fmt.Errorf("task %q not found", id)
}

// TaskAssignTeam sets a task's team.
func TaskAssignTeam(ctx context.Context, args map[string]any) (string, error) {
	if err := taskReady(ctx, toolTaskAssignTeam); err != nil {
		return "", err
	}
	taskID := stringArg(args, argTaskID)
	teamID := stringArg(args, argTeamID)
	if taskBackend.AssignTaskTeam(taskID, teamID) {
		return fmt.Sprintf("Assigned task %s to team %s", taskID, teamID), nil
	}
	return "", fmt.Errorf("task %q not found", taskID)
}

// TeamCreate adds one team and returns its id and name.
func TeamCreate(ctx context.Context, args map[string]any) (string, error) {
	if err := teamReady(ctx, toolTeamCreate); err != nil {
		return "", err
	}
	team := teamBackend.CreateTeam(stringArg(args, argName))
	return fmt.Sprintf("Created team %s: %s", team.ID, team.Name), nil
}

// TeamGet returns one team. An unknown id is `team "<id>" not found`.
func TeamGet(ctx context.Context, args map[string]any) (string, error) {
	if err := teamReady(ctx, toolTeamGet); err != nil {
		return "", err
	}
	id := stringArg(args, argTeamID)
	team, ok := teamBackend.GetTeam(id)
	if !ok {
		return "", fmt.Errorf("team %q not found", id)
	}
	return fmt.Sprintf("Team: %s\nName: %s\nStatus: %s\nTasks: %d",
		team.ID, team.Name, team.Status, team.TaskCount), nil
}

// TeamList returns every team, one line each, or `No teams.`.
func TeamList(ctx context.Context, _ map[string]any) (string, error) {
	if err := teamReady(ctx, toolTeamList); err != nil {
		return "", err
	}
	teams := teamBackend.ListTeams()
	if len(teams) == 0 {
		return msgNoTeams, nil
	}
	var sb strings.Builder
	for _, team := range teams {
		fmt.Fprintf(&sb, "%s | %s | %s | tasks=%d\n", team.ID, team.Name, team.Status, team.TaskCount)
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// TeamAddTask adds a task id to a team.
func TeamAddTask(ctx context.Context, args map[string]any) (string, error) {
	if err := teamReady(ctx, toolTeamAddTask); err != nil {
		return "", err
	}
	teamID := stringArg(args, argTeamID)
	taskID := stringArg(args, argTaskID)
	if teamBackend.AddTeamTask(teamID, taskID) {
		return fmt.Sprintf("Added task %s to team %s", taskID, teamID), nil
	}
	return "", fmt.Errorf("team %q not found", teamID)
}

// TeamDelete marks one team deleted.
func TeamDelete(ctx context.Context, args map[string]any) (string, error) {
	if err := teamReady(ctx, toolTeamDelete); err != nil {
		return "", err
	}
	id := stringArg(args, argTeamID)
	if teamBackend.DeleteTeam(id) {
		return fmt.Sprintf("Deleted team %s", id), nil
	}
	return "", fmt.Errorf("team %q not found", id)
}
