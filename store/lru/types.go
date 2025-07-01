package lru

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

// Cache lru cache
type Cache struct {
	size int
	lru  *lru.ARCCache
	mu   sync.RWMutex // Protects list operations for concurrency safety
}
