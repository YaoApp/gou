package client

import (
	"github.com/yaoapp/gou/mcp/types"
)

// OnEvent registers an event handler for a specific event type
func (c *Client) OnEvent(eventType string, handler func(event types.Event)) {
	if c.MCPClient == nil {
		return
	}

	// For now, this is a no-op as the underlying client may not support event handling
	// TODO: Implement actual event handling when mcp-go API is clarified
}

// OnNotification registers a notification handler for a specific method
func (c *Client) OnNotification(method string, handler types.NotificationHandler) {
	if c.MCPClient == nil {
		return
	}

	// For now, this is a no-op as the underlying client may not support notification handling
	// TODO: Implement actual notification handling when mcp-go API is clarified
}

// OnError registers an error handler
func (c *Client) OnError(handler types.ErrorHandler) {
	if c.MCPClient == nil {
		return
	}

	// For now, this is a no-op as the underlying client may not support error handling
	// TODO: Implement actual error handling when mcp-go API is clarified
}
