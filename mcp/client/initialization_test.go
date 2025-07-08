package client

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

func TestInitialize(t *testing.T) {
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

			// Clean up after test
			defer func() {
				client.Disconnect(ctx)
				cancel()
			}()

			// Test initialize without connection first
			_, err := client.Initialize(ctx)
			if err == nil {
				t.Errorf("Expected error when initializing without connection")
				return
			}
			if !containsString(err.Error(), "MCP client not connected") {
				t.Errorf("Expected error message to contain 'MCP client not connected', got '%s'", err.Error())
			}
			logTestInfo(t, "Initialize without connection failed as expected: %v", err)

			// Try to connect
			err = client.Connect(ctx)
			if err != nil {
				logTestInfo(t, "Connection failed (expected): %v", err)
				return
			}
			logTestInfo(t, "Connection succeeded")

			// Now test initialization
			response, err := client.Initialize(ctx)

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

			// For non-error cases, initialization should succeed
			if err != nil {
				t.Errorf("Initialization failed unexpectedly: %v", err)
				return
			}

			// If initialization succeeded, validate the response
			if response == nil {
				t.Errorf("Expected non-nil response")
				return
			}

			// Test that the result is stored in the client
			if client.GetInitResult() == nil {
				t.Errorf("Expected initialization result to be stored in client")
			}
			if !client.IsInitialized() {
				t.Errorf("Expected client to be initialized")
			}
			if client.State() != types.StateInitialized {
				t.Errorf("Expected state to be initialized, got %v", client.State())
			}

			// Verify the stored result matches the returned response
			storedResult := client.GetInitResult()
			if storedResult.ProtocolVersion != response.ProtocolVersion {
				t.Errorf("Stored result protocol version mismatch: got %s, expected %s",
					storedResult.ProtocolVersion, response.ProtocolVersion)
			}
			if storedResult.ServerInfo.Name != response.ServerInfo.Name {
				t.Errorf("Stored result server name mismatch: got %s, expected %s",
					storedResult.ServerInfo.Name, response.ServerInfo.Name)
			}

			logTestInfo(t, "Initialization succeeded")
			logTestInfo(t, "Protocol Version: %s", response.ProtocolVersion)
			logTestInfo(t, "Server Info: %s v%s", response.ServerInfo.Name, response.ServerInfo.Version)

			// Validate response fields
			if response.ProtocolVersion == "" {
				t.Errorf("Expected non-empty protocol version")
			}
			if response.ServerInfo.Name == "" {
				t.Errorf("Expected non-empty server name")
			}
		})
	}
}

func TestInitialized(t *testing.T) {
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

			// Clean up after test
			defer client.Disconnect(ctx)

			// Test initialized without connection first
			err := client.Initialized(ctx)
			if err == nil {
				t.Errorf("Expected error when calling Initialized without connection")
				return
			}
			if !containsString(err.Error(), "MCP client not connected") {
				t.Errorf("Expected error message to contain 'MCP client not connected', got '%s'", err.Error())
			}
			logTestInfo(t, "Initialized without connection failed as expected: %v", err)

			// Try to connect
			err = client.Connect(ctx)
			if err != nil {
				logTestInfo(t, "Connection failed (expected): %v", err)
				return
			}
			logTestInfo(t, "Connection succeeded")

			// Now test Initialized call
			err = client.Initialized(ctx)

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

			// Initialized should always succeed if client is connected (it's a no-op)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			logTestInfo(t, "Initialized called successfully")
		})
	}
}

func TestConvertServerCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		input    mcp.ServerCapabilities
		expected types.ServerCapabilities
	}{
		{
			name: "Empty capabilities",
			input: mcp.ServerCapabilities{
				Experimental: make(map[string]interface{}),
			},
			expected: types.ServerCapabilities{
				Experimental: make(map[string]interface{}),
			},
		},
		{
			name: "Resources capability",
			input: mcp.ServerCapabilities{
				Resources: &struct {
					Subscribe   bool `json:"subscribe,omitempty"`
					ListChanged bool `json:"listChanged,omitempty"`
				}{
					Subscribe:   true,
					ListChanged: true,
				},
				Experimental: make(map[string]interface{}),
			},
			expected: types.ServerCapabilities{
				Resources: &types.ResourcesCapability{
					Subscribe:   true,
					ListChanged: true,
				},
				Experimental: make(map[string]interface{}),
			},
		},
		{
			name: "Tools capability",
			input: mcp.ServerCapabilities{
				Tools: &struct {
					ListChanged bool `json:"listChanged,omitempty"`
				}{
					ListChanged: true,
				},
				Experimental: make(map[string]interface{}),
			},
			expected: types.ServerCapabilities{
				Tools: &types.ToolsCapability{
					ListChanged: true,
				},
				Experimental: make(map[string]interface{}),
			},
		},
		{
			name: "Prompts capability",
			input: mcp.ServerCapabilities{
				Prompts: &struct {
					ListChanged bool `json:"listChanged,omitempty"`
				}{
					ListChanged: true,
				},
				Experimental: make(map[string]interface{}),
			},
			expected: types.ServerCapabilities{
				Prompts: &types.PromptsCapability{
					ListChanged: true,
				},
				Experimental: make(map[string]interface{}),
			},
		},
		{
			name: "Logging capability",
			input: mcp.ServerCapabilities{
				Logging:      &struct{}{},
				Experimental: make(map[string]interface{}),
			},
			expected: types.ServerCapabilities{
				Logging:      &types.LoggingCapability{},
				Experimental: make(map[string]interface{}),
			},
		},
		{
			name: "All capabilities",
			input: mcp.ServerCapabilities{
				Resources: &struct {
					Subscribe   bool `json:"subscribe,omitempty"`
					ListChanged bool `json:"listChanged,omitempty"`
				}{
					Subscribe:   true,
					ListChanged: false,
				},
				Tools: &struct {
					ListChanged bool `json:"listChanged,omitempty"`
				}{
					ListChanged: true,
				},
				Prompts: &struct {
					ListChanged bool `json:"listChanged,omitempty"`
				}{
					ListChanged: false,
				},
				Logging: &struct{}{},
				Experimental: map[string]interface{}{
					"custom": "value",
				},
			},
			expected: types.ServerCapabilities{
				Resources: &types.ResourcesCapability{
					Subscribe:   true,
					ListChanged: false,
				},
				Tools: &types.ToolsCapability{
					ListChanged: true,
				},
				Prompts: &types.PromptsCapability{
					ListChanged: false,
				},
				Logging: &types.LoggingCapability{},
				Experimental: map[string]interface{}{
					"custom": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertServerCapabilities(tt.input)

			// Compare Resources
			if (result.Resources == nil) != (tt.expected.Resources == nil) {
				t.Errorf("Resources capability mismatch: got %v, expected %v", result.Resources, tt.expected.Resources)
			}
			if result.Resources != nil && tt.expected.Resources != nil {
				if result.Resources.Subscribe != tt.expected.Resources.Subscribe {
					t.Errorf("Resources.Subscribe mismatch: got %v, expected %v", result.Resources.Subscribe, tt.expected.Resources.Subscribe)
				}
				if result.Resources.ListChanged != tt.expected.Resources.ListChanged {
					t.Errorf("Resources.ListChanged mismatch: got %v, expected %v", result.Resources.ListChanged, tt.expected.Resources.ListChanged)
				}
			}

			// Compare Tools
			if (result.Tools == nil) != (tt.expected.Tools == nil) {
				t.Errorf("Tools capability mismatch: got %v, expected %v", result.Tools, tt.expected.Tools)
			}
			if result.Tools != nil && tt.expected.Tools != nil {
				if result.Tools.ListChanged != tt.expected.Tools.ListChanged {
					t.Errorf("Tools.ListChanged mismatch: got %v, expected %v", result.Tools.ListChanged, tt.expected.Tools.ListChanged)
				}
			}

			// Compare Prompts
			if (result.Prompts == nil) != (tt.expected.Prompts == nil) {
				t.Errorf("Prompts capability mismatch: got %v, expected %v", result.Prompts, tt.expected.Prompts)
			}
			if result.Prompts != nil && tt.expected.Prompts != nil {
				if result.Prompts.ListChanged != tt.expected.Prompts.ListChanged {
					t.Errorf("Prompts.ListChanged mismatch: got %v, expected %v", result.Prompts.ListChanged, tt.expected.Prompts.ListChanged)
				}
			}

			// Compare Logging
			if (result.Logging == nil) != (tt.expected.Logging == nil) {
				t.Errorf("Logging capability mismatch: got %v, expected %v", result.Logging, tt.expected.Logging)
			}

			// Compare Experimental
			if len(result.Experimental) != len(tt.expected.Experimental) {
				t.Errorf("Experimental length mismatch: got %d, expected %d", len(result.Experimental), len(tt.expected.Experimental))
			}
			for key, expectedValue := range tt.expected.Experimental {
				if actualValue, exists := result.Experimental[key]; !exists || actualValue != expectedValue {
					t.Errorf("Experimental[%s] mismatch: got %v, expected %v", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestConvertClientCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		input    types.ClientCapabilities
		expected mcp.ClientCapabilities
	}{
		{
			name: "Empty capabilities",
			input: types.ClientCapabilities{
				Experimental: make(map[string]interface{}),
			},
			expected: mcp.ClientCapabilities{
				Experimental: make(map[string]interface{}),
			},
		},
		{
			name: "Sampling capability",
			input: types.ClientCapabilities{
				Sampling:     &types.SamplingCapability{},
				Experimental: make(map[string]interface{}),
			},
			expected: mcp.ClientCapabilities{
				Sampling:     &struct{}{},
				Experimental: make(map[string]interface{}),
			},
		},
		{
			name: "Roots capability",
			input: types.ClientCapabilities{
				Roots: &types.RootsCapability{
					ListChanged: true,
				},
				Experimental: make(map[string]interface{}),
			},
			expected: mcp.ClientCapabilities{
				Roots: &struct {
					ListChanged bool `json:"listChanged,omitempty"`
				}{
					ListChanged: true,
				},
				Experimental: make(map[string]interface{}),
			},
		},
		{
			name: "All capabilities",
			input: types.ClientCapabilities{
				Sampling: &types.SamplingCapability{},
				Roots: &types.RootsCapability{
					ListChanged: false,
				},
				Experimental: map[string]interface{}{
					"custom": "value",
				},
			},
			expected: mcp.ClientCapabilities{
				Sampling: &struct{}{},
				Roots: &struct {
					ListChanged bool `json:"listChanged,omitempty"`
				}{
					ListChanged: false,
				},
				Experimental: map[string]interface{}{
					"custom": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertClientCapabilities(tt.input)

			// Compare Sampling
			if (result.Sampling == nil) != (tt.expected.Sampling == nil) {
				t.Errorf("Sampling capability mismatch: got %v, expected %v", result.Sampling, tt.expected.Sampling)
			}

			// Compare Roots
			if (result.Roots == nil) != (tt.expected.Roots == nil) {
				t.Errorf("Roots capability mismatch: got %v, expected %v", result.Roots, tt.expected.Roots)
			}
			if result.Roots != nil && tt.expected.Roots != nil {
				if result.Roots.ListChanged != tt.expected.Roots.ListChanged {
					t.Errorf("Roots.ListChanged mismatch: got %v, expected %v", result.Roots.ListChanged, tt.expected.Roots.ListChanged)
				}
			}

			// Compare Experimental
			if len(result.Experimental) != len(tt.expected.Experimental) {
				t.Errorf("Experimental length mismatch: got %d, expected %d", len(result.Experimental), len(tt.expected.Experimental))
			}
			for key, expectedValue := range tt.expected.Experimental {
				if actualValue, exists := result.Experimental[key]; !exists || actualValue != expectedValue {
					t.Errorf("Experimental[%s] mismatch: got %v, expected %v", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestInitializeWithDifferentCapabilities(t *testing.T) {
	testCases := getStandardTransportTestCases()

	capabilityTests := []struct {
		name              string
		enableSampling    bool
		enableRoots       bool
		rootsListChanged  bool
		enableElicitation bool
	}{
		{
			name:              "Client with sampling only",
			enableSampling:    true,
			enableRoots:       false,
			rootsListChanged:  false,
			enableElicitation: false,
		},
		{
			name:              "Client with roots only",
			enableSampling:    false,
			enableRoots:       true,
			rootsListChanged:  true,
			enableElicitation: false,
		},
		{
			name:              "Client with all capabilities",
			enableSampling:    true,
			enableRoots:       true,
			rootsListChanged:  true,
			enableElicitation: true,
		},
	}

	for _, testCase := range testCases {
		for _, capTest := range capabilityTests {
			t.Run(testCase.Name+" - "+capTest.name, func(t *testing.T) {
				// Skip test if configuration is not available
				if testCase.ShouldSkip {
					t.Skip(testCase.SkipReason)
					return
				}

				// Create modified DSL with different capabilities
				dsl := *testCase.DSL // Copy the DSL
				dsl.EnableSampling = capTest.enableSampling
				dsl.EnableRoots = capTest.enableRoots
				dsl.RootsListChanged = capTest.rootsListChanged
				dsl.EnableElicitation = capTest.enableElicitation

				// Create client
				client := &Client{DSL: &dsl}

				// Create context with timeout
				ctx, cancel := createTestContext(testCase.Timeout)
				defer cancel()

				// Clean up after test
				defer client.Disconnect(ctx)

				// Try to connect
				err := client.Connect(ctx)
				if err != nil {
					logTestInfo(t, "Connection failed (expected): %v", err)
					return
				}
				logTestInfo(t, "Connection succeeded")

				// Test client capabilities generation
				clientCaps := dsl.GetClientCapabilities()
				logTestInfo(t, "Client capabilities: Sampling=%v, Roots=%v",
					clientCaps.Sampling != nil, clientCaps.Roots != nil)

				// Test initialization (may fail due to no actual server)
				_, err = client.Initialize(ctx)
				if err != nil {
					logTestInfo(t, "Initialization failed (expected): %v", err)
				} else {
					logTestInfo(t, "Initialization succeeded")
				}
			})
		}
	}
}

// TestInitializationResultStorage tests the storage and retrieval of initialization results
func TestInitializationResultStorage(t *testing.T) {
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

			// Clean up after test
			defer client.Disconnect(ctx)

			// Test initial state
			if client.GetInitResult() != nil {
				t.Errorf("Expected initial initialization result to be nil")
			}
			if client.IsInitialized() {
				t.Errorf("Expected client to not be initialized initially")
			}

			// Try to connect
			err := client.Connect(ctx)
			if err != nil {
				logTestInfo(t, "Connection failed (expected): %v", err)
				return
			}
			logTestInfo(t, "Connection succeeded")

			// Try to initialize
			response, err := client.Initialize(ctx)
			if err != nil {
				logTestInfo(t, "Initialization failed (expected): %v", err)
				return
			}

			// Test that result is stored
			if client.GetInitResult() == nil {
				t.Errorf("Expected initialization result to be stored")
			}
			if !client.IsInitialized() {
				t.Errorf("Expected client to be initialized")
			}

			// Test that stored result matches response
			storedResult := client.GetInitResult()
			if storedResult != response {
				t.Errorf("Stored result pointer should match returned response")
			}

			logTestInfo(t, "Initialization result stored successfully")
		})
	}
}

// TestInitializationResultClear tests clearing of initialization results on disconnect
func TestInitializationResultClear(t *testing.T) {
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

				// Try to connect and initialize
				err := client.Connect(ctx)
				if err != nil {
					logTestInfo(t, "Connection failed (expected): %v", err)
					return
				}
				logTestInfo(t, "Connection succeeded")

				_, err = client.Initialize(ctx)
				if err != nil {
					logTestInfo(t, "Initialization failed (expected): %v", err)
					// Clean up
					client.Disconnect(ctx)
					return
				}
				logTestInfo(t, "Initialization succeeded")

				// Verify initialization result is stored
				if client.GetInitResult() == nil {
					t.Errorf("Expected initialization result to be stored before disconnect")
				}
				if !client.IsInitialized() {
					t.Errorf("Expected client to be initialized before disconnect")
				}

				// Test disconnection
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

				logTestInfo(t, "Initialization result cleared successfully")
			})
		}
	}
}

// TestClientStateTransitions tests state transitions with initialization
func TestClientStateTransitions(t *testing.T) {
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

			// Clean up after test
			defer client.Disconnect(ctx)

			// Test initial state
			if client.State() != types.StateDisconnected {
				t.Errorf("Expected initial state to be disconnected, got %v", client.State())
			}
			if client.IsInitialized() {
				t.Errorf("Expected client to not be initialized initially")
			}

			// Try to connect
			err := client.Connect(ctx)
			if err != nil {
				logTestInfo(t, "Connection failed (expected): %v", err)
				return
			}
			logTestInfo(t, "Connection succeeded")

			// Test connected state
			if client.State() != types.StateConnected {
				t.Errorf("Expected state to be connected after connection, got %v", client.State())
			}
			if client.IsInitialized() {
				t.Errorf("Expected client to not be initialized after connection")
			}

			// Try to initialize
			_, err = client.Initialize(ctx)
			if err != nil {
				logTestInfo(t, "Initialization failed (expected): %v", err)
				return
			}
			logTestInfo(t, "Initialization succeeded")

			// Test initialized state
			if client.State() != types.StateInitialized {
				t.Errorf("Expected state to be initialized after initialization, got %v", client.State())
			}
			if !client.IsInitialized() {
				t.Errorf("Expected client to be initialized after initialization")
			}

			// Test disconnect
			err = client.Disconnect(ctx)
			if err != nil {
				t.Errorf("Disconnect failed: %v", err)
			}

			// Test disconnected state
			if client.State() != types.StateDisconnected {
				t.Errorf("Expected state to be disconnected after disconnect, got %v", client.State())
			}
			if client.IsInitialized() {
				t.Errorf("Expected client to not be initialized after disconnect")
			}

			logTestInfo(t, "State transitions completed successfully")
		})
	}
}

// TestGetInitResultMethods tests the GetInitResult and IsInitialized methods
func TestGetInitResultMethods(t *testing.T) {
	dsl := createStdioTestDSL()
	client := &Client{DSL: dsl}

	// Test methods on uninitialized client
	if client.GetInitResult() != nil {
		t.Errorf("Expected GetInitResult to return nil for uninitialized client")
	}
	if client.IsInitialized() {
		t.Errorf("Expected IsInitialized to return false for uninitialized client")
	}

	// Simulate initialization by setting InitResult directly
	mockResult := &types.InitializeResponse{
		ProtocolVersion: "test-protocol",
		ServerInfo: types.ServerInfo{
			Name:    "Test Server",
			Version: "1.0.0",
		},
		Capabilities: types.ServerCapabilities{
			Experimental: make(map[string]interface{}),
		},
	}
	client.InitResult = mockResult

	// Test methods on initialized client
	if client.GetInitResult() != mockResult {
		t.Errorf("Expected GetInitResult to return the stored result")
	}
	if !client.IsInitialized() {
		t.Errorf("Expected IsInitialized to return true for initialized client")
	}

	// Clear result
	client.InitResult = nil

	// Test methods after clearing
	if client.GetInitResult() != nil {
		t.Errorf("Expected GetInitResult to return nil after clearing")
	}
	if client.IsInitialized() {
		t.Errorf("Expected IsInitialized to return false after clearing")
	}

	logTestInfo(t, "GetInitResult and IsInitialized methods work correctly")
}
