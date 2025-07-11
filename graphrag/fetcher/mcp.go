package fetcher

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/mcp"
	mcpTypes "github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/kun/any"
)

// MCP the mcp client for fetching URLs
type MCP struct {
	ID                  string            `json:"id"`                             // ID the mcp id
	Tool                string            `json:"tool"`                           // Tool the tool name
	ArgumentsMapping    map[string]string `json:"arguments_mapping,omitempty"`    // ArgumentsMapping the arguments mapping
	ResultMapping       map[string]string `json:"result_mapping,omitempty"`       // ResultMapping the result mapping
	NotificationMapping map[string]string `json:"notification_mapping,omitempty"` // NotificationMapping the notification mapping
	client              mcp.Client
}

// MCPOptions the mcp options
type MCPOptions struct {
	ID                  string            `json:"id"`                             // ID the mcp id
	Tool                string            `json:"tool"`                           // Tool the tool name
	ArgumentsMapping    map[string]string `json:"arguments_mapping,omitempty"`    // ArgumentsMapping the arguments mapping
	NotificationMapping map[string]string `json:"notification_mapping,omitempty"` // NotificationMapping the notification mapping
	ResultMapping       map[string]string `json:"result_mapping,omitempty"`       // ResultMapping the result mapping
}

// NewMCP creates a new MCP fetcher for fetching URLs using MCP client
func NewMCP(options *MCPOptions) (*MCP, error) {
	client, err := mcp.Select(options.ID)
	if err != nil {
		return nil, err
	}
	return &MCP{
		client:              client,
		ID:                  options.ID,
		Tool:                options.Tool,
		ArgumentsMapping:    options.ArgumentsMapping,
		NotificationMapping: options.NotificationMapping,
		ResultMapping:       options.ResultMapping,
	}, nil
}

// Fetch implements the Fetcher interface for MCP-based URL fetching
func (m *MCP) Fetch(ctx context.Context, url string, callback ...types.FetcherProgress) (string, string, error) {
	// Set up notification handler
	m.client.OnNotification("progress", func(ctx context.Context, notification mcpTypes.Message) error {
		m.reportProgress(notification, callback...)
		return nil
	})

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Starting URL fetch",
			"progress": 0.0,
			"url":      url,
		},
	}, callback...)

	// Get arguments using mapping
	arguments, err := m.getArguments(url)
	if err != nil {
		m.reportProgress(mcpTypes.Message{
			Method: "progress",
			Params: map[string]interface{}{
				"message":  "Failed to map arguments",
				"progress": 0.0,
				"error":    err.Error(),
				"url":      url,
			},
		}, callback...)
		return "", "", fmt.Errorf("failed to map arguments: %w", err)
	}

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Calling MCP tool",
			"progress": 0.2,
			"url":      url,
		},
	}, callback...)

	// Call MCP tool
	toolResult, err := m.client.CallTool(ctx, m.Tool, arguments)
	if err != nil {
		m.reportProgress(mcpTypes.Message{
			Method: "progress",
			Params: map[string]interface{}{
				"message":  "MCP tool call failed",
				"progress": 0.0,
				"error":    err.Error(),
				"url":      url,
			},
		}, callback...)
		return "", "", fmt.Errorf("MCP tool call failed: %w", err)
	}

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Processing result",
			"progress": 0.8,
			"url":      url,
		},
	}, callback...)

	// Convert result using mapping
	content, mimeType, err := m.getResult(toolResult)
	if err != nil {
		m.reportProgress(mcpTypes.Message{
			Method: "progress",
			Params: map[string]interface{}{
				"message":  "Failed to map result",
				"progress": 0.0,
				"error":    err.Error(),
				"url":      url,
			},
		}, callback...)
		return "", "", fmt.Errorf("failed to map result: %w", err)
	}

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "URL fetch completed",
			"progress": 1.0,
			"url":      url,
			"bytes":    int64(len(content)),
		},
	}, callback...)

	return content, mimeType, nil
}

// reportProgress report the progress
func (m *MCP) reportProgress(notification mcpTypes.Message, callback ...types.FetcherProgress) {
	if len(callback) == 0 || m.NotificationMapping == nil {
		return
	}

	// Convert notification to map for binding
	notificationData := map[string]interface{}{
		"notification": notification,
	}

	// Default mapping values
	defaultMapping := map[string]interface{}{
		"status":   types.FetcherStatusPending,
		"message":  "",
		"progress": 0.0,
		"url":      "",
		"bytes":    int64(0),
	}

	// Apply notification mapping
	for key, template := range m.NotificationMapping {
		if value := helper.Bind(template, notificationData); value != nil {
			defaultMapping[key] = value
		}
	}

	// Extract mapped values
	var status types.FetcherStatus
	var message string
	var progress float64
	var url string
	var bytes int64

	if s, ok := defaultMapping["status"]; ok {
		if statusStr, ok := s.(string); ok {
			status = types.FetcherStatus(statusStr)
		} else {
			status = types.FetcherStatusPending
		}
	}

	if m, ok := defaultMapping["message"]; ok {
		if messageStr, ok := m.(string); ok {
			message = messageStr
		}
	}

	if p, ok := defaultMapping["progress"]; ok {
		if progressFloat, ok := p.(float64); ok {
			progress = progressFloat
		} else if progressInt, ok := p.(int); ok {
			progress = float64(progressInt)
		}
	}

	if u, ok := defaultMapping["url"]; ok {
		if urlStr, ok := u.(string); ok {
			url = urlStr
		}
	}

	if b, ok := defaultMapping["bytes"]; ok {
		if bytesInt, ok := b.(int64); ok {
			bytes = bytesInt
		} else if bytesInt, ok := b.(int); ok {
			bytes = int64(bytesInt)
		}
	}

	// Create payload and call callbacks
	payload := types.FetcherPayload{
		Status:   status,
		Message:  message,
		Progress: progress,
		URL:      url,
		Bytes:    bytes,
	}

	for _, cb := range callback {
		if cb != nil {
			cb(status, payload)
		}
	}
}

// getArguments get the arguments for MCP tool call
func (m *MCP) getArguments(url string) (map[string]interface{}, error) {
	if m.ArgumentsMapping == nil {
		// Default mapping if no custom mapping provided
		return map[string]interface{}{
			"url": url,
		}, nil
	}

	// Create data context for binding
	bindingData := map[string]interface{}{
		"url": url,
	}

	// Apply argument mapping
	arguments := make(map[string]interface{})
	for key, template := range m.ArgumentsMapping {
		if value := helper.Bind(template, bindingData); value != nil {
			arguments[key] = value
		}
	}

	return arguments, nil
}

// getResult get the result from MCP tool response
func (m *MCP) getResult(result interface{}) (string, string, error) {
	if m.ResultMapping == nil {
		// Default mapping if no custom mapping provided
		// Assume result is in the format: {"content": "...", "mime_type": "..."}
		if resultMap, ok := result.(map[string]interface{}); ok {
			content := ""
			mimeType := "text/plain"

			if c, ok := resultMap["content"].(string); ok {
				content = c
			}
			if mt, ok := resultMap["mime_type"].(string); ok {
				mimeType = mt
			}

			return content, mimeType, nil
		}

		// If result is not a map, try to convert to string
		return fmt.Sprintf("%v", result), "text/plain", nil
	}

	// Convert result to map and flatten for binding
	var resultData map[string]interface{}

	// Try to convert to map first, with error handling
	func() {
		defer func() {
			if r := recover(); r != nil {
				// If conversion fails, wrap in a map
				wrappedData := map[string]interface{}{
					"result": result,
				}
				wrappedRes := any.Of(wrappedData)
				resultData = wrappedRes.Map().MapStrAny.Dot()
			}
		}()

		res := any.Of(result)
		mapResult := res.Map()
		if mapResult.MapStrAny != nil {
			// If it's already a map, use it directly
			resultData = mapResult.MapStrAny.Dot()
		} else {
			// If not a map/struct, wrap it in a map and flatten
			wrappedData := map[string]interface{}{
				"result": result,
			}
			wrappedRes := any.Of(wrappedData)
			resultData = wrappedRes.Map().MapStrAny.Dot()
		}
	}()

	// Default values
	content := ""
	mimeType := "text/plain"

	// Apply result mapping
	for key, template := range m.ResultMapping {
		value := helper.Bind(template, resultData)
		if value != nil {
			switch key {
			case "content":
				if contentStr, ok := value.(string); ok {
					content = contentStr
				}
			case "mime_type":
				if mimeTypeStr, ok := value.(string); ok {
					mimeType = mimeTypeStr
				}
			}
		}
	}

	return content, mimeType, nil
}
