package process

import (
	"context"

	"github.com/yaoapp/gou/mcp/types"
)

// CreateSampling creates a sampling request if supported by the server
func (c *Client) CreateSampling(ctx context.Context, request types.SamplingRequest) (*types.SamplingResponse, error) {
	// TODO: Implement process-based create sampling
	// This will call a Yao process like: process.New("mcp.client.sampling.create", clientID, request)
	return nil, nil
}
