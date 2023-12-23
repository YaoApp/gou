package store

// Key the cache
func (cache *Cache) Key() string {
	return cache.key
}

// Dispose the cache
func (cache *Cache) Dispose() {
	for _, ctx := range cache.contexts {
		ctx.Context.Close()
		ctx = nil
	}

	// if iso, has := Isolates.Get(cache.key); has {
	// 	iso.Dispose()
	// 	iso = nil
	// }

	cache.contexts = nil
	cache = nil
}
