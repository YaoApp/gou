package client

import (
	"context"
	"fmt"
	"strings"

	gohttp "net/http"

	goclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/gou/mcp/types"
)

// Connect establishes connection to the MCP server
func (c *Client) Connect(ctx context.Context, options ...types.ConnectionOptions) error {
	if c.DSL == nil {
		return fmt.Errorf("DSL configuration is nil")
	}

	// If already connected, return success
	if c.MCPClient != nil {
		return nil
	}

	// Merge connection options
	var opts types.ConnectionOptions
	if len(options) > 0 {
		opts = options[0]
	}

	// Create connection based on transport type
	switch c.DSL.Transport {
	case types.TransportStdio:
		return c.connectStdio()

	case types.TransportSSE:
		return c.connectSSE(opts)

	case types.TransportHTTP:
		return c.connectHTTP(opts)

	default:
		return fmt.Errorf("unsupported transport type: %s", c.DSL.Transport)
	}
}

// connectStdio creates a stdio connection
func (c *Client) connectStdio() error {
	client, err := goclient.NewStdioMCPClient(c.DSL.Command, c.DSL.GetEnvs(), c.DSL.Arguments...)
	if err != nil {
		return fmt.Errorf("failed to create stdio client: %w", err)
	}

	c.MCPClient = client
	return nil
}

// connectSSE creates a SSE connection
func (c *Client) connectSSE(opts types.ConnectionOptions) error {
	// Get Authorization Token
	authorizationToken := c.DSL.GetAuthorizationToken()

	// Prepare headers
	headers := make(map[string]string)
	if authorizationToken != "" {
		headers["Authorization"] = authorizationToken
	}

	// Merge custom headers from options
	for key, value := range opts.Headers {
		headers[key] = value
	}

	https := strings.HasPrefix(c.DSL.URL, "https://")
	proxy := http.GetProxy(https)
	tr := http.GetTransport(https, proxy)
	httpClient := &gohttp.Client{Transport: tr}

	sseTransport, err := transport.NewSSE(c.DSL.URL,
		transport.WithHeaders(headers),
		transport.WithHTTPClient(httpClient),
	)
	if err != nil {
		return fmt.Errorf("failed to create SSE transport: %w", err)
	}

	c.MCPClient = goclient.NewClient(sseTransport)
	return nil
}

// connectHTTP creates a HTTP connection
func (c *Client) connectHTTP(opts types.ConnectionOptions) error {
	// Get Authorization Token
	authorizationToken := c.DSL.GetAuthorizationToken()

	// Timeout
	timeout := c.DSL.GetTimeout()

	// Prepare headers
	headers := make(map[string]string)
	if authorizationToken != "" {
		headers["Authorization"] = authorizationToken
	}

	// Merge custom headers from options
	for key, value := range opts.Headers {
		headers[key] = value
	}

	https := strings.HasPrefix(c.DSL.URL, "https://")
	proxy := http.GetProxy(https)
	tr := http.GetTransport(https, proxy)
	httpClient := &gohttp.Client{Transport: tr}

	// Create the HTTP transport
	httpTransport, err := transport.NewStreamableHTTP(c.DSL.URL,
		// Set timeout
		transport.WithHTTPTimeout(timeout),

		// Set custom headers
		transport.WithHTTPHeaders(headers),

		// With custom HTTP client
		transport.WithHTTPBasicClient(httpClient),
	)

	if err != nil {
		return fmt.Errorf("failed to create HTTP transport: %w", err)
	}

	c.MCPClient = goclient.NewClient(httpTransport)
	return nil
}

// Disconnect closes the connection to the MCP server
func (c *Client) Disconnect(ctx context.Context) error {
	if c.MCPClient == nil {
		return nil // Already disconnected
	}

	err := c.MCPClient.Close()
	c.MCPClient = nil // Clear the client reference
	return err
}

// IsConnected checks if the client is connected to the server
func (c *Client) IsConnected() bool {
	return c.MCPClient != nil
}

// State returns the current connection state
func (c *Client) State() types.ConnectionState {
	if c.MCPClient == nil {
		return types.StateDisconnected
	}

	return types.StateConnected
}
