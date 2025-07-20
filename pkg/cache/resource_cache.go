package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ResourceCache provides intelligent caching for resource processing results
type ResourceCache struct {
	mu          sync.RWMutex
	cache       map[string]*CacheEntry
	maxSize     int
	defaultTTL  time.Duration
	accessOrder []string // For LRU eviction
}

// CacheEntry represents a cached resource result
type CacheEntry struct {
	Key          string        `json:"key"`
	ActionID     string        `json:"action_id"`
	ResourceType string        `json:"resource_type"`
	InputHash    string        `json:"input_hash"`
	Result       interface{}   `json:"result"`
	CreatedAt    time.Time     `json:"created_at"`
	LastAccessed time.Time     `json:"last_accessed"`
	TTL          time.Duration `json:"ttl"`
	AccessCount  int64         `json:"access_count"`
	HitCount     int64         `json:"hit_count"`
}

// NewResourceCache creates a new resource cache
func NewResourceCache(maxSize int, defaultTTL time.Duration) *ResourceCache {
	return &ResourceCache{
		cache:       make(map[string]*CacheEntry),
		maxSize:     maxSize,
		defaultTTL:  defaultTTL,
		accessOrder: make([]string, 0),
	}
}

// GenerateKey generates a cache key based on resource input
func (rc *ResourceCache) GenerateKey(actionID, resourceType string, input interface{}) (string, error) {
	// Create a hash of the input to detect changes
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input for cache key: %w", err)
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s:", actionID, resourceType)))
	hasher.Write(inputJSON)
	hash := hex.EncodeToString(hasher.Sum(nil))

	return fmt.Sprintf("%s:%s:%s", actionID, resourceType, hash[:16]), nil
}

// Get retrieves a cached result if it exists and is still valid
func (rc *ResourceCache) Get(key string) (*CacheEntry, bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	entry, exists := rc.cache[key]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Since(entry.CreatedAt) > entry.TTL {
		delete(rc.cache, key)
		rc.removeFromAccessOrder(key)
		return nil, false
	}

	// Update access information
	entry.LastAccessed = time.Now()
	entry.AccessCount++
	entry.HitCount++

	// Move to front of access order (LRU)
	rc.moveToFront(key)

	return entry, true
}

// Set stores a result in the cache
func (rc *ResourceCache) Set(key, actionID, resourceType string, input, result interface{}) error {
	return rc.SetWithTTL(key, actionID, resourceType, input, result, rc.defaultTTL)
}

// SetWithTTL stores a result in the cache with a specific TTL
func (rc *ResourceCache) SetWithTTL(key, actionID, resourceType string, input, result interface{}, ttl time.Duration) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Generate input hash for change detection
	inputHash, err := rc.generateInputHash(input)
	if err != nil {
		return fmt.Errorf("failed to generate input hash: %w", err)
	}

	// Create cache entry
	entry := &CacheEntry{
		Key:          key,
		ActionID:     actionID,
		ResourceType: resourceType,
		InputHash:    inputHash,
		Result:       result,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		TTL:          ttl,
		AccessCount:  1,
		HitCount:     0,
	}

	// Check if we need to evict entries
	if len(rc.cache) >= rc.maxSize {
		rc.evictLRU()
	}

	// Store the entry
	rc.cache[key] = entry
	rc.moveToFront(key)

	return nil
}

// IsValid checks if a cached entry is still valid for the given input
func (rc *ResourceCache) IsValid(key string, input interface{}) bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.cache[key]
	if !exists {
		return false
	}

	// Check expiration
	if time.Since(entry.CreatedAt) > entry.TTL {
		return false
	}

	// Check if input has changed
	currentInputHash, err := rc.generateInputHash(input)
	if err != nil {
		return false
	}

	return entry.InputHash == currentInputHash
}

// Invalidate removes a specific cache entry
func (rc *ResourceCache) Invalidate(key string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	delete(rc.cache, key)
	rc.removeFromAccessOrder(key)
}

// InvalidateByActionID removes all cache entries for a specific action ID
func (rc *ResourceCache) InvalidateByActionID(actionID string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	keysToRemove := make([]string, 0)
	for key, entry := range rc.cache {
		if entry.ActionID == actionID {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		delete(rc.cache, key)
		rc.removeFromAccessOrder(key)
	}
}

// InvalidateByResourceType removes all cache entries for a specific resource type
func (rc *ResourceCache) InvalidateByResourceType(resourceType string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	keysToRemove := make([]string, 0)
	for key, entry := range rc.cache {
		if entry.ResourceType == resourceType {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		delete(rc.cache, key)
		rc.removeFromAccessOrder(key)
	}
}

// Clear removes all cache entries
func (rc *ResourceCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache = make(map[string]*CacheEntry)
	rc.accessOrder = make([]string, 0)
}

// Stats returns cache statistics
func (rc *ResourceCache) Stats() CacheStats {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	totalHits := int64(0)
	totalAccesses := int64(0)
	oldestEntry := time.Now()
	newestEntry := time.Time{}

	for _, entry := range rc.cache {
		totalHits += entry.HitCount
		totalAccesses += entry.AccessCount

		if entry.CreatedAt.Before(oldestEntry) {
			oldestEntry = entry.CreatedAt
		}
		if entry.CreatedAt.After(newestEntry) {
			newestEntry = entry.CreatedAt
		}
	}

	hitRate := float64(0)
	if totalAccesses > 0 {
		hitRate = float64(totalHits) / float64(totalAccesses) * 100
	}

	return CacheStats{
		Size:          len(rc.cache),
		MaxSize:       rc.maxSize,
		HitRate:       hitRate,
		TotalHits:     totalHits,
		TotalAccesses: totalAccesses,
		OldestEntry:   oldestEntry,
		NewestEntry:   newestEntry,
	}
}

// Helper methods

func (rc *ResourceCache) generateInputHash(input interface{}) (string, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	hasher.Write(inputJSON)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (rc *ResourceCache) evictLRU() {
	if len(rc.accessOrder) == 0 {
		return
	}

	// Remove the least recently used entry (last in access order)
	lruKey := rc.accessOrder[len(rc.accessOrder)-1]
	delete(rc.cache, lruKey)
	rc.accessOrder = rc.accessOrder[:len(rc.accessOrder)-1]
}

func (rc *ResourceCache) moveToFront(key string) {
	// Remove from current position
	rc.removeFromAccessOrder(key)

	// Add to front
	rc.accessOrder = append([]string{key}, rc.accessOrder...)
}

func (rc *ResourceCache) removeFromAccessOrder(key string) {
	for i, k := range rc.accessOrder {
		if k == key {
			rc.accessOrder = append(rc.accessOrder[:i], rc.accessOrder[i+1:]...)
			break
		}
	}
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	Size          int       `json:"size"`
	MaxSize       int       `json:"max_size"`
	HitRate       float64   `json:"hit_rate"`
	TotalHits     int64     `json:"total_hits"`
	TotalAccesses int64     `json:"total_accesses"`
	OldestEntry   time.Time `json:"oldest_entry"`
	NewestEntry   time.Time `json:"newest_entry"`
}

// CacheConfig contains cache configuration options
type CacheConfig struct {
	MaxSize    int           `json:"max_size"`
	DefaultTTL time.Duration `json:"default_ttl"`

	// Resource-specific TTLs
	HTTPTTL   time.Duration `json:"http_ttl"`
	LLMTTL    time.Duration `json:"llm_ttl"`
	PythonTTL time.Duration `json:"python_ttl"`
	ExecTTL   time.Duration `json:"exec_ttl"`
}

// DefaultCacheConfig returns a sensible default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxSize:    1000,
		DefaultTTL: 1 * time.Hour,
		HTTPTTL:    30 * time.Minute, // HTTP responses can be cached longer
		LLMTTL:     15 * time.Minute, // LLM responses may vary
		PythonTTL:  10 * time.Minute, // Python scripts may have side effects
		ExecTTL:    5 * time.Minute,  // Exec commands may have side effects
	}
}

// GetTTLForResourceType returns the appropriate TTL for a resource type
func (config CacheConfig) GetTTLForResourceType(resourceType string) time.Duration {
	switch resourceType {
	case "HTTP":
		return config.HTTPTTL
	case "LLM":
		return config.LLMTTL
	case "Python":
		return config.PythonTTL
	case "Exec":
		return config.ExecTTL
	default:
		return config.DefaultTTL
	}
}
