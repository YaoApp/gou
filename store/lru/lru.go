package lru

import (
	"fmt"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/yaoapp/kun/log"
)

// New create a new LRU cache
func New(size int) (*Cache, error) {
	return NewWithOption(Option{Size: size})
}

// NewWithOption create a new LRU cache with options
func NewWithOption(opt Option) (*Cache, error) {
	size := opt.Size
	if size <= 0 {
		size = 10240 // Default size
	}
	cache := &Cache{size: size, prefix: opt.Prefix}
	lruCache, err := lru.NewARC(size)
	if err != nil {
		return nil, err
	}
	cache.lru = lruCache
	return cache, nil
}

// prefixKey adds the prefix to a key
func (cache *Cache) prefixKey(key string) string {
	if cache.prefix == "" {
		return key
	}
	return cache.prefix + key
}

// unprefixKey removes the prefix from a key
func (cache *Cache) unprefixKey(key string) string {
	if cache.prefix == "" {
		return key
	}
	return strings.TrimPrefix(key, cache.prefix)
}

// Get looks up a key's value from the cache.
func (cache *Cache) Get(key string) (value interface{}, ok bool) {
	raw, found := cache.lru.Get(cache.prefixKey(key))
	if !found {
		return nil, false
	}

	e, ok := raw.(*entry)
	if !ok {
		// Legacy value without TTL wrapper
		return raw, true
	}

	// Check expiration
	if e.isExpired() {
		cache.lru.Remove(cache.prefixKey(key))
		return nil, false
	}

	return e.Value, true
}

// Set adds a value to the cache with optional TTL.
func (cache *Cache) Set(key string, value interface{}, ttl time.Duration) error {
	e := &entry{Value: value}
	if ttl > 0 {
		exp := time.Now().Add(ttl)
		e.ExpiredAt = &exp
	}
	cache.lru.Add(cache.prefixKey(key), e)
	return nil
}

// Del remove is used to purge a key from the cache
// Supports wildcard pattern with * (e.g., "user:123:*")
func (cache *Cache) Del(key string) error {
	// Check if key contains wildcard
	if strings.Contains(key, "*") {
		return cache.delPattern(key)
	}
	cache.lru.Remove(cache.prefixKey(key))
	return nil
}

// delPattern deletes all keys matching the pattern
func (cache *Cache) delPattern(pattern string) error {
	// Add prefix to pattern
	fullPattern := cache.prefixKey(pattern)
	// Convert pattern to prefix (only supports suffix wildcard for now)
	prefix := strings.TrimSuffix(fullPattern, "*")

	keys := cache.lru.Keys()
	for _, k := range keys {
		keyStr, ok := k.(string)
		if !ok {
			keyStr = fmt.Sprintf("%v", k)
		}
		if matchPattern(keyStr, fullPattern, prefix) {
			cache.lru.Remove(k)
		}
	}
	return nil
}

// matchPattern checks if a key matches the pattern
func matchPattern(key, pattern, prefix string) bool {
	// Simple prefix matching for patterns ending with *
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(key, prefix)
	}
	// Exact match
	return key == pattern
}

// Has check if the cache is exist ( without updating recency or frequency )
func (cache *Cache) Has(key string) bool {
	prefixedKey := cache.prefixKey(key)
	raw, found := cache.lru.Peek(prefixedKey)
	if !found {
		return false
	}

	e, ok := raw.(*entry)
	if !ok {
		return true
	}

	if e.isExpired() {
		cache.lru.Remove(prefixedKey)
		return false
	}

	return true
}

// Len returns the number of cached entries
// Optional pattern parameter supports * wildcard (e.g., "user:*")
func (cache *Cache) Len(pattern ...string) int {
	// Build full pattern with prefix
	var fullPattern string
	if len(pattern) > 0 && pattern[0] != "" {
		fullPattern = cache.prefixKey(pattern[0])
	} else if cache.prefix != "" {
		fullPattern = cache.prefix + "*"
	}

	count := 0
	now := time.Now()
	keys := cache.lru.Keys()

	// Get prefix for pattern matching
	var pat, prefix string
	hasPattern := fullPattern != ""
	if hasPattern {
		pat = fullPattern
		prefix = strings.TrimSuffix(pat, "*")
	}

	for _, key := range keys {
		keyStr, ok := key.(string)
		if !ok {
			keyStr = fmt.Sprintf("%v", key)
		}

		// Filter by pattern if provided
		if hasPattern && !matchPattern(keyStr, pat, prefix) {
			continue
		}

		if raw, found := cache.lru.Peek(key); found {
			if e, ok := raw.(*entry); ok {
				if e.ExpiredAt != nil && now.After(*e.ExpiredAt) {
					continue // Skip expired
				}
			}
			count++
		}
	}
	return count
}

// Keys returns all the cached keys (excludes expired entries)
// Optional pattern parameter supports * wildcard (e.g., "user:*")
func (cache *Cache) Keys(pattern ...string) []string {
	keys := cache.lru.Keys()
	res := []string{}
	now := time.Now()

	// Build full pattern with prefix
	var fullPattern string
	if len(pattern) > 0 && pattern[0] != "" {
		fullPattern = cache.prefixKey(pattern[0])
	} else if cache.prefix != "" {
		fullPattern = cache.prefix + "*"
	}

	// Get pattern and prefix if provided
	var pat, prefix string
	hasPattern := fullPattern != ""
	if hasPattern {
		pat = fullPattern
		prefix = strings.TrimSuffix(pat, "*")
	}

	prefixLen := len(cache.prefix)

	for _, key := range keys {
		keystr, ok := key.(string)
		if !ok {
			keystr = fmt.Sprintf("%v", key)
		}

		// Filter by pattern if provided
		if hasPattern && !matchPattern(keystr, pat, prefix) {
			continue
		}

		// Check expiration
		if raw, found := cache.lru.Peek(key); found {
			if e, ok := raw.(*entry); ok {
				if e.ExpiredAt != nil && now.After(*e.ExpiredAt) {
					continue // Skip expired
				}
			}
		}

		// Remove prefix from returned keys
		if prefixLen > 0 && len(keystr) >= prefixLen {
			keystr = keystr[prefixLen:]
		}
		res = append(res, keystr)
	}
	return res
}

// Clear is used to clear the cache
// If prefix is set, only clears keys with that prefix
func (cache *Cache) Clear() {
	if cache.prefix == "" {
		cache.lru.Purge()
		return
	}

	// Only clear keys with the prefix
	keys := cache.lru.Keys()
	for _, k := range keys {
		keyStr, ok := k.(string)
		if !ok {
			keyStr = fmt.Sprintf("%v", k)
		}
		if strings.HasPrefix(keyStr, cache.prefix) {
			cache.lru.Remove(k)
		}
	}
}

// GetSet looks up a key's value from the cache. if does not exist add to the cache
func (cache *Cache) GetSet(key string, ttl time.Duration, getValue func(key string) (interface{}, error)) (interface{}, error) {
	value, ok := cache.Get(key)
	if ok {
		return value, nil
	}

	value, err := getValue(key)
	if err != nil {
		return nil, err
	}
	cache.Set(key, value, ttl)
	return value, nil
}

// GetDel looks up a key's value from the cache, then remove it.
func (cache *Cache) GetDel(key string) (value interface{}, ok bool) {
	value, ok = cache.Get(key)
	if !ok {
		return nil, false
	}
	cache.lru.Remove(cache.prefixKey(key))
	return value, true
}

// GetMulti mulit get values
func (cache *Cache) GetMulti(keys []string) map[string]interface{} {
	values := map[string]interface{}{}
	for _, key := range keys {
		value, _ := cache.Get(key)
		values[key] = value
	}
	return values
}

// SetMulti mulit set values
func (cache *Cache) SetMulti(values map[string]interface{}, ttl time.Duration) {
	for key, value := range values {
		cache.Set(key, value, ttl)
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
		value, ok := cache.Get(key)
		if !ok {
			var err error
			value, err = getValue(key)
			if err != nil {
				log.Error("GetSetMulti Set %s: %s", key, err.Error())
			} else {
				cache.Set(key, value, ttl)
			}
		}
		values[key] = value
	}
	return values
}

// getEntry gets the raw entry, creating one if needed for list operations
func (cache *Cache) getEntry(key string) (*entry, bool) {
	prefixedKey := cache.prefixKey(key)
	raw, found := cache.lru.Get(prefixedKey)
	if !found {
		return nil, false
	}

	e, ok := raw.(*entry)
	if !ok {
		// Legacy value, wrap it
		return &entry{Value: raw}, true
	}

	if e.isExpired() {
		cache.lru.Remove(prefixedKey)
		return nil, false
	}

	return e, true
}

// Push adds values to the end of a list
func (cache *Cache) Push(key string, values ...interface{}) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	prefixedKey := cache.prefixKey(key)
	var list []interface{}
	if e, ok := cache.getEntry(key); ok {
		if existingList, ok := e.Value.([]interface{}); ok {
			list = make([]interface{}, len(existingList))
			copy(list, existingList)
		}
	}
	list = append(list, values...)
	cache.lru.Add(prefixedKey, &entry{Value: list})
	return nil
}

// Pop removes and returns an element from a list
func (cache *Cache) Pop(key string, position int) (interface{}, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	prefixedKey := cache.prefixKey(key)
	e, ok := cache.getEntry(key)
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	originalList, ok := e.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("key is not a list")
	}

	if len(originalList) == 0 {
		return nil, fmt.Errorf("list is empty")
	}

	var value interface{}
	var list []interface{}
	if position == 1 { // pop from end
		value = originalList[len(originalList)-1]
		list = make([]interface{}, len(originalList)-1)
		copy(list, originalList[:len(originalList)-1])
	} else { // pop from beginning
		value = originalList[0]
		list = make([]interface{}, len(originalList)-1)
		copy(list, originalList[1:])
	}

	if len(list) == 0 {
		cache.lru.Remove(prefixedKey)
	} else {
		cache.lru.Add(prefixedKey, &entry{Value: list})
	}

	return value, nil
}

// Pull removes all occurrences of a value from a list
func (cache *Cache) Pull(key string, value interface{}) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	prefixedKey := cache.prefixKey(key)
	e, ok := cache.getEntry(key)
	if !ok {
		return nil
	}

	list, ok := e.Value.([]interface{})
	if !ok {
		return fmt.Errorf("key is not a list")
	}

	var newList []interface{}
	for _, item := range list {
		if !isEqual(item, value) {
			newList = append(newList, item)
		}
	}

	if len(newList) == 0 {
		cache.lru.Remove(prefixedKey)
	} else {
		cache.lru.Add(prefixedKey, &entry{Value: newList})
	}

	return nil
}

// PullAll removes all occurrences of multiple values from a list
func (cache *Cache) PullAll(key string, values []interface{}) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	prefixedKey := cache.prefixKey(key)
	e, ok := cache.getEntry(key)
	if !ok {
		return nil
	}

	list, ok := e.Value.([]interface{})
	if !ok {
		return fmt.Errorf("key is not a list")
	}

	var newList []interface{}
	for _, item := range list {
		shouldRemove := false
		for _, removeValue := range values {
			if isEqual(item, removeValue) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			newList = append(newList, item)
		}
	}

	if len(newList) == 0 {
		cache.lru.Remove(prefixedKey)
	} else {
		cache.lru.Add(prefixedKey, &entry{Value: newList})
	}

	return nil
}

// AddToSet adds values to a list only if they don't already exist
func (cache *Cache) AddToSet(key string, values ...interface{}) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	prefixedKey := cache.prefixKey(key)
	var list []interface{}
	if e, ok := cache.getEntry(key); ok {
		if existingList, ok := e.Value.([]interface{}); ok {
			list = make([]interface{}, len(existingList))
			copy(list, existingList)
		}
	}

	for _, value := range values {
		exists := false
		for _, item := range list {
			if isEqual(item, value) {
				exists = true
				break
			}
		}
		if !exists {
			list = append(list, value)
		}
	}

	cache.lru.Add(prefixedKey, &entry{Value: list})
	return nil
}

// ArrayLen returns the length of a list
func (cache *Cache) ArrayLen(key string) int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	e, ok := cache.getEntry(key)
	if !ok {
		return 0
	}

	list, ok := e.Value.([]interface{})
	if !ok {
		return 0
	}

	return len(list)
}

// ArrayGet returns an element at the specified index
func (cache *Cache) ArrayGet(key string, index int) (interface{}, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	e, ok := cache.getEntry(key)
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	list, ok := e.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("key is not a list")
	}

	if index < 0 || index >= len(list) {
		return nil, fmt.Errorf("index out of range")
	}

	return list[index], nil
}

// ArraySet sets an element at the specified index
func (cache *Cache) ArraySet(key string, index int, value interface{}) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	prefixedKey := cache.prefixKey(key)
	e, ok := cache.getEntry(key)
	if !ok {
		return fmt.Errorf("key not found")
	}

	originalList, ok := e.Value.([]interface{})
	if !ok {
		return fmt.Errorf("key is not a list")
	}

	if index < 0 || index >= len(originalList) {
		return fmt.Errorf("index out of range")
	}

	// Create a copy and modify it
	list := make([]interface{}, len(originalList))
	copy(list, originalList)
	list[index] = value
	cache.lru.Add(prefixedKey, &entry{Value: list})
	return nil
}

// ArraySlice returns a slice of the list
func (cache *Cache) ArraySlice(key string, skip, limit int) ([]interface{}, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	e, ok := cache.getEntry(key)
	if !ok {
		return []interface{}{}, nil
	}

	list, ok := e.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("key is not a list")
	}

	if skip >= len(list) || skip < 0 {
		return []interface{}{}, nil
	}

	end := skip + limit
	if end > len(list) {
		end = len(list)
	}

	// Return a copy to prevent external modification
	result := make([]interface{}, end-skip)
	copy(result, list[skip:end])
	return result, nil
}

// ArrayPage returns a page of the list
func (cache *Cache) ArrayPage(key string, page, pageSize int) ([]interface{}, error) {
	if page < 1 || pageSize < 1 {
		return []interface{}{}, nil
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	e, ok := cache.getEntry(key)
	if !ok {
		return []interface{}{}, nil
	}

	list, ok := e.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("key is not a list")
	}

	skip := (page - 1) * pageSize
	if skip >= len(list) || skip < 0 {
		return []interface{}{}, nil
	}

	end := skip + pageSize
	if end > len(list) {
		end = len(list)
	}

	// Return a copy to prevent external modification
	result := make([]interface{}, end-skip)
	copy(result, list[skip:end])
	return result, nil
}

// ArrayAll returns all elements in the list
func (cache *Cache) ArrayAll(key string) ([]interface{}, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	e, ok := cache.getEntry(key)
	if !ok {
		return []interface{}{}, nil
	}

	list, ok := e.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("key is not a list")
	}

	// Return a copy to prevent external modification
	result := make([]interface{}, len(list))
	copy(result, list)
	return result, nil
}

// Incr increments a numeric value and returns the new value
func (cache *Cache) Incr(key string, delta int64) (int64, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	prefixedKey := cache.prefixKey(key)
	var current int64
	if e, ok := cache.getEntry(key); ok {
		current = toInt64(e.Value)
	}

	newValue := current + delta
	cache.lru.Add(prefixedKey, &entry{Value: newValue})
	return newValue, nil
}

// Decr decrements a numeric value and returns the new value
func (cache *Cache) Decr(key string, delta int64) (int64, error) {
	return cache.Incr(key, -delta)
}

// toInt64 converts an interface{} to int64
func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		return int64(n)
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

// isEqual compares two values for equality
func isEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
