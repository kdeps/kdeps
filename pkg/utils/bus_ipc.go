package utils

import (
	"fmt"
	"net/rpc"
	"time"

	"github.com/kdeps/kdeps/pkg/bus"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// BusIPCManager manages IPC communication through the bus service
type BusIPCManager struct {
	client *rpc.Client
	logger *logging.Logger
}

// NewBusIPCManager creates a new bus IPC manager
func NewBusIPCManager(logger *logging.Logger) (*BusIPCManager, error) {
	client, err := bus.StartBusClient()
	if err != nil {
		return nil, fmt.Errorf("failed to start bus client: %w", err)
	}

	return &BusIPCManager{
		client: client,
		logger: logger,
	}, nil
}

// Close closes the bus connection
func (b *BusIPCManager) Close() error {
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}

// SignalResourceCompletion replaces timestamp-based completion signaling
func (b *BusIPCManager) SignalResourceCompletion(resourceID, resourceType, status string, data map[string]interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["resourceType"] = resourceType

	return bus.SignalResourceCompletion(b.client, resourceID, status, data)
}

// WaitForResourceCompletion replaces WaitForTimestampChange
func (b *BusIPCManager) WaitForResourceCompletion(resourceID string, timeoutSeconds int64) error {
	state, err := bus.WaitForResourceCompletion(b.client, resourceID, timeoutSeconds)
	if err != nil {
		return err
	}

	if state.Status == "failed" {
		return fmt.Errorf("resource %s failed", resourceID)
	}

	b.logger.Info("Resource completed via bus", "resourceID", resourceID, "status", state.Status)
	return nil
}

// SignalCleanup replaces file-based cleanup signaling
func (b *BusIPCManager) SignalCleanup(cleanupType, message string, data map[string]interface{}) error {
	eventType := "cleanup"
	if cleanupType == "docker" {
		eventType = "dockercleanup"
	}

	return bus.PublishEvent(b.client, eventType, message, "", data)
}

// WaitForCleanup replaces WaitForFileReady for cleanup files
func (b *BusIPCManager) WaitForCleanup(timeoutSeconds int64) error {
	return bus.WaitForCleanupSignal(b.client, b.logger, timeoutSeconds)
}

// SignalFileReady replaces file creation for signaling
func (b *BusIPCManager) SignalFileReady(filepath, operation string, data map[string]interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["filepath"] = filepath
	data["operation"] = operation

	return bus.PublishEvent(b.client, "file_ready", fmt.Sprintf("File %s ready for %s", filepath, operation), filepath, data)
}

// WaitForFileReady replaces the file-based WaitForFileReady function
func (b *BusIPCManager) WaitForFileReady(filepath string, timeoutSeconds int64) error {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	b.logger.Debug("Waiting for file ready signal via bus", "file", filepath)

	start := time.Now()
	return bus.WaitForEvents(b.client, b.logger, func(event bus.Event) bool {
		if time.Since(start) > timeout {
			return true // Stop and let timeout be handled by WaitForEvents
		}

		if event.Type == "file_ready" {
			if eventFilepath, ok := event.Data["filepath"].(string); ok && eventFilepath == filepath {
				b.logger.Info("File ready signal received via bus", "file", filepath)
				return true
			}
		}
		return false
	})
}

// Legacy wrapper functions for backwards compatibility

// WaitForFileReadyLegacy provides backwards compatibility with the old file-based approach
// This should be used during transition period only
func WaitForFileReadyLegacy(fs afero.Fs, filepath string, logger *logging.Logger) error {
	// Try bus-based approach first
	busManager, err := NewBusIPCManager(logger)
	if err != nil {
		logger.Debug("Bus not available, falling back to file-based approach", "error", err)
		return WaitForFileReady(fs, filepath, logger)
	}
	defer busManager.Close()

	// Set a shorter timeout for bus-based approach and fallback to file-based
	err = busManager.WaitForFileReady(filepath, 2)
	if err != nil {
		logger.Debug("Bus-based wait failed, falling back to file-based approach", "error", err)
		return WaitForFileReady(fs, filepath, logger)
	}

	return nil
}

// CreateFilesWithBusSignal creates files and signals via bus
func CreateFilesWithBusSignal(fs afero.Fs, busManager *BusIPCManager, files []string) error {
	err := CreateFiles(fs, nil, files)
	if err != nil {
		return err
	}

	// Signal file creation via bus
	for _, file := range files {
		if busManager != nil {
			if err := busManager.SignalFileReady(file, "create", nil); err != nil {
				// Log error but don't fail the operation
				busManager.logger.Warn("Failed to signal file creation via bus", "file", file, "error", err)
			}
		}
	}

	return nil
}
