package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/yaoapp/gou/mcp/types"
)

// progressMutex protects progress-related operations
var progressMutex sync.RWMutex

// CancelRequest cancels a specific request
func (c *Client) CancelRequest(ctx context.Context, requestID interface{}) error {
	if c.MCPClient == nil {
		return fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	// TODO: Implement actual request cancellation when mcp-go API is available
	// This would typically involve sending a JSON-RPC notification to the server

	return nil
}

// CreateProgress creates a progress token for tracking
func (c *Client) CreateProgress(ctx context.Context, total uint64) (uint64, error) {
	if c.MCPClient == nil {
		return 0, fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return 0, fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	progressMutex.Lock()
	defer progressMutex.Unlock()

	// Initialize map if it's nil
	if c.progressTokens == nil {
		c.progressTokens = make(map[uint64]*types.Progress)
	}

	// Initialize nextProgressToken if it's 0
	if c.nextProgressToken == 0 {
		c.nextProgressToken = 1
	}

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

	// TODO: Implement actual progress creation when mcp-go API is available
	// This would typically involve sending a progress notification to the server

	return token, nil
}

// UpdateProgress updates the progress for a given token
func (c *Client) UpdateProgress(ctx context.Context, token uint64, progress uint64) error {
	if c.MCPClient == nil {
		return fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	progressMutex.Lock()
	defer progressMutex.Unlock()

	// Initialize map if it's nil
	if c.progressTokens == nil {
		c.progressTokens = make(map[uint64]*types.Progress)
	}

	// Check if progress token exists
	prog, exists := c.progressTokens[token]
	if !exists {
		return fmt.Errorf("progress token %d not found", token)
	}

	// Update progress
	// Note: We don't store the current progress in the Progress struct
	// as it's not part of the original type definition

	// TODO: Implement actual progress updating when mcp-go API is available
	// This would typically involve sending a progress notification to the server

	// For now, we simulate by checking if progress is complete
	if progress >= prog.Total {
		// Remove completed progress
		delete(c.progressTokens, token)
	}

	return nil
}

// GetProgress returns the current progress for a given token
func (c *Client) GetProgress(token uint64) (*types.Progress, error) {
	progressMutex.RLock()
	defer progressMutex.RUnlock()

	// Initialize map if it's nil
	if c.progressTokens == nil {
		return nil, fmt.Errorf("progress token %d not found", token)
	}

	progress, exists := c.progressTokens[token]
	if !exists {
		return nil, fmt.Errorf("progress token %d not found", token)
	}

	return progress, nil
}

// ListProgress returns all active progress tokens
func (c *Client) ListProgress() map[uint64]*types.Progress {
	progressMutex.RLock()
	defer progressMutex.RUnlock()

	// Initialize map if it's nil
	if c.progressTokens == nil {
		return make(map[uint64]*types.Progress)
	}

	result := make(map[uint64]*types.Progress, len(c.progressTokens))
	for token, progress := range c.progressTokens {
		result[token] = progress
	}

	return result
}

// CompleteProgress marks a progress token as complete and removes it
func (c *Client) CompleteProgress(ctx context.Context, token uint64) error {
	if c.MCPClient == nil {
		return fmt.Errorf("MCP client not initialized")
	}

	if !c.IsInitialized() {
		return fmt.Errorf("MCP client not initialized - call Initialize() first")
	}

	progressMutex.Lock()
	defer progressMutex.Unlock()

	// Initialize map if it's nil
	if c.progressTokens == nil {
		c.progressTokens = make(map[uint64]*types.Progress)
	}

	// Check if progress token exists
	_, exists := c.progressTokens[token]
	if !exists {
		return fmt.Errorf("progress token %d not found", token)
	}

	// Remove progress token
	delete(c.progressTokens, token)

	// TODO: Implement actual progress completion when mcp-go API is available
	// This would typically involve sending a progress completion notification to the server

	return nil
}
