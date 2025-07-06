package converter

import (
	"context"
	"io"
	"os"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/mcp"
	mcpTypes "github.com/yaoapp/gou/mcp/types"
)

// MCPClient the mcp client
type MCPClient struct {
	ID                  string            `json:"id"`                             // ID the mcp id
	Tool                string            `json:"tool"`                           // Tool the tool name
	Arguments           []interface{}     `json:"arguments,omitempty"`            // Arguments the tool arguments
	ResultMapping       map[string]string `json:"result_mapping,omitempty"`       // ResultMapping the result mapping
	NotificationMapping map[string]string `json:"notification_mapping,omitempty"` // NotificationMapping the notification mapping
	client              mcp.Client
}

// MCPOptions the mcp options
type MCPOptions struct {
	ID                  string            `json:"id"`                             // ID the mcp id
	Tool                string            `json:"tool"`                           // Tool the tool name
	Arguments           []interface{}     `json:"arguments,omitempty"`            // Arguments the tool arguments
	NotificationMapping map[string]string `json:"notification_mapping,omitempty"` // NotificationMapping the notification mapping
	ResultMapping       map[string]string `json:"result_mapping,omitempty"`       // ResultMapping the result mapping
}

// NewMCPClient convert the file to plain text using the mcp client
func NewMCPClient(options *MCPOptions) *MCPClient {
	client, err := mcp.Select(options.ID)
	if err != nil {
		return nil
	}
	return &MCPClient{client: client, ID: options.ID, Tool: options.Tool, Arguments: options.Arguments, NotificationMapping: options.NotificationMapping, ResultMapping: options.ResultMapping}
}

// Convert convert the file to plain text
func (c *MCPClient) Convert(ctx context.Context, file string, callback ...types.ConverterProgress) (*types.ConvertResult, error) {

	args := make([]interface{}, len(c.Arguments)+1)
	args[0] = file
	copy(args[1:], c.Arguments)

	// Notify the progress
	if len(callback) > 0 {
		c.client.OnNotification("progress", func(ctx context.Context, notification mcpTypes.Message) error {
			callback[0]("pending", types.ConverterPayload{})
			return nil
		})
	}

	// Call the tool
	response, err := c.client.CallTool(ctx, c.Tool, args)
	if err != nil {
		return nil, err
	}

	// Parse the response
	result := &types.ConvertResult{}
	for _, content := range response.Content {
		if content.Type == "text" {
			result.Text += content.Text
		}
	}

	return result, nil
}

// ConvertStream convert the stream to plain text
func (c *MCPClient) ConvertStream(ctx context.Context, stream io.ReadSeeker, callback ...types.ConverterProgress) (*types.ConvertResult, error) {

	// Save the file to a temporary file
	tempFile, err := os.CreateTemp("", "gou-graphrag-converter-*.txt")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempFile.Name())

	// Write the stream to the temporary file
	_, err = io.Copy(tempFile, stream)
	if err != nil {
		return nil, err
	}

	return c.Convert(ctx, tempFile.Name())
}
