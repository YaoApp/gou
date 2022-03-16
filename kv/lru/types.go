package lru

import lru "github.com/hashicorp/golang-lru"

// Cache lru cache
type Cache struct {
	size int
	lru  *lru.ARCCache
}
