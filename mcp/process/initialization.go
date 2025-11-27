package process

import (
	"context"
	"sync"

	"github.com/yaoapp/gou/mcp/types"
)

// Global mapping registry for all process-based MCP clients
var (
	mappingRegistry     = make(map[string]*types.MappingData)
	mappingRegistryLock sync.RWMutex
)

// SetMapping sets the mapping data for a specific client
func SetMapping(clientID string, mapping *types.MappingData) {
	mappingRegistryLock.Lock()
	defer mappingRegistryLock.Unlock()
	mappingRegistry[clientID] = mapping
}

// GetMapping gets the mapping data for a specific client
func GetMapping(clientID string) (*types.MappingData, bool) {
	mappingRegistryLock.RLock()
	defer mappingRegistryLock.RUnlock()
	mapping, exists := mappingRegistry[clientID]
	return mapping, exists
}

// RemoveMapping removes the mapping data for a specific client
func RemoveMapping(clientID string) {
	mappingRegistryLock.Lock()
	defer mappingRegistryLock.Unlock()
	delete(mappingRegistry, clientID)
}

// Initialize initializes the MCP client and exchanges capabilities with the server
func (c *Client) Initialize(ctx context.Context) (*types.InitializeResponse, error) {
	// TODO: Implement process-based initialization
	// This will call a Yao process like: process.New("mcp.client.initialize", clientID)
	return nil, nil
}

// Initialized sends the initialized notification to the server
func (c *Client) Initialized(ctx context.Context) error {
	// TODO: Implement process-based initialized notification
	// This will call a Yao process like: process.New("mcp.client.initialized", clientID)
	return nil
}
