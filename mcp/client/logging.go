package client

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/mcp/types"
)

// SetLogLevel sets the log level for the server
func (c *Client) SetLogLevel(ctx context.Context, level types.LogLevel) error {
	if c.MCPClient == nil {
		return fmt.Errorf("MCP client not initialized")
	}

	// For now, this is a no-op as the underlying client may not support log level setting
	// TODO: Implement actual log level setting when mcp-go API is clarified
	return nil
}
