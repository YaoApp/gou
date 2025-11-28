package process

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/mcp/types"
)

// CancelRequest cancels a specific request
func (c *Client) CancelRequest(ctx context.Context, requestID interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find the cancel function for this request
	cancelFunc, exists := c.activeRequests[requestID]
	if !exists {
		return fmt.Errorf("request %v not found or already completed", requestID)
	}

	// Call the cancel function
	cancelFunc()

	// Remove from active requests
	delete(c.activeRequests, requestID)

	return nil
}

// CreateProgress creates a progress token for tracking
func (c *Client) CreateProgress(ctx context.Context, total uint64) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate a new progress token
	token := c.nextProgressToken
	c.nextProgressToken++

	// Create progress entry
	progress := &types.Progress{
		Token: token,
		Total: total,
	}

	// Store progress in the client
	c.progressTokens[token] = progress

	// TODO: Implement process-based create progress
	// This will call a Yao process like: process.New("mcp.client.progress.create", clientID, total)

	return token, nil
}

// UpdateProgress updates the progress for a given token
func (c *Client) UpdateProgress(ctx context.Context, token uint64, progress uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if progress token exists
	prog, exists := c.progressTokens[token]
	if !exists {
		return fmt.Errorf("progress token %d not found", token)
	}

	// TODO: Implement process-based update progress
	// This will call a Yao process like: process.New("mcp.client.progress.update", clientID, token, progress)

	// Remove completed progress
	if progress >= prog.Total {
		delete(c.progressTokens, token)
	}

	return nil
}

// GetProgress returns the current progress for a given token
func (c *Client) GetProgress(token uint64) (*types.Progress, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	progress, exists := c.progressTokens[token]
	if !exists {
		return nil, fmt.Errorf("progress token %d not found", token)
	}

	return progress, nil
}

// ListProgress returns all active progress tokens
func (c *Client) ListProgress() map[uint64]*types.Progress {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[uint64]*types.Progress, len(c.progressTokens))
	for token, progress := range c.progressTokens {
		result[token] = progress
	}

	return result
}

// CompleteProgress marks a progress token as complete and removes it
func (c *Client) CompleteProgress(ctx context.Context, token uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if progress token exists
	_, exists := c.progressTokens[token]
	if !exists {
		return fmt.Errorf("progress token %d not found", token)
	}

	// Remove progress token
	delete(c.progressTokens, token)

	// TODO: Implement process-based complete progress
	// This will call a Yao process like: process.New("mcp.client.progress.complete", clientID, token)

	return nil
}
