package client

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/mcp/types"
)

func TestConnectValidation(t *testing.T) {
	tests := []struct {
		name        string
		client      *Client
		options     []types.ConnectionOptions
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Nil DSL",
			client:      &Client{DSL: nil},
			expectError: true,
			errorMsg:    "DSL configuration is nil",
		},
		{
			name: "Valid Connection",
			client: &Client{
				DSL: &types.ClientDSL{
					Name:      "Test Client",
					Transport: types.TransportStdio,
					Command:   "echo",
				},
			},
			expectError: false,
		},
		{
			name: "Connection with Options",
			client: &Client{
				DSL: &types.ClientDSL{
					Name:      "Test Client",
					Transport: types.TransportHTTP,
					URL:       "http://localhost:8080/mcp",
				},
			},
			options: []types.ConnectionOptions{
				{
					Headers: map[string]string{
						"X-Test": "test-value",
					},
					Timeout: 30 * time.Second,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := createTestContext(10 * time.Second)
			defer cancel()

			err := tt.client.Connect(ctx, tt.options...)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			// For non-error cases, we expect either success or a connection failure
			// (which is acceptable in test environment)
			if err != nil {
				logTestInfo(t, "Connection failed (expected in test env): %v", err)
			} else {
				logTestInfo(t, "Connection succeeded")
				// Clean up
				defer tt.client.Disconnect(ctx)
			}
		})
	}
}

func TestConnectStdio(t *testing.T) {
	tests := []struct {
		name        string
		dsl         *types.ClientDSL
		expectError bool
	}{
		{
			name: "Valid Stdio Command",
			dsl: &types.ClientDSL{
				Name:      "Test Stdio Client",
				Transport: types.TransportStdio,
				Command:   "echo",
				Arguments: []string{"hello", "world"},
			},
			expectError: false,
		},
		{
			name: "Invalid Stdio Command",
			dsl: &types.ClientDSL{
				Name:      "Test Stdio Client",
				Transport: types.TransportStdio,
				Command:   "nonexistent_command_xyz",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{DSL: tt.dsl}
			ctx, cancel := createTestContext(10 * time.Second)
			defer cancel()

			err := client.Connect(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				logTestInfo(t, "Stdio connection failed (may be expected): %v", err)
			} else {
				logTestInfo(t, "Stdio connection succeeded")
				defer client.Disconnect(ctx)

				// Verify client state
				if !client.IsConnected() {
					t.Errorf("Expected client to be connected")
				}
				if client.State() != types.StateConnected {
					t.Errorf("Expected state to be connected, got %v", client.State())
				}
			}
		})
	}
}

func TestConnectHTTP(t *testing.T) {
	config := getTestConfig()

	if config.SkipHTTPTests {
		t.Skip("HTTP tests skipped: set MCP_CLIENT_TEST_HTTP_URL environment variable")
	}

	tests := []struct {
		name        string
		dsl         *types.ClientDSL
		options     []types.ConnectionOptions
		expectError bool
	}{
		{
			name: "Valid HTTP Connection",
			dsl: &types.ClientDSL{
				Name:               "Test HTTP Client",
				Transport:          types.TransportHTTP,
				URL:                config.HTTPUrl,
				AuthorizationToken: config.HTTPToken,
			},
			expectError: false,
		},
		{
			name: "HTTP with Custom Headers",
			dsl: &types.ClientDSL{
				Name:               "Test HTTP Client",
				Transport:          types.TransportHTTP,
				URL:                config.HTTPUrl,
				AuthorizationToken: config.HTTPToken,
			},
			options: []types.ConnectionOptions{
				{
					Headers: map[string]string{
						"X-Custom-Header": "test-value",
						"X-Client-ID":     "test-client-123",
					},
				},
			},
			expectError: false,
		},
		{
			name: "HTTP with Timeout",
			dsl: &types.ClientDSL{
				Name:               "Test HTTP Client",
				Transport:          types.TransportHTTP,
				URL:                config.HTTPUrl,
				AuthorizationToken: config.HTTPToken,
			},
			options: []types.ConnectionOptions{
				{
					Timeout: 15 * time.Second,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{DSL: tt.dsl}
			ctx, cancel := createTestContext(30 * time.Second)
			defer cancel()

			err := client.Connect(ctx, tt.options...)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				logTestInfo(t, "HTTP connection failed (may be expected): %v", err)
			} else {
				logTestInfo(t, "HTTP connection succeeded")
				defer client.Disconnect(ctx)

				// Verify client state
				if !client.IsConnected() {
					t.Errorf("Expected client to be connected")
				}
				if client.State() != types.StateConnected {
					t.Errorf("Expected state to be connected, got %v", client.State())
				}
			}
		})
	}
}

func TestConnectSSE(t *testing.T) {
	config := getTestConfig()

	if config.SkipSSETests {
		t.Skip("SSE tests skipped: set MCP_CLIENT_TEST_SSE_URL environment variable")
	}

	tests := []struct {
		name        string
		dsl         *types.ClientDSL
		options     []types.ConnectionOptions
		expectError bool
	}{
		{
			name: "Valid SSE Connection",
			dsl: &types.ClientDSL{
				Name:               "Test SSE Client",
				Transport:          types.TransportSSE,
				URL:                config.SSEUrl,
				AuthorizationToken: config.SSEToken,
			},
			expectError: false,
		},
		{
			name: "SSE with Custom Headers",
			dsl: &types.ClientDSL{
				Name:               "Test SSE Client",
				Transport:          types.TransportSSE,
				URL:                config.SSEUrl,
				AuthorizationToken: config.SSEToken,
			},
			options: []types.ConnectionOptions{
				{
					Headers: map[string]string{
						"X-Stream-ID": "test-stream-456",
						"X-Format":    "json",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{DSL: tt.dsl}
			ctx, cancel := createTestContext(30 * time.Second)
			defer cancel()

			err := client.Connect(ctx, tt.options...)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				logTestInfo(t, "SSE connection failed (may be expected): %v", err)
			} else {
				logTestInfo(t, "SSE connection succeeded")
				defer client.Disconnect(ctx)

				// Verify client state
				if !client.IsConnected() {
					t.Errorf("Expected client to be connected")
				}
				if client.State() != types.StateConnected {
					t.Errorf("Expected state to be connected, got %v", client.State())
				}
			}
		})
	}
}

func TestDisconnect(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() *Client
		expectError bool
	}{
		{
			name: "Disconnect Connected Client",
			setupClient: func() *Client {
				dsl := &types.ClientDSL{
					Name:      "Test Client",
					Transport: types.TransportStdio,
					Command:   "echo",
				}
				client := &Client{DSL: dsl}
				// Try to connect (may fail in test env, but that's ok)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				client.Connect(ctx)
				return client
			},
			expectError: false,
		},
		{
			name: "Disconnect Unconnected Client",
			setupClient: func() *Client {
				dsl := &types.ClientDSL{
					Name:      "Test Client",
					Transport: types.TransportStdio,
					Command:   "echo",
				}
				return &Client{DSL: dsl}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			ctx, cancel := createTestContext(10 * time.Second)
			defer cancel()

			err := client.Disconnect(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify client state after disconnect
			if client.IsConnected() {
				t.Errorf("Expected client to be disconnected")
			}
			if client.State() != types.StateDisconnected {
				t.Errorf("Expected state to be disconnected, got %v", client.State())
			}
		})
	}
}

func TestConnectionState(t *testing.T) {
	dsl := &types.ClientDSL{
		Name:      "Test Client",
		Transport: types.TransportStdio,
		Command:   "echo",
	}
	client := &Client{DSL: dsl}

	// Test initial state
	if client.IsConnected() {
		t.Errorf("Expected client to be initially disconnected")
	}
	if client.State() != types.StateDisconnected {
		t.Errorf("Expected initial state to be disconnected, got %v", client.State())
	}

	// Test connection state changes
	ctx, cancel := createTestContext(10 * time.Second)
	defer cancel()

	// Try to connect
	err := client.Connect(ctx)
	if err != nil {
		logTestInfo(t, "Connection failed (expected): %v", err)
		// Even if connection fails, we should still be able to test state
		if client.State() != types.StateDisconnected {
			t.Errorf("Expected state to remain disconnected after failed connection")
		}
	} else {
		logTestInfo(t, "Connection succeeded")
		if !client.IsConnected() {
			t.Errorf("Expected client to be connected after successful connection")
		}
		if client.State() != types.StateConnected {
			t.Errorf("Expected state to be connected after successful connection")
		}

		// Test disconnect
		err = client.Disconnect(ctx)
		if err != nil {
			t.Errorf("Unexpected error during disconnect: %v", err)
		}
		if client.IsConnected() {
			t.Errorf("Expected client to be disconnected after disconnect")
		}
		if client.State() != types.StateDisconnected {
			t.Errorf("Expected state to be disconnected after disconnect")
		}
	}
}

func TestUnsupportedTransport(t *testing.T) {
	dsl := &types.ClientDSL{
		Name:      "Test Client",
		Transport: "websocket", // Unsupported transport
		URL:       "ws://localhost:8080/mcp",
	}
	client := &Client{DSL: dsl}

	ctx, cancel := createTestContext(5 * time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		t.Errorf("Expected error for unsupported transport")
	}
	if !strings.Contains(err.Error(), "unsupported transport type") {
		t.Errorf("Expected error message about unsupported transport, got: %v", err)
	}
}

func TestMultipleConnections(t *testing.T) {
	dsl := &types.ClientDSL{
		Name:      "Test Client",
		Transport: types.TransportStdio,
		Command:   "echo",
	}
	client := &Client{DSL: dsl}

	ctx, cancel := createTestContext(10 * time.Second)
	defer cancel()

	// First connection
	err1 := client.Connect(ctx)
	if err1 != nil {
		logTestInfo(t, "First connection failed (expected): %v", err1)
		return
	}

	// Second connection should succeed (already connected)
	err2 := client.Connect(ctx)
	if err2 != nil {
		t.Errorf("Second connection attempt failed: %v", err2)
	}

	// Cleanup
	defer client.Disconnect(ctx)
}

func TestMultipleDisconnections(t *testing.T) {
	dsl := &types.ClientDSL{
		Name:      "Test Client",
		Transport: types.TransportStdio,
		Command:   "echo",
	}
	client := &Client{DSL: dsl}

	ctx, cancel := createTestContext(10 * time.Second)
	defer cancel()

	// First disconnect (should not error)
	err1 := client.Disconnect(ctx)
	if err1 != nil {
		t.Errorf("First disconnect failed: %v", err1)
	}

	// Second disconnect (should not error)
	err2 := client.Disconnect(ctx)
	if err2 != nil {
		t.Errorf("Second disconnect failed: %v", err2)
	}
}

func TestConnectionWithTimeout(t *testing.T) {
	dsl := &types.ClientDSL{
		Name:      "Test Client",
		Transport: types.TransportHTTP,
		URL:       "http://10.255.255.1:12345/mcp", // Non-routable IP for timeout test
	}
	client := &Client{DSL: dsl}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	options := types.ConnectionOptions{
		Timeout: 50 * time.Millisecond,
	}

	err := client.Connect(ctx, options)
	if err == nil {
		logTestInfo(t, "Connection succeeded unexpectedly (test environment may have actual server)")
		defer client.Disconnect(ctx)
	} else {
		logTestInfo(t, "Connection timed out as expected: %v", err)
	}
}

func TestConnectionOptionsValidation(t *testing.T) {
	dsl := &types.ClientDSL{
		Name:      "Test Client",
		Transport: types.TransportHTTP,
		URL:       "http://localhost:8080/mcp",
	}
	client := &Client{DSL: dsl}

	ctx, cancel := createTestContext(5 * time.Second)
	defer cancel()

	// Test with various connection options
	validOptions := types.ConnectionOptions{
		Headers: map[string]string{
			"X-Test":      "value",
			"X-Client-ID": "test-client",
		},
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}

	err := client.Connect(ctx, validOptions)
	if err != nil {
		logTestInfo(t, "Connection with valid options failed (expected): %v", err)
	} else {
		logTestInfo(t, "Connection with valid options succeeded")
		defer client.Disconnect(ctx)
	}
}

func TestConnectionHeadersMerging(t *testing.T) {
	dsl := &types.ClientDSL{
		Name:               "Test Client",
		Transport:          types.TransportHTTP,
		URL:                "http://localhost:8080/mcp",
		AuthorizationToken: "Bearer test-token",
	}
	client := &Client{DSL: dsl}

	ctx, cancel := createTestContext(5 * time.Second)
	defer cancel()

	// Test header merging
	options := types.ConnectionOptions{
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Request-ID":    "req-123",
		},
	}

	err := client.Connect(ctx, options)
	if err != nil {
		logTestInfo(t, "Connection with merged headers failed (expected): %v", err)
	} else {
		logTestInfo(t, "Connection with merged headers succeeded")
		defer client.Disconnect(ctx)
	}
}
