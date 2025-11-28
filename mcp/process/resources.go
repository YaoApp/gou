package process

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/gou/process"
)

// ListResources lists all available resources
func (c *Client) ListResources(ctx context.Context, cursor string) (*types.ListResourcesResponse, error) {
	// Get mapping data from registry
	mapping, ok := GetMapping(c.DSL.ID)
	if !ok {
		return &types.ListResourcesResponse{
			Resources: []types.Resource{},
		}, nil
	}

	// Convert ResourceSchema to Resource
	resources := make([]types.Resource, 0, len(mapping.Resources))
	for _, resSchema := range mapping.Resources {
		resource := types.Resource{
			URI:         resSchema.URI,
			Name:        resSchema.Name,
			Description: resSchema.Description,
			MimeType:    resSchema.MimeType,
			Meta:        resSchema.Meta,
		}
		resources = append(resources, resource)
	}

	return &types.ListResourcesResponse{
		Resources: resources,
	}, nil
}

// ReadResource reads a specific resource by calling the mapped Yao process
func (c *Client) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResponse, error) {
	// Get mapping data from registry
	mapping, ok := GetMapping(c.DSL.ID)
	if !ok {
		return nil, fmt.Errorf("no mapping found for client: %s", c.DSL.ID)
	}

	// Find the resource by URI (exact match or template match)
	var resourceSchema *types.ResourceSchema
	var uriParams map[string]interface{}

	for _, res := range mapping.Resources {
		// Try exact match first
		if res.URI == uri {
			resourceSchema = res
			uriParams = make(map[string]interface{})
			break
		}

		// Try template match if URI contains {param}
		if strings.Contains(res.URI, "{") {
			params, err := extractURIParams(res.URI, uri)
			if err == nil && params != nil {
				resourceSchema = res
				// Convert string map to interface{} map
				uriParams = make(map[string]interface{}, len(params))
				for k, v := range params {
					uriParams[k] = v
				}
				break
			}
		}
	}

	if resourceSchema == nil {
		return nil, fmt.Errorf("resource not found: %s", uri)
	}

	// Extract query parameters from URI (if any)
	queryParams := make(map[string]interface{})
	if idx := strings.Index(uri, "?"); idx != -1 {
		queryStr := uri[idx+1:]
		for _, pair := range strings.Split(queryStr, "&") {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				queryParams[kv[0]] = kv[1]
			}
		}
	}

	// Merge URI params and query params
	allParams := make(map[string]interface{})
	for k, v := range uriParams {
		allParams[k] = v
	}
	for k, v := range queryParams {
		allParams[k] = v
	}

	// Extract process arguments based on x-process-args mapping
	procArgs, err := extractResourceArgs(resourceSchema.ProcessArgs, uri, resourceSchema.URI, allParams)
	if err != nil {
		return nil, fmt.Errorf("failed to extract resource arguments: %w", err)
	}

	// Call the mapped Yao process with extracted arguments
	proc := process.New(resourceSchema.Process, procArgs...).WithContext(ctx)
	err = proc.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to execute process %s: %w", resourceSchema.Process, err)
	}
	defer proc.Release()

	// Get the result
	result := proc.Value()

	// Convert result to ResourceContent
	// For now, assume the result is text/json
	var text string
	if str, ok := result.(string); ok {
		text = str
	} else {
		// Try to marshal to JSON if not a string
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource content: %w", err)
		}
		text = string(jsonBytes)
	}

	content := types.ResourceContent{
		URI:      uri,
		MimeType: resourceSchema.MimeType,
		Text:     text,
	}

	return &types.ReadResourceResponse{
		Contents: []types.ResourceContent{content},
	}, nil
}

// SubscribeResource is not supported in process transport
func (c *Client) SubscribeResource(ctx context.Context, uri string) error {
	return fmt.Errorf("resource subscription not supported in process transport")
}

// UnsubscribeResource is not supported in process transport
func (c *Client) UnsubscribeResource(ctx context.Context, uri string) error {
	return fmt.Errorf("resource unsubscription not supported in process transport")
}
