package client

import (
	"context"
	"strings"
	"testing"

	"github.com/yaoapp/gou/mcp/types"
)

// TestLoggingComplete tests all logging-related functionality in one connected session
func TestLoggingComplete(t *testing.T) {
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

			// Connect once
			err := client.Connect(ctx)
			if err != nil {
				logTestInfo(t, "Connection failed (expected): %v", err)
				return
			}
			defer client.Disconnect(ctx)

			// Initialize once
			_, err = client.Initialize(ctx)
			if err != nil {
				logTestInfo(t, "Initialization failed (expected): %v", err)
				return
			}

			// Run all logging tests in sequence
			t.Run("SetLogLevel", func(t *testing.T) {
				testSetLogLevelCore(ctx, t, client)
			})

			t.Run("GetLogLevel", func(t *testing.T) {
				testGetLogLevelCore(ctx, t, client)
			})

			t.Run("SetLogLevelVariousLevels", func(t *testing.T) {
				testSetLogLevelVariousLevelsCore(ctx, t, client)
			})

			logTestInfo(t, "All logging tests completed for %s", testCase.Name)
		})
	}
}

// Core test functions that operate on an already connected client
func testSetLogLevelCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test SetLogLevel
	err := client.SetLogLevel(ctx, types.LogLevelDebug)
	if err != nil {
		// Check if it's because server doesn't support logging
		if strings.Contains(err.Error(), "server does not support logging") {
			logTestInfo(t, "Server does not support logging (expected): %v", err)
			return
		}
		logTestInfo(t, "SetLogLevel failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "SetLogLevel succeeded")

	// Verify the log level was set
	currentLevel := client.GetLogLevel()
	if currentLevel != types.LogLevelDebug {
		t.Errorf("Expected log level %s, got %s", types.LogLevelDebug, currentLevel)
	}
}

func testGetLogLevelCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test GetLogLevel
	currentLevel := client.GetLogLevel()
	logTestInfo(t, "Current log level: %s", currentLevel)

	// Log level should be a valid value
	validLevels := []types.LogLevel{
		types.LogLevelDebug,
		types.LogLevelInfo,
		types.LogLevelNotice,
		types.LogLevelWarning,
		types.LogLevelError,
		types.LogLevelCritical,
		types.LogLevelAlert,
		types.LogLevelEmergency,
	}

	found := false
	for _, level := range validLevels {
		if currentLevel == level {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Invalid log level: %s", currentLevel)
	}
}

func testSetLogLevelVariousLevelsCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test various log levels
	logLevels := []types.LogLevel{
		types.LogLevelError,
		types.LogLevelWarning,
		types.LogLevelInfo,
		types.LogLevelDebug,
		types.LogLevelCritical,
		types.LogLevelAlert,
		types.LogLevelEmergency,
		types.LogLevelNotice,
	}

	for _, level := range logLevels {
		err := client.SetLogLevel(ctx, level)
		if err != nil {
			if strings.Contains(err.Error(), "server does not support logging") {
				logTestInfo(t, "Server does not support logging (expected): %v", err)
				return
			}
			logTestInfo(t, "SetLogLevel to %s failed (may be expected): %v", level, err)
			continue
		}

		// Verify the log level was set
		currentLevel := client.GetLogLevel()
		if currentLevel != level {
			t.Errorf("Expected log level %s, got %s", level, currentLevel)
		}

		logTestInfo(t, "Successfully set log level to: %s", level)
	}
}

// Error condition tests - these test error conditions so they need separate connections
func TestLoggingErrorConditions(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			t.Run("WithoutInitialization", func(t *testing.T) {
				testLoggingWithoutInitialization(t, testCase)
			})

			t.Run("WithoutConnection", func(t *testing.T) {
				testLoggingWithoutConnection(t, testCase)
			})
		})
	}
}

func testLoggingWithoutInitialization(t *testing.T, testCase TransportTestCase) {
	t.Helper()

	// Create client
	client := &Client{DSL: testCase.DSL}

	// Create context with timeout
	ctx, cancel := createTestContext(testCase.Timeout)
	defer cancel()

	// Connect but don't initialize
	err := client.Connect(ctx)
	if err != nil {
		logTestInfo(t, "Connection failed (expected): %v", err)
		return
	}
	defer client.Disconnect(ctx)

	// Test SetLogLevel without initialization
	err = client.SetLogLevel(ctx, types.LogLevelDebug)
	if err == nil {
		t.Errorf("Expected error when calling SetLogLevel without initialization")
	} else if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("Expected error about not being initialized for SetLogLevel, got: %v", err)
	} else {
		logTestInfo(t, "SetLogLevel correctly failed without initialization: %v", err)
	}

	// GetLogLevel should still work as it's a local operation
	level := client.GetLogLevel()
	if level == "" {
		t.Errorf("GetLogLevel should return a default value even without initialization")
	} else {
		logTestInfo(t, "GetLogLevel works without initialization, returned: %s", level)
	}
}

func testLoggingWithoutConnection(t *testing.T, testCase TransportTestCase) {
	t.Helper()

	// Create client without connection
	client := &Client{DSL: testCase.DSL}

	// Create context with timeout
	ctx, cancel := createTestContext(5)
	defer cancel()

	// Test SetLogLevel without connection
	err := client.SetLogLevel(ctx, types.LogLevelDebug)
	if err == nil {
		t.Errorf("Expected error when calling SetLogLevel without connection")
	} else if !strings.Contains(err.Error(), "MCP client not initialized") {
		t.Errorf("Expected error about MCP client not initialized for SetLogLevel, got: %v", err)
	} else {
		logTestInfo(t, "SetLogLevel correctly failed without connection: %v", err)
	}

	// GetLogLevel should still work as it's a local operation
	level := client.GetLogLevel()
	if level == "" {
		t.Errorf("GetLogLevel should return a default value even without connection")
	} else {
		logTestInfo(t, "GetLogLevel works without connection, returned: %s", level)
	}
}
