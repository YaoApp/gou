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

// Push adds values to the end of a list
func (cache *Cache) Push(key string, values ...interface{}) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	var list []interface{}
	if existing, ok := cache.lru.Get(key); ok {
		if existingList, ok := existing.([]interface{}); ok {
			// Create a copy to avoid modifying the original
			list = make([]interface{}, len(existingList))
			copy(list, existingList)
		}
	}
	list = append(list, values...)
	cache.lru.Add(key, list)
	return nil
}

// Pop removes and returns an element from a list
func (cache *Cache) Pop(key string, position int) (interface{}, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	existing, ok := cache.lru.Get(key)
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	originalList, ok := existing.([]interface{})
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
		cache.lru.Remove(key)
	} else {
		cache.lru.Add(key, list)
	}

	return value, nil
}

// Pull removes all occurrences of a value from a list
func (cache *Cache) Pull(key string, value interface{}) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	existing, ok := cache.lru.Get(key)
	if !ok {
		return nil
	}

	list, ok := existing.([]interface{})
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
		cache.lru.Remove(key)
	} else {
		cache.lru.Add(key, newList)
	}

	return nil
}

// PullAll removes all occurrences of multiple values from a list
func (cache *Cache) PullAll(key string, values []interface{}) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	existing, ok := cache.lru.Get(key)
	if !ok {
		return nil
	}

	list, ok := existing.([]interface{})
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
		cache.lru.Remove(key)
	} else {
		cache.lru.Add(key, newList)
	}

	return nil
}

// AddToSet adds values to a list only if they don't already exist
func (cache *Cache) AddToSet(key string, values ...interface{}) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	var list []interface{}
	if existing, ok := cache.lru.Get(key); ok {
		if existingList, ok := existing.([]interface{}); ok {
			// Create a copy to avoid modifying the original
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

	cache.lru.Add(key, list)
	return nil
}

// ArrayLen returns the length of a list
func (cache *Cache) ArrayLen(key string) int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	existing, ok := cache.lru.Get(key)
	if !ok {
		return 0
	}

	list, ok := existing.([]interface{})
	if !ok {
		return 0
	}

	return len(list)
}

// ArrayGet returns an element at the specified index
func (cache *Cache) ArrayGet(key string, index int) (interface{}, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	existing, ok := cache.lru.Get(key)
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	list, ok := existing.([]interface{})
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

	existing, ok := cache.lru.Get(key)
	if !ok {
		return fmt.Errorf("key not found")
	}

	originalList, ok := existing.([]interface{})
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
	cache.lru.Add(key, list)
	return nil
}

// ArraySlice returns a slice of the list
func (cache *Cache) ArraySlice(key string, skip, limit int) ([]interface{}, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	existing, ok := cache.lru.Get(key)
	if !ok {
		return []interface{}{}, nil
	}

	list, ok := existing.([]interface{})
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

	existing, ok := cache.lru.Get(key)
	if !ok {
		return []interface{}{}, nil
	}

	list, ok := existing.([]interface{})
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

	existing, ok := cache.lru.Get(key)
	if !ok {
		return []interface{}{}, nil
	}

	list, ok := existing.([]interface{})
	if !ok {
		return nil, fmt.Errorf("key is not a list")
	}

	// Return a copy to prevent external modification
	result := make([]interface{}, len(list))
	copy(result, list)
	return result, nil
}

// isEqual compares two values for equality
func isEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
