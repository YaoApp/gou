# Store Package - List Operations

The Store package provides a unified interface for key-value storage with support for MongoDB-style list operations across multiple backend implementations.

## Features

- **Unified Interface**: Consistent API across LRU cache, Redis, MongoDB, and Badger implementations
- **Complete API**: 25+ methods covering key-value and list operations
- **MongoDB-style API**: Familiar operations for developers using MongoDB
- **List Operations**: Full support for array/list data structures
- **Pagination Support**: Built-in pagination with `ArrayPage` and `ArraySlice`
- **Thread Safety**: Full concurrency support with comprehensive stress testing
- **Type Safety**: Strongly typed interface with Go generics support
- **Embedded Storage**: Badger support for applications requiring local persistence

## Supported Backends

- **LRU Cache**: In-memory cache with LRU eviction
- **Redis**: Distributed cache using Redis lists
- **MongoDB**: Document-based storage using MongoDB arrays
- **Badger**: Embedded key-value database with persistent storage

## Complete API Reference

### Key-Value Operations

#### Get

Get a value by key.

```go
value, ok := store.Get("mykey")
if ok {
    fmt.Printf("Value: %v\n", value)
}
```

#### Set

Set a value with optional TTL.

```go
// Set without expiration
err := store.Set("mykey", "myvalue", 0)

// Set with 1 hour expiration
err := store.Set("mykey", "myvalue", time.Hour)
```

#### Del

Delete a key.

```go
err := store.Del("mykey")
```

#### Has

Check if a key exists.

```go
exists := store.Has("mykey")
```

#### Len

Get the total number of keys in the store.

```go
count := store.Len()
```

#### Keys

Get all keys in the store.

```go
allKeys := store.Keys()
```

#### Clear

Remove all keys from the store.

```go
store.Clear()
```

#### GetSet

Get a value, or set it if it doesn't exist (atomic operation).

```go
value, err := store.GetSet("mykey", time.Hour, func(key string) (interface{}, error) {
    return "default_value", nil
})
```

#### GetDel

Get a value and delete it atomically.

```go
value, ok := store.GetDel("mykey")
```

### Batch Operations

#### GetMulti

Get multiple values at once.

```go
keys := []string{"key1", "key2", "key3"}
values := store.GetMulti(keys)
for key, value := range values {
    fmt.Printf("%s: %v\n", key, value)
}
```

#### SetMulti

Set multiple key-value pairs at once.

```go
values := map[string]interface{}{
    "key1": "value1",
    "key2": 42,
    "key3": true,
}
store.SetMulti(values, time.Hour)
```

#### DelMulti

Delete multiple keys at once.

```go
keys := []string{"key1", "key2", "key3"}
store.DelMulti(keys)
```

#### GetSetMulti

Get multiple values, setting defaults for missing ones.

```go
keys := []string{"key1", "key2", "key3"}
values := store.GetSetMulti(keys, time.Hour, func(key string) (interface{}, error) {
    return fmt.Sprintf("default_%s", key), nil
})
```

## List Operations API

### Basic List Operations

#### Push

Add elements to the end of a list (similar to MongoDB `$push`).

```go
err := store.Push("mylist", "item1", "item2", "item3")
```

#### Pop

Remove and return an element from a list.

```go
// Pop from end (position = 1)
value, err := store.Pop("mylist", 1)

// Pop from beginning (position = -1)
value, err := store.Pop("mylist", -1)
```

**Parameters:**

- `position = 1`: Remove from the end of the list (like RPOP in Redis)
- `position = -1`: Remove from the beginning of the list (like LPOP in Redis)

**Returns:** The removed element and an error if the list is empty or doesn't exist.

#### ArrayLen

Get the length of a list.

```go
length := store.ArrayLen("mylist")
```

### Element Access

#### ArrayGet

Get an element at a specific index.

```go
value, err := store.ArrayGet("mylist", 0) // Get first element
value, err := store.ArrayGet("mylist", -1) // Get last element (if supported)
```

**Parameters:**

- `index`: Zero-based index. Most implementations support positive indices only.

**Returns:** The element at the specified index, or an error if index is out of range.

#### ArraySet

Set an element at a specific index.

```go
err := store.ArraySet("mylist", 1, "new_value")
```

**Parameters:**

- `index`: Zero-based index. Must be within the current list bounds.
- `value`: New value to set at the specified index.

**Note:** This operation modifies an existing element and cannot extend the list.

#### ArrayAll

Get all elements in a list.

```go
allItems, err := store.ArrayAll("mylist")
```

### Pagination

#### ArrayPage

Get a specific page of elements (page starts from 1).

```go
// Get page 1 with 10 items per page
page1, err := store.ArrayPage("mylist", 1, 10)

// Get page 2 with 10 items per page
page2, err := store.ArrayPage("mylist", 2, 10)
```

**Parameters:**

- `page`: Page number starting from 1
- `pageSize`: Number of items per page

**Returns:** Elements for the specified page. Returns empty slice if page is beyond list bounds.

#### ArraySlice

Get a slice of elements with skip and limit.

```go
// Skip 5 elements, return next 10
slice, err := store.ArraySlice("mylist", 5, 10)
```

**Parameters:**

- `skip`: Number of elements to skip from the beginning
- `limit`: Maximum number of elements to return

**Returns:** A slice of elements. Returns empty slice if skip is beyond list bounds.

### Element Removal

#### Pull

Remove all occurrences of a specific value (similar to MongoDB `$pull`).

```go
err := store.Pull("mylist", "unwanted_value")
```

**Parameters:**

- `value`: The value to remove from the list

**Behavior:** Removes ALL occurrences of the specified value. This is an atomic operation.

#### PullAll

Remove all occurrences of multiple values (similar to MongoDB `$pullAll`).

```go
err := store.PullAll("mylist", []interface{}{"value1", "value2", "value3"})
```

**Parameters:**

- `values`: Slice of values to remove from the list

**Behavior:** Removes ALL occurrences of ANY of the specified values. More efficient than multiple `Pull` calls.

### Set Operations

#### AddToSet

Add elements only if they don't already exist (similar to MongoDB `$addToSet`).

```go
err := store.AddToSet("mylist", "unique1", "unique2", "existing_value")
```

**Parameters:**

- `values`: Variable number of values to add

**Behavior:**

- Only adds values that don't already exist in the list
- Maintains list uniqueness
- Multiple values can be added in a single atomic operation
- Duplicate values in the input are also deduplicated

## Usage Examples

### Basic List Operations

```go
package main

import (
    "fmt"
    "github.com/yaoapp/gou/store"
)

func main() {
    // Create a store instance (LRU example)
    s, err := store.New(nil, store.Option{"size": 1000})
    if err != nil {
        panic(err)
    }

    // Add items to a list
    err = s.Push("fruits", "apple", "banana", "cherry")
    if err != nil {
        panic(err)
    }

    // Get list length
    fmt.Printf("List length: %d\n", s.ArrayLen("fruits")) // Output: 3

    // Get all items
    items, err := s.ArrayAll("fruits")
    if err != nil {
        panic(err)
    }
    fmt.Printf("All items: %v\n", items) // Output: [apple banana cherry]

    // Get specific item
    item, err := s.ArrayGet("fruits", 1)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Second item: %v\n", item) // Output: banana
}
```

### Pagination Example

```go
// Add many items
for i := 0; i < 100; i++ {
    s.Push("numbers", i)
}

// Get first page (10 items per page)
page1, err := s.ArrayPage("numbers", 1, 10)
if err != nil {
    panic(err)
}
fmt.Printf("Page 1: %v\n", page1) // Output: [0 1 2 3 4 5 6 7 8 9]

// Get second page
page2, err := s.ArrayPage("numbers", 2, 10)
if err != nil {
    panic(err)
}
fmt.Printf("Page 2: %v\n", page2) // Output: [10 11 12 13 14 15 16 17 18 19]
```

### Set Operations Example

```go
// Add unique items only
err = s.AddToSet("tags", "go", "database", "cache", "go") // "go" won't be added twice
if err != nil {
    panic(err)
}

tags, err := s.ArrayAll("tags")
if err != nil {
    panic(err)
}
fmt.Printf("Unique tags: %v\n", tags) // Output: [go database cache]
```

## Implementation Details

### LRU Cache Implementation

- Stores lists as `[]interface{}` in memory
- Fast access but limited by memory
- No persistence across restarts
- **Thread-safe operations** with read-write mutex protection
- **Copy-on-write** semantics to prevent data races
- Returns data copies to prevent external modification

### Redis Implementation

- Uses Redis LIST commands (RPUSH, LPOP, LRANGE, etc.)
- Distributed and persistent
- Atomic operations
- JSON serialization for complex data types
- Uses Lua scripts for complex operations like `AddToSet`

### MongoDB Implementation

- Uses MongoDB array operators ($push, $pull, $addToSet, etc.)
- Document-based storage with atomic updates
- Aggregation pipeline for complex queries
- Native support for array operations
- Persistent and scalable

### Badger Implementation

- Embedded key-value database with no external dependencies
- JSON serialization for List operations
- File-based persistent storage
- **Thread-safe operations** with read-write mutex protection
- Automatic directory creation for database path
- High-performance LSM-tree based storage engine
- Configurable database path (relative to application root)

## Configuration

### LRU Cache

```go
store, err := store.New(nil, store.Option{"size": 10000})
```

### Redis

```go
// Redis connector must be configured first
store, err := store.New(redisConnector, store.Option{})
```

### MongoDB

```go
// MongoDB connector must be configured first
store, err := store.New(mongoConnector, store.Option{})
```

### Badger

```go
// Default path (relative to application root)
store, err := store.New(nil, store.Option{"driver": "badger", "path": "badger/db"})

// Absolute path
store, err := store.New(nil, store.Option{"driver": "badger", "path": "/var/lib/myapp/badger"})

// Current directory relative path
store, err := store.New(nil, store.Option{"driver": "badger", "path": "./data/badger"})
```

## Testing

Run the comprehensive test suite:

```bash
# Set up test environment
source env.local.sh

# Run all tests
go test ./store -v

# Run specific test
go test ./store -v -run TestLRU

# Run concurrency tests
go test ./store -v -run TestLRUConcurrency
go test ./store -v -run TestRedisConcurrency
go test ./store -v -run TestMongoConcurrency
go test ./store -v -run TestBadgerConcurrency

# Run benchmarks
go test ./store -bench=BenchmarkLRU -v
go test ./store -bench=BenchmarkRedis -v
go test ./store -bench=BenchmarkMongo -v
go test ./store -bench=BenchmarkBadger -v
```

The test suite covers:

- All list operations across all backends
- Edge cases and error conditions
- Pagination functionality
- Set operations and uniqueness
- **Concurrency stress testing** (100 goroutines)
- **Memory leak detection** with 10MB threshold
- **Goroutine leak detection**
- Performance characteristics and benchmarks

## Concurrency and Thread Safety

All store implementations are designed to be **thread-safe** and support concurrent access:

### LRU Cache

- Uses `sync.RWMutex` for reader-writer lock protection
- Read operations (Get, ArrayGet, ArrayLen, etc.) use read locks for better concurrency
- Write operations (Set, Push, Pop, etc.) use exclusive write locks
- Copy-on-write semantics prevent data races during list modifications
- Returns defensive copies to prevent external modification

### Redis

- Inherently thread-safe due to Redis's single-threaded nature
- All operations are atomic at the Redis server level
- Network I/O is handled safely by the Redis client library
- Lua scripts ensure atomicity for complex operations like `AddToSet`

### MongoDB

- Thread-safe through MongoDB driver's connection pooling
- All array operations use MongoDB's atomic update operators
- Document-level locking ensures consistency
- Aggregation pipelines are atomic and isolated

### Badger

- Uses `sync.RWMutex` for reader-writer lock protection
- Read operations use read locks for better concurrency
- Write operations use exclusive write locks
- JSON serialization ensures data consistency
- Embedded database eliminates network-related concurrency issues
- Persistent storage with crash recovery

### Stress Testing

The store package includes comprehensive concurrency tests:

- **100 concurrent goroutines** performing mixed read/write operations
- **Memory leak detection** with sub-10MB growth threshold
- **Goroutine leak detection** ensuring proper cleanup
- **Race condition detection** using Go's race detector

Example concurrent usage:

```go
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()

        // Safe concurrent operations
        store.Push("shared_list", fmt.Sprintf("item_%d", id))
        value, _ := store.Pop("shared_list", 1)
        store.ArrayLen("shared_list")
    }(i)
}
wg.Wait()
```

## Error Handling

All list operations return appropriate errors for:

- Non-existent keys
- Index out of range
- Type mismatches
- Backend connection issues

Example error handling:

```go
value, err := store.ArrayGet("mylist", 10)
if err != nil {
    if strings.Contains(err.Error(), "index out of range") {
        fmt.Println("Index is too large")
    } else {
        fmt.Printf("Other error: %v\n", err)
    }
}
```

## Performance Considerations

### LRU Cache

- **Best for**: Frequent access, small datasets
- **O(1)** for most operations
- Memory limited

### Redis

- **Best for**: Distributed applications, medium to large datasets
- **O(1)** for Push/Pop operations
- **O(N)** for Pull operations
- Network latency considerations

### MongoDB

- **Best for**: Complex queries, very large datasets
- **O(1)** for indexed operations
- **O(N)** for array operations
- Supports complex aggregation queries

### Badger

- **Best for**: Embedded applications, single-node deployments
- **O(log N)** for most operations (LSM-tree based)
- **O(N)** for list operations (JSON serialization)
- No network latency, embedded storage
- Persistent across application restarts
- Suitable for applications requiring local data persistence

## Migration Guide

If you're upgrading from the basic Store interface, the new list operations are additive and won't break existing code. Simply start using the new methods where needed.

## License

This package is part of the GOU framework and follows the same licensing terms.
