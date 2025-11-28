package process_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

func TestListTools(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("should return tools from dsl MCP", func(t *testing.T) {
		// Get dsl client using Select
		client, err := mcp.Select("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Test ListTools
		resp, err := client.ListTools(context.Background(), "")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 4, len(resp.Tools), "DSL should have 4 tools")

		// Verify tool details
		toolMap := make(map[string]*types.Tool)
		for i := range resp.Tools {
			toolMap[resp.Tools[i].Name] = &resp.Tools[i]
		}

		// Verify validate_model
		validateTool, ok := toolMap["validate_model"]
		assert.True(t, ok, "Should have validate_model tool")
		assert.Equal(t, "validate_model", validateTool.Name)
		assert.NotEmpty(t, validateTool.Description)
		assert.NotEmpty(t, validateTool.InputSchema, "Should have input schema")

		// Verify format_flow
		formatTool, ok := toolMap["format_flow"]
		assert.True(t, ok, "Should have format_flow tool")
		assert.Equal(t, "format_flow", formatTool.Name)
		assert.NotEmpty(t, formatTool.InputSchema)

		// Verify analyze_api
		analyzeTool, ok := toolMap["analyze_api"]
		assert.True(t, ok, "Should have analyze_api tool")

		// Verify test_nested (for testing nested field extraction)
		nestedTool, ok := toolMap["test_nested"]
		assert.True(t, ok, "Should have test_nested tool")
		assert.Equal(t, "test_nested", nestedTool.Name)
		assert.NotEmpty(t, nestedTool.InputSchema)
		assert.Equal(t, "analyze_api", analyzeTool.Name)
		assert.NotEmpty(t, analyzeTool.InputSchema)
	})

	t.Run("should return tools from echo MCP", func(t *testing.T) {
		// Get echo client
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		resp, err := client.ListTools(context.Background(), "")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 2, len(resp.Tools), "Echo should have 2 tools")

		// Verify tool details
		toolMap := make(map[string]*types.Tool)
		for i := range resp.Tools {
			toolMap[resp.Tools[i].Name] = &resp.Tools[i]
		}

		// Verify ping tool
		pingTool, ok := toolMap["ping"]
		assert.True(t, ok, "Should have ping tool")
		assert.Equal(t, "ping", pingTool.Name)
		assert.NotEmpty(t, pingTool.InputSchema, "Ping should have input schema")

		// Verify status tool
		statusTool, ok := toolMap["status"]
		assert.True(t, ok, "Should have status tool")
		assert.Equal(t, "status", statusTool.Name)
		assert.NotEmpty(t, statusTool.InputSchema, "Status should have input schema")
	})

	t.Run("should return tools from customer MCP", func(t *testing.T) {
		// Get customer client
		client, err := mcp.Select("customer")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		resp, err := client.ListTools(context.Background(), "")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 2, len(resp.Tools), "Customer should have 2 tools")

		// Verify tool details
		toolMap := make(map[string]*types.Tool)
		for i := range resp.Tools {
			toolMap[resp.Tools[i].Name] = &resp.Tools[i]
		}

		// Verify create_customer tool
		createTool, ok := toolMap["create_customer"]
		assert.True(t, ok, "Should have create_customer tool")
		assert.Equal(t, "create_customer", createTool.Name)
		assert.NotEmpty(t, createTool.InputSchema, "Should have input schema")

		// Verify update_customer tool
		updateTool, ok := toolMap["update_customer"]
		assert.True(t, ok, "Should have update_customer tool")
		assert.Equal(t, "update_customer", updateTool.Name)
		assert.NotEmpty(t, updateTool.InputSchema, "Should have input schema")
	})
}
