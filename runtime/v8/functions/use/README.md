# Use Function

Universal resource management function that automatically calls `__release()` on objects after use, ensuring immediate resource cleanup without waiting for V8 garbage collection.

## When to Use

### Use `Use()` for:

- **Automatic cleanup**: Resources are released immediately after the callback completes
- **Simplified code**: Less boilerplate than `try-finally`
- **Guaranteed cleanup**: Works even when exceptions are thrown

### Use `try-finally` with manual `Release()` for:

- **Critical memory situations**: When you need explicit control over when resources are freed
- **Performance-sensitive code**: Avoid callback overhead
- **Complex control flow**: When the resource lifetime doesn't fit a simple callback pattern

## Features

- ✅ Immediate resource cleanup (doesn't wait for GC)
- ✅ Works with any constructor
- ✅ Supports multiple arguments
- ✅ Error-safe (always releases resources)
- ✅ Nested usage support

## Usage

### Basic Usage

```javascript
Use(Constructor, ...args, (instance) => {
  // Use the instance
  return result;
});
// instance.__release() is called automatically
```

### With MCP Client

```javascript
Use(MCP, "client_id", (client) => {
  const tools = client.ListTools();
  const result = client.CallTool("tool_name", { arg: "value" });
  return result;
});
// client.Release() is called automatically
```

### Nested Resources

```javascript
Use(MCP, "dsl", (dslClient) => {
  return Use(MCP, "customer", (customerClient) => {
    const tools = dslClient.ListTools();
    const resources = customerClient.ListResources();
    return { tools, resources };
  });
});
// Both clients are released in reverse order (LIFO)
```

### Multiple Constructor Arguments

```javascript
Use(SomeClass, arg1, arg2, arg3, (instance) => {
  return instance.doSomething();
});
```

### Error Handling

Errors thrown inside `Use()` callbacks properly propagate to the caller, and resources are still cleaned up:

```javascript
try {
  Use(MCP, "client_id", (client) => {
    throw new Error("Something went wrong");
  });
} catch (error) {
  // Error is caught
  // client.__release() was still called before the error propagated
}
```

The `Use()` function ensures that `__release()` is always called, even when errors occur, by calling it immediately after the callback completes and before re-throwing any exception.

## How It Works

1. `Use()` takes a constructor and arguments
2. The last argument must be a callback function
3. The constructor is called with `new` and the provided arguments
4. The callback is executed with the created instance
5. **After the callback completes (or throws), `__release()` is automatically called on the instance**
6. Errors in `__release()` are silently ignored

### Release Methods

Objects must provide a `__release()` method for `Use()` to work:

- **`__release()`** - Internal cleanup method called by:

  - `Use()` function (immediate, automatic)
  - V8 garbage collector (delayed, when object is collected)

- **`Release()`** - Public cleanup method for manual use:
  - Called explicitly in `try-finally` blocks
  - Provides immediate cleanup when needed
  - Offers explicit control over resource lifetime

**Important**: `Use()` **only** calls `__release()`, not `Release()`. This separation ensures:

- Clear distinction between automatic and manual cleanup
- `Release()` remains available for explicit control
- No confusion about which method to use

### Memory Management

**Important**: `Use()` provides **immediate** resource cleanup:

```javascript
// ❌ BAD: Memory accumulates until GC
for (let i = 0; i < 10000; i++) {
  const client = new MCP("dsl");
  client.ListTools();
  // Waits for V8 GC to cleanup - may run out of memory!
}

// ✅ GOOD: Immediate cleanup with Use()
for (let i = 0; i < 10000; i++) {
  Use(MCP, "dsl", (client) => {
    client.ListTools();
  });
  // Cleaned up immediately after each iteration
}

// ✅ ALSO GOOD: Explicit cleanup with try-finally
for (let i = 0; i < 10000; i++) {
  const client = new MCP("dsl");
  try {
    client.ListTools();
  } finally {
    client.Release(); // Explicit immediate cleanup
  }
}
```

## Comparison: Use() vs try-finally

### Option 1: `Use()` - Automatic Cleanup (Recommended)

**When to use**: Most cases, especially when you want clean code and immediate cleanup.

```javascript
function test() {
  return Use(MCP, "dsl", (client) => {
    const tools = client.ListTools();
    return tools;
  });
  // client.__release() or client.Release() called automatically here
}
```

**Pros**:

- ✅ Shorter, cleaner code
- ✅ Impossible to forget cleanup
- ✅ Works with nested resources

**Cons**:

- ❌ Slight callback nesting overhead
- ❌ Less explicit control

### Option 2: `try-finally` with `Release()` - Manual Cleanup

**When to use**: Critical memory scenarios, or when you need explicit control.

```javascript
function test() {
  const client = new MCP("dsl");
  try {
    const tools = client.ListTools();
    return tools;
  } finally {
    client.Release(); // Explicit, immediate cleanup
  }
}
```

**Pros**:

- ✅ Explicit control over cleanup timing
- ✅ No callback overhead
- ✅ Better for complex control flow

**Cons**:

- ❌ More boilerplate code
- ❌ Easy to forget cleanup
- ❌ Nested resources become verbose

### Option 3: No Cleanup - Rely on GC (Not Recommended)

**When to use**: Never in production. Only for quick scripts or debugging.

```javascript
function test() {
  const client = new MCP("dsl");
  const tools = client.ListTools();
  return tools;
  // client waits for V8 GC to call __release() - SLOW and unpredictable!
}
```

**Why avoid**:

- ❌ Memory accumulates until GC runs
- ❌ Unpredictable timing
- ❌ May cause out-of-memory in loops

## API Reference

### Signature

```typescript
function Use<T>(
  Constructor: new (...args: any[]) => T,
  ...args: [...ConstructorArgs, (instance: T) => any]
): any;
```

### Parameters

- `Constructor`: A constructor function (must be callable with `new`)
- `...args`: Arguments for the constructor, followed by the callback function
- `callback`: Function that receives the created instance (last argument)

### Return Value

Returns the value returned by the callback function.

### Error Handling

- If the constructor fails: Error is thrown, no cleanup needed
- If the callback throws: Error is propagated, but `__release()` is still called
- If `__release()` throws: Error is silently ignored

## Implementation Details

- Uses V8 Go API for optimal performance
- Zero JavaScript code injection
- Thread-safe resource management
- **Requires objects to have a `__release()` method**
- LIFO cleanup order for nested `Use()` calls
- Only calls `__release()`, never `Release()` (separation of concerns)

## Testing

Run tests:

```bash
cd gou/runtime/v8/functions/use
go test -v
```

All tests use mock constructors and don't require external dependencies.
