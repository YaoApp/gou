package lru

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

// Cache lru cache with TTL support
type Cache struct {
	size int
	lru  *lru.ARCCache
	mu   sync.RWMutex // Protects list operations for concurrency safety
}

// entry wraps a value with optional expiration time
type entry struct {
	Value     interface{}
	ExpiredAt *time.Time
}

// isExpired checks if the entry has expired
func (e *entry) isExpired() bool {
	if e.ExpiredAt == nil {
		return false
	}
	return time.Now().After(*e.ExpiredAt)
}
