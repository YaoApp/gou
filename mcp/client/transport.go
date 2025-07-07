package client

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/mcp/types"
)

// Send sends a message through the transport
func (c *Client) Send(ctx context.Context, message types.Message) error {
	if c.MCPClient == nil {
		return fmt.Errorf("MCP client not initialized")
	}

	// For now, this is a no-op as the underlying client may not support direct message sending
	// TODO: Implement actual message sending when mcp-go API is clarified
	return nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.MCPClient == nil {
		return fmt.Errorf("MCP client not initialized")
	}

	err := c.MCPClient.Close()
	c.MCPClient = nil  // Clear the client reference
	c.InitResult = nil // Clear the initialization result
	return err
}
