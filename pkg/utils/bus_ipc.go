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
	resilientClient *bus.ResilientClient
	logger          *logging.Logger
}

// NewBusIPCManager creates a new bus IPC manager with resilient client
func NewBusIPCManager(logger *logging.Logger) (*BusIPCManager, error) {
	resilientClient, err := bus.NewResilientClient(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create resilient bus client: %w", err)
	}

	return &BusIPCManager{
		resilientClient: resilientClient,
		logger:          logger,
	}, nil
}

// NewBusIPCManagerWithConfig creates a bus IPC manager with custom configuration
func NewBusIPCManagerWithConfig(logger *logging.Logger, poolSize int, retryConfig bus.RetryConfig) (*BusIPCManager, error) {
	resilientClient, err := bus.NewResilientClientWithConfig(logger, poolSize, retryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create resilient bus client: %w", err)
	}

	return &BusIPCManager{
		resilientClient: resilientClient,
		logger:          logger,
	}, nil
}

// Close closes the bus connection
func (b *BusIPCManager) Close() error {
	if b.resilientClient != nil {
		return b.resilientClient.Close()
	}
	return nil
}

// SignalResourceCompletion replaces timestamp-based completion signaling
func (b *BusIPCManager) SignalResourceCompletion(resourceID, resourceType, status string, data map[string]interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["resourceType"] = resourceType

	return b.resilientClient.SignalResourceCompletion(resourceID, status, data)
}

// WaitForResourceCompletion replaces WaitForTimestampChange
func (b *BusIPCManager) WaitForResourceCompletion(resourceID string, timeoutSeconds int64) error {
	state, err := b.resilientClient.WaitForResourceCompletion(resourceID, timeoutSeconds)
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

	return b.resilientClient.PublishEvent(eventType, message, "", data)
}

// WaitForCleanup replaces WaitForFileReady for cleanup files
func (b *BusIPCManager) WaitForCleanup(timeoutSeconds int64) error {
	return b.resilientClient.WaitForCleanupSignal(timeoutSeconds)
}

// SignalFileReady replaces file creation for signaling
func (b *BusIPCManager) SignalFileReady(filepath, operation string, data map[string]interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["filepath"] = filepath
	data["operation"] = operation

	return b.resilientClient.PublishEvent("file_ready", fmt.Sprintf("File %s ready for %s", filepath, operation), filepath, data)
}

// WaitForFileReady replaces the file-based WaitForFileReady function
func (b *BusIPCManager) WaitForFileReady(filepath string, timeoutSeconds int64) error {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	b.logger.Debug("Waiting for file ready signal via bus", "file", filepath)

	// Use resilient client with built-in retry and circuit breaking
	start := time.Now()
	return b.resilientClient.ExecuteWithRetry(func(client *rpc.Client) error {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for file %s", filepath)
		}

		return bus.WaitForEvents(client, b.logger, func(event bus.Event) bool {
			if time.Since(start) > timeout {
				return true // Stop and let timeout be handled
			}

			if event.Type == "file_ready" {
				if eventFilepath, ok := event.Data["filepath"].(string); ok && eventFilepath == filepath {
					b.logger.Info("File ready signal received via bus", "file", filepath)
					return true
				}
			}
			return false
		})
	})
}

// HealthCheck performs a health check on the bus service
func (b *BusIPCManager) HealthCheck() (*bus.HealthStatus, error) {
	return b.resilientClient.HealthCheck()
}

// GetMetrics returns bus client metrics for monitoring
func (b *BusIPCManager) GetMetrics() map[string]interface{} {
	return b.resilientClient.GetMetrics()
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
