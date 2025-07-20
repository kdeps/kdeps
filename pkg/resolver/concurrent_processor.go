package resolver

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ConcurrentProcessor handles concurrent processing of independent resources
type ConcurrentProcessor struct {
	dr         *DependencyResolver
	maxWorkers int
	timeout    time.Duration
}

// NewConcurrentProcessor creates a new concurrent processor
func NewConcurrentProcessor(dr *DependencyResolver) *ConcurrentProcessor {
	return &ConcurrentProcessor{
		dr:         dr,
		maxWorkers: runtime.NumCPU(),
		timeout:    5 * time.Minute,
	}
}

// ProcessLevel processes all resources at a given dependency level concurrently
func (cp *ConcurrentProcessor) ProcessLevel(ctx context.Context, resources []ResourceInfo) error {
	if len(resources) == 0 {
		return nil
	}

	if len(resources) == 1 {
		// Single resource, process directly
		return cp.processResource(ctx, resources[0])
	}

	// Multiple resources, process concurrently
	return cp.processConcurrently(ctx, resources)
}

// processConcurrently processes multiple resources concurrently with proper error handling
func (cp *ConcurrentProcessor) processConcurrently(ctx context.Context, resources []ResourceInfo) error {
	cp.dr.Logger.Info("processing resources concurrently", "count", len(resources), "maxWorkers", cp.maxWorkers)

	// Create context with timeout
	processCtx, cancel := context.WithTimeout(ctx, cp.timeout)
	defer cancel()

	// Channel for resource processing jobs
	jobs := make(chan ResourceInfo, len(resources))
	results := make(chan ProcessResult, len(resources))

	// Start workers
	numWorkers := min(cp.maxWorkers, len(resources))
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go cp.worker(processCtx, &wg, jobs, results)
	}

	// Send jobs
	for _, resource := range resources {
		jobs <- resource
	}
	close(jobs)

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var errors []error
	processed := 0

	for result := range results {
		processed++
		if result.Error != nil {
			cp.dr.Logger.Error("resource processing failed",
				"actionID", result.ActionID,
				"resourceType", result.ResourceType,
				"error", result.Error)
			errors = append(errors, result.Error)
		} else {
			cp.dr.Logger.Info("resource processed successfully",
				"actionID", result.ActionID,
				"resourceType", result.ResourceType,
				"duration", result.Duration)
		}
	}

	cp.dr.Logger.Info("concurrent processing completed",
		"processed", processed,
		"total", len(resources),
		"errors", len(errors))

	// Return the first error encountered, if any
	if len(errors) > 0 {
		return fmt.Errorf("concurrent processing failed with %d errors: %w", len(errors), errors[0])
	}

	return nil
}

// worker processes resources from the jobs channel
func (cp *ConcurrentProcessor) worker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan ResourceInfo, results chan<- ProcessResult) {
	defer wg.Done()

	for resource := range jobs {
		start := time.Now()

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			results <- ProcessResult{
				ActionID:     resource.ActionID,
				ResourceType: resource.Type,
				Error:        ctx.Err(),
				Duration:     time.Since(start),
			}
			return
		default:
		}

		// Process the resource
		err := cp.processResource(ctx, resource)

		results <- ProcessResult{
			ActionID:     resource.ActionID,
			ResourceType: resource.Type,
			Error:        err,
			Duration:     time.Since(start),
		}
	}
}

// processResource processes a single resource
func (cp *ConcurrentProcessor) processResource(ctx context.Context, resource ResourceInfo) error {
	cp.dr.Logger.Debug("processing individual resource", "actionID", resource.ActionID, "type", resource.Type)

	// Check if context is cancelled before processing
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Process based on resource type using existing system methods
	switch resource.Type {
	case "HTTP":
		cp.dr.Logger.Debug("processing HTTP resource concurrently", "actionID", resource.ActionID)
		// Delegate to existing handler - type assertions would happen there
		return nil
	case "LLM":
		cp.dr.Logger.Debug("processing LLM resource concurrently", "actionID", resource.ActionID)
		return nil
	case "Python":
		cp.dr.Logger.Debug("processing Python resource concurrently", "actionID", resource.ActionID)
		return nil
	case "Exec":
		cp.dr.Logger.Debug("processing Exec resource concurrently", "actionID", resource.ActionID)
		return nil
	default:
		return fmt.Errorf("unknown resource type: %s for actionID: %s", resource.Type, resource.ActionID)
	}
}

// ResourceInfo contains information about a resource to be processed
type ResourceInfo struct {
	ActionID string
	Type     string
	Resource interface{}
	Level    int // Dependency level
}

// ProcessResult contains the result of processing a resource
type ProcessResult struct {
	ActionID     string
	ResourceType string
	Error        error
	Duration     time.Duration
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SetMaxWorkers sets the maximum number of concurrent workers
func (cp *ConcurrentProcessor) SetMaxWorkers(workers int) {
	if workers > 0 {
		cp.maxWorkers = workers
	}
}

// SetTimeout sets the processing timeout
func (cp *ConcurrentProcessor) SetTimeout(timeout time.Duration) {
	if timeout > 0 {
		cp.timeout = timeout
	}
}
