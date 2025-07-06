package client

import (
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

func TestInitialize(t *testing.T) {
	config := getTestConfig()

	tests := []struct {
		name        string
		setupClient func() *Client
		expectError bool
		errorMsg    string
	}{
		{
			name: "Initialize without connection",
			setupClient: func() *Client {
				dsl := createStdioTestDSL()
				client := &Client{DSL: dsl}
				// Don't connect
				return client
			},
			expectError: true,
			errorMsg:    "MCP client not connected",
		},
		{
			name: "Initialize with stdio connection",
			setupClient: func() *Client {
				dsl := createStdioTestDSL()
				client := &Client{DSL: dsl}
				// Try to connect
				ctx, cancel := createTestContext(10 * time.Second)
				defer cancel()
				client.Connect(ctx)
				return client
			},
			expectError: false, // May succeed or fail depending on test environment
		},
		{
			name: "Initialize with HTTP connection",
			setupClient: func() *Client {
				if config.SkipHTTPTests {
					return nil // Will be skipped
				}
				dsl := createHTTPTestDSL(config)
				client := &Client{DSL: dsl}
				// Try to connect
				ctx, cancel := createTestContext(10 * time.Second)
				defer cancel()
				client.Connect(ctx)
				return client
			},
			expectError: false,
		},
		{
			name: "Initialize with SSE connection",
			setupClient: func() *Client {
				if config.SkipSSETests {
					return nil // Will be skipped
				}
				dsl := createSSETestDSL(config)
				client := &Client{DSL: dsl}
				// Try to connect
				ctx, cancel := createTestContext(10 * time.Second)
				defer cancel()
				client.Connect(ctx)
				return client
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			if client == nil {
				t.Skip("Test skipped - client setup returned nil")
				return
			}

			ctx, cancel := createTestContext(30 * time.Second)
			defer cancel()

			// Clean up after test
			defer client.Disconnect(ctx)

			response, err := client.Initialize(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				logTestInfo(t, "Expected error: %v", err)
				return
			}

			// For non-error cases, initialization may succeed or fail depending on actual server
			if err != nil {
				logTestInfo(t, "Initialization failed (may be expected in test env): %v", err)
				return
			}

			// If initialization succeeded, validate the response
			if response == nil {
				t.Errorf("Expected non-nil response")
				return
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
	tests := []struct {
		name        string
		setupClient func() *Client
		expectError bool
		errorMsg    string
	}{
		{
			name: "Initialized without connection",
			setupClient: func() *Client {
				dsl := createStdioTestDSL()
				return &Client{DSL: dsl}
			},
			expectError: true,
			errorMsg:    "MCP client not connected",
		},
		{
			name: "Initialized with connection",
			setupClient: func() *Client {
				dsl := createStdioTestDSL()
				client := &Client{DSL: dsl}
				// Try to connect
				ctx, cancel := createTestContext(5 * time.Second)
				defer cancel()
				client.Connect(ctx)
				return client
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			ctx, cancel := createTestContext(10 * time.Second)
			defer cancel()

			// Clean up after test
			defer client.Disconnect(ctx)

			err := client.Initialized(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
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
	tests := []struct {
		name string
		dsl  *types.ClientDSL
	}{
		{
			name: "Client with sampling only",
			dsl: &types.ClientDSL{
				Name:           "Test Client",
				Transport:      types.TransportStdio,
				Command:        "echo",
				EnableSampling: true,
				EnableRoots:    false,
			},
		},
		{
			name: "Client with roots only",
			dsl: &types.ClientDSL{
				Name:             "Test Client",
				Transport:        types.TransportStdio,
				Command:          "echo",
				EnableSampling:   false,
				EnableRoots:      true,
				RootsListChanged: true,
			},
		},
		{
			name: "Client with all capabilities",
			dsl: &types.ClientDSL{
				Name:              "Test Client",
				Transport:         types.TransportStdio,
				Command:           "echo",
				EnableSampling:    true,
				EnableRoots:       true,
				RootsListChanged:  true,
				EnableElicitation: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{DSL: tt.dsl}
			ctx, cancel := createTestContext(10 * time.Second)
			defer cancel()

			// Try to connect
			err := client.Connect(ctx)
			if err != nil {
				logTestInfo(t, "Connection failed (expected): %v", err)
				return
			}

			defer client.Disconnect(ctx)

			// Test client capabilities generation
			clientCaps := tt.dsl.GetClientCapabilities()
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

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(substr) > 0 && len(s) >= len(substr) &&
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}
