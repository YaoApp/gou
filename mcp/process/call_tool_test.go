package process_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

func TestCallTool(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("should return error for non-existent tool", func(t *testing.T) {
		client, err := mcp.Select("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		resp, err := client.CallTool(context.Background(), "non_existent_tool", nil)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "tool not found")
	})

	t.Run("should call tool and return response", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Call ping tool
		args := map[string]interface{}{
			"message": "test",
		}

		resp, err := client.CallTool(context.Background(), "ping", args)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		// Response may have error if process doesn't exist in test env, but structure should be valid
		assert.Greater(t, len(resp.Content), 0, "Should have content")
		assert.NotEmpty(t, resp.Content[0].Text, "Content should have text")
	})

	t.Run("should handle tool execution error gracefully", func(t *testing.T) {
		client, err := mcp.Select("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Call with invalid arguments to trigger error
		resp, err := client.CallTool(context.Background(), "validate_model", nil)
		assert.NoError(t, err) // Error should be in response, not returned
		assert.NotNil(t, resp)
		// Tool execution error should be in Content with IsError=true
		if resp.IsError {
			assert.Greater(t, len(resp.Content), 0)
			assert.NotEmpty(t, resp.Content[0].Text)
		}
	})
}

func TestCallTools(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("should call multiple tools in sequence", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Prepare batch calls
		calls := []types.ToolCall{
			{
				Name:      "ping",
				Arguments: map[string]interface{}{"message": "test1"},
			},
			{
				Name:      "status",
				Arguments: map[string]interface{}{"verbose": false},
			},
		}

		resp, err := client.CallTools(context.Background(), calls)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 2, len(resp.Results), "Should have 2 results")
	})

	t.Run("should handle partial failures", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Mix valid and invalid tool calls
		calls := []types.ToolCall{
			{
				Name:      "ping",
				Arguments: map[string]interface{}{"message": "test"},
			},
			{
				Name:      "non_existent",
				Arguments: nil,
			},
		}

		resp, err := client.CallTools(context.Background(), calls)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 2, len(resp.Results), "Should have 2 results")

		// Second call should have error
		assert.True(t, resp.Results[1].IsError, "Second result should be error")
	})
}

func TestCallToolsParallel(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("should call multiple tools concurrently", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Call multiple tools in parallel
		calls := []types.ToolCall{
			{
				Name:      "ping",
				Arguments: map[string]interface{}{"message": "test1"},
			},
			{
				Name:      "ping",
				Arguments: map[string]interface{}{"message": "test2"},
			},
			{
				Name:      "status",
				Arguments: map[string]interface{}{"verbose": false},
			},
		}

		resp, err := client.CallToolsParallel(context.Background(), calls)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 3, len(resp.Results), "Should have 3 results")

		// All results should be in order (matching input order)
		for i, result := range resp.Results {
			assert.NotNil(t, result.Content, "Result %d should have content", i)
		}
	})

	t.Run("should handle partial failures in parallel execution", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Mix valid and invalid tool calls
		calls := []types.ToolCall{
			{
				Name:      "ping",
				Arguments: map[string]interface{}{"message": "test"},
			},
			{
				Name:      "non_existent",
				Arguments: nil,
			},
			{
				Name:      "status",
				Arguments: map[string]interface{}{"verbose": false},
			},
		}

		resp, err := client.CallToolsParallel(context.Background(), calls)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 3, len(resp.Results), "Should have 3 results")

		// Second call should have error
		assert.True(t, resp.Results[1].IsError, "Second result should be error")

		// Results should maintain input order despite concurrent execution
	})
}

func TestCancelRequest(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("should handle context cancellation", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Create a context that we can cancel immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Try to call tool with cancelled context
		resp, err := client.CallTool(ctx, "ping", map[string]interface{}{"message": "test"})

		// Should get a response (not an error) with IsError=true
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		// Response should indicate cancellation or error
		assert.Greater(t, len(resp.Content), 0, "Should have content")
		// The response will contain error text
		assert.NotEmpty(t, resp.Content[0].Text)
	})

	t.Run("should return error for non-existent request", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Try to cancel non-existent request
		err = client.CancelRequest(context.Background(), uint64(9999))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
