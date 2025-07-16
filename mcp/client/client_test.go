package client

import (
	"testing"

	"github.com/yaoapp/gou/mcp/types"
	gouTypes "github.com/yaoapp/gou/types"
)

// TestNew tests the New function for creating MCP clients
func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		dsl         *types.ClientDSL
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Nil DSL",
			dsl:         nil,
			expectError: true,
			errorMsg:    "DSL cannot be nil",
		},
		{
			name: "Missing Name",
			dsl: &types.ClientDSL{
				Transport: types.TransportStdio,
				Command:   "echo",
			},
			expectError: true,
			errorMsg:    "client name is required",
		},
		{
			name: "Missing Command for Stdio",
			dsl: &types.ClientDSL{
				Name:      "Test Client",
				Transport: types.TransportStdio,
			},
			expectError: true,
			errorMsg:    "command is required for stdio transport",
		},
		{
			name: "Missing URL for HTTP",
			dsl: &types.ClientDSL{
				Name:      "Test Client",
				Transport: types.TransportHTTP,
			},
			expectError: true,
			errorMsg:    "URL is required for http transport",
		},
		{
			name: "Missing URL for SSE",
			dsl: &types.ClientDSL{
				Name:      "Test Client",
				Transport: types.TransportSSE,
			},
			expectError: true,
			errorMsg:    "URL is required for sse transport",
		},
		{
			name: "Unsupported Transport",
			dsl: &types.ClientDSL{
				Name:      "Test Client",
				Transport: "websocket", // Not a supported transport
			},
			expectError: true,
			errorMsg:    "unsupported transport type: websocket",
		},
		{
			name: "Valid Stdio DSL",
			dsl: &types.ClientDSL{
				Name:           "Test Stdio Client",
				Transport:      types.TransportStdio,
				Command:        "echo",
				Arguments:      []string{"hello"},
				EnableSampling: true,
			},
			expectError: false,
		},
		{
			name: "Valid HTTP DSL",
			dsl: &types.ClientDSL{
				Name:      "Test HTTP Client",
				Transport: types.TransportHTTP,
				URL:       "http://localhost:8080/mcp",
			},
			expectError: false,
		},
		{
			name: "Valid SSE DSL",
			dsl: &types.ClientDSL{
				Name:      "Test SSE Client",
				Transport: types.TransportSSE,
				URL:       "http://localhost:8080/sse",
			},
			expectError: false,
		},
		{
			name: "Complete DSL with All Fields",
			dsl: &types.ClientDSL{
				ID:        "complete-client",
				Name:      "Complete Test Client",
				Version:   "2.0.0",
				Transport: types.TransportHTTP,
				MetaInfo: gouTypes.MetaInfo{
					Label:       "Complete Client",
					Description: "A complete test client",
				},
				URL:                "https://api.example.com/mcp",
				AuthorizationToken: "Bearer test-token",
				EnableSampling:     true,
				EnableRoots:        true,
				RootsListChanged:   true,
				EnableElicitation:  false,
				Timeout:            "60s",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.dsl)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				logTestInfo(t, "Expected error: %v", err)
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("Expected client to be non-nil")
				return
			}

			// Verify client state
			assertClientState(t, client, false, types.StateDisconnected)

			// Verify DSL is correctly stored
			if client.DSL != tt.dsl {
				t.Error("Client DSL does not match input DSL")
			}

			logTestInfo(t, "Created client: %s (transport: %s)", tt.dsl.Name, tt.dsl.Transport)
		})
	}
}

// // TestClientConnection tests client connection functionality
// func TestClientConnection(t *testing.T) {
// 	config := getTestConfig()
// 	printTestConfig(t, config)

// 	t.Run("Stdio Connection", func(t *testing.T) {
// 		skipTestIfNoConfig(t, config.SkipStdioTests, "Stdio tests are disabled")

// 		dsl := createStdioTestDSL()
// 		client, err := New(dsl)
// 		if err != nil {
// 			t.Fatalf("Failed to create client: %v", err)
// 		}

// 		testBasicConnection(t, client, "stdio")
// 	})

// 	t.Run("HTTP Connection", func(t *testing.T) {
// 		skipTestIfNoConfig(t, config.SkipHTTPTests, "HTTP tests skipped: set MCP_CLIENT_TEST_HTTP_URL environment variable")

// 		dsl := createHTTPTestDSL(config)
// 		client, err := New(dsl)
// 		if err != nil {
// 			t.Fatalf("Failed to create client: %v", err)
// 		}

// 		testBasicConnection(t, client, "HTTP")
// 	})

// 	t.Run("SSE Connection", func(t *testing.T) {
// 		skipTestIfNoConfig(t, config.SkipSSETests, "SSE tests skipped: set MCP_CLIENT_TEST_SSE_URL environment variable")

// 		dsl := createSSETestDSL(config)
// 		client, err := New(dsl)
// 		if err != nil {
// 			t.Fatalf("Failed to create client: %v", err)
// 		}

// 		testBasicConnection(t, client, "SSE")
// 	})
// }

// // testBasicConnection tests basic connection functionality
// func testBasicConnection(t *testing.T, client *Client, transportName string) {
// 	t.Helper()

// 	ctx, cancel := createTestContext(30 * time.Second)
// 	defer cancel()

// 	dslInfo := getDSLInfo(client.DSL)
// 	logTestInfo(t, "Testing %s connection with DSL: %+v", transportName, dslInfo)

// 	// Test initial state
// 	assertClientState(t, client, false, types.StateDisconnected)

// 	// Test connection
// 	err := client.Connect(ctx)
// 	if err != nil {
// 		logTestInfo(t, "%s connection failed (may be expected): %v", transportName, err)
// 		// Connection failure is often expected in test environment
// 		return
// 	}

// 	logTestInfo(t, "%s connection successful", transportName)
// 	assertClientState(t, client, true, types.StateConnected)

// 	// Test disconnect
// 	err = client.Disconnect(ctx)
// 	if err != nil {
// 		t.Errorf("Failed to disconnect: %v", err)
// 	}

// 	assertClientState(t, client, false, types.StateDisconnected)
// 	logTestInfo(t, "%s disconnection successful", transportName)
// }

// // TestClientConnectionWithOptions tests connection with various options
// func TestClientConnectionWithOptions(t *testing.T) {
// 	config := getTestConfig()

// 	testCases := []struct {
// 		name        string
// 		createDSL   func() *types.ClientDSL
// 		skipTest    bool
// 		skipReason  string
// 		testOptions []TestConnectionOptions
// 	}{
// 		{
// 			name:       "HTTP with Options",
// 			createDSL:  func() *types.ClientDSL { return createHTTPTestDSL(config) },
// 			skipTest:   config.SkipHTTPTests,
// 			skipReason: "HTTP tests skipped: set MCP_CLIENT_TEST_HTTP_URL environment variable",
// 			testOptions: []TestConnectionOptions{
// 				{
// 					WithSessionID: true,
// 					SessionID:     "http-test-session-123",
// 				},
// 				{
// 					WithCustomHeaders: true,
// 					CustomHeaders: map[string]string{
// 						"X-Test-Mode":      "unit-test",
// 						"X-Client-Version": "1.0.0",
// 					},
// 				},
// 				{
// 					WithSessionID:     true,
// 					WithCustomHeaders: true,
// 					WithTimeout:       true,
// 					SessionID:         "http-full-test-session",
// 					CustomHeaders: map[string]string{
// 						"X-Request-ID":  "req-http-12345",
// 						"X-Environment": "test",
// 					},
// 					ConnectionTimeout: 10 * time.Second,
// 				},
// 			},
// 		},
// 		{
// 			name:       "SSE with Options",
// 			createDSL:  func() *types.ClientDSL { return createSSETestDSL(config) },
// 			skipTest:   config.SkipSSETests,
// 			skipReason: "SSE tests skipped: set MCP_CLIENT_TEST_SSE_URL environment variable",
// 			testOptions: []TestConnectionOptions{
// 				{
// 					WithSessionID: true,
// 					SessionID:     "sse-test-session-456",
// 				},
// 				{
// 					WithCustomHeaders: true,
// 					CustomHeaders: map[string]string{
// 						"X-Stream-Mode": "live",
// 						"X-Format":      "json",
// 					},
// 				},
// 				{
// 					WithSessionID:     true,
// 					WithCustomHeaders: true,
// 					SessionID:         "sse-full-test-session",
// 					CustomHeaders: map[string]string{
// 						"X-Request-ID":    "req-sse-67890",
// 						"X-Connection-ID": "conn-sse-test",
// 					},
// 				},
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			skipTestIfNoConfig(t, tc.skipTest, tc.skipReason)

// 			dsl := tc.createDSL()
// 			client, err := New(dsl)
// 			if err != nil {
// 				t.Fatalf("Failed to create client: %v", err)
// 			}

// 			for i, testOpts := range tc.testOptions {
// 				t.Run(tc.name+"-Option-"+string(rune(i+1)), func(t *testing.T) {
// 					testConnectionWithOptions(t, client, testOpts)
// 				})
// 			}
// 		})
// 	}
// }

// // testConnectionWithOptions tests connection with specific options
// func testConnectionWithOptions(t *testing.T, client *Client, opts TestConnectionOptions) {
// 	t.Helper()

// 	ctx, cancel := createTestContext(30 * time.Second)
// 	defer cancel()

// 	connOpts := createConnectionOptions(opts)
// 	logTestInfo(t, "Testing connection with options: %+v", connOpts)

// 	// Test initial state
// 	assertClientState(t, client, false, types.StateDisconnected)

// 	// Test connection with options
// 	err := client.Connect(ctx, connOpts)
// 	if err != nil {
// 		logTestInfo(t, "Connection with options failed (may be expected): %v", err)
// 		return
// 	}

// 	logTestInfo(t, "Connection with options successful")
// 	assertClientState(t, client, true, types.StateConnected)

// 	// Test disconnect
// 	err = client.Disconnect(ctx)
// 	if err != nil {
// 		t.Errorf("Failed to disconnect: %v", err)
// 	}

// 	assertClientState(t, client, false, types.StateDisconnected)
// 	logTestInfo(t, "Disconnection successful")
// }

// // TestClientReconnection tests reconnection scenarios
// func TestClientReconnection(t *testing.T) {
// 	// Test with stdio (most reliable for testing)
// 	dsl := createStdioTestDSL()
// 	client, err := New(dsl)
// 	if err != nil {
// 		t.Fatalf("Failed to create client: %v", err)
// 	}

// 	ctx, cancel := createTestContext(30 * time.Second)
// 	defer cancel()

// 	// Test multiple connect calls (should not error)
// 	err = client.Connect(ctx)
// 	if err != nil {
// 		logTestInfo(t, "Initial connection failed (expected in test env): %v", err)
// 		return
// 	}

// 	// Second connect call should succeed (already connected)
// 	err = client.Connect(ctx)
// 	if err != nil {
// 		t.Errorf("Second connect call failed: %v", err)
// 	}

// 	assertClientState(t, client, true, types.StateConnected)

// 	// Test disconnect
// 	err = client.Disconnect(ctx)
// 	if err != nil {
// 		t.Errorf("Failed to disconnect: %v", err)
// 	}

// 	// Test reconnect
// 	err = client.Connect(ctx)
// 	if err != nil {
// 		logTestInfo(t, "Reconnection failed (expected in test env): %v", err)
// 		return
// 	}

// 	// Final cleanup
// 	client.Disconnect(ctx)
// }

// // TestClientMultipleDisconnect tests multiple disconnect calls
// func TestClientMultipleDisconnect(t *testing.T) {
// 	dsl := createStdioTestDSL()
// 	client, err := New(dsl)
// 	if err != nil {
// 		t.Fatalf("Failed to create client: %v", err)
// 	}

// 	ctx, cancel := createTestContext(10 * time.Second)
// 	defer cancel()

// 	// Test disconnect on unconnected client (should not error)
// 	err = client.Disconnect(ctx)
// 	if err != nil {
// 		t.Errorf("Disconnect on unconnected client failed: %v", err)
// 	}

// 	assertClientState(t, client, false, types.StateDisconnected)

// 	// Try to connect and then test multiple disconnects
// 	err = client.Connect(ctx)
// 	if err != nil {
// 		logTestInfo(t, "Connection failed (expected): %v", err)
// 		return
// 	}

// 	// Multiple disconnect calls should not error
// 	err = client.Disconnect(ctx)
// 	if err != nil {
// 		t.Errorf("First disconnect failed: %v", err)
// 	}

// 	err = client.Disconnect(ctx)
// 	if err != nil {
// 		t.Errorf("Second disconnect failed: %v", err)
// 	}

// 	assertClientState(t, client, false, types.StateDisconnected)
// }
