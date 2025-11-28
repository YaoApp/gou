package process

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/mcp/types"
)

// ListPrompts lists all available prompts
func (c *Client) ListPrompts(ctx context.Context, cursor string) (*types.ListPromptsResponse, error) {
	// Get mapping data from registry
	mapping, ok := GetMapping(c.DSL.ID)
	if !ok {
		return &types.ListPromptsResponse{
			Prompts: []types.Prompt{},
		}, nil
	}

	// Convert PromptSchema to Prompt
	prompts := make([]types.Prompt, 0, len(mapping.Prompts))
	for _, promptSchema := range mapping.Prompts {
		prompt := types.Prompt{
			Name:        promptSchema.Name,
			Description: promptSchema.Description,
			Arguments:   promptSchema.Arguments,
		}
		prompts = append(prompts, prompt)
	}

	return &types.ListPromptsResponse{
		Prompts: prompts,
	}, nil
}

// GetPrompt gets a specific prompt with given arguments and renders the template
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*types.GetPromptResponse, error) {
	// Get mapping data from registry
	mapping, ok := GetMapping(c.DSL.ID)
	if !ok {
		return nil, fmt.Errorf("no mapping found for client: %s", c.DSL.ID)
	}

	// Find the prompt by name
	var promptSchema *types.PromptSchema
	for _, p := range mapping.Prompts {
		if p.Name == name {
			promptSchema = p
			break
		}
	}

	if promptSchema == nil {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}

	// Simple template rendering: replace {{variable}} with values from arguments
	template := promptSchema.Template
	for key, value := range arguments {
		placeholder := fmt.Sprintf("{{%s}}", key)
		var strValue string
		if str, ok := value.(string); ok {
			strValue = str
		} else {
			strValue = fmt.Sprintf("%v", value)
		}
		template = strings.ReplaceAll(template, placeholder, strValue)
	}

	// Create the prompt message
	message := types.PromptMessage{
		Role: types.PromptRoleUser,
		Content: types.PromptContent{
			Type: types.PromptContentTypeText,
			Text: template,
		},
	}

	return &types.GetPromptResponse{
		Description: promptSchema.Description,
		Messages:    []types.PromptMessage{message},
	}, nil
}
