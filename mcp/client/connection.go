package client

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		return c.connectStdio(ctx)

	case types.TransportSSE:
		return c.connectSSE(ctx, opts)

	case types.TransportHTTP:
		return c.connectHTTP(ctx, opts)

	default:
		return fmt.Errorf("unsupported transport type: %s", c.DSL.Transport)
	}
}

// connectStdio creates a stdio connection with process tracking
func (c *Client) connectStdio(_ context.Context) error {
	// For testing purposes, we'll use the existing mcp-go client
	// but also track the process if possible
	client, err := goclient.NewStdioMCPClient(c.DSL.Command, c.DSL.GetEnvs(), c.DSL.Arguments...)
	if err != nil {
		return fmt.Errorf("failed to create stdio client: %w", err)
	}

	c.MCPClient = client

	// Note: We can't easily get the process from mcp-go client
	// This is a limitation of the current mcp-go library
	// For now, we'll rely on the improved timeout mechanism

	return nil
}

// connectSSE creates a SSE connection
func (c *Client) connectSSE(ctx context.Context, opts types.ConnectionOptions) error {
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

	var err error
	c.MCPClient, err = goclient.NewSSEMCPClient(c.DSL.URL,
		transport.WithHeaders(headers),
		transport.WithHTTPClient(httpClient),
	)

	err = c.MCPClient.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start SSE client: %w", err)
	}

	return err
}

// connectHTTP creates a HTTP connection
func (c *Client) connectHTTP(_ context.Context, opts types.ConnectionOptions) error {
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

	// Create a reasonable timeout for the operation
	disconnectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try graceful close with timeout
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		errChan <- c.MCPClient.Close()
	}()

	var err error
	select {
	case err = <-errChan:
		// Graceful close completed
	case <-disconnectCtx.Done():
		// Timeout or context cancellation
		err = disconnectCtx.Err()
	}

	// Clean up references regardless of close result
	c.MCPClient = nil
	c.InitResult = nil

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

	// Check if initialized
	if c.InitResult != nil {
		return types.StateInitialized
	}

	return types.StateConnected
}
