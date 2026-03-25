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

package federation

import (
	"sync"
	"time"
)

// cacheEntry holds a cached Capability with its expiry time.
type cacheEntry struct {
	capability *Capability
	expiresAt  time.Time
}

// RegistryCache is an in-memory LRU-style cache for agent capabilities.
// It stores capabilities keyed by their canonical URN string, with per-entry TTL.
// A background goroutine evicts expired entries every 5 minutes.
type RegistryCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	stopCh  chan struct{}
}

// NewRegistryCache creates a RegistryCache with the given TTL and starts the
// background cleanup goroutine. Call Stop() when done to release the goroutine.
func NewRegistryCache(ttl time.Duration) *RegistryCache {
	rc := &RegistryCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
		stopCh:  make(chan struct{}),
	}
	go rc.cleanupLoop()
	return rc
}

// Get retrieves the cached Capability for the given URN string.
// Returns ErrCacheMiss if the entry is absent or expired.
func (rc *RegistryCache) Get(urnStr string) (*Capability, error) {
	rc.mu.RLock()
	e, ok := rc.entries[urnStr]
	rc.mu.RUnlock()

	if !ok || time.Now().After(e.expiresAt) {
		return nil, ErrCacheMiss
	}
	return e.capability, nil
}

// Set stores the Capability in the cache under the given URN string.
// The entry expires after the cache TTL.
func (rc *RegistryCache) Set(urnStr string, cap *Capability) {
	rc.mu.Lock()
	rc.entries[urnStr] = &cacheEntry{
		capability: cap,
		expiresAt:  time.Now().Add(rc.ttl),
	}
	rc.mu.Unlock()
}

// Invalidate removes a specific URN from the cache.
func (rc *RegistryCache) Invalidate(urnStr string) {
	rc.mu.Lock()
	delete(rc.entries, urnStr)
	rc.mu.Unlock()
}

// Clear removes all entries from the cache.
func (rc *RegistryCache) Clear() {
	rc.mu.Lock()
	rc.entries = make(map[string]*cacheEntry)
	rc.mu.Unlock()
}

// Stop halts the background cleanup goroutine. Safe to call multiple times.
func (rc *RegistryCache) Stop() {
	select {
	case <-rc.stopCh:
		// already stopped
	default:
		close(rc.stopCh)
	}
}

// cleanupLoop runs every 5 minutes and removes expired cache entries.
func (rc *RegistryCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rc.evictExpired()
		case <-rc.stopCh:
			return
		}
	}
}

// evictExpired removes all entries whose TTL has elapsed.
func (rc *RegistryCache) evictExpired() {
	now := time.Now()
	rc.mu.Lock()
	for k, e := range rc.entries {
		if now.After(e.expiresAt) {
			delete(rc.entries, k)
		}
	}
	rc.mu.Unlock()
}
