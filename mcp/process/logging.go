package process

import (
	"context"

	"github.com/yaoapp/gou/mcp/types"
)

// SetLogLevel sets the log level for the server
func (c *Client) SetLogLevel(ctx context.Context, level types.LogLevel) error {
	// TODO: Implement process-based set log level
	// This will call a Yao process like: process.New("mcp.client.logging.setlevel", clientID, level)
	c.mu.Lock()
	c.currentLogLevel = level
	c.mu.Unlock()
	return nil
}

// GetLogLevel returns the current log level
func (c *Client) GetLogLevel() types.LogLevel {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.currentLogLevel == "" {
		return types.LogLevelInfo
	}
	return c.currentLogLevel
}
