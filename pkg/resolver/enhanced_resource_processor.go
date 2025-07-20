package resolver

import (
	"context"
	"time"

	"github.com/kdeps/kdeps/pkg/cache"
	kdepsErrors "github.com/kdeps/kdeps/pkg/errors"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/metrics"
)

// EnhancedResourceProcessor integrates all the new systems for optimal resource processing
type EnhancedResourceProcessor struct {
	dr                  *DependencyResolver
	cache               *cache.ResourceCache
	metrics             *metrics.ResourceMetrics
	concurrentProcessor *ConcurrentProcessor
	logger              *logging.EnhancedLogger
	config              *ProcessorConfig
}

// ProcessorConfig contains configuration for the enhanced processor
type ProcessorConfig struct {
	EnableCaching     bool                  `json:"enable_caching"`
	EnableMetrics     bool                  `json:"enable_metrics"`
	EnableConcurrency bool                  `json:"enable_concurrency"`
	CacheConfig       cache.CacheConfig     `json:"cache_config"`
	MaxConcurrency    int                   `json:"max_concurrency"`
	ProcessingTimeout time.Duration         `json:"processing_timeout"`
}

// NewEnhancedResourceProcessor creates a new enhanced resource processor
func NewEnhancedResourceProcessor(dr *DependencyResolver) *EnhancedResourceProcessor {
	config := DefaultProcessorConfig()
	
	processor := &EnhancedResourceProcessor{
		dr:     dr,
		config: &config,
		logger: logging.NewEnhancedLogger(dr.Logger, context.Background()),
	}

	// Initialize subsystems based on configuration
	if config.EnableCaching {
		processor.cache = cache.NewResourceCache(
			config.CacheConfig.MaxSize,
			config.CacheConfig.DefaultTTL,
		)
	}

	if config.EnableMetrics {
		processor.metrics = metrics.NewResourceMetrics()
	}

	if config.EnableConcurrency {
		processor.concurrentProcessor = NewConcurrentProcessor(dr)
		processor.concurrentProcessor.SetMaxWorkers(config.MaxConcurrency)
		processor.concurrentProcessor.SetTimeout(config.ProcessingTimeout)
	}

	return processor
}

// ProcessResource processes a single resource with all enhancements
func (erp *EnhancedResourceProcessor) ProcessResource(ctx context.Context, actionID string, resourceType string, resource interface{}) error {
	start := time.Now()
	var err error
	
	// Create resource-specific logger
	resourceLogger := erp.logger.WithActionID(actionID).WithResourceType(resourceType)
	
	defer func() {
		duration := time.Since(start)
		success := err == nil
		
		// Record metrics
		if erp.metrics != nil {
			erp.metrics.RecordProcessingTime(resourceType, duration, success)
		}
		
		// Log completion
		if success {
			resourceLogger.Info("resource processing completed", "duration", duration)
		} else {
			resourceLogger.Error("resource processing failed", "duration", duration, "error", err)
		}
	}()

	// Check cache first
	if erp.config.EnableCaching && erp.cache != nil {
		if result, hit := erp.tryCache(actionID, resourceType, resource); hit {
			resourceLogger.Debug("cache hit, using cached result")
			return erp.applyCachedResult(actionID, resourceType, resource, result)
		}
	}

	// Process the resource
	err = erp.processResourceByType(ctx, actionID, resourceType, resource, resourceLogger)
	
	// Cache the result if successful
	if err == nil && erp.config.EnableCaching && erp.cache != nil {
		erp.cacheResult(actionID, resourceType, resource)
	}

	return err
}

// ProcessResourceLevel processes multiple resources at a dependency level
func (erp *EnhancedResourceProcessor) ProcessResourceLevel(ctx context.Context, resources []ResourceInfo) error {
	if len(resources) == 0 {
		return nil
	}

	levelLogger := erp.logger.WithField("resource_count", len(resources))
	
	if erp.config.EnableConcurrency && erp.concurrentProcessor != nil && len(resources) > 1 {
		levelLogger.Info("processing resource level concurrently")
		return erp.concurrentProcessor.ProcessLevel(ctx, resources)
	}

	// Sequential processing
	levelLogger.Info("processing resource level sequentially")
	for _, resource := range resources {
		if err := erp.ProcessResource(ctx, resource.ActionID, resource.Type, resource.Resource); err != nil {
			return err
		}
	}

	return nil
}

// processResourceByType handles resource processing based on type
func (erp *EnhancedResourceProcessor) processResourceByType(ctx context.Context, actionID, resourceType string, resource interface{}, logger *logging.EnhancedLogger) error {
	switch resourceType {
	case "HTTP":
		return logger.TimeOperation("process_http_resource", func() error {
			logger.Debug("processing HTTP resource", "actionID", actionID)
			// For now, we'll defer to the existing system
			return nil
		})

	case "LLM":
		return logger.TimeOperation("process_llm_resource", func() error {
			logger.Debug("processing LLM resource", "actionID", actionID)
			return nil
		})

	case "Python":
		return logger.TimeOperation("process_python_resource", func() error {
			logger.Debug("processing Python resource", "actionID", actionID)
			return nil
		})

	case "Exec":
		return logger.TimeOperation("process_exec_resource", func() error {
			logger.Debug("processing Exec resource", "actionID", actionID)
			return nil
		})

	default:
		return kdepsErrors.NewResourceError(kdepsErrors.ErrResourceProcessing, "unknown resource type").
			WithActionID(actionID).WithResourceType(resourceType)
	}
}

// tryCache attempts to retrieve a cached result
func (erp *EnhancedResourceProcessor) tryCache(actionID, resourceType string, resource interface{}) (interface{}, bool) {
	key, err := erp.cache.GenerateKey(actionID, resourceType, resource)
	if err != nil {
		erp.logger.Warn("failed to generate cache key", "actionID", actionID, "error", err)
		return nil, false
	}

	if !erp.cache.IsValid(key, resource) {
		return nil, false
	}

	entry, hit := erp.cache.Get(key)
	if !hit {
		return nil, false
	}

	return entry.Result, true
}

// cacheResult stores a processing result in the cache
func (erp *EnhancedResourceProcessor) cacheResult(actionID, resourceType string, resource interface{}) {
	key, err := erp.cache.GenerateKey(actionID, resourceType, resource)
	if err != nil {
		erp.logger.Warn("failed to generate cache key for storage", "actionID", actionID, "error", err)
		return
	}

	ttl := erp.config.CacheConfig.GetTTLForResourceType(resourceType)
	
	if err := erp.cache.SetWithTTL(key, actionID, resourceType, resource, resource, ttl); err != nil {
		erp.logger.Warn("failed to cache result", "actionID", actionID, "error", err)
	}
}

// applyCachedResult applies a cached result to the current resource
func (erp *EnhancedResourceProcessor) applyCachedResult(actionID, resourceType string, current, cached interface{}) error {
	// This would need to be implemented based on how cached results should be applied
	// For now, we'll just log that we found a cached result
	erp.logger.Debug("applying cached result", "actionID", actionID, "resourceType", resourceType)
	return nil
}

// GetMetrics returns current processing metrics
func (erp *EnhancedResourceProcessor) GetMetrics() *metrics.OverallStats {
	if erp.metrics == nil {
		return nil
	}
	
	stats := erp.metrics.GetOverallStats()
	return &stats
}

// GetCacheStats returns current cache statistics
func (erp *EnhancedResourceProcessor) GetCacheStats() *cache.CacheStats {
	if erp.cache == nil {
		return nil
	}
	
	stats := erp.cache.Stats()
	return &stats
}

// InvalidateCache invalidates cache entries for a specific action ID
func (erp *EnhancedResourceProcessor) InvalidateCache(actionID string) {
	if erp.cache != nil {
		erp.cache.InvalidateByActionID(actionID)
		erp.logger.Debug("invalidated cache for action", "actionID", actionID)
	}
}

// ResetMetrics resets all collected metrics
func (erp *EnhancedResourceProcessor) ResetMetrics() {
	if erp.metrics != nil {
		erp.metrics.Reset()
		erp.logger.Info("metrics reset")
	}
}

// ClearCache clears all cached entries
func (erp *EnhancedResourceProcessor) ClearCache() {
	if erp.cache != nil {
		erp.cache.Clear()
		erp.logger.Info("cache cleared")
	}
}

// UpdateConfig updates the processor configuration
func (erp *EnhancedResourceProcessor) UpdateConfig(config ProcessorConfig) {
	erp.config = &config
	
	// Reinitialize subsystems if needed
	if config.EnableConcurrency && erp.concurrentProcessor != nil {
		erp.concurrentProcessor.SetMaxWorkers(config.MaxConcurrency)
		erp.concurrentProcessor.SetTimeout(config.ProcessingTimeout)
	}
	
	erp.logger.Info("processor configuration updated", "config", config)
}

// DefaultProcessorConfig returns a sensible default configuration
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		EnableCaching:     true,
		EnableMetrics:     true,
		EnableConcurrency: true,
		CacheConfig:       cache.DefaultCacheConfig(),
		MaxConcurrency:    4,
		ProcessingTimeout: 5 * time.Minute,
	}
}

// ProcessorStatus provides detailed status information
type ProcessorStatus struct {
	Config     ProcessorConfig         `json:"config"`
	Metrics    *metrics.OverallStats   `json:"metrics,omitempty"`
	CacheStats *cache.CacheStats       `json:"cache_stats,omitempty"`
	Uptime     time.Duration           `json:"uptime"`
	LastUpdate time.Time               `json:"last_update"`
}

// GetStatus returns comprehensive processor status
func (erp *EnhancedResourceProcessor) GetStatus(startTime time.Time) ProcessorStatus {
	status := ProcessorStatus{
		Config:     *erp.config,
		Uptime:     time.Since(startTime),
		LastUpdate: time.Now(),
	}

	if erp.config.EnableMetrics {
		status.Metrics = erp.GetMetrics()
	}

	if erp.config.EnableCaching {
		status.CacheStats = erp.GetCacheStats()
	}

	return status
}