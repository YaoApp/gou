package process

import (
	"context"

	"github.com/yaoapp/gou/mcp/types"
)

// Connect establishes connection to the MCP server via Yao Process
func (c *Client) Connect(ctx context.Context, options ...types.ConnectionOptions) error {
	// TODO: Implement process-based connection
	// This will call a Yao process to establish connection
	return nil
}

// Disconnect closes the connection to the MCP server
func (c *Client) Disconnect(ctx context.Context) error {
	// TODO: Implement process-based disconnection
	return nil
}

// IsConnected checks if the client is connected to the server
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// State returns the current connection state
func (c *Client) State() types.ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return types.StateDisconnected
	}

	if c.InitResult != nil {
		return types.StateInitialized
	}

	return types.StateConnected
}
