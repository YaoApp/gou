package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/yaoapp/gou/mcp/types"
)

// eventMutex protects event handler operations
var eventMutex sync.RWMutex

// OnEvent registers an event handler for a specific event type
func (c *Client) OnEvent(eventType string, handler func(event types.Event)) {
	eventMutex.Lock()
	defer eventMutex.Unlock()

	// Initialize event handlers for this type if not exists
	if c.eventHandlers == nil {
		c.eventHandlers = make(map[string][]func(event types.Event))
	}

	// Add handler to the list
	c.eventHandlers[eventType] = append(c.eventHandlers[eventType], handler)

	// TODO: Implement actual event handling when mcp-go API is available
	// This would typically involve registering with the underlying MCP client
}

// OnNotification registers a notification handler for a specific method
func (c *Client) OnNotification(method string, handler types.NotificationHandler) {
	eventMutex.Lock()
	defer eventMutex.Unlock()

	// Initialize notification handlers for this method if not exists
	if c.notificationHandlers == nil {
		c.notificationHandlers = make(map[string][]types.NotificationHandler)
	}

	// Add handler to the list
	c.notificationHandlers[method] = append(c.notificationHandlers[method], handler)

	// TODO: Implement actual notification handling when mcp-go API is available
	// This would typically involve registering with the underlying MCP client
}

// OnError registers an error handler
func (c *Client) OnError(handler types.ErrorHandler) {
	eventMutex.Lock()
	defer eventMutex.Unlock()

	// Initialize error handlers if not exists
	if c.errorHandlers == nil {
		c.errorHandlers = []types.ErrorHandler{}
	}

	// Add handler to the list
	c.errorHandlers = append(c.errorHandlers, handler)

	// TODO: Implement actual error handling when mcp-go API is available
	// This would typically involve registering with the underlying MCP client
}

// RemoveEventHandler removes an event handler for a specific event type
func (c *Client) RemoveEventHandler(eventType string, handler func(event types.Event)) {
	eventMutex.Lock()
	defer eventMutex.Unlock()

	if c.eventHandlers == nil {
		return
	}

	handlers, exists := c.eventHandlers[eventType]
	if !exists {
		return
	}

	// Remove the handler by finding it and removing from slice
	for i, h := range handlers {
		// Note: This is a simplified comparison - in practice you might need
		// a more sophisticated way to identify and remove specific handlers
		if &h == &handler {
			c.eventHandlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	// Clean up empty handler list
	if len(c.eventHandlers[eventType]) == 0 {
		delete(c.eventHandlers, eventType)
	}
}

// RemoveNotificationHandler removes a notification handler for a specific method
func (c *Client) RemoveNotificationHandler(method string, handler types.NotificationHandler) {
	eventMutex.Lock()
	defer eventMutex.Unlock()

	if c.notificationHandlers == nil {
		return
	}

	handlers, exists := c.notificationHandlers[method]
	if !exists {
		return
	}

	// Remove the handler by finding it and removing from slice
	for i, h := range handlers {
		// Note: This is a simplified comparison - in practice you might need
		// a more sophisticated way to identify and remove specific handlers
		if &h == &handler {
			c.notificationHandlers[method] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	// Clean up empty handler list
	if len(c.notificationHandlers[method]) == 0 {
		delete(c.notificationHandlers, method)
	}
}

// TriggerEvent triggers all registered event handlers for a specific event type
func (c *Client) TriggerEvent(event types.Event) {
	eventMutex.RLock()
	defer eventMutex.RUnlock()

	if c.eventHandlers == nil {
		return
	}

	handlers, exists := c.eventHandlers[event.Type]
	if !exists {
		return
	}

	// Trigger all handlers for this event type
	for _, handler := range handlers {
		go func(h func(event types.Event)) {
			defer func() {
				if r := recover(); r != nil {
					// Handle panic in event handler gracefully
					c.TriggerErrorStandard(context.Background(), fmt.Errorf("event handler panic: %v", r))
				}
			}()
			h(event)
		}(handler)
	}
}

// TriggerNotification triggers all registered notification handlers for a specific method
func (c *Client) TriggerNotification(ctx context.Context, notification types.Message) {
	eventMutex.RLock()
	defer eventMutex.RUnlock()

	if c.notificationHandlers == nil {
		return
	}

	handlers, exists := c.notificationHandlers[notification.Method]
	if !exists {
		return
	}

	// Trigger all handlers for this method
	for _, handler := range handlers {
		go func(h types.NotificationHandler) {
			defer func() {
				if r := recover(); r != nil {
					// Handle panic in notification handler gracefully
					c.TriggerErrorStandard(ctx, fmt.Errorf("notification handler panic: %v", r))
				}
			}()
			h(ctx, notification)
		}(handler)
	}
}

// TriggerErrorStandard triggers all registered error handlers with a standard error
func (c *Client) TriggerErrorStandard(ctx context.Context, err error) {
	eventMutex.RLock()
	defer eventMutex.RUnlock()

	if c.errorHandlers == nil {
		return
	}

	// Trigger all error handlers
	for _, handler := range c.errorHandlers {
		go func(h types.ErrorHandler) {
			defer func() {
				if r := recover(); r != nil {
					// If error handler itself panics, we can't do much more
					// Just continue with other handlers
				}
			}()
			h(ctx, err)
		}(handler)
	}
}

// ClearAllHandlers removes all registered handlers
func (c *Client) ClearAllHandlers() {
	eventMutex.Lock()
	defer eventMutex.Unlock()

	c.eventHandlers = make(map[string][]func(event types.Event))
	c.notificationHandlers = make(map[string][]types.NotificationHandler)
	c.errorHandlers = []types.ErrorHandler{}
}

// GetEventHandlers returns a copy of current event handlers (for debugging/inspection)
func (c *Client) GetEventHandlers() map[string]int {
	eventMutex.RLock()
	defer eventMutex.RUnlock()

	result := make(map[string]int)
	for eventType, handlers := range c.eventHandlers {
		result[eventType] = len(handlers)
	}
	return result
}

// GetNotificationHandlers returns a copy of current notification handlers (for debugging/inspection)
func (c *Client) GetNotificationHandlers() map[string]int {
	eventMutex.RLock()
	defer eventMutex.RUnlock()

	result := make(map[string]int)
	for method, handlers := range c.notificationHandlers {
		result[method] = len(handlers)
	}
	return result
}
