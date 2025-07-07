package client

import (
	"fmt"

	goclient "github.com/mark3labs/mcp-go/client"

	"github.com/yaoapp/gou/mcp/types"
)

// Client the MCP Client
type Client struct {
	DSL        *types.ClientDSL
	MCPClient  *goclient.Client
	InitResult *types.InitializeResponse // Store the initialization result
}

// New create a new MCP Client (without establishing connection)
func New(dsl *types.ClientDSL) (*Client, error) {
	// Validate DSL
	if dsl == nil {
		return nil, fmt.Errorf("DSL cannot be nil")
	}

	if dsl.Name == "" {
		return nil, fmt.Errorf("client name is required")
	}

	// Validate transport-specific requirements
	switch dsl.Transport {
	case types.TransportStdio:
		if dsl.Command == "" {
			return nil, fmt.Errorf("command is required for stdio transport")
		}
	case types.TransportSSE, types.TransportHTTP:
		if dsl.URL == "" {
			return nil, fmt.Errorf("URL is required for %s transport", dsl.Transport)
		}
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", dsl.Transport)
	}

	// Create client without establishing connection
	client := &Client{
		DSL:        dsl,
		MCPClient:  nil, // Will be created when Connect() is called
		InitResult: nil, // Will be set when Initialize() is called
	}

	return client, nil
}
