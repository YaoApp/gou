package process

import (
	"context"

	"github.com/yaoapp/gou/mcp/types"
)

// ListTools requests a list of available tools from the server
func (c *Client) ListTools(ctx context.Context, cursor string) (*types.ListToolsResponse, error) {
	// TODO: Implement process-based list tools
	// This will call a Yao process like: process.New("mcp.client.tools.list", clientID, cursor)
	return nil, nil
}

// CallTool invokes a specific tool on the server
func (c *Client) CallTool(ctx context.Context, name string, arguments interface{}) (*types.CallToolResponse, error) {
	// TODO: Implement process-based call tool
	// This will call a Yao process like: process.New("mcp.client.tools.call", clientID, name, arguments)
	return nil, nil
}

// CallToolsBatch calls multiple tools in sequence
func (c *Client) CallToolsBatch(ctx context.Context, tools []types.ToolCall) (*types.CallToolsBatchResponse, error) {
	// TODO: Implement process-based batch call tools
	// This will call a Yao process like: process.New("mcp.client.tools.batch", clientID, tools)
	return nil, nil
}
