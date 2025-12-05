package process_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

// TestCallToolWithExtraArgs tests the extra arguments feature
func TestCallToolWithExtraArgs(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("CallTool with extra arguments", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Call tool with extra arguments that will be appended to process call
		args := map[string]interface{}{
			"message": "test",
		}

		// Extra arguments will be appended after the mapped arguments
		extraCtx := map[string]interface{}{
			"user_id": "123",
			"session": "abc",
		}

		resp, err := client.CallTool(context.Background(), "ping", args, extraCtx, "extra_param")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Greater(t, len(resp.Content), 0, "Should have content")
	})

	t.Run("CallTool without extra arguments (backward compatibility)", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		args := map[string]interface{}{
			"message": "test",
		}

		// Call without extra args - should work as before
		resp, err := client.CallTool(context.Background(), "ping", args)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Greater(t, len(resp.Content), 0, "Should have content")
	})
}

// TestCallToolsWithExtraArgs tests the extra arguments feature for batch calls
func TestCallToolsWithExtraArgs(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("CallTools with extra arguments", func(t *testing.T) {
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

		// Extra arguments will be appended to each tool call
		extraCtx := map[string]interface{}{
			"user_id": "456",
			"tenant":  "default",
		}

		resp, err := client.CallTools(context.Background(), calls, extraCtx)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 2, len(resp.Results), "Should have 2 results")
	})

	t.Run("CallTools without extra arguments (backward compatibility)", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		calls := []types.ToolCall{
			{
				Name:      "ping",
				Arguments: map[string]interface{}{"message": "test"},
			},
		}

		// Call without extra args - should work as before
		resp, err := client.CallTools(context.Background(), calls)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 1, len(resp.Results))
	})
}

// TestCallToolsParallelWithExtraArgs tests the extra arguments feature for parallel calls
func TestCallToolsParallelWithExtraArgs(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("CallToolsParallel with extra arguments", func(t *testing.T) {
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

		// Extra arguments shared across all parallel calls
		extraCtx := map[string]interface{}{
			"request_id": "req-789",
			"trace_id":   "trace-xyz",
		}

		resp, err := client.CallToolsParallel(context.Background(), calls, extraCtx)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 3, len(resp.Results), "Should have 3 results")

		// All results should be present (order matches input)
		for i, result := range resp.Results {
			assert.NotNil(t, result.Content, "Result %d should have content", i)
		}
	})

	t.Run("CallToolsParallel without extra arguments (backward compatibility)", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		calls := []types.ToolCall{
			{
				Name:      "ping",
				Arguments: map[string]interface{}{"message": "test"},
			},
		}

		// Call without extra args - should work as before
		resp, err := client.CallToolsParallel(context.Background(), calls)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 1, len(resp.Results))
	})
}

// TestExtraArgsWithParameterMapping tests extra args combined with x-process-args mapping
func TestExtraArgsWithParameterMapping(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("Extra args appended after mapped arguments", func(t *testing.T) {
		client, err := mcp.Select("customer")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Tool with x-process-args: ["$name", "$email", "$phone", "$status"]
		args := map[string]interface{}{
			"name":   "Test Customer",
			"email":  "test@example.com",
			"phone":  "555-0100",
			"status": "active",
		}

		// Extra arguments will be appended after the mapped args
		extraCtx := map[string]interface{}{
			"user_id":    "admin",
			"created_by": "system",
		}

		result, err := client.CallTool(context.Background(), "create_customer", args, extraCtx)

		// Should not have parameter extraction errors
		if err != nil {
			assert.NotContains(t, err.Error(), "extracting arguments")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
		}
	})

	t.Run("Multiple extra args of different types", func(t *testing.T) {
		client, err := mcp.Select("customer")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		args := map[string]interface{}{
			"name":  "Test Customer",
			"email": "test@example.com",
		}

		// Pass multiple extra args of different types
		// They will all be appended in order
		result, err := client.CallTool(
			context.Background(),
			"create_customer",
			args,
			"string_param",
			123,
			true,
			map[string]interface{}{"key": "value"},
		)

		// Should handle mixed types
		if err != nil {
			assert.NotContains(t, err.Error(), "extracting arguments")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
		}
	})
}

// TestExtraArgsWithDefaultBehavior tests extra args with tools that have no x-process-args
func TestExtraArgsWithDefaultBehavior(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("Extra args with default behavior (no x-process-args)", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Tools without x-process-args pass entire args object + extra args
		args := map[string]interface{}{
			"message": "test",
		}

		extraCtx := map[string]interface{}{
			"context": "test_context",
		}

		result, err := client.CallTool(context.Background(), "ping", args, extraCtx)

		// Should work with default behavior
		if err != nil {
			assert.NotContains(t, err.Error(), "extracting arguments")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
		}
	})
}
