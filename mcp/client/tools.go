package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

// ListTools requests a list of available tools from the server
func (c *Client) ListTools(ctx context.Context, cursor string) (*types.ListToolsResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return nil, fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports tools
	initResult := c.GetInitResult()
	if initResult.Capabilities.Tools == nil {
		return nil, fmt.Errorf("server does not support tools")
	}

	// Create list tools request
	request := mcp.ListToolsRequest{
		PaginatedRequest: mcp.PaginatedRequest{
			Request: mcp.Request{
				Method: "tools/list",
			},
			Params: mcp.PaginatedParams{
				Cursor: mcp.Cursor(cursor),
			},
		},
	}

	// Call the MCP API
	result, err := c.MCPClient.ListTools(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert mcp-go result to our types
	tools := make([]types.Tool, len(result.Tools))
	for i, tool := range result.Tools {
		// Convert InputSchema to json.RawMessage
		schemaBytes, err := json.Marshal(tool.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tool input schema: %w", err)
		}

		tools[i] = types.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: json.RawMessage(schemaBytes),
		}
	}

	response := &types.ListToolsResponse{
		Tools:      tools,
		NextCursor: string(result.NextCursor),
	}

	return response, nil
}

// CallTool invokes a specific tool on the server
func (c *Client) CallTool(ctx context.Context, name string, arguments interface{}) (*types.CallToolResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return nil, fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports tools
	initResult := c.GetInitResult()
	if initResult.Capabilities.Tools == nil {
		return nil, fmt.Errorf("server does not support tools")
	}

	// Create call tool request
	request := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	}

	// Call the MCP API
	result, err := c.MCPClient.CallTool(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	// Convert mcp-go result to our types
	contents := make([]types.ToolContent, len(result.Content))
	for i, content := range result.Content {
		// Convert content based on its type
		if textContent, ok := mcp.AsTextContent(content); ok {
			contents[i] = types.ToolContent{
				Type: types.ToolContentTypeText,
				Text: textContent.Text,
			}
		} else if imageContent, ok := mcp.AsImageContent(content); ok {
			contents[i] = types.ToolContent{
				Type:     types.ToolContentTypeImage,
				Data:     imageContent.Data,
				MimeType: imageContent.MIMEType,
			}
		} else {
			// Handle unknown content type gracefully
			contents[i] = types.ToolContent{
				Type: types.ToolContentTypeText,
				Text: fmt.Sprintf("Unsupported content type: %v", content),
			}
		}
	}

	response := &types.CallToolResponse{
		Content: contents,
		IsError: result.IsError,
	}

	return response, nil
}

// CallTools calls multiple tools in sequence
// Tools are executed one by one, ensuring order and avoiding race conditions
func (c *Client) CallTools(ctx context.Context, tools []types.ToolCall) (*types.CallToolsResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return nil, fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports tools
	initResult := c.GetInitResult()
	if initResult.Capabilities.Tools == nil {
		return nil, fmt.Errorf("server does not support tools")
	}

	// Call each tool individually (sequential processing)
	results := make([]types.CallToolResponse, len(tools))
	for i, tool := range tools {
		result, err := c.CallTool(ctx, tool.Name, tool.Arguments)
		if err != nil {
			// Create error response for failed tool call
			results[i] = types.CallToolResponse{
				Content: []types.ToolContent{
					{
						Type: types.ToolContentTypeText,
						Text: fmt.Sprintf("Error calling tool %s: %v", tool.Name, err),
					},
				},
				IsError: true,
			}
		} else {
			results[i] = *result
		}
	}

	return &types.CallToolsResponse{
		Results: results,
	}, nil
}

// CallToolsParallel calls multiple tools concurrently
// All tools are executed in parallel for better performance
// Note: Results order matches the input order, but execution is concurrent
func (c *Client) CallToolsParallel(ctx context.Context, tools []types.ToolCall) (*types.CallToolsResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return nil, fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports tools
	initResult := c.GetInitResult()
	if initResult.Capabilities.Tools == nil {
		return nil, fmt.Errorf("server does not support tools")
	}

	// Call tools concurrently
	results := make([]types.CallToolResponse, len(tools))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, tool := range tools {
		wg.Add(1)
		go func(idx int, t types.ToolCall) {
			defer wg.Done()

			result, err := c.CallTool(ctx, t.Name, t.Arguments)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results[idx] = types.CallToolResponse{
					Content: []types.ToolContent{
						{
							Type: types.ToolContentTypeText,
							Text: fmt.Sprintf("Error calling tool %s: %v", t.Name, err),
						},
					},
					IsError: true,
				}
			} else {
				results[idx] = *result
			}
		}(i, tool)
	}

	wg.Wait()

	return &types.CallToolsResponse{
		Results: results,
	}, nil
}
