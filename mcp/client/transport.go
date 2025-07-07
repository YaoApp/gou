package client

import (
	"context"
	"fmt"
	"time"

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

	// Create a context with reasonable timeout for Close operation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use a goroutine to handle the potentially blocking Close() call
	errChan := make(chan error, 1)

	go func() {
		errChan <- c.MCPClient.Close()
	}()

	var err error
	select {
	case err = <-errChan:
		// Close completed normally
	case <-ctx.Done():
		// Context cancelled or timed out
		err = ctx.Err()
		// Note: We still clean up the client reference even if Close() times out
	}

	c.MCPClient = nil  // Clear the client reference
	c.InitResult = nil // Clear the initialization result
	return err
}
