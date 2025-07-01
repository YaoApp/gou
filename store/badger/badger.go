package badger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/yaoapp/gou/application"
)

// Badger the badger store
type Badger struct {
	db   *badger.DB
	path string
	mu   sync.RWMutex // Protects operations for concurrency safety
}

// New create a new badger store
func New(path string) (*Badger, error) {
	// Handle relative and absolute paths
	var dbPath string
	if strings.HasPrefix(path, "/") {
		// Absolute path
		dbPath = path
	} else if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		// Relative path
		dbPath = path
	} else {
		// Relative to project root - use application root
		root := application.App.Root()
		dbPath = filepath.Join(root, path)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %v", dbPath, err)
	}

	// Open badger database
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil // Disable badger logs to avoid noise

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger database: %v", err)
	}

	return &Badger{
		db:   db,
		path: dbPath,
	}, nil
}

// Close close the badger database
func (b *Badger) Close() error {
	return b.db.Close()
}

// Key-Value Operations

// Get get a value by key
func (b *Badger) Get(key string) (value interface{}, ok bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result interface{}
	var found bool

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}

		return item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, &result); err != nil {
				return err
			}
			found = true
			return nil
		})
	})

	if err != nil {
		return nil, false
	}

	return result, found
}

// Set set a key-value pair with optional TTL
func (b *Badger) Set(key string, value interface{}, ttl time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %v", err)
	}

	return b.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), data)
		if ttl > 0 {
			entry = entry.WithTTL(ttl)
		}
		return txn.SetEntry(entry)
	})
}

// Del delete a key
func (b *Badger) Del(key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Has check if a key exists
func (b *Badger) Has(key string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var exists bool
	b.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		exists = (err == nil)
		return nil
	})
	return exists
}

// Len get the number of keys in the store
func (b *Badger) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	count := 0
	b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // Only need keys for counting
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})
	return count
}

// Keys get all keys in the store
func (b *Badger) Keys() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var keys []string
	b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // Only need keys
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			keys = append(keys, key)
		}
		return nil
	})
	return keys
}

// Clear remove all keys from the store
func (b *Badger) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			if err := txn.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetSet get a value or set it if it doesn't exist
func (b *Badger) GetSet(key string, ttl time.Duration, getValue func(key string) (interface{}, error)) (interface{}, error) {
	// First try to get the value
	if value, ok := b.Get(key); ok {
		return value, nil
	}

	// Generate new value
	newValue, err := getValue(key)
	if err != nil {
		return nil, err
	}

	// Set the new value
	if err := b.Set(key, newValue, ttl); err != nil {
		return nil, err
	}

	return newValue, nil
}

// GetDel get a value and delete it atomically
func (b *Badger) GetDel(key string) (value interface{}, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var result interface{}
	var found bool

	err := b.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}

		err = item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, &result); err != nil {
				return err
			}
			found = true
			return nil
		})
		if err != nil {
			return err
		}

		// Delete the key after getting the value
		return txn.Delete([]byte(key))
	})

	if err != nil {
		return nil, false
	}

	return result, found
}

// GetMulti get multiple values at once
func (b *Badger) GetMulti(keys []string) map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make(map[string]interface{})

	b.db.View(func(txn *badger.Txn) error {
		for _, key := range keys {
			item, err := txn.Get([]byte(key))
			if err != nil {
				continue // Skip missing keys
			}

			item.Value(func(val []byte) error {
				var value interface{}
				if err := json.Unmarshal(val, &value); err == nil {
					result[key] = value
				}
				return nil
			})
		}
		return nil
	})

	return result
}

// SetMulti set multiple key-value pairs at once
func (b *Badger) SetMulti(values map[string]interface{}, ttl time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.db.Update(func(txn *badger.Txn) error {
		for key, value := range values {
			data, err := json.Marshal(value)
			if err != nil {
				continue // Skip invalid values
			}

			entry := badger.NewEntry([]byte(key), data)
			if ttl > 0 {
				entry = entry.WithTTL(ttl)
			}

			if err := txn.SetEntry(entry); err != nil {
				return err
			}
		}
		return nil
	})
}

// DelMulti delete multiple keys at once
func (b *Badger) DelMulti(keys []string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.db.Update(func(txn *badger.Txn) error {
		for _, key := range keys {
			txn.Delete([]byte(key))
		}
		return nil
	})
}

// GetSetMulti get multiple values, setting defaults for missing ones
func (b *Badger) GetSetMulti(keys []string, ttl time.Duration, getValue func(key string) (interface{}, error)) map[string]interface{} {
	result := make(map[string]interface{})

	for _, key := range keys {
		value, err := b.GetSet(key, ttl, getValue)
		if err == nil {
			result[key] = value
		}
	}

	return result
}

// List Operations

// getList safely gets a list from storage
func (b *Badger) getList(key string) ([]interface{}, error) {
	var list []interface{}

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // Return empty list for non-existent keys
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &list)
		})
	})

	return list, err
}

// setList safely sets a list to storage
func (b *Badger) setList(key string, list []interface{}) error {
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// Push add elements to the end of a list
func (b *Badger) Push(key string, values ...interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	list, err := b.getList(key)
	if err != nil {
		return err
	}

	list = append(list, values...)
	return b.setList(key, list)
}

// Pop remove and return an element from a list
func (b *Badger) Pop(key string, position int) (interface{}, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	list, err := b.getList(key)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("list is empty")
	}

	var result interface{}
	var index int
	if position == 1 {
		// Pop from end
		index = len(list) - 1
	} else if position == -1 {
		// Pop from beginning
		index = 0
	} else {
		return nil, fmt.Errorf("invalid position: %d", position)
	}

	// Get the value to return
	result = list[index]

	// Remove element from list
	if index == 0 {
		list = list[1:]
	} else if index == len(list)-1 {
		list = list[:len(list)-1]
	} else {
		list = append(list[:index], list[index+1:]...)
	}

	return result, b.setList(key, list)
}

// Pull remove all occurrences of a specific value
func (b *Badger) Pull(key string, value interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	list, err := b.getList(key)
	if err != nil {
		return err
	}

	// Remove all occurrences of the value
	var newList []interface{}
	for _, item := range list {
		// Deep comparison using JSON
		itemData, _ := json.Marshal(item)
		valueData, _ := json.Marshal(value)
		if string(itemData) != string(valueData) {
			newList = append(newList, item)
		}
	}

	return b.setList(key, newList)
}

// PullAll remove all occurrences of multiple values
func (b *Badger) PullAll(key string, values []interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	list, err := b.getList(key)
	if err != nil {
		return err
	}

	// Create a set of values to remove for efficient lookup
	valuesToRemove := make(map[string]bool)
	for _, v := range values {
		data, _ := json.Marshal(v)
		valuesToRemove[string(data)] = true
	}

	// Remove all occurrences of the values
	var newList []interface{}
	for _, item := range list {
		itemData, _ := json.Marshal(item)
		if !valuesToRemove[string(itemData)] {
			newList = append(newList, item)
		}
	}

	return b.setList(key, newList)
}

// AddToSet add elements only if they don't already exist
func (b *Badger) AddToSet(key string, values ...interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	list, err := b.getList(key)
	if err != nil {
		return err
	}

	// Create a set of existing values for efficient lookup
	existingValues := make(map[string]bool)
	for _, item := range list {
		data, _ := json.Marshal(item)
		existingValues[string(data)] = true
	}

	// Add only unique values
	for _, value := range values {
		valueData, _ := json.Marshal(value)
		if !existingValues[string(valueData)] {
			list = append(list, value)
			existingValues[string(valueData)] = true
		}
	}

	return b.setList(key, list)
}

// ArrayLen get the length of a list
func (b *Badger) ArrayLen(key string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	list, err := b.getList(key)
	if err != nil {
		return 0
	}
	return len(list)
}

// ArrayGet get an element at a specific index
func (b *Badger) ArrayGet(key string, index int) (interface{}, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	list, err := b.getList(key)
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(list) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}

	return list[index], nil
}

// ArraySet set an element at a specific index
func (b *Badger) ArraySet(key string, index int, value interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	list, err := b.getList(key)
	if err != nil {
		return err
	}

	if index < 0 || index >= len(list) {
		return fmt.Errorf("index out of range: %d", index)
	}

	// Set the value
	list[index] = value

	return b.setList(key, list)
}

// ArraySlice get a slice of elements with skip and limit
func (b *Badger) ArraySlice(key string, skip, limit int) ([]interface{}, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	list, err := b.getList(key)
	if err != nil {
		return nil, err
	}

	if skip < 0 {
		skip = 0
	}
	if skip >= len(list) {
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

// ArrayPage get a specific page of elements
func (b *Badger) ArrayPage(key string, page, pageSize int) ([]interface{}, error) {
	if page < 1 {
		page = 1
	}
	skip := (page - 1) * pageSize
	return b.ArraySlice(key, skip, pageSize)
}

// ArrayAll get all elements in a list
func (b *Badger) ArrayAll(key string) ([]interface{}, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	list, err := b.getList(key)
	if err != nil {
		return nil, err
	}

	// Return a copy to prevent external modification
	result := make([]interface{}, len(list))
	copy(result, list)
	return result, nil
}
