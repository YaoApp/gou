package converter

import (
	"context"
	"encoding/base64"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/mcp"
	mcpTypes "github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/kun/any"
)

// MCP the mcp client
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

// NewMCP convert the file to plain text using the mcp client
func NewMCP(options *MCPOptions) (*MCP, error) {
	client, err := mcp.Select(options.ID)
	if err != nil {
		return nil, err
	}
	return &MCP{client: client, ID: options.ID, Tool: options.Tool, ArgumentsMapping: options.ArgumentsMapping, NotificationMapping: options.NotificationMapping, ResultMapping: options.ResultMapping}, nil
}

// Convert convert the file to plain text
func (m *MCP) Convert(ctx context.Context, file string, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	// Set up notification handler
	m.client.OnNotification("progress", func(ctx context.Context, notification mcpTypes.Message) error {
		m.reportProgress(notification, callback...)
		return nil
	})

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Reading file",
			"progress": 0.1,
		},
	}, callback...)

	// Read file content
	content, err := os.ReadFile(file)
	if err != nil {
		m.reportProgress(mcpTypes.Message{
			Method: "progress",
			Params: map[string]interface{}{
				"message":  "Failed to read file",
				"progress": 0.0,
				"error":    err.Error(),
			},
		}, callback...)
		return nil, err
	}

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Creating data URI",
			"progress": 0.2,
		},
	}, callback...)

	// Create data URI with content type
	dataURI, err := m.createDataURI(file, content)
	if err != nil {
		m.reportProgress(mcpTypes.Message{
			Method: "progress",
			Params: map[string]interface{}{
				"message":  "Failed to create data URI",
				"progress": 0.0,
				"error":    err.Error(),
			},
		}, callback...)
		return nil, err
	}

	// Get arguments using mapping
	arguments, err := m.getArguments(dataURI)
	if err != nil {
		m.reportProgress(mcpTypes.Message{
			Method: "progress",
			Params: map[string]interface{}{
				"message":  "Failed to map arguments",
				"progress": 0.0,
				"error":    err.Error(),
			},
		}, callback...)
		return nil, err
	}

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Calling MCP tool",
			"progress": 0.4,
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
			},
		}, callback...)
		return nil, err
	}

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Processing result",
			"progress": 0.8,
		},
	}, callback...)

	// Convert result using mapping
	result, err := m.getResult(toolResult)
	if err != nil {
		m.reportProgress(mcpTypes.Message{
			Method: "progress",
			Params: map[string]interface{}{
				"message":  "Failed to map result",
				"progress": 0.0,
				"error":    err.Error(),
			},
		}, callback...)
		return nil, err
	}

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Conversion completed",
			"progress": 1.0,
		},
	}, callback...)

	return result, nil
}

// ConvertStream convert the stream to plain text
func (m *MCP) ConvertStream(ctx context.Context, stream io.ReadSeeker, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Saving stream to temporary file",
			"progress": 0.0,
		},
	}, callback...)

	// Save stream to temporary file
	tempFile, err := m.saveStreamToTempFile(stream)
	if err != nil {
		m.reportProgress(mcpTypes.Message{
			Method: "progress",
			Params: map[string]interface{}{
				"message":  "Failed to save stream to temp file",
				"progress": 0.0,
				"error":    err.Error(),
			},
		}, callback...)
		return nil, err
	}
	defer os.Remove(tempFile) // Clean up temp file

	m.reportProgress(mcpTypes.Message{
		Method: "progress",
		Params: map[string]interface{}{
			"message":  "Processing temporary file",
			"progress": 0.1,
		},
	}, callback...)

	// Call Convert method with the temporary file
	return m.Convert(ctx, tempFile, callback...)
}

// saveStreamToTempFile saves the stream to a temporary file
func (m *MCP) saveStreamToTempFile(stream io.ReadSeeker) (string, error) {
	tempFile, err := os.CreateTemp("", "mcp_input_*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, stream)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// reportProgress report the progress
func (m *MCP) reportProgress(notification mcpTypes.Message, callback ...types.ConverterProgress) {
	if len(callback) == 0 || m.NotificationMapping == nil {
		return
	}

	// Convert notification to map for binding
	notificationData := map[string]interface{}{
		"notification": notification,
	}

	// Default mapping values
	defaultMapping := map[string]interface{}{
		"status":   types.ConverterStatusPending,
		"message":  "",
		"progress": 0.0,
	}

	// Apply notification mapping
	for key, template := range m.NotificationMapping {
		if value := helper.Bind(template, notificationData); value != nil {
			defaultMapping[key] = value
		}
	}

	// Extract mapped values
	var status types.ConverterStatus
	var message string
	var progress float64

	if s, ok := defaultMapping["status"]; ok {
		if statusStr, ok := s.(string); ok {
			status = types.ConverterStatus(statusStr)
		} else {
			status = types.ConverterStatusPending
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

	// Create payload and call callbacks
	payload := types.ConverterPayload{
		Status:   status,
		Message:  message,
		Progress: progress,
	}

	for _, cb := range callback {
		if cb != nil {
			cb(status, payload)
		}
	}
}

// createDataURI creates a Data URI from file path and content
func (m *MCP) createDataURI(filePath string, content []byte) (string, error) {
	// Detect content type
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		// Try to detect by content
		contentType = http.DetectContentType(content)
	}
	if contentType == "" {
		// Default to binary
		contentType = "application/octet-stream"
	}

	// Encode content to base64
	base64Content := base64.StdEncoding.EncodeToString(content)

	// Create data URI: data:contentType;base64,xxxxx
	dataURI := "data:" + contentType + ";base64," + base64Content

	return dataURI, nil
}

// getArguments get the arguments
func (m *MCP) getArguments(dataURI string) (map[string]interface{}, error) {
	if m.ArgumentsMapping == nil {
		return map[string]interface{}{}, nil
	}

	// Create data context for binding
	bindingData := map[string]interface{}{
		"data_uri": dataURI,
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

// getResult get the result
func (m *MCP) getResult(result interface{}) (*types.ConvertResult, error) {
	if m.ResultMapping == nil {
		// Default mapping if no custom mapping provided
		return &types.ConvertResult{
			Text:     "",
			Metadata: map[string]interface{}{},
		}, nil
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

	// Default result structure
	convertResult := &types.ConvertResult{
		Text:     "",
		Metadata: map[string]interface{}{},
	}

	// Apply result mapping
	for key, template := range m.ResultMapping {
		value := helper.Bind(template, resultData)
		if value != nil {
			switch key {
			case "text":
				if textStr, ok := value.(string); ok {
					convertResult.Text = textStr
				}
			case "metadata":
				if metadataMap, ok := value.(map[string]interface{}); ok {
					convertResult.Metadata = metadataMap
				}
			default:
				// Add other mapped fields to metadata
				if convertResult.Metadata == nil {
					convertResult.Metadata = make(map[string]interface{})
				}
				convertResult.Metadata[key] = value
			}
		}
	}

	return convertResult, nil
}
