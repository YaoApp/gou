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

	if !c.IsInitialized() {
		return fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports logging
	initResult := c.GetInitResult()
	if initResult.Capabilities.Logging == nil {
		return fmt.Errorf("server does not support logging")
	}

	// Store the current log level in the client
	// In a real implementation, this would send a request to the server
	// For now, we simulate the functionality by storing the level
	c.currentLogLevel = level

	// TODO: Implement actual log level setting when mcp-go API is available
	// This would typically involve sending a JSON-RPC request to the server

	return nil
}

// GetLogLevel returns the current log level
func (c *Client) GetLogLevel() types.LogLevel {
	// Return default level if not set
	if c.currentLogLevel == "" {
		return types.LogLevelInfo
	}
	return c.currentLogLevel
}
