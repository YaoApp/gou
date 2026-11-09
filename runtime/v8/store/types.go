package store

import (
	"sync"

	"rogchap.com/v8go"
)

// Store the sync map
type Store struct {
	data  map[string]IStore
	mutex *sync.Mutex
}

// IStore the interface of store
type IStore interface {
	Key() string
	Dispose()
}

// Isolate v8 Isolate
type Isolate struct {
	*v8go.Isolate
	Status   uint8
	Template *v8go.ObjectTemplate
}

// Context runtime context
type Context struct {
	script  string // Script ID
	isolate string // Isolate ID
	*v8go.Context
}

// Cache the cache
type Cache struct {
	key      string
	contexts map[string]*Context
}
