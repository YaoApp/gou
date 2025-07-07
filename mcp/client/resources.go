package client

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

// ListResources lists all available resources
func (c *Client) ListResources(ctx context.Context, cursor string) (*types.ListResourcesResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return nil, fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports resources
	initResult := c.GetInitResult()
	if initResult.Capabilities.Resources == nil {
		return nil, fmt.Errorf("server does not support resources")
	}

	// Create list resources request
	request := mcp.ListResourcesRequest{
		PaginatedRequest: mcp.PaginatedRequest{
			Request: mcp.Request{
				Method: "resources/list",
			},
			Params: mcp.PaginatedParams{
				Cursor: mcp.Cursor(cursor),
			},
		},
	}

	// Call the MCP API
	result, err := c.MCPClient.ListResources(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Convert mcp-go result to our types
	resources := make([]types.Resource, len(result.Resources))
	for i, resource := range result.Resources {
		resources[i] = types.Resource{
			URI:         resource.URI,
			Name:        resource.Name,
			Description: resource.Description,
			MimeType:    resource.MIMEType,
		}
	}

	response := &types.ListResourcesResponse{
		Resources:  resources,
		NextCursor: string(result.NextCursor),
	}

	return response, nil
}

// ReadResource reads a specific resource
func (c *Client) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return nil, fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports resources
	initResult := c.GetInitResult()
	if initResult.Capabilities.Resources == nil {
		return nil, fmt.Errorf("server does not support resources")
	}

	// Create read resource request
	request := mcp.ReadResourceRequest{
		Request: mcp.Request{
			Method: "resources/read",
		},
		Params: mcp.ReadResourceParams{
			URI: uri,
		},
	}

	// Call the MCP API
	result, err := c.MCPClient.ReadResource(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}

	// Convert mcp-go result to our types
	contents := make([]types.ResourceContent, len(result.Contents))
	for i, content := range result.Contents {
		// Check the type of content and convert accordingly
		if textContent, ok := mcp.AsTextResourceContents(content); ok {
			contents[i] = types.ResourceContent{
				URI:      textContent.URI,
				MimeType: textContent.MIMEType,
				Text:     textContent.Text,
			}
		} else if blobContent, ok := mcp.AsBlobResourceContents(content); ok {
			// Decode base64 string to []byte
			blobData, err := base64.StdEncoding.DecodeString(blobContent.Blob)
			if err != nil {
				return nil, fmt.Errorf("failed to decode blob content: %w", err)
			}
			contents[i] = types.ResourceContent{
				URI:      blobContent.URI,
				MimeType: blobContent.MIMEType,
				Blob:     blobData,
			}
		} else {
			return nil, fmt.Errorf("unsupported resource content type")
		}
	}

	response := &types.ReadResourceResponse{
		Contents: contents,
	}

	return response, nil
}

// SubscribeResource subscribes to resource updates
func (c *Client) SubscribeResource(ctx context.Context, uri string) error {
	if c.MCPClient == nil {
		return fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports resource subscriptions
	initResult := c.GetInitResult()
	if initResult.Capabilities.Resources == nil {
		return fmt.Errorf("server does not support resources")
	}

	if !initResult.Capabilities.Resources.Subscribe {
		return fmt.Errorf("server does not support resource subscriptions")
	}

	// Create subscribe request
	request := mcp.SubscribeRequest{
		Request: mcp.Request{
			Method: "resources/subscribe",
		},
		Params: mcp.SubscribeParams{
			URI: uri,
		},
	}

	// Call the MCP API
	err := c.MCPClient.Subscribe(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to subscribe to resource: %w", err)
	}

	return nil
}

// UnsubscribeResource unsubscribes from resource updates
func (c *Client) UnsubscribeResource(ctx context.Context, uri string) error {
	if c.MCPClient == nil {
		return fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// Check if server supports resource subscriptions
	initResult := c.GetInitResult()
	if initResult.Capabilities.Resources == nil {
		return fmt.Errorf("server does not support resources")
	}

	if !initResult.Capabilities.Resources.Subscribe {
		return fmt.Errorf("server does not support resource subscriptions")
	}

	// Create unsubscribe request
	request := mcp.UnsubscribeRequest{
		Request: mcp.Request{
			Method: "resources/unsubscribe",
		},
		Params: mcp.UnsubscribeParams{
			URI: uri,
		},
	}

	// Call the MCP API
	err := c.MCPClient.Unsubscribe(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from resource: %w", err)
	}

	return nil
}
