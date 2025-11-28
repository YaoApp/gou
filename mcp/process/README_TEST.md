# Process Package Tests

This package contains tests for the MCP Process Client implementation.

## Test Structure

### Test Environment Setup

- **Package**: `process_test` (to avoid circular dependencies)
- **Test Data**: Uses real MCP configurations from `gou-dev-app/mcps/`
- **Setup**: `TestMain` initializes the application environment
- **Helper Functions**:
  - `Prepare(t)`: Loads all test MCP clients (dsl, echo, customer)
  - `Clean()`: Unloads all test clients

### Test Files

1. **`process_test.go`**
   - Test environment setup
   - Common helper functions (`Prepare`, `Clean`)

2. **`resources_test.go`**
   - `TestListResources`: Verify resource listing from customer MCP
   - `TestSubscribeResource`: Verify subscription not supported

3. **`tools_test.go`**
   - `TestListTools`: Verify tool listing from dsl, echo, customer MCPs
   - `TestCallTool`: Verify tool calling not implemented

4. **`prompts_test.go`**
   - `TestListPrompts`: Verify prompt listing
   - `TestGetPrompt`: Verify prompt retrieval and template rendering

## Test Data

Tests use real MCP configurations from `gou-dev-app`:

- **dsl.mcp.yao**: 3 tools (validate_model, format_flow, analyze_api)
- **echo.mcp.yao**: 2 tools (ping, status)
- **customer.mcp.yao**: 2 tools + 2 resources

## Running Tests

```bash
# Run all process tests
cd gou/mcp/process
go test -v

# Run specific test
go test -v -run TestListTools

# With custom application path
GOU_TEST_APPLICATION=/path/to/gou-dev-app go test -v
```

## Test Pattern

All tests follow this pattern:

```go
func TestSomething(t *testing.T) {
    Prepare(t)          // Load MCP clients
    defer Clean()       // Cleanup

    client, err := mcp.Select("client-id")
    assert.NoError(t, err)

    // Test client functionality
}
```

## Coverage

- ✅ ListResources
- ✅ ListTools
- ✅ ListPrompts
- ✅ GetPrompt
- ✅ Subscribe/Unsubscribe (error cases)
- ✅ CallTool (not implemented)

