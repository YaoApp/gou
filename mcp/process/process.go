package process

import (
	"context"
	"sync"

	"github.com/yaoapp/gou/mcp/types"
	gouTypes "github.com/yaoapp/gou/types"
)

// Client implements MCP Client interface using Yao Process calls
type Client struct {
	DSL        *types.ClientDSL
	InitResult *types.InitializeResponse

	// Client state
	connected            bool
	currentLogLevel      types.LogLevel
	progressTokens       map[uint64]*types.Progress
	eventHandlers        map[string][]func(event types.Event)
	notificationHandlers map[string][]types.NotificationHandler
	errorHandlers        []types.ErrorHandler
	nextProgressToken    uint64

	// Request cancellation support
	activeRequests map[interface{}]context.CancelFunc
	nextRequestID  uint64

	// Mutex for thread safety
	mu sync.RWMutex
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

// New creates a new Process-based MCP Client
func New(dsl *types.ClientDSL) (*Client, error) {
	if dsl == nil {
		return nil, nil
	}

	client := &Client{
		DSL:                  dsl,
		InitResult:           nil,
		connected:            false,
		currentLogLevel:      types.LogLevelInfo,
		progressTokens:       make(map[uint64]*types.Progress),
		eventHandlers:        make(map[string][]func(event types.Event)),
		notificationHandlers: make(map[string][]types.NotificationHandler),
		errorHandlers:        []types.ErrorHandler{},
		nextProgressToken:    1,
		activeRequests:       make(map[interface{}]context.CancelFunc),
		nextRequestID:        1,
	}

	return client, nil
}

// GetMetaInfo returns the meta information of the client
func (c *Client) GetMetaInfo() gouTypes.MetaInfo {
	if c.DSL == nil {
		return gouTypes.MetaInfo{}
	}
	return c.DSL.MetaInfo
}
