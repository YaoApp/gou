package client

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/mcp/types"
)

// CreateSampling creates a sampling request if supported by the server
func (c *Client) CreateSampling(ctx context.Context, request types.SamplingRequest) (*types.SamplingResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	// For now, return empty response to avoid compilation errors
	// TODO: Implement actual sampling when mcp-go API is clarified
	response := &types.SamplingResponse{
		Model:      request.Model,
		Role:       "assistant",
		Content:    types.SamplingContent{Type: "text", Text: ""},
		StopReason: "",
	}

	return response, nil
}
