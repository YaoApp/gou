package mcp

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/application/disk"
	"github.com/yaoapp/gou/mcp/process"
	"github.com/yaoapp/gou/mcp/types"
)

func init() {
	// Initialize application for process tests if not already done
	// This uses the same logic as mcp_test.go TestMain
	if application.App == nil {
		root := os.Getenv("GOU_TEST_APPLICATION")
		if root == "" {
			root = "../.." // Default to gou root directory
		}

		diskApp, err := disk.Open(root)
		if err == nil {
			application.Load(diskApp)
		}
	}
}

// TestLoadProcessClient tests loading process-based MCP clients with mapping
func TestLoadProcessClient(t *testing.T) {
	// Skip if application not initialized
	if application.App == nil {
		t.Skip("Application not initialized, skipping process client tests")
	}

	t.Run("Load DSL MCP Client with tools mapping", func(t *testing.T) {
		client, err := LoadClient("mcps/dsl.mcp.yao", "dsl")
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, Exists("dsl"))

		// Check if mapping was loaded
		mapping, err := GetClientMapping("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, mapping)

		// Verify tools were loaded
		assert.Greater(t, len(mapping.Tools), 0, "Should have loaded tools")
		assert.Contains(t, mapping.Tools, "validate_model", "Should have validate_model tool")
		assert.Contains(t, mapping.Tools, "format_flow", "Should have format_flow tool")
		assert.Contains(t, mapping.Tools, "analyze_api", "Should have analyze_api tool")

		// Check tool details
		validateTool := mapping.Tools["validate_model"]
		assert.Equal(t, "validate_model", validateTool.Name)
		assert.Equal(t, "scripts.dsl.ValidateModel", validateTool.Process)
		assert.NotEmpty(t, validateTool.InputSchema)
	})

	t.Run("Load Echo MCP Client with tools mapping", func(t *testing.T) {
		client, err := LoadClient("mcps/echo.mcp.yao", "echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, Exists("echo"))

		// Check if mapping was loaded
		mapping, err := GetClientMapping("echo")
		assert.NoError(t, err)
		assert.NotNil(t, mapping)

		// Verify tools were loaded
		assert.Equal(t, 2, len(mapping.Tools), "Should have 2 tools (ping, status)")
		assert.Contains(t, mapping.Tools, "ping", "Should have ping tool")
		assert.Contains(t, mapping.Tools, "status", "Should have status tool")

		// Check ping tool has output schema
		pingTool := mapping.Tools["ping"]
		assert.NotEmpty(t, pingTool.InputSchema)
		assert.NotEmpty(t, pingTool.OutputSchema, "Ping should have output schema")
	})

	t.Run("Load Customer MCP Client with tools and resources", func(t *testing.T) {
		client, err := LoadClient("mcps/customer.mcp.yao", "customer")
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, Exists("customer"))

		// Check if mapping was loaded
		mapping, err := GetClientMapping("customer")
		assert.NoError(t, err)
		assert.NotNil(t, mapping)

		// Verify tools were loaded
		assert.Equal(t, 2, len(mapping.Tools), "Should have 2 tools")
		assert.Contains(t, mapping.Tools, "create_customer")
		assert.Contains(t, mapping.Tools, "update_customer")

		// Verify resources were loaded
		assert.Equal(t, 2, len(mapping.Resources), "Should have 2 resources")
		assert.Contains(t, mapping.Resources, "detail")
		assert.Contains(t, mapping.Resources, "list")

		// Check resource details
		detailResource := mapping.Resources["detail"]
		assert.Equal(t, "detail", detailResource.Name)
		assert.Equal(t, "models.customer.Find", detailResource.Process)
		assert.Equal(t, "customers://{id}", detailResource.URI)

		listResource := mapping.Resources["list"]
		assert.Equal(t, "list", listResource.Name)
		assert.Equal(t, "models.customer.Paginate", listResource.Process)
		assert.Equal(t, "customers://list", listResource.URI)
		assert.Equal(t, 4, len(listResource.Parameters), "List resource should have 4 parameters")
	})

	t.Run("Unload process client should clean up mapping", func(t *testing.T) {
		// Create a custom mapping in memory instead of loading from file
		dsl := `{
			"transport": "process",
			"name": "test-unload",
			"tools": {"test_tool": "scripts.test.Tool"}
		}`

		customMapping := &MappingData{
			Tools: map[string]*ToolSchema{
				"test_tool": {
					Name:        "test_tool",
					Process:     "scripts.test.Tool",
					InputSchema: []byte(`{"type": "object"}`),
				},
			},
			Resources: map[string]*ResourceSchema{},
			Prompts:   map[string]*PromptSchema{},
		}

		// Load client with custom mapping
		client, err := LoadClientSource(dsl, "test-unload", customMapping)
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Verify mapping exists
		_, err = GetClientMapping("test-unload")
		assert.NoError(t, err)

		// Unload client
		UnloadClient("test-unload")
		assert.False(t, Exists("test-unload"))

		// Mapping should be cleaned up
		_, err = GetClientMapping("test-unload")
		assert.Error(t, err)
	})

	// Cleanup
	t.Cleanup(func() {
		UnloadClient("dsl")
		UnloadClient("echo")
		UnloadClient("customer")
	})
}

// TestProcessClientMapping tests direct mapping registry operations
func TestProcessClientMapping(t *testing.T) {
	t.Run("SetMapping and GetMapping", func(t *testing.T) {
		clientID := "test-mapping"

		// Create test mapping
		testMapping := &MappingData{
			Tools: map[string]*ToolSchema{
				"test_tool": {
					Name:        "test_tool",
					Description: "Test tool",
					Process:     "scripts.test.Tool",
					InputSchema: []byte(`{"type": "object"}`),
				},
			},
			Resources: map[string]*ResourceSchema{},
			Prompts:   map[string]*PromptSchema{},
		}

		// Set mapping
		process.SetMapping(clientID, testMapping)

		// Get mapping
		retrieved, exists := process.GetMapping(clientID)
		assert.True(t, exists)
		assert.NotNil(t, retrieved)
		assert.Equal(t, 1, len(retrieved.Tools))
		assert.Equal(t, "scripts.test.Tool", retrieved.Tools["test_tool"].Process)

		// Cleanup
		process.RemoveMapping(clientID)
	})

	t.Run("UpdateClientMapping", func(t *testing.T) {
		// Create a client with initial mapping
		dsl := `{
			"transport": "process",
			"name": "test-update",
			"tools": {"tool1": "scripts.test.Tool1"}
		}`

		initialMapping := &MappingData{
			Tools: map[string]*ToolSchema{
				"tool1": {
					Name:        "tool1",
					Process:     "scripts.test.Tool1",
					InputSchema: []byte(`{"type": "object"}`),
				},
			},
			Resources: map[string]*ResourceSchema{},
			Prompts:   map[string]*PromptSchema{},
		}

		client, err := LoadClientSource(dsl, "test-update", initialMapping)
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Add a new tool dynamically
		newTool := &ToolSchema{
			Name:        "tool2",
			Description: "Dynamically added tool",
			Process:     "scripts.test.Tool2",
			InputSchema: []byte(`{"type": "object"}`),
		}

		err = UpdateClientMapping("test-update", map[string]*ToolSchema{"tool2": newTool}, nil, nil)
		assert.NoError(t, err)

		// Verify tool was added
		mapping, err := GetClientMapping("test-update")
		assert.NoError(t, err)
		assert.Contains(t, mapping.Tools, "tool1", "Original tool should remain")
		assert.Contains(t, mapping.Tools, "tool2", "New tool should be added")
		assert.Equal(t, "scripts.test.Tool2", mapping.Tools["tool2"].Process)

		// Cleanup
		UnloadClient("test-update")
	})

	t.Run("RemoveClientMappingItems", func(t *testing.T) {
		// Create a client with multiple tools
		dsl := `{
			"transport": "process",
			"name": "test-remove",
			"tools": {"tool1": "scripts.test.Tool1", "tool2": "scripts.test.Tool2"}
		}`

		initialMapping := &MappingData{
			Tools: map[string]*ToolSchema{
				"tool1": {
					Name:        "tool1",
					Process:     "scripts.test.Tool1",
					InputSchema: []byte(`{"type": "object"}`),
				},
				"tool2": {
					Name:        "tool2",
					Process:     "scripts.test.Tool2",
					InputSchema: []byte(`{"type": "object"}`),
				},
			},
			Resources: map[string]*ResourceSchema{},
			Prompts:   map[string]*PromptSchema{},
		}

		client, err := LoadClientSource(dsl, "test-remove", initialMapping)
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Remove one tool
		err = RemoveClientMappingItems("test-remove", []string{"tool1"}, nil, nil)
		assert.NoError(t, err)

		// Verify tool was removed
		mapping, err := GetClientMapping("test-remove")
		assert.NoError(t, err)
		assert.NotContains(t, mapping.Tools, "tool1", "Removed tool should not exist")
		assert.Contains(t, mapping.Tools, "tool2", "Other tool should remain")

		// Cleanup
		UnloadClient("test-remove")
	})
}

// TestLoadClientSourceWithMapping tests loading from source with custom mapping
func TestLoadClientSourceWithMapping(t *testing.T) {
	t.Run("Load process client with provided mapping data", func(t *testing.T) {
		dsl := `{
			"transport": "process",
			"name": "test-custom-mapping",
			"tools": {
				"custom_tool": "scripts.custom.Tool"
			}
		}`

		// Create custom mapping
		customMapping := &MappingData{
			Tools: map[string]*ToolSchema{
				"custom_tool": {
					Name:        "custom_tool",
					Description: "Custom tool",
					Process:     "scripts.custom.Tool",
					InputSchema: []byte(`{"type": "object", "properties": {"input": {"type": "string"}}}`),
				},
			},
			Resources: map[string]*ResourceSchema{},
			Prompts:   map[string]*PromptSchema{},
		}

		// Load with custom mapping
		client, err := LoadClientSource(dsl, "test-custom", customMapping)
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Verify custom mapping was used
		mapping, err := GetClientMapping("test-custom")
		assert.NoError(t, err)
		assert.NotNil(t, mapping)
		assert.Equal(t, 1, len(mapping.Tools))
		assert.Contains(t, mapping.Tools, "custom_tool")

		// Cleanup
		UnloadClient("test-custom")
	})

	t.Run("Load process client without mapping data", func(t *testing.T) {
		dsl := `{
			"transport": "process",
			"name": "test-no-mapping"
		}`

		// Load without tools/resources/prompts - should create empty mapping
		client, err := LoadClientSource(dsl, "test-no-mapping")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Should have empty mapping
		mapping, err := GetClientMapping("test-no-mapping")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(mapping.Tools))
		assert.Equal(t, 0, len(mapping.Resources))
		assert.Equal(t, 0, len(mapping.Prompts))

		// Cleanup
		UnloadClient("test-no-mapping")
	})
}

// Helper type aliases for cleaner test code
type (
	MappingData    = types.MappingData
	ToolSchema     = types.ToolSchema
	ResourceSchema = types.ResourceSchema
	PromptSchema   = types.PromptSchema
)
