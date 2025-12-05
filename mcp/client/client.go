package client

import (
	"fmt"

	goclient "github.com/mark3labs/mcp-go/client"

	"github.com/yaoapp/gou/mcp/types"
	gouTypes "github.com/yaoapp/gou/types"
)

// Client the MCP Client
type Client struct {
	DSL        *types.ClientDSL
	MCPClient  *goclient.Client
	InitResult *types.InitializeResponse // Store the initialization result

	// Additional client state
	currentLogLevel      types.LogLevel
	progressTokens       map[uint64]*types.Progress
	eventHandlers        map[string][]func(event types.Event)
	notificationHandlers map[string][]types.NotificationHandler
	errorHandlers        []types.ErrorHandler
	nextProgressToken    uint64
}

// Info returns basic client information
func (c *Client) Info() *types.ClientInfo {
	if c.DSL == nil {
		return &types.ClientInfo{}
	}
	return &types.ClientInfo{
		ID:          c.DSL.ID,
		Name:        c.DSL.Name,
		Version:     c.DSL.Version,
		Type:        c.DSL.Type,
		Transport:   c.DSL.Transport,
		Label:       c.DSL.Label,
		Description: c.DSL.Description,
	}
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

		// Initialize additional fields
		currentLogLevel:      types.LogLevelInfo,
		progressTokens:       make(map[uint64]*types.Progress),
		eventHandlers:        make(map[string][]func(event types.Event)),
		notificationHandlers: make(map[string][]types.NotificationHandler),
		errorHandlers:        []types.ErrorHandler{},
		nextProgressToken:    1,
	}

	return client, nil
}

// GetMetaInfo returns the meta information of the client
func (c *Client) GetMetaInfo() gouTypes.MetaInfo {
	return c.DSL.MetaInfo
}
