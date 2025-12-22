package xun

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

// DefaultTableName is the default table name for the store
const DefaultTableName = "__store_default"

// DefaultCacheSize is the default LRU cache size
const DefaultCacheSize = 10240

// DefaultCleanupInterval is the default interval for cleanup goroutine
const DefaultCleanupInterval = time.Minute * 5

// DefaultPersistInterval is the default interval for async persistence
const DefaultPersistInterval = time.Minute * 1

// Store xun database store with LRU cache and async persistence
// Architecture:
// - Read: cache-first, lazy load from DB on cache miss
// - Write: write to cache immediately, async persist to DB
// - LRU auto-eviction when cache is full
// - Background worker for periodic persistence and cleanup
type Store struct {
	connector       string
	tableName       string
	cache           *lru.ARCCache
	cacheSize       int
	cleanupInterval time.Duration
	persistInterval time.Duration

	// Dirty tracking for async persistence
	dirty     map[string]*dirtyEntry
	dirtyMu   sync.RWMutex
	deleted   map[string]bool // Track deleted keys
	deletedMu sync.RWMutex

	mu         sync.RWMutex
	stopWorker chan struct{}
	workerDone chan struct{}
}

// cacheEntry represents a cached value with expiration
type cacheEntry struct {
	Value     interface{}
	ExpiredAt *time.Time
	Type      string // "value" or "list"
}

// dirtyEntry represents a dirty cache entry pending persistence
type dirtyEntry struct {
	Key       string
	Value     interface{}
	Type      string // "value" or "list"
	ExpiredAt *time.Time
	UpdatedAt time.Time
}

// Option xun store option
type Option struct {
	Table           string        `json:"table,omitempty"`            // Table name, default: __store_default
	Connector       string        `json:"connector,omitempty"`        // Database connector, default: default
	CacheSize       int           `json:"cache_size,omitempty"`       // LRU cache size, default: 10240
	CleanupInterval time.Duration `json:"cleanup_interval,omitempty"` // Cleanup interval, default: 5 minutes
	PersistInterval time.Duration `json:"persist_interval,omitempty"` // Persist interval, default: 1 minute
}

// StoreRecord represents a record in the store table
type StoreRecord struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Type      string      `json:"type"` // "value" or "list"
	ExpiredAt *time.Time  `json:"expired_at,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}
