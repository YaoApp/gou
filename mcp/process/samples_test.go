package process_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/mcp/process"
	"github.com/yaoapp/gou/mcp/types"
)

func TestListSamples(t *testing.T) {
	Prepare(t)
	defer Clean()

	ctx := context.Background()

	t.Run("should list samples for DSL validate_model tool", func(t *testing.T) {
		client, err := mcp.Select("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// List samples for validate_model tool
		resp, err := client.ListSamples(ctx, types.SampleTool, "validate_model")
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		// If samples exist, verify structure
		if resp.Total > 0 {
			assert.Equal(t, resp.Total, len(resp.Samples), "Total should match sample count")

			// Verify first sample structure
			sample := resp.Samples[0]
			assert.Equal(t, 0, sample.Index, "First sample should have index 0")
			assert.Equal(t, "validate_model", sample.ItemName)
			assert.NotNil(t, sample.Input, "Sample should have input")
		}
	})

	t.Run("should return empty list for tool without samples", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// List samples for a tool that likely has no samples
		resp, err := client.ListSamples(ctx, types.SampleTool, "nonexistent_tool")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 0, resp.Total)
		assert.Equal(t, 0, len(resp.Samples))
	})

	t.Run("should handle customer tool samples", func(t *testing.T) {
		client, err := mcp.Select("customer")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// List samples for create_customer
		resp, err := client.ListSamples(ctx, types.SampleTool, "create_customer")
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify response structure regardless of whether samples exist
		assert.GreaterOrEqual(t, resp.Total, 0)
	})
}

func TestMultiLevelPath(t *testing.T) {
	Prepare(t)
	defer Clean()

	// Load foo.bar client for testing
	_, err := mcp.LoadClient("mcps/foo/bar.mcp.yao", "")
	if err != nil {
		t.Fatalf("Failed to load foo.bar client: %v", err)
	}
	defer mcp.UnloadClient("foo.bar")

	// Test multi-level MCP ID path conversion (foo.bar → foo/bar)
	client, err := mcp.Select("foo.bar")
	if err != nil {
		t.Fatalf("Failed to select foo.bar MCP client: %v", err)
	}

	processClient, ok := client.(*process.Client)
	if !ok {
		t.Fatalf("Expected process.Client, got %T", client)
	}

	// Test ListSamples with multi-level path
	t.Run("should list samples with multi-level path", func(t *testing.T) {
		resp, err := processClient.ListSamples(context.Background(), types.SampleTool, "test_action")
		if err != nil {
			t.Fatalf("ListSamples failed: %v", err)
		}

		if resp.Total != 2 {
			t.Errorf("Expected 2 samples, got %d", resp.Total)
		}

		if len(resp.Samples) != 2 {
			t.Errorf("Expected 2 samples, got %d", len(resp.Samples))
		}

		// Verify first sample content
		if resp.Samples[0].Index != 0 {
			t.Errorf("Expected index 0, got %d", resp.Samples[0].Index)
		}
		if resp.Samples[0].ItemName != "test_action" {
			t.Errorf("Expected itemName 'test_action', got %s", resp.Samples[0].ItemName)
		}
		if msg, ok := resp.Samples[0].Input["message"].(string); !ok || msg != "hello world" {
			t.Errorf("Expected input message 'hello world', got %v", resp.Samples[0].Input["message"])
		}
	})

	// Test GetSample with multi-level path
	t.Run("should get specific sample with multi-level path", func(t *testing.T) {
		sample, err := processClient.GetSample(context.Background(), types.SampleTool, "test_action", 1)
		if err != nil {
			t.Fatalf("GetSample failed: %v", err)
		}

		if sample.Index != 1 {
			t.Errorf("Expected index 1, got %d", sample.Index)
		}
		if msg, ok := sample.Input["message"].(string); !ok || msg != "test foo.bar path" {
			t.Errorf("Expected input message 'test foo.bar path', got %v", sample.Input["message"])
		}
	})
}

func TestGetSample(t *testing.T) {
	Prepare(t)
	defer Clean()

	ctx := context.Background()

	t.Run("should get specific sample by index", func(t *testing.T) {
		client, err := mcp.Select("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// First list samples to know how many exist
		listResp, err := client.ListSamples(ctx, types.SampleTool, "validate_model")
		assert.NoError(t, err)

		if listResp.Total > 0 {
			// Get first sample
			sample, err := client.GetSample(ctx, types.SampleTool, "validate_model", 0)
			assert.NoError(t, err)
			assert.NotNil(t, sample)
			assert.Equal(t, 0, sample.Index)
			assert.Equal(t, "validate_model", sample.ItemName)
			assert.NotNil(t, sample.Input)

			// Verify it matches the first sample from ListSamples
			assert.Equal(t, listResp.Samples[0].Input, sample.Input)
		}
	})

	t.Run("should return error for invalid index", func(t *testing.T) {
		client, err := mcp.Select("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Try to get sample with negative index
		sample, err := client.GetSample(ctx, types.SampleTool, "validate_model", -1)
		assert.Error(t, err)
		assert.Nil(t, sample)
		assert.Contains(t, err.Error(), "invalid sample index")
	})

	t.Run("should return error for out of range index", func(t *testing.T) {
		client, err := mcp.Select("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Try to get sample with very large index
		sample, err := client.GetSample(ctx, types.SampleTool, "validate_model", 999999)
		assert.Error(t, err)
		assert.Nil(t, sample)
		assert.Contains(t, err.Error(), "sample not found")
	})

	t.Run("should return error for nonexistent tool", func(t *testing.T) {
		client, err := mcp.Select("echo")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Try to get sample for tool that doesn't exist
		sample, err := client.GetSample(ctx, types.SampleTool, "nonexistent_tool", 0)
		assert.Error(t, err)
		assert.Nil(t, sample)
		assert.Contains(t, err.Error(), "no samples found")
	})
}

func TestSamplesMultipleClients(t *testing.T) {
	Prepare(t)
	defer Clean()

	ctx := context.Background()

	t.Run("should handle samples across different clients", func(t *testing.T) {
		// Test DSL client
		dslClient, err := mcp.Select("dsl")
		assert.NoError(t, err)
		dslResp, err := dslClient.ListSamples(ctx, types.SampleTool, "validate_model")
		assert.NoError(t, err)
		assert.NotNil(t, dslResp)

		// Test Customer client
		customerClient, err := mcp.Select("customer")
		assert.NoError(t, err)
		customerResp, err := customerClient.ListSamples(ctx, types.SampleTool, "create_customer")
		assert.NoError(t, err)
		assert.NotNil(t, customerResp)

		// Test Echo client
		echoClient, err := mcp.Select("echo")
		assert.NoError(t, err)
		echoResp, err := echoClient.ListSamples(ctx, types.SampleTool, "ping")
		assert.NoError(t, err)
		assert.NotNil(t, echoResp)
	})
}

func TestResourceSamples(t *testing.T) {
	Prepare(t)
	defer Clean()

	ctx := context.Background()

	t.Run("should list resource samples", func(t *testing.T) {
		client, err := mcp.Select("customer")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// List samples for detail resource
		resp, err := client.ListSamples(ctx, types.SampleResource, "detail")
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		if resp.Total > 0 {
			assert.Equal(t, resp.Total, len(resp.Samples))

			// Verify first sample structure
			sample := resp.Samples[0]
			assert.Equal(t, 0, sample.Index)
			assert.Equal(t, "detail", sample.ItemName)
			assert.NotEmpty(t, sample.URI, "Resource sample should have URI")
			assert.NotNil(t, sample.Data, "Resource sample should have Data")

			// Verify URI format
			assert.Contains(t, sample.URI, "customers://")
		}
	})

	t.Run("should get specific resource sample", func(t *testing.T) {
		client, err := mcp.Select("customer")
		assert.NoError(t, err)

		// Get first sample from list resource
		sample, err := client.GetSample(ctx, types.SampleResource, "list", 0)
		assert.NoError(t, err)
		assert.NotNil(t, sample)

		assert.Equal(t, 0, sample.Index)
		assert.Equal(t, "list", sample.ItemName)
		assert.NotEmpty(t, sample.URI)
		assert.Contains(t, sample.URI, "customers://list")
		assert.NotNil(t, sample.Data)
	})

	t.Run("should return empty for non-existent resource samples", func(t *testing.T) {
		client, err := mcp.Select("dsl")
		assert.NoError(t, err)

		// DSL doesn't have resources, so no resource samples
		resp, err := client.ListSamples(ctx, types.SampleResource, "nonexistent")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.Samples)
	})
}
