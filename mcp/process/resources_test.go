package process_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

func TestListResources(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("should return resources from customer MCP", func(t *testing.T) {
		// Get customer client using Select
		client, err := mcp.Select("customer")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Test ListResources
		resp, err := client.ListResources(context.Background(), "")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 2, len(resp.Resources), "Customer should have 2 resources")

		// Verify resource details
		resourceMap := make(map[string]*types.Resource)
		for i := range resp.Resources {
			resourceMap[resp.Resources[i].Name] = &resp.Resources[i]
		}

		// Verify detail resource
		detailRes, ok := resourceMap["detail"]
		assert.True(t, ok, "Should have detail resource")
		assert.Equal(t, "detail", detailRes.Name)
		assert.Equal(t, "customers://{id}", detailRes.URI)
		assert.NotEmpty(t, detailRes.Description)
		assert.Equal(t, "application/json", detailRes.MimeType)

		// Verify list resource
		listRes, ok := resourceMap["list"]
		assert.True(t, ok, "Should have list resource")
		assert.Equal(t, "list", listRes.Name)
		assert.Equal(t, "customers://list", listRes.URI)
		assert.NotEmpty(t, listRes.Description)
		assert.Equal(t, "application/json", listRes.MimeType)
	})

	t.Run("should return empty list for MCP without resources", func(t *testing.T) {
		// Get echo client (no resources, only tools)
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		resp, err := client.ListResources(context.Background(), "")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 0, len(resp.Resources), "Echo should have no resources")
	})
}

func TestSubscribeResource(t *testing.T) {
	Prepare(t)
	defer Clean()

	// Get customer client using Select
	client, err := mcp.Select("customer")
	assert.NoError(t, err)
	assert.NotNil(t, client)

	t.Run("should return error for subscribe", func(t *testing.T) {
		err := client.SubscribeResource(context.Background(), "customers://1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("should return error for unsubscribe", func(t *testing.T) {
		err := client.UnsubscribeResource(context.Background(), "customers://1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})
}
