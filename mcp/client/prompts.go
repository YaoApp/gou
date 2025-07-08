package client

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

// ListPrompts lists all available prompts
func (c *Client) ListPrompts(ctx context.Context, cursor string) (*types.ListPromptsResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return nil, fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports prompts
	initResult := c.GetInitResult()
	if initResult.Capabilities.Prompts == nil {
		return nil, fmt.Errorf("server does not support prompts")
	}

	// Create list prompts request
	request := mcp.ListPromptsRequest{
		PaginatedRequest: mcp.PaginatedRequest{
			Request: mcp.Request{
				Method: "prompts/list",
			},
			Params: mcp.PaginatedParams{
				Cursor: mcp.Cursor(cursor),
			},
		},
	}

	// Call the MCP API
	result, err := c.MCPClient.ListPrompts(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	// Convert mcp-go result to our types
	prompts := make([]types.Prompt, len(result.Prompts))
	for i, prompt := range result.Prompts {
		// Convert arguments
		arguments := make([]types.PromptArgument, len(prompt.Arguments))
		for j, arg := range prompt.Arguments {
			arguments[j] = types.PromptArgument{
				Name:        arg.Name,
				Description: arg.Description,
				Required:    arg.Required,
			}
		}

		prompts[i] = types.Prompt{
			Name:        prompt.Name,
			Description: prompt.Description,
			Arguments:   arguments,
		}
	}

	response := &types.ListPromptsResponse{
		Prompts:    prompts,
		NextCursor: string(result.NextCursor),
	}

	return response, nil
}

// GetPrompt gets a specific prompt with given arguments
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*types.GetPromptResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return nil, fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports prompts
	initResult := c.GetInitResult()
	if initResult.Capabilities.Prompts == nil {
		return nil, fmt.Errorf("server does not support prompts")
	}

	// Convert arguments to string map
	stringArgs := make(map[string]string)
	for k, v := range arguments {
		stringArgs[k] = fmt.Sprintf("%v", v)
	}

	// Create get prompt request
	request := mcp.GetPromptRequest{
		Request: mcp.Request{
			Method: "prompts/get",
		},
		Params: mcp.GetPromptParams{
			Name:      name,
			Arguments: stringArgs,
		},
	}

	// Call the MCP API
	result, err := c.MCPClient.GetPrompt(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	// Convert mcp-go result to our types
	messages := make([]types.PromptMessage, len(result.Messages))
	for i, message := range result.Messages {
		// Convert role
		var role types.PromptRole
		if string(message.Role) == "user" {
			role = types.PromptRoleUser
		} else if string(message.Role) == "assistant" {
			role = types.PromptRoleAssistant
		} else {
			role = types.PromptRoleUser // default fallback
		}

		// Convert content based on its type
		var content types.PromptContent

		if textContent, ok := mcp.AsTextContent(message.Content); ok {
			content = types.PromptContent{
				Type: types.PromptContentTypeText,
				Text: textContent.Text,
			}
		} else if imageContent, ok := mcp.AsImageContent(message.Content); ok {
			content = types.PromptContent{
				Type:     types.PromptContentTypeImage,
				Data:     imageContent.Data,
				MimeType: imageContent.MIMEType,
			}
		} else {
			// Handle unknown content type gracefully
			content = types.PromptContent{
				Type: types.PromptContentTypeText,
				Text: fmt.Sprintf("Unsupported content type: %v", message.Content),
			}
		}

		messages[i] = types.PromptMessage{
			Role:    role,
			Content: content,
		}
	}

	response := &types.GetPromptResponse{
		Description: result.Description,
		Messages:    messages,
	}

	return response, nil
}
