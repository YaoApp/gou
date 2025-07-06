package client

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/mcp/types"
)

// ListPrompts lists all available prompts
func (c *Client) ListPrompts(ctx context.Context, cursor string) (*types.ListPromptsResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	// For now, return empty response to avoid compilation errors
	// TODO: Implement actual prompt listing when mcp-go API is clarified
	response := &types.ListPromptsResponse{
		Prompts:    []types.Prompt{},
		NextCursor: "",
	}

	return response, nil
}

// GetPrompt gets a specific prompt with given arguments
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*types.GetPromptResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	// For now, return empty response to avoid compilation errors
	// TODO: Implement actual prompt getting when mcp-go API is clarified
	response := &types.GetPromptResponse{
		Description: "",
		Messages:    []types.PromptMessage{},
	}

	return response, nil
}
