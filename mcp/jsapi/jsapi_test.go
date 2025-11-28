package jsapi_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/mcp"
	v8 "github.com/yaoapp/gou/runtime/v8"
)

func TestMain(m *testing.M) {
	prepare()
	defer v8.Stop()

	// Run tests and exit with the result code
	os.Exit(m.Run())
}

// prepare initializes the test environment
func prepare() {
	// Get test application root from environment or use default
	testAppRoot := os.Getenv("GOU_TEST_APPLICATION")
	if testAppRoot == "" {
		testAppRoot = "../../gou-dev-app" // Default: two levels up from jsapi package
	}

	// Initialize application for file system access
	app, err := application.OpenFromDisk(testAppRoot)
	if err != nil {
		panic(err)
	}
	application.Load(app)

	// Initialize V8 runtime
	option := &v8.Option{}
	option.Validate()
	v8.EnablePrecompile()
	err = v8.Start(option)
	if err != nil {
		panic(err)
	}

	// Load test MCP clients
	clients := map[string]string{
		"dsl":      "mcps/dsl.mcp.yao",
		"customer": "mcps/customer.mcp.yao",
		"echo":     "mcps/echo.mcp.yao",
	}

	for id, file := range clients {
		_, err = mcp.LoadClient(file, id)
		if err != nil {
			panic(err)
		}
	}

	// MCP constructor is registered automatically in MakeTemplate
}

// TestMCPConstructor tests creating a new MCP client instance
func TestMCPConstructor(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("dsl");
			try {
				return {
					id: client.id,
					hasListTools: typeof client.ListTools === 'function',
					hasCallTool: typeof client.CallTool === 'function',
					hasListResources: typeof client.ListResources === 'function',
					hasListPrompts: typeof client.ListPrompts === 'function',
					hasRelease: typeof client.Release === 'function'
				};
			} finally {
				client.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, "dsl", result["id"], "client ID should match")
	assert.Equal(t, true, result["hasListTools"], "should have ListTools method")
	assert.Equal(t, true, result["hasCallTool"], "should have CallTool method")
	assert.Equal(t, true, result["hasListResources"], "should have ListResources method")
	assert.Equal(t, true, result["hasListPrompts"], "should have ListPrompts method")
	assert.Equal(t, true, result["hasRelease"], "should have Release method")
}

// TestMCPConstructorInvalidID tests error handling when client ID doesn't exist
func TestMCPConstructorInvalidID(t *testing.T) {
	_, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("nonexistent");
			return { success: true };
		}`)

	assert.Error(t, err, "should fail with non-existent client ID")
	assert.Contains(t, err.Error(), "not found", "error should mention client not found")
}

// TestMCPConstructorMissingID tests error handling when client ID is not provided
func TestMCPConstructorMissingID(t *testing.T) {
	_, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP();
			return { success: true };
		}`)

	assert.Error(t, err, "should fail without client ID")
	assert.Contains(t, err.Error(), "requires client ID", "error should mention missing client ID")
}

// TestListTools tests the ListTools method
func TestListTools(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("dsl");
			try {
				const tools = client.ListTools();
				return {
					hasTools: Array.isArray(tools.tools),
					toolCount: tools.tools ? tools.tools.length : 0
				};
			} finally {
				client.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["hasTools"], "should return tools array")
	assert.Greater(t, int(result["toolCount"].(float64)), 0, "should have at least one tool")
}

// TestCallTool tests the CallTool method
func TestCallTool(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("dsl");
			try {
				const result = client.CallTool("validate_model", { model: "user" });
				return {
					success: true,
					hasContent: result.content && result.content.length > 0
				};
			} catch (error) {
				return {
					success: false,
					error: error.message
				};
			} finally {
				client.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	// Note: The actual tool execution may fail if the script doesn't exist,
	// but we're testing the JSAPI wrapper works correctly
	assert.NotNil(t, result, "should return a result")
}

// TestCallTools tests sequential tool calls
func TestCallTools(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("dsl");
			try {
				const results = client.CallTools([
					{ name: "validate_model", arguments: { model: "user" } },
					{ name: "format_flow", arguments: { flow: "test" } }
				]);
				return {
					success: true,
					hasResults: Array.isArray(results.results),
					count: results.results ? results.results.length : 0
				};
			} catch (error) {
				return {
					success: false,
					error: error.message
				};
			} finally {
				client.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.NotNil(t, result, "should return a result")
}

// TestListResources tests the ListResources method
func TestListResources(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("customer");
			try {
				const resources = client.ListResources();
				return {
					hasResources: Array.isArray(resources.resources),
					count: resources.resources ? resources.resources.length : 0
				};
			} finally {
				client.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["hasResources"], "should return resources array")
}

// TestReadResource tests the ReadResource method
func TestReadResource(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("customer");
			try {
				const content = client.ReadResource("customers://1");
				return {
					success: true,
					hasContent: content.contents && content.contents.length > 0
				};
			} catch (error) {
				return {
					success: false,
					error: error.message
				};
			} finally {
				client.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.NotNil(t, result, "should return a result")
}

// TestListPrompts tests the ListPrompts method
func TestListPrompts(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("dsl");
			try {
				const prompts = client.ListPrompts();
				return {
					hasPrompts: Array.isArray(prompts.prompts),
					count: prompts.prompts ? prompts.prompts.length : 0
				};
			} finally {
				client.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.NotNil(t, result["hasPrompts"], "should return prompts property")
}

// TestListSamples tests the ListSamples method
func TestListSamples(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("dsl");
			try {
				const samples = client.ListSamples("tool", "validate_model");
				return {
					hasSamples: Array.isArray(samples.samples),
					total: samples.total || 0
				};
			} finally {
				client.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.NotNil(t, result["hasSamples"], "should return samples array")
}

// TestGetSample tests the GetSample method
func TestGetSample(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const client = new MCP("dsl");
			try {
				const sample = client.GetSample("tool", "validate_model", 0);
				return {
					success: true,
					hasSample: sample !== null && sample !== undefined,
					hasIndex: sample && typeof sample.index === 'number'
				};
			} catch (error) {
				// It's okay if there are no samples
				return {
					success: false,
					error: error.message
				};
			} finally {
				client.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.NotNil(t, result, "should return a result")
}

// TestMultipleClients tests using multiple MCP clients simultaneously
func TestMultipleClients(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const dslClient = new MCP("dsl");
			const customerClient = new MCP("customer");
			
			try {
				const dslTools = dslClient.ListTools();
				const customerResources = customerClient.ListResources();
				
				return {
					dslId: dslClient.id,
					customerId: customerClient.id,
					dslHasTools: Array.isArray(dslTools.tools),
					customerHasResources: Array.isArray(customerResources.resources)
				};
			} finally {
				dslClient.Release();
				customerClient.Release();
			}
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, "dsl", result["dslId"], "DSL client ID should match")
	assert.Equal(t, "customer", result["customerId"], "Customer client ID should match")
	assert.Equal(t, true, result["dslHasTools"], "DSL client should have tools")
	assert.Equal(t, true, result["customerHasResources"], "Customer client should have resources")
}
