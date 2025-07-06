package client

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

// Initialize initializes the MCP client and exchanges capabilities with the server
func (c *Client) Initialize(ctx context.Context) (*types.InitializeResponse, error) {
	if c.MCPClient == nil {
		return nil, fmt.Errorf("MCP client not connected - call Connect() first")
	}

	// Get our client capabilities and implementation
	clientImpl := c.DSL.GetImplementation()
	clientCaps := c.DSL.GetClientCapabilities()

	// Create initialize request with proper mcp-go types
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: types.ProtocolVersion,
			Capabilities:    convertClientCapabilities(clientCaps),
			ClientInfo: mcp.Implementation{
				Name:    clientImpl.Name,
				Version: clientImpl.Version,
			},
		},
	}

	// Call the mcp-go Initialize method
	result, err := c.MCPClient.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// Convert mcp-go result to our types
	response := &types.InitializeResponse{
		ProtocolVersion: result.ProtocolVersion,
		ServerInfo: types.ServerInfo{
			Name:    result.ServerInfo.Name,
			Version: result.ServerInfo.Version,
		},
		Capabilities: convertServerCapabilities(result.Capabilities),
	}

	return response, nil
}

// Initialized sends the initialized notification to the server
// Note: In mcp-go v0.32.0, this is typically handled automatically after Initialize
func (c *Client) Initialized(ctx context.Context) error {
	if c.MCPClient == nil {
		return fmt.Errorf("MCP client not connected")
	}

	// In mcp-go v0.32.0, the initialized notification is typically sent automatically
	// after successful initialization. This method is a no-op for compatibility.
	return nil
}

// convertServerCapabilities converts mcp.ServerCapabilities to our types.ServerCapabilities
func convertServerCapabilities(caps mcp.ServerCapabilities) types.ServerCapabilities {
	result := types.ServerCapabilities{
		Experimental: caps.Experimental,
	}

	// Convert Resources capability
	if caps.Resources != nil {
		result.Resources = &types.ResourcesCapability{
			Subscribe:   caps.Resources.Subscribe,
			ListChanged: caps.Resources.ListChanged,
		}
	}

	// Convert Tools capability
	if caps.Tools != nil {
		result.Tools = &types.ToolsCapability{
			ListChanged: caps.Tools.ListChanged,
		}
	}

	// Convert Prompts capability
	if caps.Prompts != nil {
		result.Prompts = &types.PromptsCapability{
			ListChanged: caps.Prompts.ListChanged,
		}
	}

	// Convert Logging capability
	if caps.Logging != nil {
		result.Logging = &types.LoggingCapability{}
	}

	return result
}

// convertClientCapabilities converts our types.ClientCapabilities to mcp.ClientCapabilities
func convertClientCapabilities(caps types.ClientCapabilities) mcp.ClientCapabilities {
	result := mcp.ClientCapabilities{
		Experimental: make(map[string]any),
	}

	// Convert Sampling capability
	if caps.Sampling != nil {
		result.Sampling = &struct{}{}
	}

	// Convert Roots capability
	if caps.Roots != nil {
		result.Roots = &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{
			ListChanged: caps.Roots.ListChanged,
		}
	}

	// Copy experimental capabilities
	if caps.Experimental != nil {
		result.Experimental = caps.Experimental
	}

	return result
}
