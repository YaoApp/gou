package lru

import (
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/yaoapp/kun/log"
)

// New create a new LRU cache
func New(size int) (*Cache, error) {
	cache := &Cache{}
	lru, err := lru.NewARC(size)
	if err != nil {
		return nil, err
	}
	cache.lru = lru
	return cache, nil
}

// Get looks up a key's value from the cache.
func (cache *Cache) Get(key string) (value interface{}, ok bool) {
	return cache.lru.Get(key)
}

// Set adds a value to the cache.
func (cache *Cache) Set(key string, value interface{}, ttl time.Duration) error {
	cache.lru.Add(key, value)
	return nil
}

// Del remove is used to purge a key from the cache
func (cache *Cache) Del(key string) error {
	cache.lru.Remove(key)
	return nil
}

// Has check if the cache is exist ( without updating recency or frequency )
func (cache *Cache) Has(key string) bool {
	_, has := cache.lru.Peek(key)
	return has
}

// Len returns the number of cached entries
func (cache *Cache) Len() int {
	return cache.lru.Len()
}

// Keys returns all the cached keys
func (cache *Cache) Keys() []string {
	keys := cache.lru.Keys()
	res := []string{}
	for _, key := range keys {
		keystr, ok := key.(string)
		if !ok {
			keystr = fmt.Sprintf("%v", key)
		}
		res = append(res, keystr)
	}
	return res
}

// Clear is used to clear the cache
func (cache *Cache) Clear() {
	cache.lru.Purge()
}

// GetSet looks up a key's value from the cache. if does not exist add to the cache
func (cache *Cache) GetSet(key string, ttl time.Duration, getValue func(key string) (interface{}, error)) (interface{}, error) {
	value, ok := cache.lru.Get(key)
	if !ok {
		var err error
		value, err = getValue(key)
		if err != nil {
			return nil, err
		}
		cache.Set(key, value, ttl)
	}
	return value, nil
}

// GetDel looks up a key's value from the cache, then remove it.
func (cache *Cache) GetDel(key string) (value interface{}, ok bool) {
	value, ok = cache.lru.Get(key)
	if !ok {
		return nil, false
	}
	cache.lru.Remove(key)
	return value, true
}

// GetMulti mulit get values
func (cache *Cache) GetMulti(keys []string) map[string]interface{} {
	values := map[string]interface{}{}
	for _, key := range keys {
		value, _ := cache.lru.Get(key)
		values[key] = value
	}
	return values
}

// SetMulti mulit set values
func (cache *Cache) SetMulti(values map[string]interface{}, ttl time.Duration) {
	for key, value := range values {
		cache.lru.Add(key, value)
	}
}

// DelMulti mulit remove values
func (cache *Cache) DelMulti(keys []string) {
	for _, key := range keys {
		cache.lru.Remove(key)
	}
}

// GetSetMulti mulit get values, if does not exist add to the cache
func (cache *Cache) GetSetMulti(keys []string, ttl time.Duration, getValue func(key string) (interface{}, error)) map[string]interface{} {
	values := map[string]interface{}{}
	for _, key := range keys {
		value, ok := cache.lru.Get(key)
		if !ok {
			var err error
			value, err = getValue(key)
			if err != nil {
				log.Error("GetSetMulti Set %s: %s", key, err.Error())
			}
		}
		values[key] = value
	}
	return values
}
