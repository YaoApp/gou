package process

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/mcp/types"
)

// OnEvent registers an event handler for a specific event type
func (c *Client) OnEvent(eventType string, handler func(event types.Event)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add handler to the list
	c.eventHandlers[eventType] = append(c.eventHandlers[eventType], handler)

	// TODO: Implement actual event handling via Yao process
	// This might involve registering with a process-based event system
}

// OnNotification registers a notification handler for a specific method
func (c *Client) OnNotification(method string, handler types.NotificationHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add handler to the list
	c.notificationHandlers[method] = append(c.notificationHandlers[method], handler)

	// TODO: Implement actual notification handling via Yao process
}

// OnError registers an error handler
func (c *Client) OnError(handler types.ErrorHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add handler to the list
	c.errorHandlers = append(c.errorHandlers, handler)

	// TODO: Implement actual error handling via Yao process
}

// RemoveEventHandler removes an event handler for a specific event type
func (c *Client) RemoveEventHandler(eventType string, handler func(event types.Event)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	handlers, exists := c.eventHandlers[eventType]
	if !exists {
		return
	}

	// Remove the handler
	for i, h := range handlers {
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
	c.mu.Lock()
	defer c.mu.Unlock()

	handlers, exists := c.notificationHandlers[method]
	if !exists {
		return
	}

	// Remove the handler
	for i, h := range handlers {
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
	c.mu.RLock()
	handlers := c.eventHandlers[event.Type]
	c.mu.RUnlock()

	// Trigger all handlers for this event type
	for _, handler := range handlers {
		go func(h func(event types.Event)) {
			defer func() {
				if r := recover(); r != nil {
					c.TriggerErrorStandard(context.Background(), fmt.Errorf("event handler panic: %v", r))
				}
			}()
			h(event)
		}(handler)
	}
}

// TriggerNotification triggers all registered notification handlers for a specific method
func (c *Client) TriggerNotification(ctx context.Context, notification types.Message) {
	c.mu.RLock()
	handlers := c.notificationHandlers[notification.Method]
	c.mu.RUnlock()

	// Trigger all handlers for this method
	for _, handler := range handlers {
		go func(h types.NotificationHandler) {
			defer func() {
				if r := recover(); r != nil {
					c.TriggerErrorStandard(ctx, fmt.Errorf("notification handler panic: %v", r))
				}
			}()
			h(ctx, notification)
		}(handler)
	}
}

// TriggerErrorStandard triggers all registered error handlers with a standard error
func (c *Client) TriggerErrorStandard(ctx context.Context, err error) {
	c.mu.RLock()
	handlers := c.errorHandlers
	c.mu.RUnlock()

	// Trigger all error handlers
	for _, handler := range handlers {
		go func(h types.ErrorHandler) {
			defer func() {
				if r := recover(); r != nil {
					// If error handler itself panics, we can't do much more
				}
			}()
			h(ctx, err)
		}(handler)
	}
}

// ClearAllHandlers removes all registered handlers
func (c *Client) ClearAllHandlers() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.eventHandlers = make(map[string][]func(event types.Event))
	c.notificationHandlers = make(map[string][]types.NotificationHandler)
	c.errorHandlers = []types.ErrorHandler{}
}

// GetEventHandlers returns a copy of current event handlers (for debugging/inspection)
func (c *Client) GetEventHandlers() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]int)
	for eventType, handlers := range c.eventHandlers {
		result[eventType] = len(handlers)
	}
	return result
}

// GetNotificationHandlers returns a copy of current notification handlers (for debugging/inspection)
func (c *Client) GetNotificationHandlers() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]int)
	for method, handlers := range c.notificationHandlers {
		result[method] = len(handlers)
	}
	return result
}
