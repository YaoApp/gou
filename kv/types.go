package kv

import "time"

// Store The interface of a key-value store
type Store interface {
	Get(key string) (value interface{}, ok bool)
	Set(key string, value interface{}, ttl time.Duration)
	Del(key string)
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
