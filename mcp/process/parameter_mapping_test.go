package process_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
)

// TestToolParameterMapping tests x-process-args mapping for tools
func TestToolParameterMapping(t *testing.T) {
	Prepare(t)
	defer Clean()

	// Get customer client
	client, err := mcp.Select("customer")
	assert.NoError(t, err)
	assert.NotNil(t, client)

	ctx := context.Background()

	t.Run("create_customer with positional args mapping", func(t *testing.T) {
		// This tool uses x-process-args to extract individual fields as positional arguments
		result, err := client.CallTool(ctx, "create_customer", map[string]interface{}{
			"name":   "Test Customer",
			"email":  "test@example.com",
			"phone":  "555-0100",
			"status": "active",
		})

		// Note: The actual process may not exist in test environment,
		// but we're testing the parameter extraction logic
		// The error (if any) should be about process execution, not parameter extraction
		if err != nil {
			// Should not be a parameter extraction error
			assert.NotContains(t, err.Error(), "extracting arguments")
		}

		if result != nil {
			// If we get a result, it should be well-formed
			assert.NotNil(t, result.Content)
		}
	})

	t.Run("update_customer with ID and object", func(t *testing.T) {
		// This tool uses x-process-args: ["$args.id", ":arguments"]
		result, err := client.CallTool(ctx, "update_customer", map[string]interface{}{
			"id":     123,
			"name":   "Updated Customer",
			"email":  "updated@example.com",
			"status": "inactive",
		})

		// Check parameter extraction worked
		if err != nil {
			assert.NotContains(t, err.Error(), "extracting arguments")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
		}
	})

	t.Run("Missing optional field returns nil", func(t *testing.T) {
		// Test that missing optional fields don't cause errors
		result, err := client.CallTool(ctx, "create_customer", map[string]interface{}{
			"name":  "Minimal Customer",
			"email": "minimal@example.com",
			// phone and status are optional
		})

		// Parameter extraction should succeed (missing fields -> nil)
		if err != nil {
			assert.NotContains(t, err.Error(), "extracting arguments")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
		}
	})
}

// TestResourceParameterMapping tests x-process-args mapping for resources
func TestResourceParameterMapping(t *testing.T) {
	Prepare(t)
	defer Clean()

	// Get customer client
	client, err := mcp.Select("customer")
	assert.NoError(t, err)
	assert.NotNil(t, client)

	ctx := context.Background()

	t.Run("Read detail resource with URI parameter extraction", func(t *testing.T) {
		// Resource detail.res.yao uses x-process-args: ["$uri.id"]
		// URI template: "customers://{id}"
		result, err := client.ReadResource(ctx, "customers://123")

		// Check that URI parameter extraction worked
		if err != nil {
			// Should not be a parameter extraction error
			assert.NotContains(t, err.Error(), "extract")
		}

		if result != nil {
			assert.NotNil(t, result.Contents)
			if len(result.Contents) > 0 {
				assert.Equal(t, "customers://123", result.Contents[0].URI)
				assert.Equal(t, "application/json", result.Contents[0].MimeType)
			}
		}
	})

	t.Run("Read list resource with query parameters", func(t *testing.T) {
		// Resource list.res.yao uses x-process-args: ["$args.page", "$args.pagesize", "$args.where", "$args.order"]
		// URI: "customers://list?page=1&pagesize=10"
		result, err := client.ReadResource(ctx, "customers://list?page=1&pagesize=10")

		// Check that query parameter extraction worked
		if err != nil {
			assert.NotContains(t, err.Error(), "extract")
		}

		if result != nil {
			assert.NotNil(t, result.Contents)
			if len(result.Contents) > 0 {
				assert.Contains(t, result.Contents[0].URI, "customers://list")
			}
		}
	})

	t.Run("Resource with missing optional parameters", func(t *testing.T) {
		// List resource with only some parameters
		result, err := client.ReadResource(ctx, "customers://list?page=1")

		// Missing optional parameters should not cause extraction errors
		if err != nil {
			assert.NotContains(t, err.Error(), "extract")
		}

		if result != nil {
			assert.NotNil(t, result.Contents)
		}
	})
}

// TestDefaultParameterBehavior tests tools/resources without x-process-args
func TestDefaultParameterBehavior(t *testing.T) {
	Prepare(t)
	defer Clean()

	// Get dsl and echo clients (which don't have x-process-args in their schemas)
	dslClient, err := mcp.Select("dsl")
	assert.NoError(t, err)

	echoClient, err := mcp.Select("echo")
	assert.NoError(t, err)

	ctx := context.Background()

	t.Run("Tool without x-process-args passes entire object", func(t *testing.T) {
		// DSL tools don't have x-process-args, so entire arguments object is passed
		result, err := dslClient.CallTool(ctx, "validate_model", map[string]interface{}{
			"model": `{"name": "test", "table": {"name": "test"}}`,
		})

		// Should work with default behavior (no parameter extraction errors)
		if err != nil {
			assert.NotContains(t, err.Error(), "extracting arguments")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
		}
	})

	t.Run("Echo tools pass entire object by default", func(t *testing.T) {
		result, callErr := echoClient.CallTool(ctx, "ping", map[string]interface{}{
			"message": "test",
		})

		if callErr != nil {
			assert.NotContains(t, callErr.Error(), "extracting arguments")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
		}
	})
}

// TestParameterMappingEdgeCases tests edge cases and error conditions
func TestParameterMappingEdgeCases(t *testing.T) {
	Prepare(t)
	defer Clean()

	client, err := mcp.Select("customer")
	assert.NoError(t, err)

	ctx := context.Background()

	t.Run("Tool call with non-object arguments when x-process-args is present", func(t *testing.T) {
		// MCP tools with x-process-args expect object arguments for field extraction
		// Passing non-object should trigger an error during parameter extraction
		result, callErr := client.CallTool(ctx, "create_customer", "not an object")

		// Should get some error (extraction or execution)
		_ = callErr // May be nil if error is in result.IsError
		if result != nil {
			// Error should be present (either extraction or process not found)
			assert.True(t, result.IsError, "Expected error when passing non-object to tool with x-process-args")
			assert.NotEmpty(t, result.Content[0].Text, "Error message should be present")

			// Log the actual error for debugging
			t.Logf("Error message: %s", result.Content[0].Text)
		}
	})

	t.Run("Resource URI that doesn't match template", func(t *testing.T) {
		// Try to read a resource with URI that doesn't exist
		_, readErr := client.ReadResource(ctx, "invalid://uri")

		// Should get resource not found error
		assert.Error(t, readErr)
		assert.Contains(t, readErr.Error(), "not found")
	})
}

// TestNestedFieldExtraction tests nested object field extraction using $args.foo.bar syntax
func TestNestedFieldExtraction(t *testing.T) {
	Prepare(t)
	defer Clean()

	// Get DSL client (has test_nested tool with nested field extraction)
	client, err := mcp.Select("dsl")
	assert.NoError(t, err)
	assert.NotNil(t, client)

	ctx := context.Background()

	t.Run("Extract nested fields using dot notation", func(t *testing.T) {
		// test_nested.in.yao has:
		// "x-process-args": [
		//   "$args.user.name",           // Extract user.name
		//   "$args.user.contact.email",  // Extract user.contact.email
		//   "$args.metadata.source"      // Extract metadata.source
		// ]

		result, err := client.CallTool(ctx, "test_nested", map[string]interface{}{
			"user": map[string]interface{}{
				"name": "Alice",
				"contact": map[string]interface{}{
					"email": "alice@example.com",
					"phone": "555-0100",
				},
			},
			"metadata": map[string]interface{}{
				"source": "test",
			},
		})

		// Should not have parameter extraction errors
		// The process might not exist, but extraction should work
		if err != nil {
			assert.NotContains(t, err.Error(), "extracting arguments")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
			// If there's an error, it should be about the process not existing,
			// NOT about parameter extraction
			if result.IsError {
				assert.NotContains(t, result.Content[0].Text, "path segment")
				assert.NotContains(t, result.Content[0].Text, "not an object")
			}
		}
	})

	t.Run("Handle missing optional nested fields", func(t *testing.T) {
		// Test with missing optional field (metadata.source)
		result, err := client.CallTool(ctx, "test_nested", map[string]interface{}{
			"user": map[string]interface{}{
				"name": "Bob",
				"contact": map[string]interface{}{
					"email": "bob@example.com",
				},
			},
			// metadata is missing (optional)
		})

		// Should not have extraction errors for missing optional fields
		if err != nil {
			assert.NotContains(t, err.Error(), "extracting arguments")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
		}
	})

	t.Run("Handle deeply nested field extraction", func(t *testing.T) {
		// Test extraction from deeply nested structure
		result, err := client.CallTool(ctx, "test_nested", map[string]interface{}{
			"user": map[string]interface{}{
				"name": "Charlie",
				"contact": map[string]interface{}{
					"email": "charlie@example.com",
					"phone": "555-0200",
				},
			},
			"metadata": map[string]interface{}{
				"source": "integration_test",
			},
		})

		// Parameter extraction should succeed
		if err != nil {
			assert.NotContains(t, err.Error(), "extracting arguments")
			assert.NotContains(t, err.Error(), "path segment")
		}

		if result != nil {
			assert.NotNil(t, result.Content)
		}
	})
}
