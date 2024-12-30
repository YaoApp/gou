package plan

import (
	"fmt"
	"sync"
)

// MemorySharedSpace implements SharedSpace interface using in-memory storage
type MemorySharedSpace struct {
	mu          sync.RWMutex
	data        map[string]interface{}
	subscribers map[string][]func(key string, value interface{})
	subMu       sync.RWMutex
}

// NewMemorySharedSpace creates a new MemorySharedSpace instance
func NewMemorySharedSpace() *MemorySharedSpace {
	return &MemorySharedSpace{
		data:        make(map[string]interface{}),
		subscribers: make(map[string][]func(key string, value interface{})),
	}
}

// Set stores a value in the shared space
func (m *MemorySharedSpace) Set(key string, value interface{}) error {
	m.mu.Lock()
	m.data[key] = value
	m.mu.Unlock()

	// Notify subscribers
	m.subMu.RLock()
	if callbacks, exists := m.subscribers[key]; exists {
		for _, callback := range callbacks {
			go callback(key, value)
		}
	}
	m.subMu.RUnlock()

	return nil
}

// Get retrieves a value from the shared space
func (m *MemorySharedSpace) Get(key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key %s not found", key)
	}
	return value, nil
}

// Delete removes a value from the shared space
func (m *MemorySharedSpace) Delete(key string) error {
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()

	// Notify subscribers of deletion
	m.subMu.RLock()
	if callbacks, exists := m.subscribers[key]; exists {
		for _, callback := range callbacks {
			go callback(key, nil)
		}
	}
	m.subMu.RUnlock()

	return nil
}

// Clear removes all values from the shared space
func (m *MemorySharedSpace) Clear() error {
	m.mu.Lock()
	m.data = make(map[string]interface{})
	m.mu.Unlock()

	// Notify all subscribers of clearing
	m.subMu.RLock()
	for key, callbacks := range m.subscribers {
		for _, callback := range callbacks {
			go callback(key, nil)
		}
	}
	m.subMu.RUnlock()

	return nil
}

// Subscribe subscribes to changes in the shared space
func (m *MemorySharedSpace) Subscribe(key string, callback func(key string, value interface{})) error {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	m.subscribers[key] = append(m.subscribers[key], callback)
	return nil
}

// Unsubscribe removes a subscription
func (m *MemorySharedSpace) Unsubscribe(key string) error {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	delete(m.subscribers, key)
	return nil
}
