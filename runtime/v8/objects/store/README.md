# Store JavaScript API Documentation

The Store JavaScript API provides a unified interface for key-value storage with MongoDB-style list operations in Yao applications. This API supports LRU cache, Redis, MongoDB, and Xun (database-backed) backends through a consistent JavaScript interface.

## Table of Contents

- [Installation & Setup](#installation--setup)
- [TypeScript Integration](#typescript-integration)
- [API Reference](#api-reference)
  - [Key-Value Operations](#key-value-operations)
  - [List Operations](#list-operations)
- [Usage Examples](#usage-examples)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)

## Installation & Setup

The Store API is automatically available in Yao JavaScript runtime. No additional installation is required.

```javascript
// Create a store instance
const store = new Store("cache_name");
```

## TypeScript Integration

For TypeScript development, you can define the Store interface:

```typescript
interface YaoStore {
  // Key-Value Operations
  Set(key: string, value: any, ttl?: number): void;
  Get(key: string): any;
  GetSet(key: string, getValue: (key: string) => any, ttl?: number): any;
  GetDel(key: string): any;
  Has(key: string): boolean;
  Del(key: string): void;
  Keys(): string[];
  Len(): number;
  Clear(): void;

  // Batch Operations
  SetMulti(values: { [key: string]: any }, ttl?: number): void;
  GetMulti(keys: string[]): { [key: string]: any };
  DelMulti(keys: string[]): void;
  GetSetMulti(
    keys: string[],
    getValue: (key: string) => any,
    ttl?: number
  ): { [key: string]: any };

  // List Operations
  Push(key: string, ...values: any[]): void;
  Pop(key: string, position: number): any;
  Pull(key: string, value: any): void;
  PullAll(key: string, ...values: any[]): void;
  AddToSet(key: string, ...values: any[]): void;
  ArrayLen(key: string): number;
  ArrayGet(key: string, index: number): any;
  ArraySet(key: string, index: number, value: any): void;
  ArraySlice(key: string, skip: number, limit: number): any[];
  ArrayPage(key: string, page: number, pageSize: number): any[];
  ArrayAll(key: string): any[];
}

declare const Store: {
  new (name: string): YaoStore;
};

// Usage in TypeScript
const store: YaoStore = new Store("my_cache");
```

## API Reference

### Key-Value Operations

#### Set(key, value, ttl?)

Set a key-value pair with optional TTL (Time To Live) in seconds.

```typescript
store.Set("user:123", { name: "John", age: 30 });
store.Set("session:abc", "active", 3600); // Expires in 1 hour
```

#### Get(key)

Get a value by key. Returns `undefined` if key doesn't exist.

```typescript
const user = store.Get("user:123");
if (user) {
  console.log(user.name); // "John"
}
```

#### GetSet(key, getValue, ttl?)

Get a value or set it if it doesn't exist (cache-aside pattern).

```typescript
const userData = store.GetSet(
  "user:123",
  (key: string) => {
    // This function is called only if the key doesn't exist
    return fetchUserFromDatabase(key);
  },
  3600
);
```

#### GetDel(key)

Get a value and delete it atomically.

```typescript
const token = store.GetDel("temp:token:abc");
```

#### Has(key)

Check if a key exists.

```typescript
if (store.Has("user:123")) {
  console.log("User exists in cache");
}
```

#### Del(key)

Delete a key.

```typescript
store.Del("user:123");
```

#### Keys()

Get all keys in the store.

```typescript
const allKeys: string[] = store.Keys();
console.log("Total keys:", allKeys.length);
```

#### Len()

Get the total number of keys.

```typescript
const count: number = store.Len();
```

#### Clear()

Remove all keys from the store.

```typescript
store.Clear();
```

### Batch Operations

#### SetMulti(values, ttl?)

Set multiple key-value pairs at once.

```typescript
const values = {
  "user:123": { name: "John", age: 30 },
  "user:456": { name: "Jane", age: 25 },
  "config:theme": "dark",
};
store.SetMulti(values, 3600); // Set all with 1 hour TTL
```

#### GetMulti(keys)

Get multiple values at once.

```typescript
const keys = ["user:123", "user:456", "config:theme"];
const values = store.GetMulti(keys);
console.log(values); // { "user:123": {...}, "user:456": {...}, "config:theme": "dark" }
```

#### DelMulti(keys)

Delete multiple keys at once.

```typescript
const keys = ["user:123", "user:456", "temp:data"];
store.DelMulti(keys);
```

#### GetSetMulti(keys, getValue, ttl?)

Get multiple values or set them if they don't exist.

```typescript
const keys = ["user:123", "user:456"];
const values = store.GetSetMulti(
  keys,
  (key: string) => {
    return fetchUserFromDatabase(key);
  },
  3600
);
```

### List Operations

#### Push(key, ...values)

Add elements to the end of a list.

```typescript
store.Push("fruits", "apple", "banana", "cherry");
store.Push("numbers", 1, 2, 3, 4, 5);
```

#### Pop(key, position)

Remove and return an element from a list.

```typescript
// Pop from end (position = 1)
const lastItem = store.Pop("fruits", 1);

// Pop from beginning (position = -1)
const firstItem = store.Pop("fruits", -1);
```

#### Pull(key, value)

Remove all occurrences of a specific value.

```typescript
store.Pull("fruits", "apple"); // Removes all "apple" entries
```

#### PullAll(key, ...values)

Remove all occurrences of multiple values.

```typescript
store.PullAll("fruits", "apple", "banana", "cherry");
```

#### AddToSet(key, ...values)

Add elements only if they don't already exist (ensures uniqueness).

```typescript
store.AddToSet("tags", "javascript", "typescript", "javascript");
// Only unique values are added
```

#### ArrayLen(key)

Get the length of a list.

```typescript
const count: number = store.ArrayLen("fruits");
```

#### ArrayGet(key, index)

Get an element at a specific index.

```typescript
const firstFruit = store.ArrayGet("fruits", 0);
const lastFruit = store.ArrayGet("fruits", -1); // If supported
```

#### ArraySet(key, index, value)

Set an element at a specific index.

```typescript
store.ArraySet("fruits", 1, "orange"); // Replace element at index 1
```

#### ArrayAll(key)

Get all elements in a list.

```typescript
const allFruits: any[] = store.ArrayAll("fruits");
```

#### ArraySlice(key, skip, limit)

Get a slice of elements with skip and limit.

```typescript
// Skip 5 elements, return next 10
const slice: any[] = store.ArraySlice("items", 5, 10);
```

#### ArrayPage(key, page, pageSize)

Get a specific page of elements (page starts from 1).

```typescript
// Get page 1 with 10 items per page
const page1: any[] = store.ArrayPage("items", 1, 10);

// Get page 2 with 10 items per page
const page2: any[] = store.ArrayPage("items", 2, 10);
```

## Usage Examples

### Basic Key-Value Usage

```typescript
// User session management
function manageUserSession(userId: string, sessionData: any) {
  const store = new Store("sessions");

  // Set session with 1 hour expiry
  store.Set(`user:${userId}`, sessionData, 3600);

  // Check if user is logged in
  if (store.Has(`user:${userId}`)) {
    console.log("User is active");
  }

  // Get session data
  const session = store.Get(`user:${userId}`);
  return session;
}
```

### Shopping Cart Implementation

```typescript
function ShoppingCart(userId: string) {
  const store = new Store("shopping_carts");
  const key = `cart:${userId}`;

  return {
    addItem(productId: string, quantity: number = 1) {
      for (let i = 0; i < quantity; i++) {
        store.Push(key, productId);
      }
    },

    removeItem(productId: string) {
      store.Pull(key, productId);
    },

    getItems(): string[] {
      return store.ArrayAll(key) || [];
    },

    getItemCount(): number {
      return store.ArrayLen(key);
    },

    clear() {
      store.Del(key);
    },

    getPage(page: number, pageSize: number = 10): string[] {
      return store.ArrayPage(key, page, pageSize);
    },
  };
}

// Usage
const cart = ShoppingCart("user123");
cart.addItem("product456", 2);
cart.addItem("product789");
console.log("Items:", cart.getItems());
console.log("Total items:", cart.getItemCount());
```

### Tag Management System

```typescript
function TagManager(entityType: string) {
  const store = new Store("tags");

  return {
    addTags(entityId: string, ...tags: string[]) {
      const key = `${entityType}:${entityId}:tags`;
      store.AddToSet(key, ...tags); // Ensures uniqueness
    },

    removeTags(entityId: string, ...tags: string[]) {
      const key = `${entityType}:${entityId}:tags`;
      store.PullAll(key, ...tags);
    },

    getTags(entityId: string): string[] {
      const key = `${entityType}:${entityId}:tags`;
      return store.ArrayAll(key) || [];
    },

    hasTag(entityId: string, tag: string): boolean {
      const tags = this.getTags(entityId);
      return tags.includes(tag);
    },
  };
}

// Usage
const articleTags = TagManager("article");
articleTags.addTags("123", "javascript", "typescript", "programming");
articleTags.addTags("123", "javascript"); // Won't create duplicate
console.log("Tags:", articleTags.getTags("123"));
```

### Activity Feed with Pagination

```typescript
function ActivityFeed(userId: string) {
  const store = new Store("activity_feeds");
  const key = `feed:${userId}`;

  return {
    addActivity(activity: any) {
      // Add timestamp
      const timestampedActivity = {
        ...activity,
        timestamp: Date.now(),
      };
      store.Push(key, timestampedActivity);

      // Keep only latest 1000 activities
      const total = store.ArrayLen(key);
      if (total > 1000) {
        // Remove oldest activities
        const excess = total - 1000;
        for (let i = 0; i < excess; i++) {
          store.Pop(key, -1); // Pop from beginning
        }
      }
    },

    getActivities(page: number = 1, pageSize: number = 20): any[] {
      return store.ArrayPage(key, page, pageSize);
    },

    getLatest(count: number = 10): any[] {
      return store.ArraySlice(key, 0, count);
    },

    getTotalCount(): number {
      return store.ArrayLen(key);
    },
  };
}

// Usage
const feed = ActivityFeed("user123");
feed.addActivity({ type: "login", message: "User logged in" });
feed.addActivity({ type: "post", message: "Created new post" });

const recentActivities = feed.getLatest(5);
const page1 = feed.getActivities(1, 10);
```

### Caching with GetSet Pattern

```typescript
interface User {
  id: string;
  name: string;
  email: string;
}

function UserService() {
  const store = new Store("users");

  return {
    getUser(userId: string): User | null {
      return store.GetSet(
        `user:${userId}`,
        (key: string) => {
          // This function is called only if cache miss
          console.log(`Fetching user ${userId} from database...`);
          return fetchUserFromDatabase(userId);
        },
        3600 // Cache for 1 hour
      );
    },

    updateUser(user: User) {
      // Update database first
      updateUserInDatabase(user);

      // Update cache
      store.Set(`user:${user.id}`, user, 3600);
    },

    deleteUser(userId: string) {
      // Delete from database
      deleteUserFromDatabase(userId);

      // Remove from cache
      store.Del(`user:${userId}`);
    },
  };
}
```

## Error Handling

```typescript
function safeStoreOperation() {
  const store = new Store("my_cache");

  try {
    // Operations that might fail
    const value = store.ArrayGet("list", 100); // Index might be out of range
    const popped = store.Pop("empty_list", 1); // List might be empty

    return value;
  } catch (error) {
    console.error("Store operation failed:", error);
    return null;
  }
}
```

## Best Practices

### 1. Use Meaningful Key Names

```typescript
// Good
store.Set("user:session:123", sessionData);
store.Push("notifications:user:456", notification);

// Avoid
store.Set("u123", sessionData);
store.Push("n", notification);
```

### 2. Set Appropriate TTL

```typescript
// Short-lived data
store.Set("temp:token", token, 300); // 5 minutes

// Session data
store.Set("session:user", session, 3600); // 1 hour

// Long-term cache
store.Set("user:profile", profile, 86400); // 24 hours
```

### 3. Handle Cache Misses Gracefully

```typescript
function getDataWithFallback(key: string) {
  const cached = store.Get(key);
  if (cached !== undefined) {
    return cached;
  }

  // Fallback to database or API
  const fresh = fetchFromSource(key);
  store.Set(key, fresh, 3600);
  return fresh;
}
```

### 4. Use Lists for Ordered Data

```typescript
// Recent searches (ordered by time)
store.Push("recent_searches:user123", searchQuery);

// Keep only last 10 searches
if (store.ArrayLen("recent_searches:user123") > 10) {
  store.Pop("recent_searches:user123", -1); // Remove oldest
}
```

### 5. Use AddToSet for Unique Collections

```typescript
// User interests (no duplicates)
store.AddToSet("interests:user123", "programming", "music", "travel");

// Tags (unique)
store.AddToSet("article:tags:456", "javascript", "tutorial", "beginner");
```

### 6. Implement Pagination for Large Lists

```typescript
function getPaginatedResults(key: string, page: number, pageSize: number = 20) {
  const total = store.ArrayLen(key);
  const totalPages = Math.ceil(total / pageSize);
  const items = store.ArrayPage(key, page, pageSize);

  return {
    items,
    pagination: {
      page,
      pageSize,
      total,
      totalPages,
      hasNext: page < totalPages,
      hasPrev: page > 1,
    },
  };
}
```

## Performance Considerations

- **LRU Cache**: Fastest for frequent access, limited by memory
- **Redis**: Good for distributed applications, network latency considerations
- **MongoDB**: Best for complex queries and very large datasets
- **Xun**: Database-backed storage with LRU cache layer, leverages existing database infrastructure

Choose the appropriate backend based on your application's requirements and configure the store accordingly in your Yao application.

### Xun Store Features

The Xun store provides database-backed persistence with performance optimization:

- **LRU Cache Layer**: Fast reads from in-memory cache
- **Async Persistence**: Batch writes to reduce database load (configurable interval)
- **Lazy Loading**: Data loaded from database on first access
- **TTL Support**: Automatic expiration with background cleanup
- **Multi-Database**: Supports MySQL, PostgreSQL, SQLite via Xun connectors

**Note**: Up to `persist_interval` seconds of data may be lost on application crash.
