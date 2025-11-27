package process

import (
	"context"

	"github.com/yaoapp/gou/mcp/types"
)

// ListResources lists all available resources
func (c *Client) ListResources(ctx context.Context, cursor string) (*types.ListResourcesResponse, error) {
	// TODO: Implement process-based list resources
	// This will call a Yao process like: process.New("mcp.client.resources.list", clientID, cursor)
	return nil, nil
}

// ReadResource reads a specific resource
func (c *Client) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResponse, error) {
	// TODO: Implement process-based read resource
	// This will call a Yao process like: process.New("mcp.client.resources.read", clientID, uri)
	return nil, nil
}

// SubscribeResource subscribes to resource updates
func (c *Client) SubscribeResource(ctx context.Context, uri string) error {
	// TODO: Implement process-based subscribe resource
	// This will call a Yao process like: process.New("mcp.client.resources.subscribe", clientID, uri)
	return nil
}

// UnsubscribeResource unsubscribes from resource updates
func (c *Client) UnsubscribeResource(ctx context.Context, uri string) error {
	// TODO: Implement process-based unsubscribe resource
	// This will call a Yao process like: process.New("mcp.client.resources.unsubscribe", clientID, uri)
	return nil
}
