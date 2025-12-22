package store

import (
	"time"

	"github.com/yaoapp/gou/types"
)

// Store The interface of a key-value store
type Store interface {
	Get(key string) (value interface{}, ok bool)
	Set(key string, value interface{}, ttl time.Duration) error
	Del(key string) error // Supports wildcard pattern with * (e.g., "user:123:*")
	Has(key string) bool
	Len() int
	Keys() []string
	Clear()
	GetSet(key string, ttl time.Duration, getValue func(key string) (interface{}, error)) (interface{}, error)
	GetDel(key string) (value interface{}, ok bool)
	GetMulti(keys []string) map[string]interface{}
	SetMulti(values map[string]interface{}, ttl time.Duration)
	DelMulti(keys []string)
	GetSetMulti(keys []string, ttl time.Duration, getValue func(key string) (interface{}, error)) map[string]interface{}

	// Atomic counter operations
	Incr(key string, delta int64) (int64, error) // Increment a numeric value, returns new value
	Decr(key string, delta int64) (int64, error) // Decrement a numeric value, returns new value

	// List operations - MongoDB-style API
	Push(key string, values ...interface{}) error
	Pop(key string, position int) (interface{}, error) // position: 1=last, -1=first
	Pull(key string, value interface{}) error
	PullAll(key string, values []interface{}) error
	AddToSet(key string, values ...interface{}) error
	ArrayLen(key string) int
	ArrayGet(key string, index int) (interface{}, error)
	ArraySet(key string, index int, value interface{}) error
	ArraySlice(key string, skip, limit int) ([]interface{}, error)
	ArrayPage(key string, page, pageSize int) ([]interface{}, error)
	ArrayAll(key string) ([]interface{}, error)
}

// Instance the kv-store setting
type Instance struct {
	types.MetaInfo
	Name      string                 `json:"name"`
	Connector string                 `json:"connector,omitempty"`
	Type      string                 `json:"type,omitempty"` // warning: type is deprecated in the future new version
	Option    map[string]interface{} `json:"option,omitempty"`
}

// Option the store option
type Option map[string]interface{}
