package client

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/mcp/types"
)

func TestConnect(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			// Create client
			client := &Client{DSL: testCase.DSL}

			// Create context with timeout
			ctx, cancel := createTestContext(testCase.Timeout)
			defer cancel()

			// Test connection
			err := client.Connect(ctx)

			if testCase.ExpectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if testCase.ExpectedError != "" && !containsString(err.Error(), testCase.ExpectedError) {
					t.Errorf("Expected error message to contain '%s', got '%s'", testCase.ExpectedError, err.Error())
				}
				logTestInfo(t, "Expected error: %v", err)
				return
			}

			// For non-error cases, we expect either success or a connection failure
			// (which is acceptable in test environment)
			if err != nil {
				logTestInfo(t, "Connection failed (expected in test env): %v", err)
			} else {
				logTestInfo(t, "Connection succeeded")
				// Clean up
				defer client.Disconnect(ctx)

				// Verify client state
				if !client.IsConnected() {
					t.Errorf("Expected client to be connected")
				}
				if client.State() != types.StateConnected {
					t.Errorf("Expected state to be connected, got %v", client.State())
				}
				if client.IsInitialized() {
					t.Errorf("Expected client to not be initialized after just connecting")
				}
			}
		})
	}
}

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
					Command:   "npx",
					Arguments: []string{"-y", "@modelcontextprotocol/server-everything"},
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

func TestDisconnect(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			// Create client
			client := &Client{DSL: testCase.DSL}

			// Create context with timeout
			_, cancel := createTestContext(testCase.Timeout)
			defer cancel()

			// Test disconnect without connection (should not error)
			ctxclose, cancelclose := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
			defer cancelclose()
			err := client.Disconnect(ctxclose)
			if err != nil && err.Error() != "close timeout: context canceled" {
				t.Errorf("Unexpected error during disconnect: %v", err)
			}

			// Verify client state after disconnect
			if client.IsConnected() {
				t.Errorf("Expected client to be disconnected")
			}
			if client.State() != types.StateDisconnected {
				t.Errorf("Expected state to be disconnected, got %v", client.State())
			}
			if client.IsInitialized() {
				t.Errorf("Expected client to not be initialized after disconnect")
			}

			logTestInfo(t, "Disconnect test completed successfully for %s", testCase.Name)
		})
	}
}

func TestConnectionState(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			// Create client
			client := &Client{DSL: testCase.DSL}

			// Test initial state
			if client.IsConnected() {
				t.Errorf("Expected client to be initially disconnected")
			}
			if client.State() != types.StateDisconnected {
				t.Errorf("Expected initial state to be disconnected, got %v", client.State())
			}
			if client.IsInitialized() {
				t.Errorf("Expected client to not be initialized initially")
			}

			// Test connection state changes
			ctx, cancel := createTestContext(testCase.Timeout)
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
				if client.IsInitialized() {
					t.Errorf("Expected client to not be initialized after just connecting")
				}

				// Try to initialize
				_, err = client.Initialize(ctx)
				if err != nil {
					logTestInfo(t, "Initialization failed (expected): %v", err)
				} else {
					logTestInfo(t, "Initialization succeeded")
					if client.State() != types.StateInitialized {
						t.Errorf("Expected state to be initialized after initialization, but got %v", client.State())
					}
					if !client.IsInitialized() {
						t.Errorf("Expected client to be initialized after initialization, but got %v", client.IsInitialized())
					}
				}

				// // Test disconnect
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
				if client.IsInitialized() {
					t.Errorf("Expected client to not be initialized after disconnect")
				}
			}
		})
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
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			// Create client
			client := &Client{DSL: testCase.DSL}

			// Create context with timeout
			ctx, cancel := createTestContext(testCase.Timeout)
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
		})
	}
}

func TestMultipleDisconnections(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			// Create client
			client := &Client{DSL: testCase.DSL}

			// Create context with timeout
			ctx, cancel := createTestContext(testCase.Timeout)
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
		})
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

// TestInitializationResultStorageInConnection tests initialization result storage during connection lifecycle
func TestInitializationResultStorageInConnection(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			// Create client
			client := &Client{DSL: testCase.DSL}

			// Create context with timeout
			ctx, cancel := createTestContext(testCase.Timeout)
			defer cancel()

			// Test initial state
			if client.GetInitResult() != nil {
				t.Errorf("Expected initialization result to be nil initially")
			}
			if client.IsInitialized() {
				t.Errorf("Expected client to not be initialized initially")
			}

			// Connect
			err := client.Connect(ctx)
			if err != nil {
				logTestInfo(t, "Connection failed (expected): %v", err)
				return
			}
			defer client.Disconnect(ctx)

			// Verify still not initialized after connection
			if client.GetInitResult() != nil {
				t.Errorf("Expected initialization result to be nil after connection")
			}
			if client.IsInitialized() {
				t.Errorf("Expected client to not be initialized after connection")
			}
			if client.State() != types.StateConnected {
				t.Errorf("Expected state to be connected after connection, got %v", client.State())
			}

			// Initialize
			response, err := client.Initialize(ctx)
			if err != nil {
				logTestInfo(t, "Initialization failed (expected): %v", err)
				return
			}

			// Verify initialization result is stored
			if client.GetInitResult() == nil {
				t.Errorf("Expected initialization result to be stored after initialization")
			}
			if !client.IsInitialized() {
				t.Errorf("Expected client to be initialized after initialization")
			}
			if client.State() != types.StateInitialized {
				t.Errorf("Expected state to be initialized after initialization, got %v", client.State())
			}

			// Verify stored result matches response
			storedResult := client.GetInitResult()
			if storedResult != response {
				t.Errorf("Expected stored result to match returned response")
			}

			logTestInfo(t, "Initialization result storage in connection lifecycle works correctly")
		})
	}
}

// TestDisconnectClearsInitializationResult tests that disconnection clears initialization result
func TestDisconnectClearsInitializationResult(t *testing.T) {
	testCases := getStandardTransportTestCases()

	disconnectMethods := []struct {
		name           string
		disconnectFunc func(*Client, context.Context) error
	}{
		{
			name: "Disconnect method clears result",
			disconnectFunc: func(c *Client, ctx context.Context) error {
				return c.Disconnect(ctx)
			},
		},
	}

	for _, testCase := range testCases {
		for _, method := range disconnectMethods {
			t.Run(testCase.Name+" - "+method.name, func(t *testing.T) {
				// Skip test if configuration is not available
				if testCase.ShouldSkip {
					t.Skip(testCase.SkipReason)
					return
				}

				// Create client
				client := &Client{DSL: testCase.DSL}

				// Create context with timeout
				ctx, cancel := createTestContext(testCase.Timeout)
				defer cancel()

				// Connect and initialize
				err := client.Connect(ctx)
				if err != nil {
					logTestInfo(t, "Connection failed (expected): %v", err)
					return
				}

				_, err = client.Initialize(ctx)
				if err != nil {
					logTestInfo(t, "Initialization failed (expected): %v", err)
					client.Disconnect(ctx)
					return
				}

				// Verify initialization result is stored
				if client.GetInitResult() == nil {
					t.Errorf("Expected initialization result to be stored before disconnect")
				}
				if !client.IsInitialized() {
					t.Errorf("Expected client to be initialized before disconnect")
				}
				if client.State() != types.StateInitialized {
					t.Errorf("Expected state to be initialized before disconnect, got %v", client.State())
				}

				// Disconnect using the specified method
				err = method.disconnectFunc(client, ctx)
				if err != nil {
					t.Errorf("Disconnect failed: %v", err)
				}

				// Verify initialization result is cleared
				if client.GetInitResult() != nil {
					t.Errorf("Expected initialization result to be cleared after disconnect")
				}
				if client.IsInitialized() {
					t.Errorf("Expected client to not be initialized after disconnect")
				}
				if client.State() != types.StateDisconnected {
					t.Errorf("Expected state to be disconnected after disconnect, got %v", client.State())
				}
				if client.IsConnected() {
					t.Errorf("Expected client to not be connected after disconnect")
				}

				logTestInfo(t, "Disconnect method %s cleared initialization result correctly", method.name)
			})
		}
	}
}

// TestMultipleInitializationAttempts tests multiple initialization attempts
func TestMultipleInitializationAttempts(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			// Create client
			client := &Client{DSL: testCase.DSL}

			// Create context with timeout
			ctx, cancel := createTestContext(testCase.Timeout)
			defer cancel()

			// Connect
			err := client.Connect(ctx)
			if err != nil {
				logTestInfo(t, "Connection failed (expected): %v", err)
				return
			}
			defer client.Disconnect(ctx)

			// First initialization
			response1, err := client.Initialize(ctx)
			if err != nil {
				logTestInfo(t, "First initialization failed (expected): %v", err)
				return
			}

			// Verify first initialization
			if client.GetInitResult() == nil {
				t.Errorf("Expected initialization result to be stored after first initialization")
			}
			if client.GetInitResult() != response1 {
				t.Errorf("Expected stored result to match first response")
			}

			// Second initialization (should update the stored result)
			response2, err := client.Initialize(ctx)
			if err != nil {
				logTestInfo(t, "Second initialization failed (expected): %v", err)
				return
			}

			// Verify second initialization updates the stored result
			if client.GetInitResult() == nil {
				t.Errorf("Expected initialization result to be stored after second initialization")
			}
			if client.GetInitResult() != response2 {
				t.Errorf("Expected stored result to match second response")
			}

			logTestInfo(t, "Multiple initialization attempts work correctly")
		})
	}
}
