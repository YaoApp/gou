package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/gou/mcp/client"
	"github.com/yaoapp/gou/mcp/types"
)

// TestClientInitialization tests the complete client flow from DSL to initialization
func TestClientInitialization(t *testing.T) {
	fmt.Println("=== MCP Client Initialization Test ===")

	// Step 1: Create test DSL
	fmt.Println("\n1. Creating test DSL...")
	dsl := &types.ClientDSL{
		ID:          "test-client-001",
		Name:        "Test MCP Client",
		Version:     "1.0.0",
		Transport:   types.TransportStdio,
		Label:       "Test Client",
		Description: "MCP client for testing",

		// Client capability configuration
		EnableSampling:    true,
		EnableRoots:       true,
		RootsListChanged:  true,
		EnableElicitation: false,

		// Stdio transport configuration
		Command:   "echo", // Using echo command for testing (this will fail, but we can test the flow)
		Arguments: []string{"hello"},
		Env: map[string]string{
			"TEST_ENV": "test_value",
		},

		Timeout: "30s",
	}

	// Print DSL configuration
	dslJSON, _ := json.MarshalIndent(dsl, "", "  ")
	fmt.Printf("DSL Configuration:\n%s\n", string(dslJSON))

	// Step 2: Create client instance
	fmt.Println("\n2. Creating client instance...")
	mcpClient, err := client.New(dsl)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	fmt.Printf("✅ Client created successfully\n")

	// Step 3: Check client state
	fmt.Println("\n3. Checking client state...")
	fmt.Printf("Is Connected: %v\n", mcpClient.IsConnected())
	fmt.Printf("Connection State: %v\n", mcpClient.State())

	// Step 4: Test DSL methods
	fmt.Println("\n4. Testing DSL methods...")
	fmt.Printf("Client Name: %s\n", dsl.Name)
	fmt.Printf("Client Version: %s\n", dsl.GetVersion())
	fmt.Printf("Client Envs: %v\n", dsl.GetEnvs())
	fmt.Printf("Client Timeout: %v\n", dsl.GetTimeout())
	fmt.Printf("Client Auth Token: %s\n", dsl.GetAuthorizationToken())

	// Test Implementation information
	impl := dsl.GetImplementation()
	fmt.Printf("Implementation: Name=%s, Version=%s\n", impl.Name, impl.Version)

	// Test client capabilities
	caps := dsl.GetClientCapabilities()
	fmt.Printf("Client Capabilities:\n")
	fmt.Printf("  Sampling: %v\n", caps.Sampling != nil)
	fmt.Printf("  Roots: %v\n", caps.Roots != nil)
	if caps.Roots != nil {
		fmt.Printf("    ListChanged: %v\n", caps.Roots.ListChanged)
	}
	fmt.Printf("  Elicitation: %v\n", caps.Elicitation != nil)

	// Step 5: Attempt to connect (this might fail because echo is not a real MCP server)
	fmt.Println("\n5. Attempting to connect...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with custom headers
	connectOptions := types.ConnectionOptions{
		Headers: map[string]string{
			"Mcp-Session-Id":   "test-session-123",
			"X-Client-Version": "1.0.0",
			"X-Custom-Header":  "test-value",
		},
		Timeout: 30 * time.Second,
	}

	fmt.Printf("Connection options: %+v\n", connectOptions)

	connectErr := mcpClient.Connect(ctx, connectOptions)
	if connectErr != nil {
		fmt.Printf("❌ Connection failed (expected): %v\n", connectErr)
		fmt.Printf("This is expected since 'echo' is not a real MCP server\n")

		// Test connection without options as well
		fmt.Println("\n5b. Attempting to connect without options...")
		connectErr2 := mcpClient.Connect(ctx)
		if connectErr2 != nil {
			fmt.Printf("❌ Connection without options also failed (expected): %v\n", connectErr2)
		}
	} else {
		fmt.Printf("✅ Connection successful\n")
		fmt.Printf("Is Connected: %v\n", mcpClient.IsConnected())
		fmt.Printf("Connection State: %v\n", mcpClient.State())

		// Step 6: Attempt to initialize (only execute if connection successful)
		fmt.Println("\n6. Attempting to initialize...")
		initResult, initErr := mcpClient.Initialize(ctx)
		if initErr != nil {
			fmt.Printf("❌ Initialization failed: %v\n", initErr)
		} else {
			fmt.Printf("✅ Initialization successful\n")
			fmt.Printf("Protocol Version: %s\n", initResult.ProtocolVersion)
			fmt.Printf("Server Info: Name=%s, Version=%s\n",
				initResult.ServerInfo.Name, initResult.ServerInfo.Version)

			// Print server capabilities
			fmt.Printf("Server Capabilities:\n")
			if initResult.Capabilities.Resources != nil {
				fmt.Printf("  Resources: Subscribe=%v, ListChanged=%v\n",
					initResult.Capabilities.Resources.Subscribe,
					initResult.Capabilities.Resources.ListChanged)
			}
			if initResult.Capabilities.Tools != nil {
				fmt.Printf("  Tools: ListChanged=%v\n",
					initResult.Capabilities.Tools.ListChanged)
			}
			if initResult.Capabilities.Prompts != nil {
				fmt.Printf("  Prompts: ListChanged=%v\n",
					initResult.Capabilities.Prompts.ListChanged)
			}
		}

		// Step 7: Call Initialized
		fmt.Println("\n7. Calling Initialized...")
		initializedErr := mcpClient.Initialized(ctx)
		if initializedErr != nil {
			fmt.Printf("❌ Initialized failed: %v\n", initializedErr)
		} else {
			fmt.Printf("✅ Initialized successful\n")
		}

		// Step 8: Cleanup connection
		fmt.Println("\n8. Cleaning up connection...")
		disconnectErr := mcpClient.Disconnect(ctx)
		if disconnectErr != nil {
			fmt.Printf("❌ Disconnect failed: %v\n", disconnectErr)
		} else {
			fmt.Printf("✅ Disconnect successful\n")
		}
	}

	fmt.Println("\n=== Test Complete ===")
}

// TestClientDSLValidation tests DSL validation
func TestClientDSLValidation(t *testing.T) {
	fmt.Println("\n=== DSL Validation Test ===")

	// Test invalid DSL configurations
	testCases := []struct {
		name string
		dsl  *types.ClientDSL
	}{
		{
			name: "Empty DSL",
			dsl:  nil,
		},
		{
			name: "Missing Name",
			dsl: &types.ClientDSL{
				Transport: types.TransportStdio,
				Command:   "echo",
			},
		},
		{
			name: "Missing Command for Stdio",
			dsl: &types.ClientDSL{
				Name:      "Test Client",
				Transport: types.TransportStdio,
			},
		},
		{
			name: "Missing URL for HTTP",
			dsl: &types.ClientDSL{
				Name:      "Test Client",
				Transport: types.TransportHTTP,
			},
		},
		{
			name: "Valid Stdio DSL",
			dsl: &types.ClientDSL{
				Name:           "Test Client",
				Transport:      types.TransportStdio,
				Command:        "echo",
				EnableSampling: true,
			},
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\nTesting: %s\n", tc.name)
		_, err := client.New(tc.dsl)
		if err != nil {
			fmt.Printf("❌ Validation failed: %v\n", err)
		} else {
			fmt.Printf("✅ Validation passed\n")
		}
	}
}

// TestConnectionOptions tests connection options functionality
func TestConnectionOptions(t *testing.T) {
	fmt.Println("\n=== Connection Options Test ===")

	// Create test DSL for HTTP transport
	dsl := &types.ClientDSL{
		Name:           "Test HTTP Client",
		Version:        "1.0.0",
		Transport:      types.TransportHTTP,
		URL:            "http://localhost:8080/mcp", // This will fail but we can test the flow
		EnableSampling: true,
	}

	fmt.Printf("Testing with HTTP transport DSL: %s\n", dsl.URL)

	// Create client instance
	mcpClient, err := client.New(dsl)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	fmt.Printf("✅ Client created successfully\n")

	// Test various connection options
	testCases := []struct {
		name    string
		options types.ConnectionOptions
	}{
		{
			name:    "No Options",
			options: types.ConnectionOptions{},
		},
		{
			name: "With Session ID",
			options: types.ConnectionOptions{
				Headers: map[string]string{
					"Mcp-Session-Id": "session-abc-123",
				},
			},
		},
		{
			name: "With Multiple Headers",
			options: types.ConnectionOptions{
				Headers: map[string]string{
					"Mcp-Session-Id":   "session-xyz-456",
					"X-Client-Version": "1.0.0",
					"X-Request-ID":     "req-12345",
					"X-Custom-Auth":    "bearer custom-token",
				},
				Timeout: 15 * time.Second,
			},
		},
		{
			name: "With Custom Timeout",
			options: types.ConnectionOptions{
				Headers: map[string]string{
					"Mcp-Session-Id": "session-timeout-test",
				},
				Timeout:    5 * time.Second,
				MaxRetries: 3,
				RetryDelay: 1 * time.Second,
			},
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		fmt.Printf("\nTesting: %s\n", tc.name)
		fmt.Printf("Options: %+v\n", tc.options)

		connectErr := mcpClient.Connect(ctx, tc.options)
		if connectErr != nil {
			fmt.Printf("❌ Connection failed (expected): %v\n", connectErr)
		} else {
			fmt.Printf("✅ Connection successful\n")
			// Clean up if connected
			mcpClient.Disconnect(ctx)
		}
	}

	fmt.Println("\n=== Connection Options Test Complete ===")
}
