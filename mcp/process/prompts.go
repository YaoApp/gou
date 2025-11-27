package process

import (
	"context"

	"github.com/yaoapp/gou/mcp/types"
)

// ListPrompts lists all available prompts
func (c *Client) ListPrompts(ctx context.Context, cursor string) (*types.ListPromptsResponse, error) {
	// TODO: Implement process-based list prompts
	// This will call a Yao process like: process.New("mcp.client.prompts.list", clientID, cursor)
	return nil, nil
}

// GetPrompt gets a specific prompt with given arguments
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*types.GetPromptResponse, error) {
	// TODO: Implement process-based get prompt
	// This will call a Yao process like: process.New("mcp.client.prompts.get", clientID, name, arguments)
	return nil, nil
}
