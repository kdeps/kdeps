package metrics

import (
	"sync"
	"time"
)

// ResourceMetrics tracks performance and usage metrics for resources
type ResourceMetrics struct {
	mu                      sync.RWMutex
	resourceProcessingTimes map[string][]time.Duration
	resourceSuccessCount    map[string]int64
	resourceErrorCount      map[string]int64
	totalProcessed          int64
	totalErrors             int64
	averageProcessingTime   time.Duration
	lastUpdated             time.Time
}

// NewResourceMetrics creates a new metrics collector
func NewResourceMetrics() *ResourceMetrics {
	return &ResourceMetrics{
		resourceProcessingTimes: make(map[string][]time.Duration),
		resourceSuccessCount:    make(map[string]int64),
		resourceErrorCount:      make(map[string]int64),
		lastUpdated:             time.Now(),
	}
}

// RecordProcessingTime records the processing time for a resource
func (rm *ResourceMetrics) RecordProcessingTime(resourceType string, duration time.Duration, success bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Record processing time
	rm.resourceProcessingTimes[resourceType] = append(rm.resourceProcessingTimes[resourceType], duration)

	// Keep only last 100 measurements per resource type
	if len(rm.resourceProcessingTimes[resourceType]) > 100 {
		rm.resourceProcessingTimes[resourceType] = rm.resourceProcessingTimes[resourceType][1:]
	}

	// Update counters
	if success {
		rm.resourceSuccessCount[resourceType]++
		rm.totalProcessed++
	} else {
		rm.resourceErrorCount[resourceType]++
		rm.totalErrors++
	}

	rm.lastUpdated = time.Now()
	rm.calculateAverageProcessingTime()
}

// calculateAverageProcessingTime calculates the overall average processing time
func (rm *ResourceMetrics) calculateAverageProcessingTime() {
	var totalTime time.Duration
	var totalMeasurements int

	for _, times := range rm.resourceProcessingTimes {
		for _, t := range times {
			totalTime += t
			totalMeasurements++
		}
	}

	if totalMeasurements > 0 {
		rm.averageProcessingTime = totalTime / time.Duration(totalMeasurements)
	}
}

// GetResourceStats returns statistics for a specific resource type
func (rm *ResourceMetrics) GetResourceStats(resourceType string) ResourceStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	times := rm.resourceProcessingTimes[resourceType]
	if len(times) == 0 {
		return ResourceStats{
			ResourceType: resourceType,
			SuccessCount: rm.resourceSuccessCount[resourceType],
			ErrorCount:   rm.resourceErrorCount[resourceType],
			TotalCount:   rm.resourceSuccessCount[resourceType] + rm.resourceErrorCount[resourceType],
		}
	}

	return ResourceStats{
		ResourceType: resourceType,
		SuccessCount: rm.resourceSuccessCount[resourceType],
		ErrorCount:   rm.resourceErrorCount[resourceType],
		TotalCount:   rm.resourceSuccessCount[resourceType] + rm.resourceErrorCount[resourceType],
		AverageTime:  rm.calculateAverage(times),
		MinTime:      rm.calculateMin(times),
		MaxTime:      rm.calculateMax(times),
		MedianTime:   rm.calculateMedian(times),
		P95Time:      rm.calculatePercentile(times, 0.95),
		P99Time:      rm.calculatePercentile(times, 0.99),
		SuccessRate:  rm.calculateSuccessRate(resourceType),
	}
}

// GetOverallStats returns overall system statistics
func (rm *ResourceMetrics) GetOverallStats() OverallStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	resourceTypeStats := make(map[string]ResourceStats)
	for resourceType := range rm.resourceProcessingTimes {
		resourceTypeStats[resourceType] = rm.GetResourceStats(resourceType)
	}

	return OverallStats{
		TotalProcessed:        rm.totalProcessed,
		TotalErrors:           rm.totalErrors,
		OverallSuccessRate:    rm.calculateOverallSuccessRate(),
		AverageProcessingTime: rm.averageProcessingTime,
		ResourceTypeStats:     resourceTypeStats,
		LastUpdated:           rm.lastUpdated,
	}
}

// Helper calculation methods
func (rm *ResourceMetrics) calculateAverage(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}
	var total time.Duration
	for _, t := range times {
		total += t
	}
	return total / time.Duration(len(times))
}

func (rm *ResourceMetrics) calculateMin(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}
	min := times[0]
	for _, t := range times[1:] {
		if t < min {
			min = t
		}
	}
	return min
}

func (rm *ResourceMetrics) calculateMax(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}
	max := times[0]
	for _, t := range times[1:] {
		if t > max {
			max = t
		}
	}
	return max
}

func (rm *ResourceMetrics) calculateMedian(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}

	// Create a copy and sort it
	sorted := make([]time.Duration, len(times))
	copy(sorted, times)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	if len(sorted)%2 == 0 {
		return (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	}
	return sorted[len(sorted)/2]
}

func (rm *ResourceMetrics) calculatePercentile(times []time.Duration, percentile float64) time.Duration {
	if len(times) == 0 {
		return 0
	}

	// Create a copy and sort it
	sorted := make([]time.Duration, len(times))
	copy(sorted, times)

	// Simple bubble sort
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	index := int(float64(len(sorted)-1) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func (rm *ResourceMetrics) calculateSuccessRate(resourceType string) float64 {
	successCount := rm.resourceSuccessCount[resourceType]
	errorCount := rm.resourceErrorCount[resourceType]
	total := successCount + errorCount

	if total == 0 {
		return 0.0
	}

	return float64(successCount) / float64(total) * 100.0
}

func (rm *ResourceMetrics) calculateOverallSuccessRate() float64 {
	if rm.totalProcessed+rm.totalErrors == 0 {
		return 0.0
	}
	return float64(rm.totalProcessed) / float64(rm.totalProcessed+rm.totalErrors) * 100.0
}

// ResourceStats contains statistics for a specific resource type
type ResourceStats struct {
	ResourceType string        `json:"resource_type"`
	SuccessCount int64         `json:"success_count"`
	ErrorCount   int64         `json:"error_count"`
	TotalCount   int64         `json:"total_count"`
	AverageTime  time.Duration `json:"average_time"`
	MinTime      time.Duration `json:"min_time"`
	MaxTime      time.Duration `json:"max_time"`
	MedianTime   time.Duration `json:"median_time"`
	P95Time      time.Duration `json:"p95_time"`
	P99Time      time.Duration `json:"p99_time"`
	SuccessRate  float64       `json:"success_rate"`
}

// OverallStats contains overall system statistics
type OverallStats struct {
	TotalProcessed        int64                    `json:"total_processed"`
	TotalErrors           int64                    `json:"total_errors"`
	OverallSuccessRate    float64                  `json:"overall_success_rate"`
	AverageProcessingTime time.Duration            `json:"average_processing_time"`
	ResourceTypeStats     map[string]ResourceStats `json:"resource_type_stats"`
	LastUpdated           time.Time                `json:"last_updated"`
}

// Reset clears all metrics
func (rm *ResourceMetrics) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.resourceProcessingTimes = make(map[string][]time.Duration)
	rm.resourceSuccessCount = make(map[string]int64)
	rm.resourceErrorCount = make(map[string]int64)
	rm.totalProcessed = 0
	rm.totalErrors = 0
	rm.averageProcessingTime = 0
	rm.lastUpdated = time.Now()
}
