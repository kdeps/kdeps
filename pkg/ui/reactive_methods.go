package ui

import (
	"fmt"
	"time"

	"github.com/kdeps/kdeps/pkg/reactive"
)

// syncFromReactiveState synchronizes UI state with reactive state
func (m *LiveGUIModel) syncFromReactiveState(state reactive.AppState) {
	// Update loading state
	if state.Loading != (m.err != nil && !m.done) {
		// Sync loading state
	}

	// Update error state
	if state.Error != "" && m.err == nil {
		m.err = fmt.Errorf("%s", state.Error)
	} else if state.Error == "" && m.err != nil {
		m.err = nil
	}

	// Sync operations from reactive state
	m.syncOperationsFromReactiveState(state.Operations)

	// Update logs
	if len(state.Logs) > len(m.dockerLogs) {
		m.syncLogsFromReactiveState(state.Logs)
	}
}

// syncOperationsFromReactiveState syncs operations from reactive state
func (m *LiveGUIModel) syncOperationsFromReactiveState(reactiveOps map[string]reactive.Operation) {
	// Update existing operations or create new ones
	for id, reactiveOp := range reactiveOps {
		found := false
		for i, op := range m.operations {
			if op.Name == id {
				// Update existing operation
				m.operations[i].Status = m.convertReactiveStatusToUIStatus(reactiveOp.Status)
				m.operations[i].Progress = reactiveOp.Progress
				m.operations[i].Description = fmt.Sprintf("%s - %s", reactiveOp.Type, reactiveOp.Status)
				found = true
				break
			}
		}

		if !found {
			// Add new operation
			newOp := Operation{
				Name:        id,
				Description: fmt.Sprintf("%s - %s", reactiveOp.Type, reactiveOp.Status),
				Status:      m.convertReactiveStatusToUIStatus(reactiveOp.Status),
				Progress:    reactiveOp.Progress,
				StartTime:   time.Unix(reactiveOp.StartTime, 0),
				Logs:        []string{},
			}

			if reactiveOp.EndTime > 0 {
				newOp.EndTime = time.Unix(reactiveOp.EndTime, 0)
			}

			m.operations = append(m.operations, newOp)
		}
	}

	// Update global progress
	m.updateGlobalProgress()
}

// syncLogsFromReactiveState syncs logs from reactive state
func (m *LiveGUIModel) syncLogsFromReactiveState(reactiveLogs []reactive.LogEntry) {
	// Clear existing logs and add new ones
	m.dockerLogs = make([]string, 0, len(reactiveLogs))

	for _, log := range reactiveLogs {
		timestamp := time.Unix(log.Timestamp, 0).Format("15:04:05")
		logLine := fmt.Sprintf("[%s] [%s] %s", timestamp, log.Level, log.Message)

		if log.Source != "" {
			logLine = fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, log.Level, log.Source, log.Message)
		}

		m.dockerLogs = append(m.dockerLogs, logLine)
	}

	// Trim logs if too many
	if len(m.dockerLogs) > m.maxLogs {
		m.dockerLogs = m.dockerLogs[len(m.dockerLogs)-m.maxLogs:]
	}

	// Update viewport
	m.updateViewport()
}

// convertReactiveStatusToUIStatus converts reactive operation status to UI status
func (m *LiveGUIModel) convertReactiveStatusToUIStatus(reactiveStatus string) OperationStatus {
	switch reactiveStatus {
	case "running":
		return StatusRunning
	case "completed":
		return StatusCompleted
	case "error":
		return StatusError
	default:
		return StatusPending
	}
}

// Reactive state update methods

// UpdateOperationReactive updates an operation in the reactive state
func (m *LiveGUIModel) UpdateOperationReactive(id, opType, status string, progress float64) {
	if m.reactiveStore == nil {
		return
	}

	op := reactive.Operation{
		ID:       id,
		Type:     opType,
		Status:   status,
		Progress: progress,
	}

	if status == "running" {
		op.StartTime = time.Now().Unix()
	} else if status == "completed" || status == "error" {
		op.EndTime = time.Now().Unix()
	}

	// Check if operation exists, update or add
	state := m.reactiveStore.GetState()
	if _, exists := state.Operations[id]; exists {
		m.reactiveStore.Dispatch(reactive.UpdateOperation(op))
	} else {
		m.reactiveStore.Dispatch(reactive.AddOperation(op))
	}
}

// AddLogReactive adds a log entry to the reactive state
func (m *LiveGUIModel) AddLogReactive(level, message, source string) {
	if m.reactiveStore == nil {
		return
	}

	logEntry := reactive.LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now().Unix(),
		Source:    source,
	}

	m.reactiveStore.Dispatch(reactive.AddLog(logEntry))
}

// SetErrorReactive sets an error in the reactive state
func (m *LiveGUIModel) SetErrorReactive(err string) {
	if m.reactiveStore == nil {
		return
	}

	m.reactiveStore.Dispatch(reactive.SetError(err))
}

// SetLoadingReactive sets the loading state in the reactive state
func (m *LiveGUIModel) SetLoadingReactive(loading bool) {
	if m.reactiveStore == nil {
		return
	}

	m.reactiveStore.Dispatch(reactive.SetLoading(loading))
}

// ClearErrorReactive clears errors in the reactive state
func (m *LiveGUIModel) ClearErrorReactive() {
	if m.reactiveStore == nil {
		return
	}

	m.reactiveStore.Dispatch(reactive.ClearError())
}

// GetReactiveStore returns the reactive store for external access
func (m *LiveGUIModel) GetReactiveStore() *reactive.Store[reactive.AppState] {
	return m.reactiveStore
}

// Close cleans up reactive subscriptions
func (m *LiveGUIModel) Close() {
	if m.subscription != nil {
		m.subscription.Unsubscribe()
	}
	if m.reactiveStore != nil {
		m.reactiveStore.Close()
	}
}
