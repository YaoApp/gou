# MCP JSAPI

JavaScript API for MCP (Model Context Protocol) clients in Yao.

## Quick Start

### Recommended: Using `Use()` for Automatic Resource Management

The `Use()` function automatically handles resource cleanup with a callback pattern:

```javascript
// Simple and clean - resources are automatically released
Use(MCP, "client_id", (client) => {
  const tools = client.ListTools();
  const result = client.CallTool("tool_name", { arg: "value" });
  return result;
});
// client.__release() is automatically called after the callback, even if an error occurs
```

### Alternative: Manual Resource Management

For cases where you need explicit control or complex control flow:

```javascript
const client = new MCP("client_id");
try {
  // Use the client...
  const result = client.CallTool("tool_name", { arg: "value" });
  return result;
} finally {
  client.Release(); // Manually release resources
}
```

## Resource Management

MCP clients hold Go resources that must be released. Two approaches:

### ✅ Recommended: `Use()` Function

**Pros:**

- Automatic cleanup - no need to remember `Release()`
- Less boilerplate code
- Always releases resources, even on errors
- Cleaner, more readable code

**When to use:**

- Most cases
- When resource lifetime matches callback scope
- When you want guaranteed cleanup with minimal code

```javascript
Use(MCP, "dsl", (client) => {
  // Use client here
  return client.CallTool("validate", { schema: data });
});
```

### Alternative: Manual `try-finally`

**Pros:**

- Explicit control over resource lifetime
- Suitable for complex control flow
- No callback nesting

**When to use:**

- Resource needs to be used across multiple scopes
- Complex error handling requirements
- Performance-critical code (minimal overhead difference)

```javascript
const client = new MCP("dsl");
try {
  // Use client
} finally {
  client.Release();
}
```

## API Reference

### Tool Operations

```javascript
// List available tools
Use(MCP, "client_id", (client) => {
  const tools = client.ListTools();
  console.log(tools.tools); // Array of tool definitions
});

// Call a single tool
Use(MCP, "client_id", (client) => {
  const result = client.CallTool("tool_name", {
    arg1: "value1",
    arg2: 123,
  });
  return result;
});

// Call multiple tools sequentially
Use(MCP, "client_id", (client) => {
  const results = client.CallTools([
    { name: "tool1", arguments: { foo: "bar" } },
    { name: "tool2", arguments: { baz: 42 } },
  ]);
  return results;
});

// Call multiple tools in parallel
Use(MCP, "client_id", (client) => {
  const results = client.CallToolsParallel([
    { name: "tool1", arguments: { foo: "bar" } },
    { name: "tool2", arguments: { baz: 42 } },
  ]);
  return results;
});
```

### Resource Operations

```javascript
// List available resources
Use(MCP, "customer", (client) => {
  const resources = client.ListResources();
  return resources.resources;
});

// Read a resource by URI
Use(MCP, "customer", (client) => {
  const content = client.ReadResource("customers://123");
  return content.contents;
});
```

### Prompt Operations

```javascript
// List available prompts
Use(MCP, "client_id", (client) => {
  const prompts = client.ListPrompts();
  return prompts.prompts;
});

// Get a prompt template with arguments
Use(MCP, "client_id", (client) => {
  const prompt = client.GetPrompt("prompt_name", {
    arg1: "value1",
  });
  return prompt.messages;
});
```

### Sample Operations

```javascript
// List samples for a tool
Use(MCP, "client_id", (client) => {
  const samples = client.ListSamples("tool", "tool_name");
  return samples.samples;
});

// Get a specific sample by index
Use(MCP, "client_id", (client) => {
  const sample = client.GetSample("tool", "tool_name", 0);
  return { input: sample.input, output: sample.output };
});

// List samples for a resource
Use(MCP, "client_id", (client) => {
  return client.ListSamples("resource", "resource_name");
});
```

## Complete Examples

### Example 1: Using `Use()` (Recommended)

```javascript
function processCustomer(customerId) {
  return Use(MCP, "customer", (mcp) => {
    // List available tools
    const tools = mcp.ListTools();
    console.log(
      "Available tools:",
      tools.tools.map((t) => t.name)
    );

    // Read customer resource
    const customer = mcp.ReadResource(`customers://${customerId}`);

    // Call a tool
    const result = mcp.CallTool("update_customer", {
      id: customerId,
      status: "active",
    });

    return result;
  }); // Automatic cleanup happens here
}
```

### Example 2: Nested MCP Clients

```javascript
function validateAndProcess(schema, customerId) {
  return Use(MCP, "dsl", (dslClient) => {
    // Validate schema
    const validation = dslClient.CallTool("validate_schema", { schema });

    if (!validation.isValid) {
      throw new Error("Invalid schema");
    }

    // Process customer with another MCP client
    return Use(MCP, "customer", (customerClient) => {
      return customerClient.CallTool("process", {
        id: customerId,
        schema: validation.normalized,
      });
    });
  });
}
```

### Example 3: Error Handling with `Use()`

```javascript
function safeToolCall(toolName, args) {
  try {
    return Use(MCP, "client_id", (client) => {
      // Errors are properly propagated
      return client.CallTool(toolName, args);
    });
    // client.__release() is still called even if an error occurs
  } catch (error) {
    console.error("Tool call failed:", error.message);
    return { error: error.message };
  }
}
```

### Example 4: Manual Resource Management

For complex scenarios where `Use()` doesn't fit:

```javascript
function complexWorkflow(customerId) {
  const mcp = new MCP("customer");

  try {
    // Step 1: Read customer
    const customer = mcp.ReadResource(`customers://${customerId}`);

    // Step 2: Some complex logic...
    if (customer.status === "inactive") {
      return null; // Early return
    }

    // Step 3: Update customer
    const result = mcp.CallTool("update_customer", {
      id: customerId,
      lastAccess: new Date().toISOString(),
    });

    return result;
  } finally {
    // Always release resources
    mcp.Release();
  }
}
```

## API Methods

### Constructor

- `new MCP(clientId: string)` - Creates a new MCP client instance

### Properties

- `id: string` - The client ID

### Tool Methods

- `ListTools(cursor?: string): ListToolsResponse` - List all available tools
- `CallTool(name: string, arguments?: object): CallToolResponse` - Call a single tool
- `CallTools(toolCalls: ToolCall[]): CallToolsResponse` - Call multiple tools sequentially
- `CallToolsParallel(toolCalls: ToolCall[]): CallToolsResponse` - Call multiple tools in parallel

### Resource Methods

- `ListResources(cursor?: string): ListResourcesResponse` - List all available resources
- `ReadResource(uri: string): ReadResourceResponse` - Read a resource by URI

### Prompt Methods

- `ListPrompts(cursor?: string): ListPromptsResponse` - List all available prompts
- `GetPrompt(name: string, arguments?: object): GetPromptResponse` - Get a prompt template

### Sample Methods

- `ListSamples(itemType: "tool"|"resource", itemName: string): ListSamplesResponse` - List samples for a tool or resource
- `GetSample(itemType: "tool"|"resource", itemName: string, index: number): SampleData` - Get a specific sample by index

### Resource Management Methods

- `Release()` - Manually release Go resources (required when not using `Use()`)
- `__release()` - Internal method called automatically by `Use()` or V8 GC (do not call directly)

## Best Practices

1. **Use `Use()` by default** - It's cleaner and safer than manual resource management
2. **Handle errors with try-catch** - Wrap `Use()` calls when you need error handling
3. **Nest `Use()` calls** - For multiple MCP clients, nest them naturally
4. **Use manual `try-finally` only when needed** - For complex control flow or explicit lifetime control
5. **Never forget to release** - If not using `Use()`, always call `Release()` in a `finally` block

## Notes

- The MCP client must be loaded in Go before creating a JavaScript instance
- All methods are synchronous in the current implementation
- Errors are thrown as JavaScript exceptions
- `Use()` ensures resources are released even when errors occur
- When using `Use()`, the callback return value becomes the return value of `Use()`
- The `__release()` method is internal - use `Release()` for manual cleanup or `Use()` for automatic cleanup
