package process_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
)

func TestListPrompts(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("should return prompts from echo MCP", func(t *testing.T) {
		// Get echo client using Select
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Test ListPrompts
		resp, err := client.ListPrompts(context.Background(), "")
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		// Echo may or may not have prompts depending on configuration
		// Just verify it doesn't error and returns a valid response
		assert.NotNil(t, resp.Prompts)
	})

	t.Run("should return prompts from dsl MCP", func(t *testing.T) {
		// Get dsl client
		client, err := mcp.Select("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		resp, err := client.ListPrompts(context.Background(), "")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Prompts)
	})
}

func TestGetPrompt(t *testing.T) {
	Prepare(t)
	defer Clean()

	// Get echo client (if it has prompts)
	client, err := mcp.Select("echo")
	assert.NoError(t, err)
	assert.NotNil(t, client)

	t.Run("should return error for non-existent prompt", func(t *testing.T) {
		_, err := client.GetPrompt(context.Background(), "non_existent_prompt", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt not found")
	})

	// Note: We can't test successful GetPrompt without knowing exact prompt names
	// from the loaded MCP files. This would require checking the mapping first.
}
