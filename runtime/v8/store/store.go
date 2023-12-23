package store

import (
	"sync"
)

// Isolates the new isolate store
var Isolates = New()

// New create a new store
func New() *Store {
	return &Store{data: map[string]IStore{}, mutex: &sync.Mutex{}}
}

// Get get a isolate
func (store *Store) Get(key string) (IStore, bool) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	v, ok := store.data[key]
	if !ok {
		return nil, false
	}
	return v.(IStore), true
}

// Add a isolate
func (store *Store) Add(data IStore) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.data[data.Key()] = data

}

// Remove a isolate
func (store *Store) Remove(key string) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	delete(store.data, key)

}

// Len the length of store
func (store *Store) Len() int {
	return len(store.data)
}

// Range traverse isolates
func (store *Store) Range(callback func(data IStore) bool) {
	for _, v := range store.data {
		if !callback(v.(IStore)) {
			break
		}
	}
}
