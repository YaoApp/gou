package store

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/any"
)

func TestLoad(t *testing.T) {

}

func TestLRU(t *testing.T) {
	lru := newStore(t, nil)
	testBasic(t, lru)
	testMulti(t, lru)
	testList(t, lru)
}

func TestRedis(t *testing.T) {
	redis := newStore(t, getConnector(t, "redis"))
	testBasic(t, redis)
	testMulti(t, redis)
	testList(t, redis)
}

func TestMongo(t *testing.T) {
	mongo := newStore(t, getConnector(t, "mongo"))
	testBasic(t, mongo)
	testMulti(t, mongo)
	testList(t, mongo)
}

func TestBadger(t *testing.T) {
	badger := newBadgerStore(t)
	testBasic(t, badger)
	testMulti(t, badger)
	testList(t, badger)
}

func TestLRUConcurrency(t *testing.T) {
	lru := newStore(t, nil)
	testConcurrency(t, lru)
	testMemoryLeak(t, lru)
	testGoroutineLeak(t, lru)
}

func TestRedisConcurrency(t *testing.T) {
	redis := newStore(t, getConnector(t, "redis"))
	testConcurrency(t, redis)
	testMemoryLeak(t, redis)
	testGoroutineLeak(t, redis)
}

func TestMongoConcurrency(t *testing.T) {
	mongo := newStore(t, getConnector(t, "mongo"))
	testConcurrency(t, mongo)
	testMemoryLeak(t, mongo)
	testGoroutineLeak(t, mongo)
}

func TestBadgerConcurrency(t *testing.T) {
	badger := newBadgerStore(t)
	testConcurrency(t, badger)
	testMemoryLeak(t, badger)
	testGoroutineLeak(t, badger)
}

func testBasic(t *testing.T, kv Store) {

	var err error
	kv.Clear()
	kv.Set("key1", "bar", 0)
	kv.Set("key2", 1024, 0)
	kv.Set("key3", 0.618, 0)
	value, ok := kv.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "bar", value)

	value, ok = kv.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 1024, any.Of(value).CInt())

	value, ok = kv.Get("key3")
	assert.True(t, ok)
	assert.Equal(t, 0.618, value)

	kv.Set("key1", "foo", 0)
	value, ok = kv.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "foo", value)
	assert.True(t, kv.Has("key1"))

	kv.Del("key1")
	_, ok = kv.Get("key1")
	assert.False(t, ok)
	assert.False(t, kv.Has("key1"))

	assert.Equal(t, 2, kv.Len())

	assert.False(t, kv.Has("key1"))
	assert.True(t, kv.Has("key2"))
	assert.True(t, kv.Has("key3"))

	assert.Contains(t, kv.Keys(), "key2")
	assert.Contains(t, kv.Keys(), "key3")
	assert.Equal(t, 2, len(kv.Keys()))

	kv.Clear()
	assert.Equal(t, 0, kv.Len())

	value, err = kv.GetSet("key1", 0, func(key string) (interface{}, error) {
		return "bar", nil
	})
	assert.Nil(t, err)
	value, ok = kv.Get("key1")
	assert.Equal(t, "bar", value)

	value, err = kv.GetSet("key1", 0, func(key string) (interface{}, error) {
		return nil, fmt.Errorf("error test")
	})
	assert.Nil(t, err)
	assert.Equal(t, "bar", value)

	value, err = kv.GetSet("key2", 0, func(key string) (interface{}, error) {
		return nil, fmt.Errorf("error test")
	})
	assert.Equal(t, "error test", err.Error())
	assert.Nil(t, value)

	value, ok = kv.GetDel("key1")
	assert.True(t, ok)
	assert.Equal(t, "bar", value)
	assert.Equal(t, 0, kv.Len())

}

func testMulti(t *testing.T, kv Store) {

	kv.SetMulti(map[string]interface{}{"key1": "foo", "key2": 1024, "key3": 0.618}, 0)
	assert.Equal(t, 3, kv.Len())

	values := kv.GetMulti([]string{"key1", "key2", "key3", "key4"})
	assert.Equal(t, "foo", values["key1"])
	assert.Equal(t, 1024, any.Of(values["key2"]).CInt())
	assert.Equal(t, 0.618, values["key3"])
	assert.Equal(t, nil, values["key4"])

	kv.DelMulti([]string{"key1", "key2", "key3"})
	assert.Equal(t, 0, kv.Len())

	values = kv.GetSetMulti([]string{"key1", "key2", "key3", "key4"}, 0, func(key string) (interface{}, error) {
		return key, nil
	})
	assert.Equal(t, "key1", values["key1"])
	assert.Equal(t, "key2", values["key2"])
	assert.Equal(t, "key3", values["key3"])
	assert.Equal(t, "key4", values["key4"])
	kv.Clear()

	values = kv.GetSetMulti([]string{"key1", "key2", "key3", "key4"}, 0, func(key string) (interface{}, error) {
		switch key {
		case "key1", "key2":
			return key, nil
		default:
			return nil, fmt.Errorf("error test")
		}
	})

	assert.Equal(t, "key1", values["key1"])
	assert.Equal(t, "key2", values["key2"])
	assert.Equal(t, nil, values["key3"])
	assert.Equal(t, nil, values["key4"])

	kv.DelMulti([]string{"key1", "key2"})
	assert.Equal(t, 0, kv.Len())
}

func newStore(t *testing.T, c connector.Connector) Store {
	store, err := New(c, Option{"size": 20480})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func newBadgerStore(t *testing.T) Store {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("badger_test_%d", time.Now().UnixNano()))

	// Create badger store with path option - like LRU with size
	store, err := New(nil, Option{"path": tempDir})
	if err != nil {
		t.Fatal(err)
	}

	// Schedule cleanup
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return store
}

func getConnector(t *testing.T, name string) connector.Connector {
	return connector.Connectors[name]
}

func prepareStores(t *testing.T) {
	stores := map[string]string{
		"cache":  filepath.Join("stores", "cache.lru.yao"),
		"share":  filepath.Join("stores", "share.redis.yao"),
		"data":   filepath.Join("stores", "data.mongo.yao"),
		"badger": filepath.Join("stores", "local.badger.yao"),
	}
	for id, file := range stores {
		_, err := Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func prepare(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	connectors := map[string]string{
		"mysql":  filepath.Join("connectors", "mysql.conn.yao"),
		"mongo":  filepath.Join("connectors", "mongo.conn.yao"),
		"redis":  filepath.Join("connectors", "redis.conn.yao"),
		"sqlite": filepath.Join("connectors", "sqlite.conn.yao"),
	}

	for id, file := range connectors {
		_, err = connector.Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func testList(t *testing.T, kv Store) {
	// Clear any existing data
	kv.Clear()

	// Test Push operation
	err := kv.Push("testlist", "item1", "item2", "item3")
	assert.Nil(t, err)

	// Test ArrayLen
	assert.Equal(t, 3, kv.ArrayLen("testlist"))

	// Test ArrayGet
	value, err := kv.ArrayGet("testlist", 0)
	assert.Nil(t, err)
	assert.Equal(t, "item1", value)

	value, err = kv.ArrayGet("testlist", 2)
	assert.Nil(t, err)
	assert.Equal(t, "item3", value)

	// Test ArraySet
	err = kv.ArraySet("testlist", 1, "modified_item2")
	assert.Nil(t, err)

	value, err = kv.ArrayGet("testlist", 1)
	assert.Nil(t, err)
	assert.Equal(t, "modified_item2", value)

	// Test ArrayAll
	allItems, err := kv.ArrayAll("testlist")
	assert.Nil(t, err)
	assert.Equal(t, 3, len(allItems))
	assert.Equal(t, "item1", allItems[0])
	assert.Equal(t, "modified_item2", allItems[1])
	assert.Equal(t, "item3", allItems[2])

	// Test ArraySlice
	slice, err := kv.ArraySlice("testlist", 1, 2)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(slice))
	assert.Equal(t, "modified_item2", slice[0])
	assert.Equal(t, "item3", slice[1])

	// Test ArrayPage
	page, err := kv.ArrayPage("testlist", 1, 2)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(page))
	assert.Equal(t, "item1", page[0])
	assert.Equal(t, "modified_item2", page[1])

	page, err = kv.ArrayPage("testlist", 2, 2)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(page))
	assert.Equal(t, "item3", page[0])

	// Test Pop operation - from end (position = 1)
	poppedValue, err := kv.Pop("testlist", 1)
	assert.Nil(t, err)
	assert.Equal(t, "item3", poppedValue)
	assert.Equal(t, 2, kv.ArrayLen("testlist"))

	// Test Pop operation - from beginning (position = -1)
	poppedValue, err = kv.Pop("testlist", -1)
	assert.Nil(t, err)
	assert.Equal(t, "item1", poppedValue)
	assert.Equal(t, 1, kv.ArrayLen("testlist"))

	// Add more items for testing removal operations
	err = kv.Push("testlist", "apple", "banana", "apple", "cherry", "apple")
	assert.Nil(t, err)
	assert.Equal(t, 6, kv.ArrayLen("testlist"))

	// Test Pull operation - remove all occurrences of "apple"
	err = kv.Pull("testlist", "apple")
	assert.Nil(t, err)
	assert.Equal(t, 3, kv.ArrayLen("testlist"))

	allItems, err = kv.ArrayAll("testlist")
	assert.Nil(t, err)
	assert.Equal(t, "modified_item2", allItems[0])
	assert.Equal(t, "banana", allItems[1])
	assert.Equal(t, "cherry", allItems[2])

	// Test PullAll operation
	err = kv.PullAll("testlist", []interface{}{"banana", "cherry"})
	assert.Nil(t, err)
	assert.Equal(t, 1, kv.ArrayLen("testlist"))

	allItems, err = kv.ArrayAll("testlist")
	assert.Nil(t, err)
	assert.Equal(t, "modified_item2", allItems[0])

	// Test AddToSet operation
	err = kv.AddToSet("testlist", "modified_item2", "new_item", "another_item")
	assert.Nil(t, err)

	// Should have 3 items now (modified_item2 was not added again)
	assert.Equal(t, 3, kv.ArrayLen("testlist"))

	allItems, err = kv.ArrayAll("testlist")
	assert.Nil(t, err)
	assert.Contains(t, allItems, "modified_item2")
	assert.Contains(t, allItems, "new_item")
	assert.Contains(t, allItems, "another_item")

	// Test empty list operations
	kv.Clear()
	assert.Equal(t, 0, kv.ArrayLen("nonexistent"))

	allItems, err = kv.ArrayAll("nonexistent")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(allItems))

	slice, err = kv.ArraySlice("nonexistent", 0, 10)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(slice))

	page, err = kv.ArrayPage("nonexistent", 1, 10)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(page))

	// Test error cases
	_, err = kv.ArrayGet("nonexistent", 0)
	assert.NotNil(t, err)

	_, err = kv.Pop("nonexistent", 1)
	assert.NotNil(t, err)
}

// testConcurrency tests concurrent operations on the store
func testConcurrency(t *testing.T, kv Store) {
	kv.Clear()

	const numGoroutines = 100
	const numOperations = 50

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	// Test concurrent basic operations
	t.Run("ConcurrentBasicOps", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("key_%d_%d", id, j)
					value := fmt.Sprintf("value_%d_%d", id, j)

					// Set
					if err := kv.Set(key, value, 0); err != nil {
						errors <- fmt.Errorf("set error: %v", err)
						return
					}

					// Get
					if val, ok := kv.Get(key); !ok || val != value {
						errors <- fmt.Errorf("get mismatch: expected %s, got %v", value, val)
						return
					}

					// Has
					if !kv.Has(key) {
						errors <- fmt.Errorf("has failed for key %s", key)
						return
					}

					// Del
					if err := kv.Del(key); err != nil {
						errors <- fmt.Errorf("del error: %v", err)
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Error(err)
		}
	})

	// Test concurrent list operations
	t.Run("ConcurrentListOps", func(t *testing.T) {
		kv.Clear()
		listKey := "concurrent_list"

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*10)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Push items
				for j := 0; j < 10; j++ {
					item := fmt.Sprintf("item_%d_%d", id, j)
					if err := kv.Push(listKey, item); err != nil {
						errors <- fmt.Errorf("push error: %v", err)
						return
					}
				}

				// Read operations
				if length := kv.ArrayLen(listKey); length < 0 {
					errors <- fmt.Errorf("unexpected length: %d", length)
					return
				}

				if _, err := kv.ArrayAll(listKey); err != nil {
					errors <- fmt.Errorf("array all error: %v", err)
					return
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Error(err)
		}

		// Verify final state
		length := kv.ArrayLen(listKey)
		assert.Equal(t, numGoroutines*10, length)
	})
}

// testMemoryLeak tests for memory leaks
func testMemoryLeak(t *testing.T, kv Store) {
	runtime.GC()
	runtime.GC() // Double GC to ensure cleanup

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform intensive operations
	const iterations = 1000
	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("leak_test_%d", i)
		value := make([]byte, 1024) // 1KB per value

		// Basic operations
		kv.Set(key, value, 0)
		kv.Get(key)
		kv.Has(key)

		// List operations
		listKey := fmt.Sprintf("list_%d", i)
		kv.Push(listKey, value, value, value)
		kv.ArrayAll(listKey)
		kv.ArrayLen(listKey)
		kv.Pop(listKey, 1)
		kv.Pull(listKey, value)

		// Cleanup
		kv.Del(key)
		kv.Del(listKey)
	}

	// Clear everything
	kv.Clear()

	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Allow GC to complete
	runtime.ReadMemStats(&m2)

	// Check memory growth
	memGrowth := int64(m2.Alloc) - int64(m1.Alloc)
	maxAllowedGrowth := int64(10 * 1024 * 1024) // 10MB threshold

	if memGrowth > maxAllowedGrowth {
		t.Errorf("Potential memory leak detected: memory grew by %d bytes (threshold: %d bytes)",
			memGrowth, maxAllowedGrowth)
	}

	t.Logf("Memory stats - Before: %d bytes, After: %d bytes, Growth: %d bytes",
		m1.Alloc, m2.Alloc, memGrowth)
}

// testGoroutineLeak tests for goroutine leaks
func testGoroutineLeak(t *testing.T, kv Store) {
	initialGoroutines := runtime.NumGoroutine()

	// Perform operations that might create goroutines
	const numOperations = 100
	var wg sync.WaitGroup

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			key := fmt.Sprintf("goroutine_test_%d", id)
			kv.Set(key, "value", time.Second)
			kv.Get(key)

			// List operations
			listKey := fmt.Sprintf("list_goroutine_%d", id)
			kv.Push(listKey, "item1", "item2")
			kv.ArrayAll(listKey)
			kv.Pop(listKey, 1)

			kv.Del(key)
			kv.Del(listKey)
		}(i)
	}

	wg.Wait()

	// Allow some time for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	goroutineGrowth := finalGoroutines - initialGoroutines

	// Allow some tolerance for background goroutines
	maxAllowedGrowth := 5

	if goroutineGrowth > maxAllowedGrowth {
		t.Errorf("Potential goroutine leak detected: %d new goroutines (threshold: %d)",
			goroutineGrowth, maxAllowedGrowth)
	}

	t.Logf("Goroutine stats - Initial: %d, Final: %d, Growth: %d",
		initialGoroutines, finalGoroutines, goroutineGrowth)
}

// benchmarkConcurrentRead benchmarks concurrent read operations
func benchmarkConcurrentRead(b *testing.B, kv Store) {
	kv.Clear()

	// Prepare data
	const numKeys = 1000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		value := fmt.Sprintf("bench_value_%d", i)
		kv.Set(key, value, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench_key_%d", i%numKeys)
			kv.Get(key)
			i++
		}
	})
}

// benchmarkConcurrentWrite benchmarks concurrent write operations
func benchmarkConcurrentWrite(b *testing.B, kv Store) {
	kv.Clear()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench_write_key_%d", i)
			value := fmt.Sprintf("bench_write_value_%d", i)
			kv.Set(key, value, 0)
			i++
		}
	})
}

// benchmarkConcurrentMixed benchmarks mixed read/write operations
func benchmarkConcurrentMixed(b *testing.B, kv Store) {
	kv.Clear()

	// Prepare some initial data
	const numInitialKeys = 100
	for i := 0; i < numInitialKeys; i++ {
		key := fmt.Sprintf("initial_key_%d", i)
		value := fmt.Sprintf("initial_value_%d", i)
		kv.Set(key, value, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%3 == 0 {
				// Write operation
				key := fmt.Sprintf("mixed_key_%d", i)
				value := fmt.Sprintf("mixed_value_%d", i)
				kv.Set(key, value, 0)
			} else {
				// Read operation
				key := fmt.Sprintf("initial_key_%d", i%numInitialKeys)
				kv.Get(key)
			}
			i++
		}
	})
}

// Benchmark tests for performance
func BenchmarkLRUConcurrentRead(b *testing.B) {
	store := newStore(&testing.T{}, nil)
	benchmarkConcurrentRead(b, store)
}

func BenchmarkLRUConcurrentWrite(b *testing.B) {
	store := newStore(&testing.T{}, nil)
	benchmarkConcurrentWrite(b, store)
}

func BenchmarkLRUConcurrentMixed(b *testing.B) {
	store := newStore(&testing.T{}, nil)
	benchmarkConcurrentMixed(b, store)
}

func BenchmarkRedisConcurrentRead(b *testing.B) {
	connectors := map[string]string{
		"redis": filepath.Join("connectors", "redis.conn.yao"),
	}
	for id, file := range connectors {
		_, err := connector.Load(file, id)
		if err != nil {
			b.Skip("Redis connector not available:", err)
			return
		}
	}
	store := newStore(&testing.T{}, getConnector(&testing.T{}, "redis"))
	benchmarkConcurrentRead(b, store)
}

func BenchmarkRedisConcurrentWrite(b *testing.B) {
	connectors := map[string]string{
		"redis": filepath.Join("connectors", "redis.conn.yao"),
	}
	for id, file := range connectors {
		_, err := connector.Load(file, id)
		if err != nil {
			b.Skip("Redis connector not available:", err)
			return
		}
	}
	store := newStore(&testing.T{}, getConnector(&testing.T{}, "redis"))
	benchmarkConcurrentWrite(b, store)
}

func BenchmarkMongoConcurrentRead(b *testing.B) {
	connectors := map[string]string{
		"mongo": filepath.Join("connectors", "mongo.conn.yao"),
	}
	for id, file := range connectors {
		_, err := connector.Load(file, id)
		if err != nil {
			b.Skip("Mongo connector not available:", err)
			return
		}
	}
	store := newStore(&testing.T{}, getConnector(&testing.T{}, "mongo"))
	benchmarkConcurrentRead(b, store)
}

func BenchmarkMongoConcurrentWrite(b *testing.B) {
	connectors := map[string]string{
		"mongo": filepath.Join("connectors", "mongo.conn.yao"),
	}
	for id, file := range connectors {
		_, err := connector.Load(file, id)
		if err != nil {
			b.Skip("Mongo connector not available:", err)
			return
		}
	}
	store := newStore(&testing.T{}, getConnector(&testing.T{}, "mongo"))
	benchmarkConcurrentWrite(b, store)
}
