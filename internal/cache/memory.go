package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MemoryManager implements an in-memory cache (temporary, will use SQLite later)
type MemoryManager struct {
	data map[string]*cacheEntry
	ttl  time.Duration
	mu   sync.RWMutex
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryManager creates a new in-memory cache manager
func NewMemoryManager(ttl time.Duration) *MemoryManager {
	return &MemoryManager{
		data: make(map[string]*cacheEntry),
		ttl:  ttl,
	}
}

// Get retrieves a value from the cache
func (m *MemoryManager) Get(key string, dest interface{}) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.data[key]
	if !ok {
		return false, nil
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return false, nil
	}

	// Unmarshal JSON
	if err := json.Unmarshal(entry.value, dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return true, nil
}

// Set stores a value in the cache
func (m *MemoryManager) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Marshal to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	m.data[key] = &cacheEntry{
		value:     data,
		expiresAt: time.Now().Add(m.ttl),
	}

	return nil
}

// Delete removes a value from the cache
func (m *MemoryManager) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

// Clear removes all entries from the cache
func (m *MemoryManager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]*cacheEntry)
	return nil
}

// CleanExpired removes all expired entries
func (m *MemoryManager) CleanExpired() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, entry := range m.data {
		if now.After(entry.expiresAt) {
			delete(m.data, key)
		}
	}

	return nil
}

// Stats returns cache statistics
func (m *MemoryManager) Stats() (total int, expired int, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.data)
	now := time.Now()

	for _, entry := range m.data {
		if now.After(entry.expiresAt) {
			expired++
		}
	}

	return total, expired, nil
}

// Close does nothing for memory cache but implements the interface
func (m *MemoryManager) Close() error {
	return nil
}
