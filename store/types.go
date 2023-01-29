package store

import "time"

// Store The interface of a key-value store
type Store interface {
	Get(key string) (value interface{}, ok bool)
	Set(key string, value interface{}, ttl time.Duration) error
	Del(key string) error
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
}

// Instance the kv-store setting
type Instance struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Connector   string                 `json:"connector,omitempty"`
	Type        string                 `json:"type,omitempty"` // warning: type is deprecated in the future new version
	Option      map[string]interface{} `json:"option,omitempty"`
}

// Option the store option
type Option map[string]interface{}
