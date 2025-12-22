package xun

import (
	"fmt"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/schema"
	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
)

// New create a new xun store with async persistence
func New(option Option) (*Store, error) {
	// Set defaults
	tableName := option.Table
	if tableName == "" {
		tableName = DefaultTableName
	}

	connector := option.Connector
	if connector == "" {
		connector = "default"
	}

	cacheSize := option.CacheSize
	if cacheSize <= 0 {
		cacheSize = DefaultCacheSize
	}

	cleanupInterval := option.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = DefaultCleanupInterval
	}

	persistInterval := option.PersistInterval
	if persistInterval <= 0 {
		persistInterval = DefaultPersistInterval
	}

	// Create LRU cache
	cache, err := lru.NewARC(cacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %v", err)
	}

	store := &Store{
		connector:       connector,
		tableName:       tableName,
		cache:           cache,
		cacheSize:       cacheSize,
		cleanupInterval: cleanupInterval,
		persistInterval: persistInterval,
		dirty:           make(map[string]*dirtyEntry),
		deleted:         make(map[string]bool),
		stopWorker:      make(chan struct{}),
		workerDone:      make(chan struct{}),
	}

	// Create table if not exists
	if err := store.ensureTable(); err != nil {
		return nil, fmt.Errorf("failed to create store table: %v", err)
	}

	// Start background worker for persistence and cleanup
	go store.backgroundWorker()

	return store, nil
}

// ensureTable creates the store table if it doesn't exist
func (store *Store) ensureTable() error {
	sch := schema.Use(store.connector)

	// Check if table exists
	exists, err := sch.TableExists(store.tableName)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// Create table
	blueprint := types.Blueprint{
		Columns: []types.Column{
			{Name: "key", Type: "string", Length: 255, Primary: true},
			{Name: "value", Type: "longText", Nullable: true},
			{Name: "type", Type: "string", Length: 20, Default: "value"},
			{Name: "expired_at", Type: "datetime", Nullable: true, Index: true},
			{Name: "created_at", Type: "datetime"},
			{Name: "updated_at", Type: "datetime"},
		},
		Indexes: []types.Index{
			{Name: "idx_expired_at", Type: "index", Columns: []string{"expired_at"}},
		},
	}

	return sch.TableCreate(store.tableName, blueprint)
}

// backgroundWorker handles periodic persistence and cleanup
func (store *Store) backgroundWorker() {
	defer close(store.workerDone)

	persistTicker := time.NewTicker(store.persistInterval)
	cleanupTicker := time.NewTicker(store.cleanupInterval)
	defer persistTicker.Stop()
	defer cleanupTicker.Stop()

	for {
		select {
		case <-store.stopWorker:
			// Final flush before exit
			store.flush()
			return
		case <-persistTicker.C:
			store.flush()
		case <-cleanupTicker.C:
			store.cleanupExpired()
		}
	}
}

// flush persists all dirty entries to database
func (store *Store) flush() {
	// Get dirty entries
	store.dirtyMu.Lock()
	store.deletedMu.Lock()
	if len(store.dirty) == 0 && len(store.deleted) == 0 {
		store.deletedMu.Unlock()
		store.dirtyMu.Unlock()
		return
	}
	store.deletedMu.Unlock()

	dirtyEntries := store.dirty
	store.dirty = make(map[string]*dirtyEntry)
	store.dirtyMu.Unlock()

	// Get deleted keys
	store.deletedMu.Lock()
	deletedKeys := store.deleted
	store.deleted = make(map[string]bool)
	store.deletedMu.Unlock()

	// Persist dirty entries
	for _, entry := range dirtyEntries {
		// Skip if key was deleted after being marked dirty
		if deletedKeys[entry.Key] {
			continue
		}

		if err := store.persistEntry(entry); err != nil {
			log.Error("Store xun persist %s: %s", entry.Key, err.Error())
			// Re-add to dirty on failure
			store.dirtyMu.Lock()
			store.dirty[entry.Key] = entry
			store.dirtyMu.Unlock()
		}
	}

	// Delete keys from database
	for key := range deletedKeys {
		if err := store.deleteFromDB(key); err != nil {
			log.Error("Store xun delete %s: %s", key, err.Error())
		}
	}
}

// persistEntry writes a single entry to database
func (store *Store) persistEntry(entry *dirtyEntry) error {
	valueBytes, err := jsoniter.Marshal(entry.Value)
	if err != nil {
		return err
	}

	var expiredAtDB interface{}
	if entry.ExpiredAt != nil {
		expiredAtDB = *entry.ExpiredAt
	}

	now := entry.UpdatedAt

	// Check if key exists in database
	count, err := capsule.Query().
		Table(store.tableName).
		Where("key", entry.Key).
		Count()

	if err != nil {
		return err
	}

	if count > 0 {
		// Update
		_, err = capsule.Query().
			Table(store.tableName).
			Where("key", entry.Key).
			Update(map[string]interface{}{
				"value":      string(valueBytes),
				"type":       entry.Type,
				"expired_at": expiredAtDB,
				"updated_at": now,
			})
	} else {
		// Insert
		err = capsule.Query().
			Table(store.tableName).
			Insert(
				[][]interface{}{{entry.Key, string(valueBytes), entry.Type, expiredAtDB, now, now}},
				[]string{"key", "value", "type", "expired_at", "created_at", "updated_at"},
			)
	}

	return err
}

// deleteFromDB removes a key from database
func (store *Store) deleteFromDB(key string) error {
	_, err := capsule.Query().
		Table(store.tableName).
		Where("key", key).
		Delete()
	return err
}

// cleanupExpired removes expired entries from cache and database
func (store *Store) cleanupExpired() {
	now := time.Now()

	// Clean from cache
	keys := store.cache.Keys()
	for _, k := range keys {
		if key, ok := k.(string); ok {
			if cached, found := store.cache.Peek(key); found {
				if entry, ok := cached.(*cacheEntry); ok {
					if entry.ExpiredAt != nil && now.After(*entry.ExpiredAt) {
						store.cache.Remove(key)
					}
				}
			}
		}
	}

	// Clean from database
	_, err := capsule.Query().
		Table(store.tableName).
		Where("expired_at", "<", now).
		WhereNotNull("expired_at").
		Delete()

	if err != nil {
		log.Error("Store xun cleanup expired: %s", err.Error())
	}
}

// Close stops the background worker and flushes pending data
func (store *Store) Close() {
	close(store.stopWorker)
	<-store.workerDone
}

// Flush forces immediate persistence of all dirty data
func (store *Store) Flush() {
	store.flush()
}

// markDirty marks a key as dirty for async persistence
func (store *Store) markDirty(key string, value interface{}, typ string, expiredAt *time.Time) {
	store.dirtyMu.Lock()
	store.dirty[key] = &dirtyEntry{
		Key:       key,
		Value:     value,
		Type:      typ,
		ExpiredAt: expiredAt,
		UpdatedAt: time.Now(),
	}
	store.dirtyMu.Unlock()

	// Remove from deleted if it was there
	store.deletedMu.Lock()
	delete(store.deleted, key)
	store.deletedMu.Unlock()
}

// markDeleted marks a key as deleted for async persistence
func (store *Store) markDeleted(key string) {
	store.deletedMu.Lock()
	store.deleted[key] = true
	store.deletedMu.Unlock()

	// Remove from dirty if it was there
	store.dirtyMu.Lock()
	delete(store.dirty, key)
	store.dirtyMu.Unlock()
}

// Get looks up a key's value from the store (cache-first, lazy load from DB)
func (store *Store) Get(key string) (value interface{}, ok bool) {
	// Check cache first
	if cached, found := store.cache.Get(key); found {
		if entry, ok := cached.(*cacheEntry); ok {
			// Check if expired
			if entry.ExpiredAt != nil && time.Now().After(*entry.ExpiredAt) {
				store.cache.Remove(key)
				return nil, false
			}
			return entry.Value, true
		}
	}

	// Lazy load from database
	row, err := capsule.Query().
		Table(store.tableName).
		Where("key", key).
		Where(func(qb query.Query) {
			qb.WhereNull("expired_at").OrWhere("expired_at", ">", time.Now())
		}).
		First()

	if err != nil {
		if !strings.Contains(err.Error(), "no rows") {
			log.Error("Store xun Get %s: %s", key, err.Error())
		}
		return nil, false
	}

	if row.IsEmpty() {
		return nil, false
	}

	// Unmarshal value
	valueStr, ok := row.Get("value").(string)
	if !ok {
		return nil, false
	}

	var result interface{}
	if err := jsoniter.UnmarshalFromString(valueStr, &result); err != nil {
		log.Error("Store xun Get unmarshal %s: %s", key, err.Error())
		return nil, false
	}

	// Get type and expiration
	typ := "value"
	if t, ok := row.Get("type").(string); ok {
		typ = t
	}

	var expiredAt *time.Time
	if exp := row.Get("expired_at"); exp != nil {
		if t, ok := exp.(time.Time); ok {
			expiredAt = &t
		}
	}

	// Add to cache (LRU will auto-evict if full)
	store.cache.Add(key, &cacheEntry{
		Value:     result,
		ExpiredAt: expiredAt,
		Type:      typ,
	})

	return result, true
}

// Set adds a value to the store (writes to cache, async persist)
func (store *Store) Set(key string, value interface{}, ttl time.Duration) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	var expiredAt *time.Time
	if ttl > 0 {
		exp := time.Now().Add(ttl)
		expiredAt = &exp
	}

	// Write to cache
	store.cache.Add(key, &cacheEntry{
		Value:     value,
		ExpiredAt: expiredAt,
		Type:      "value",
	})

	// Mark as dirty for async persistence
	store.markDirty(key, value, "value", expiredAt)

	return nil
}

// Del removes a key from the store
// Supports wildcard pattern with * (e.g., "user:123:*")
func (store *Store) Del(key string) error {
	// Check if key contains wildcard
	if strings.Contains(key, "*") {
		return store.delPattern(key)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	// Remove from cache
	store.cache.Remove(key)

	// Mark as deleted for async persistence
	store.markDeleted(key)

	return nil
}

// delPattern deletes all keys matching the pattern
func (store *Store) delPattern(pattern string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	// Convert wildcard pattern to SQL LIKE pattern
	// e.g., "user:123:*" -> "user:123:%"
	likePattern := strings.ReplaceAll(pattern, "*", "%")

	// Remove matching keys from cache
	keys := store.cache.Keys()
	prefix := strings.TrimSuffix(pattern, "*")
	for _, k := range keys {
		if key, ok := k.(string); ok {
			if strings.HasSuffix(pattern, "*") && strings.HasPrefix(key, prefix) {
				store.cache.Remove(key)
				store.markDeletedNoLock(key)
			}
		}
	}

	// Delete from database using LIKE
	_, err := capsule.Query().
		Table(store.tableName).
		Where("key", "like", likePattern).
		Delete()

	if err != nil {
		log.Error("Store xun delPattern %s: %s", pattern, err.Error())
		return err
	}

	return nil
}

// markDeletedNoLock marks a key as deleted without acquiring lock (caller must hold lock)
func (store *Store) markDeletedNoLock(key string) {
	store.deletedMu.Lock()
	store.deleted[key] = true
	store.deletedMu.Unlock()

	store.dirtyMu.Lock()
	delete(store.dirty, key)
	store.dirtyMu.Unlock()
}

// Has checks if a key exists in the store
func (store *Store) Has(key string) bool {
	// Check cache first
	if cached, found := store.cache.Get(key); found {
		if entry, ok := cached.(*cacheEntry); ok {
			if entry.ExpiredAt != nil && time.Now().After(*entry.ExpiredAt) {
				store.cache.Remove(key)
				return false
			}
			return true
		}
	}

	// Check database
	count, err := capsule.Query().
		Table(store.tableName).
		Where("key", key).
		Where(func(qb query.Query) {
			qb.WhereNull("expired_at").OrWhere("expired_at", ">", time.Now())
		}).
		Count()

	if err != nil {
		log.Error("Store xun Has %s: %s", key, err.Error())
		return false
	}

	return count > 0
}

// Len returns the number of stored entries
func (store *Store) Len() int {
	now := time.Now()
	count := 0

	// Count from cache (includes dirty data not yet persisted)
	keys := store.cache.Keys()
	for _, k := range keys {
		if key, ok := k.(string); ok {
			if cached, found := store.cache.Peek(key); found {
				if entry, ok := cached.(*cacheEntry); ok {
					if entry.ExpiredAt == nil || now.Before(*entry.ExpiredAt) {
						count++
					}
				}
			}
		}
	}

	// Also count from database for keys not in cache
	store.deletedMu.RLock()
	deletedKeys := make(map[string]bool)
	for k := range store.deleted {
		deletedKeys[k] = true
	}
	store.deletedMu.RUnlock()

	rows, err := capsule.Query().
		Table(store.tableName).
		Select("key").
		Where(func(qb query.Query) {
			qb.WhereNull("expired_at").OrWhere("expired_at", ">", time.Now())
		}).
		Get()

	if err != nil {
		return count
	}

	// Add keys from DB that are not in cache and not deleted
	for _, row := range rows {
		if key, ok := row.Get("key").(string); ok {
			if !store.cache.Contains(key) && !deletedKeys[key] {
				count++
			}
		}
	}

	return count
}

// Keys returns all the keys
func (store *Store) Keys() []string {
	now := time.Now()
	keySet := make(map[string]bool)

	// Get keys from cache
	keys := store.cache.Keys()
	for _, k := range keys {
		if key, ok := k.(string); ok {
			if cached, found := store.cache.Peek(key); found {
				if entry, ok := cached.(*cacheEntry); ok {
					if entry.ExpiredAt == nil || now.Before(*entry.ExpiredAt) {
						keySet[key] = true
					}
				}
			}
		}
	}

	// Get deleted keys
	store.deletedMu.RLock()
	deletedKeys := make(map[string]bool)
	for k := range store.deleted {
		deletedKeys[k] = true
	}
	store.deletedMu.RUnlock()

	// Get keys from database
	rows, err := capsule.Query().
		Table(store.tableName).
		Select("key").
		Where(func(qb query.Query) {
			qb.WhereNull("expired_at").OrWhere("expired_at", ">", time.Now())
		}).
		Get()

	if err == nil {
		for _, row := range rows {
			if key, ok := row.Get("key").(string); ok {
				if !deletedKeys[key] {
					keySet[key] = true
				}
			}
		}
	}

	result := make([]string, 0, len(keySet))
	for key := range keySet {
		result = append(result, key)
	}

	return result
}

// Clear removes all entries from the store
func (store *Store) Clear() {
	store.mu.Lock()
	defer store.mu.Unlock()

	// Get all keys before clearing
	keys := store.cache.Keys()
	for _, k := range keys {
		if key, ok := k.(string); ok {
			store.markDeleted(key)
		}
	}

	// Clear cache
	store.cache.Purge()

	// Clear dirty entries
	store.dirtyMu.Lock()
	store.dirty = make(map[string]*dirtyEntry)
	store.dirtyMu.Unlock()
}

// GetSet looks up a key's value from the store, if not exist add to the store
func (store *Store) GetSet(key string, ttl time.Duration, getValue func(key string) (interface{}, error)) (interface{}, error) {
	value, ok := store.Get(key)
	if ok {
		return value, nil
	}

	newValue, err := getValue(key)
	if err != nil {
		return nil, err
	}

	if err := store.Set(key, newValue, ttl); err != nil {
		return nil, err
	}

	return newValue, nil
}

// GetDel looks up a key's value from the store, then remove it
func (store *Store) GetDel(key string) (value interface{}, ok bool) {
	value, ok = store.Get(key)
	if !ok {
		return nil, false
	}

	if err := store.Del(key); err != nil {
		return value, false
	}

	return value, true
}

// GetMulti gets multiple values at once
func (store *Store) GetMulti(keys []string) map[string]interface{} {
	values := make(map[string]interface{})
	for _, key := range keys {
		value, _ := store.Get(key)
		values[key] = value
	}
	return values
}

// SetMulti sets multiple key-value pairs at once
func (store *Store) SetMulti(values map[string]interface{}, ttl time.Duration) {
	for key, value := range values {
		store.Set(key, value, ttl)
	}
}

// DelMulti deletes multiple keys at once
func (store *Store) DelMulti(keys []string) {
	for _, key := range keys {
		store.Del(key)
	}
}

// GetSetMulti gets multiple values, setting defaults for missing ones
func (store *Store) GetSetMulti(keys []string, ttl time.Duration, getValue func(key string) (interface{}, error)) map[string]interface{} {
	values := make(map[string]interface{})
	for _, key := range keys {
		value, err := store.GetSet(key, ttl, getValue)
		if err == nil {
			values[key] = value
		}
	}
	return values
}

// List Operations

// getListFromCache gets a list from cache, lazy load from DB if not found
func (store *Store) getListFromCache(key string) ([]interface{}, bool) {
	// Check cache first
	if cached, found := store.cache.Get(key); found {
		if entry, ok := cached.(*cacheEntry); ok {
			if entry.ExpiredAt != nil && time.Now().After(*entry.ExpiredAt) {
				store.cache.Remove(key)
				return nil, false
			}
			if list, ok := entry.Value.([]interface{}); ok {
				return list, true
			}
		}
	}

	// Lazy load from database
	row, err := capsule.Query().
		Table(store.tableName).
		Where("key", key).
		Where(func(qb query.Query) {
			qb.WhereNull("expired_at").OrWhere("expired_at", ">", time.Now())
		}).
		First()

	if err != nil {
		if !strings.Contains(err.Error(), "no rows") {
			log.Error("Store xun getList %s: %s", key, err.Error())
		}
		return nil, false
	}

	if row.IsEmpty() {
		return nil, false
	}

	valueStr, ok := row.Get("value").(string)
	if !ok {
		return nil, false
	}

	var list []interface{}
	if err := jsoniter.UnmarshalFromString(valueStr, &list); err != nil {
		return nil, false
	}

	// Add to cache
	store.cache.Add(key, &cacheEntry{
		Value:     list,
		ExpiredAt: nil,
		Type:      "list",
	})

	return list, true
}

// setListToCache sets a list to cache and marks dirty
func (store *Store) setListToCache(key string, list []interface{}) {
	store.cache.Add(key, &cacheEntry{
		Value:     list,
		ExpiredAt: nil,
		Type:      "list",
	})
	store.markDirty(key, list, "list", nil)
}

// Push adds values to the end of a list
func (store *Store) Push(key string, values ...interface{}) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	list, _ := store.getListFromCache(key)
	if list == nil {
		list = []interface{}{}
	}

	list = append(list, values...)
	store.setListToCache(key, list)

	return nil
}

// Pop removes and returns an element from a list
func (store *Store) Pop(key string, position int) (interface{}, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	list, found := store.getListFromCache(key)
	if !found || len(list) == 0 {
		return nil, fmt.Errorf("list is empty")
	}

	var result interface{}
	if position == 1 { // pop from end
		result = list[len(list)-1]
		list = list[:len(list)-1]
	} else { // pop from beginning
		result = list[0]
		list = list[1:]
	}

	store.setListToCache(key, list)

	return result, nil
}

// Pull removes all occurrences of a value from a list
func (store *Store) Pull(key string, value interface{}) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	list, found := store.getListFromCache(key)
	if !found {
		return nil
	}

	valueBytes, _ := jsoniter.Marshal(value)
	var newList []interface{}
	for _, item := range list {
		itemBytes, _ := jsoniter.Marshal(item)
		if string(itemBytes) != string(valueBytes) {
			newList = append(newList, item)
		}
	}

	if newList == nil {
		newList = []interface{}{}
	}

	store.setListToCache(key, newList)

	return nil
}

// PullAll removes all occurrences of multiple values from a list
func (store *Store) PullAll(key string, values []interface{}) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	list, found := store.getListFromCache(key)
	if !found {
		return nil
	}

	valuesToRemove := make(map[string]bool)
	for _, v := range values {
		data, _ := jsoniter.Marshal(v)
		valuesToRemove[string(data)] = true
	}

	var newList []interface{}
	for _, item := range list {
		itemData, _ := jsoniter.Marshal(item)
		if !valuesToRemove[string(itemData)] {
			newList = append(newList, item)
		}
	}

	if newList == nil {
		newList = []interface{}{}
	}

	store.setListToCache(key, newList)

	return nil
}

// AddToSet adds values only if they don't already exist
func (store *Store) AddToSet(key string, values ...interface{}) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	list, _ := store.getListFromCache(key)
	if list == nil {
		list = []interface{}{}
	}

	existingValues := make(map[string]bool)
	for _, item := range list {
		data, _ := jsoniter.Marshal(item)
		existingValues[string(data)] = true
	}

	for _, value := range values {
		valueData, _ := jsoniter.Marshal(value)
		if !existingValues[string(valueData)] {
			list = append(list, value)
			existingValues[string(valueData)] = true
		}
	}

	store.setListToCache(key, list)

	return nil
}

// ArrayLen returns the length of a list
func (store *Store) ArrayLen(key string) int {
	list, found := store.getListFromCache(key)
	if !found {
		return 0
	}
	return len(list)
}

// ArrayGet returns an element at the specified index
func (store *Store) ArrayGet(key string, index int) (interface{}, error) {
	list, found := store.getListFromCache(key)
	if !found {
		return nil, fmt.Errorf("key not found")
	}

	if index < 0 || index >= len(list) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}

	return list[index], nil
}

// ArraySet sets an element at the specified index
func (store *Store) ArraySet(key string, index int, value interface{}) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	list, found := store.getListFromCache(key)
	if !found {
		return fmt.Errorf("key not found")
	}

	if index < 0 || index >= len(list) {
		return fmt.Errorf("index out of range: %d", index)
	}

	// Make a copy to avoid modifying cached slice
	newList := make([]interface{}, len(list))
	copy(newList, list)
	newList[index] = value

	store.setListToCache(key, newList)

	return nil
}

// ArraySlice returns a slice of elements with skip and limit
func (store *Store) ArraySlice(key string, skip, limit int) ([]interface{}, error) {
	list, found := store.getListFromCache(key)
	if !found {
		return []interface{}{}, nil
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

	result := make([]interface{}, end-skip)
	copy(result, list[skip:end])
	return result, nil
}

// ArrayPage returns a specific page of elements
func (store *Store) ArrayPage(key string, page, pageSize int) ([]interface{}, error) {
	if page < 1 {
		page = 1
	}
	skip := (page - 1) * pageSize
	return store.ArraySlice(key, skip, pageSize)
}

// ArrayAll returns all elements in a list
func (store *Store) ArrayAll(key string) ([]interface{}, error) {
	list, found := store.getListFromCache(key)
	if !found {
		return []interface{}{}, nil
	}

	result := make([]interface{}, len(list))
	copy(result, list)
	return result, nil
}

// keyExists checks if a key exists in the database
func (store *Store) keyExists(key string) (bool, error) {
	count, err := capsule.Query().
		Table(store.tableName).
		Where("key", key).
		Count()

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Incr increments a numeric value and returns the new value
func (store *Store) Incr(key string, delta int64) (int64, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	var current int64

	// Check cache first
	if cached, found := store.cache.Get(key); found {
		if entry, ok := cached.(*cacheEntry); ok {
			current = toInt64(entry.Value)
		}
	} else {
		// Load from database
		if value, ok := store.Get(key); ok {
			current = toInt64(value)
		}
	}

	newValue := current + delta

	// Update cache
	store.cache.Add(key, &cacheEntry{
		Value:     newValue,
		ExpiredAt: nil,
		Type:      "value",
	})

	// Mark as dirty for async persistence
	store.markDirty(key, newValue, "value", nil)

	return newValue, nil
}

// Decr decrements a numeric value and returns the new value
func (store *Store) Decr(key string, delta int64) (int64, error) {
	return store.Incr(key, -delta)
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
