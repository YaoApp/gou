package client

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/yaoapp/gou/mcp/types"
	gouTypes "github.com/yaoapp/gou/types"
)

// Test environment variable names
const (
	EnvHTTPURL   = "MCP_CLIENT_TEST_HTTP_URL"
	EnvHTTPToken = "MCP_CLIENT_TEST_HTTP_AUTHORIZATION_TOKEN"
	EnvSSEURL    = "MCP_CLIENT_TEST_SSE_URL"
	EnvSSEToken  = "MCP_CLIENT_TEST_SSE_AUTHORIZATION_TOKEN"
)

// TestConfig holds test configuration
type TestConfig struct {
	HTTPUrl        string
	HTTPToken      string
	SSEUrl         string
	SSEToken       string
	SkipHTTPTests  bool
	SkipSSETests   bool
	SkipStdioTests bool
}

// getTestConfig reads test configuration from environment variables
func getTestConfig() *TestConfig {
	config := &TestConfig{
		HTTPUrl:   os.Getenv(EnvHTTPURL),
		HTTPToken: os.Getenv(EnvHTTPToken),
		SSEUrl:    os.Getenv(EnvSSEURL),
		SSEToken:  os.Getenv(EnvSSEToken),
	}

	// Determine which tests to skip based on available configuration
	config.SkipHTTPTests = config.HTTPUrl == ""
	config.SkipSSETests = config.SSEUrl == ""
	config.SkipStdioTests = false // stdio tests can always run with echo

	return config
}

// createHTTPTestDSL creates a test DSL for HTTP transport
func createHTTPTestDSL(config *TestConfig) *types.ClientDSL {
	return &types.ClientDSL{
		ID:        "test-http-client",
		Name:      "Test HTTP MCP Client",
		Version:   "1.0.0",
		Transport: types.TransportHTTP,
		MetaInfo: gouTypes.MetaInfo{
			Label:       "HTTP Test Client",
			Description: "MCP client for HTTP testing",
		},
		URL:                config.HTTPUrl,
		AuthorizationToken: config.HTTPToken,
		EnableSampling:     true,
		EnableRoots:        true,
		RootsListChanged:   true,
		Timeout:            "30s",
	}
}

// createSSETestDSL creates a test DSL for SSE transport
func createSSETestDSL(config *TestConfig) *types.ClientDSL {
	return &types.ClientDSL{
		ID:        "test-sse-client",
		Name:      "Test SSE MCP Client",
		Version:   "1.0.0",
		Transport: types.TransportSSE,
		MetaInfo: gouTypes.MetaInfo{
			Label:       "SSE Test Client",
			Description: "MCP client for SSE testing",
		},
		URL:                config.SSEUrl,
		AuthorizationToken: config.SSEToken,
		EnableSampling:     true,
		EnableRoots:        true,
		RootsListChanged:   true,
		Timeout:            "30s",
	}
}

// createStdioTestDSL creates a test DSL for stdio transport
func createStdioTestDSL() *types.ClientDSL {
	return &types.ClientDSL{
		ID:        "test-stdio-client",
		Name:      "Test Stdio MCP Client",
		Version:   "1.0.0",
		Transport: types.TransportStdio,
		MetaInfo: gouTypes.MetaInfo{
			Label:       "Stdio Test Client",
			Description: "MCP client for stdio testing",
		},
		Command:          "npx",
		Arguments:        []string{"-y", "@modelcontextprotocol/server-everything"},
		EnableSampling:   true,
		EnableRoots:      false,
		RootsListChanged: false,
		Env: map[string]string{
			"TEST_MODE": "true",
			"MCP_DEBUG": "false",
		},
		Timeout: "10s",
	}
}

// createTestContext creates a context with timeout for testing
func createTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

// skipTestIfNoConfig skips the test if required configuration is not available
func skipTestIfNoConfig(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Skip(message)
	}
}

// TestConnectionOptions provides common connection options for testing
type TestConnectionOptions struct {
	WithSessionID     bool
	WithCustomHeaders bool
	WithTimeout       bool
	SessionID         string
	CustomHeaders     map[string]string
	ConnectionTimeout time.Duration
}

// createConnectionOptions creates connection options based on test configuration
func createConnectionOptions(opts TestConnectionOptions) types.ConnectionOptions {
	connOpts := types.ConnectionOptions{
		Headers: make(map[string]string),
	}

	if opts.WithSessionID {
		sessionID := opts.SessionID
		if sessionID == "" {
			sessionID = "test-session-" + generateRandomID(8)
		}
		connOpts.Headers["Mcp-Session-Id"] = sessionID
	}

	if opts.WithCustomHeaders {
		for key, value := range opts.CustomHeaders {
			connOpts.Headers[key] = value
		}
	}

	if opts.WithTimeout {
		timeout := opts.ConnectionTimeout
		if timeout == 0 {
			timeout = 15 * time.Second
		}
		connOpts.Timeout = timeout
	}

	return connOpts
}

// generateRandomID generates a random ID for testing
func generateRandomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

// assertClientState asserts the client state matches expected value
func assertClientState(t *testing.T, client *Client, expectedConnected bool, expectedState types.ConnectionState) {
	t.Helper()

	if client.IsConnected() != expectedConnected {
		t.Errorf("Expected IsConnected() = %v, got %v", expectedConnected, client.IsConnected())
	}

	if client.State() != expectedState {
		t.Errorf("Expected State() = %v, got %v", expectedState, client.State())
	}

	// Additional checks for initialization state
	switch expectedState {
	case types.StateInitialized:
		if !client.IsInitialized() {
			t.Errorf("Expected IsInitialized() = true for state %v", expectedState)
		}
		if client.GetInitResult() == nil {
			t.Errorf("Expected GetInitResult() to be non-nil for state %v", expectedState)
		}
	case types.StateConnected:
		if client.IsInitialized() {
			t.Errorf("Expected IsInitialized() = false for state %v", expectedState)
		}
		if client.GetInitResult() != nil {
			t.Errorf("Expected GetInitResult() to be nil for state %v", expectedState)
		}
	case types.StateDisconnected:
		if client.IsInitialized() {
			t.Errorf("Expected IsInitialized() = false for state %v", expectedState)
		}
		if client.GetInitResult() != nil {
			t.Errorf("Expected GetInitResult() to be nil for state %v", expectedState)
		}
	}
}

// logTestInfo logs test information for debugging
func logTestInfo(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	if testing.Verbose() {
		t.Logf(format, args...)
	}
}

// TestDSLInfo holds information about test DSL for logging
type TestDSLInfo struct {
	Transport   types.TransportType
	Name        string
	URL         string
	HasToken    bool
	Command     string
	EnabledCaps []string
}

// getDSLInfo extracts information from DSL for logging
func getDSLInfo(dsl *types.ClientDSL) TestDSLInfo {
	info := TestDSLInfo{
		Transport: dsl.Transport,
		Name:      dsl.Name,
		URL:       dsl.URL,
		HasToken:  dsl.AuthorizationToken != "",
		Command:   dsl.Command,
	}

	// Collect enabled capabilities
	if dsl.EnableSampling {
		info.EnabledCaps = append(info.EnabledCaps, "sampling")
	}
	if dsl.EnableRoots {
		info.EnabledCaps = append(info.EnabledCaps, "roots")
	}
	if dsl.EnableElicitation {
		info.EnabledCaps = append(info.EnabledCaps, "elicitation")
	}

	return info
}

// printTestConfig prints test configuration for debugging
func printTestConfig(t *testing.T, config *TestConfig) {
	t.Helper()
	if !testing.Verbose() {
		return
	}

	t.Logf("Test Configuration:")
	t.Logf("  HTTP URL: %s", maskSensitive(config.HTTPUrl))
	t.Logf("  HTTP Token: %s", maskToken(config.HTTPToken))
	t.Logf("  SSE URL: %s", maskSensitive(config.SSEUrl))
	t.Logf("  SSE Token: %s", maskToken(config.SSEToken))
	t.Logf("  Skip HTTP Tests: %v", config.SkipHTTPTests)
	t.Logf("  Skip SSE Tests: %v", config.SkipSSETests)
	t.Logf("  Skip Stdio Tests: %v", config.SkipStdioTests)
}

// maskSensitive masks sensitive information for logging
func maskSensitive(value string) string {
	if value == "" {
		return "<not set>"
	}
	if len(value) <= 10 {
		return "***"
	}
	return value[:5] + "***" + value[len(value)-2:]
}

// maskToken masks token for logging
func maskToken(token string) string {
	if token == "" {
		return "<not set>"
	}
	return "***"
}

// getTestResourceURI returns a test resource URI
func getTestResourceURI(t *testing.T) string {
	t.Helper()
	// Return a generic test URI that should work with most MCP servers
	return "test://example/resource"
}

// TransportTestCase represents a test case for a specific transport type
type TransportTestCase struct {
	Name          string
	Transport     types.TransportType
	DSL           *types.ClientDSL
	ShouldSkip    bool
	SkipReason    string
	ExpectError   bool
	ExpectedError string
	Timeout       time.Duration
}

// getStandardTransportTestCases returns standard test cases for all transport types
func getStandardTransportTestCases() []TransportTestCase {
	config := getTestConfig()
	return []TransportTestCase{
		{
			Name:        "HTTP Transport",
			Transport:   types.TransportHTTP,
			DSL:         createHTTPTestDSL(config),
			ShouldSkip:  config.SkipHTTPTests,
			SkipReason:  "HTTP test configuration not available",
			ExpectError: false,
			Timeout:     30 * time.Second,
		},
		{
			Name:        "SSE Transport",
			Transport:   types.TransportSSE,
			DSL:         createSSETestDSL(config),
			ShouldSkip:  config.SkipSSETests,
			SkipReason:  "SSE test configuration not available",
			ExpectError: false,
			Timeout:     30 * time.Second,
		},
		{
			Name:        "STDIO Transport",
			Transport:   types.TransportStdio,
			DSL:         createStdioTestDSL(),
			ShouldSkip:  config.SkipStdioTests,
			SkipReason:  "STDIO test configuration not available",
			ExpectError: false,
			Timeout:     30 * time.Second,
		},
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
