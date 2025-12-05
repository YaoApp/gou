package process

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/gou/process"
)

// ListTools requests a list of available tools from the server
func (c *Client) ListTools(ctx context.Context, cursor string) (*types.ListToolsResponse, error) {
	// Get mapping data from registry
	mapping, ok := GetMapping(c.DSL.ID)
	if !ok {
		return &types.ListToolsResponse{
			Tools: []types.Tool{},
		}, nil
	}

	// Convert ToolSchema to Tool
	tools := make([]types.Tool, 0, len(mapping.Tools))
	for _, toolSchema := range mapping.Tools {
		tool := types.Tool{
			Name:        toolSchema.Name,
			Description: toolSchema.Description,
			InputSchema: toolSchema.InputSchema,
		}
		tools = append(tools, tool)
	}

	return &types.ListToolsResponse{
		Tools: tools,
	}, nil
}

// CallTool invokes a specific tool by calling the mapped Yao process
// extraArgs are optional additional arguments that will be appended to the process call
func (c *Client) CallTool(ctx context.Context, name string, arguments interface{}, extraArgs ...interface{}) (*types.CallToolResponse, error) {
	// Get mapping data from registry
	mapping, ok := GetMapping(c.DSL.ID)
	if !ok {
		return nil, fmt.Errorf("no mapping found for client: %s", c.DSL.ID)
	}

	// Find the tool by name
	toolSchema, ok := mapping.Tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Generate request ID and create cancellable context
	c.mu.Lock()
	requestID := c.nextRequestID
	c.nextRequestID++
	c.mu.Unlock()

	// Create cancellable context
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register the cancel function
	c.mu.Lock()
	c.activeRequests[requestID] = cancel
	c.mu.Unlock()

	// Ensure cleanup
	defer func() {
		c.mu.Lock()
		delete(c.activeRequests, requestID)
		c.mu.Unlock()
	}()

	// Extract process arguments based on x-process-args mapping
	// extraArgs will be appended after the mapped arguments
	procArgs, err := extractProcessArgs(toolSchema.ProcessArgs, arguments, extraArgs...)
	if err != nil {
		return &types.CallToolResponse{
			Content: []types.ToolContent{
				{
					Type: types.ToolContentTypeText,
					Text: fmt.Sprintf("Error extracting arguments for tool %s: %v", name, err),
				},
			},
			IsError: true,
		}, nil
	}

	// Call the mapped Yao process with extracted arguments
	proc := process.New(toolSchema.Process, procArgs...).WithContext(ctxWithCancel)
	err = proc.Execute()
	if err != nil {
		// Check if error is due to cancellation
		if ctxWithCancel.Err() == context.Canceled {
			return &types.CallToolResponse{
				Content: []types.ToolContent{
					{
						Type: types.ToolContentTypeText,
						Text: fmt.Sprintf("Tool call %s was cancelled", name),
					},
				},
				IsError: true,
			}, nil
		}

		// Return error as ToolContent
		return &types.CallToolResponse{
			Content: []types.ToolContent{
				{
					Type: types.ToolContentTypeText,
					Text: fmt.Sprintf("Error executing tool %s: %v", name, err),
				},
			},
			IsError: true,
		}, nil // Return nil error, error info in Content
	}
	defer proc.Release()

	// Get the result
	result := proc.Value()

	// Convert result to ToolContent
	content := convertToToolContent(result)

	return &types.CallToolResponse{
		Content: content,
		IsError: false,
	}, nil
}

// CallTools calls multiple tools in sequence
// Tools are executed one by one, ensuring order and avoiding race conditions
// extraArgs are optional additional arguments that will be appended to each tool's process call
func (c *Client) CallTools(ctx context.Context, tools []types.ToolCall, extraArgs ...interface{}) (*types.CallToolsResponse, error) {
	results := make([]types.CallToolResponse, len(tools))
	for i, tool := range tools {
		result, err := c.CallTool(ctx, tool.Name, tool.Arguments, extraArgs...)
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
// extraArgs are optional additional arguments that will be appended to each tool's process call
func (c *Client) CallToolsParallel(ctx context.Context, tools []types.ToolCall, extraArgs ...interface{}) (*types.CallToolsResponse, error) {
	results := make([]types.CallToolResponse, len(tools))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, tool := range tools {
		wg.Add(1)
		go func(idx int, t types.ToolCall) {
			defer wg.Done()

			result, err := c.CallTool(ctx, t.Name, t.Arguments, extraArgs...)

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

// convertToToolContent converts a process result to ToolContent array
func convertToToolContent(result interface{}) []types.ToolContent {
	if result == nil {
		return []types.ToolContent{
			{
				Type: types.ToolContentTypeText,
				Text: "",
			},
		}
	}

	// If it's already a string, return as text content
	if str, ok := result.(string); ok {
		return []types.ToolContent{
			{
				Type: types.ToolContentTypeText,
				Text: str,
			},
		}
	}

	// If it's a slice, check if it's already ToolContent[]
	if contents, ok := result.([]types.ToolContent); ok {
		return contents
	}

	// Otherwise, marshal to JSON and return as text
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return []types.ToolContent{
			{
				Type: types.ToolContentTypeText,
				Text: fmt.Sprintf("%v", result),
			},
		}
	}

	return []types.ToolContent{
		{
			Type: types.ToolContentTypeText,
			Text: string(jsonBytes),
		},
	}
}
