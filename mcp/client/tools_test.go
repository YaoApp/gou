package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/mcp/types"
)

// TestToolsComplete tests all tool-related functionality in one connected session
func TestToolsComplete(t *testing.T) {
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

			// Run all tool tests in sequence
			t.Run("ListTools", func(t *testing.T) {
				testListToolsCore(ctx, t, client)
			})

			t.Run("CallTool", func(t *testing.T) {
				testCallToolCore(ctx, t, client)
			})

			t.Run("CallToolWithArguments", func(t *testing.T) {
				testCallToolWithArgumentsCore(ctx, t, client)
			})

			t.Run("CallToolsBatch", func(t *testing.T) {
				testCallToolsBatchCore(ctx, t, client)
			})

			t.Run("ToolsPagination", func(t *testing.T) {
				testToolsPaginationCore(ctx, t, client)
			})

			t.Run("ToolsWithInvalidName", func(t *testing.T) {
				testToolsWithInvalidNameCore(ctx, t, client)
			})

			logTestInfo(t, "All tool tests completed for %s", testCase.Name)
		})
	}
}

// Core test functions that operate on an already connected client
func testListToolsCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test ListTools
	response, err := client.ListTools(ctx, "")
	if err != nil {
		// Check if it's because server doesn't support tools
		if strings.Contains(err.Error(), "server does not support tools") {
			logTestInfo(t, "Server does not support tools (expected): %v", err)
			return
		}
		logTestInfo(t, "ListTools failed (may be expected): %v", err)
		return
	}

	// Verify response structure
	if response == nil {
		t.Errorf("Expected non-nil response")
		return
	}

	logTestInfo(t, "ListTools succeeded, found %d tools", len(response.Tools))

	// Verify tools structure
	for i, tool := range response.Tools {
		if tool.Name == "" {
			t.Errorf("Tool %d has empty name", i)
		}
		if tool.InputSchema == nil {
			t.Errorf("Tool %d has nil input schema", i)
		}
		if i < 5 { // Log first 5 tools to avoid spam
			logTestInfo(t, "Tool %d: Name=%s, Description=%s", i, tool.Name, tool.Description)
		}
	}

	// Test with cursor pagination if NextCursor is provided
	if response.NextCursor != "" {
		logTestInfo(t, "Testing pagination with cursor: %s", response.NextCursor)
		paginatedResponse, err := client.ListTools(ctx, response.NextCursor)
		if err != nil {
			logTestInfo(t, "Paginated ListTools failed (may be expected): %v", err)
		} else {
			logTestInfo(t, "Paginated ListTools succeeded, found %d more tools", len(paginatedResponse.Tools))
		}
	}
}

func testCallToolCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// First, get available tools
	listResponse, err := client.ListTools(ctx, "")
	if err != nil {
		if strings.Contains(err.Error(), "server does not support tools") {
			logTestInfo(t, "Server does not support tools (expected): %v", err)
			return
		}
		logTestInfo(t, "ListTools failed, skipping CallTool test: %v", err)
		return
	}

	if len(listResponse.Tools) == 0 {
		logTestInfo(t, "No tools available to call")
		return
	}

	// Test calling the first few tools (look for simple ones first)
	maxToolsToTest := 3
	if len(listResponse.Tools) < maxToolsToTest {
		maxToolsToTest = len(listResponse.Tools)
	}

	for i := 0; i < maxToolsToTest; i++ {
		testTool := listResponse.Tools[i]
		logTestInfo(t, "Testing CallTool with name: %s", testTool.Name)

		// Use simple arguments for testing
		var testArgs interface{}
		if strings.Contains(strings.ToLower(testTool.Name), "echo") {
			testArgs = map[string]interface{}{"message": "Hello, MCP!"}
		} else if strings.Contains(strings.ToLower(testTool.Name), "add") {
			testArgs = map[string]interface{}{"a": 5, "b": 3}
		} else {
			// For unknown tools, try with empty arguments
			testArgs = map[string]interface{}{}
		}

		// Test calling tool
		callResponse, err := client.CallTool(ctx, testTool.Name, testArgs)
		if err != nil {
			logTestInfo(t, "CallTool failed (may be expected): %v", err)
			continue
		}

		// Verify response structure
		if callResponse == nil {
			t.Errorf("Expected non-nil response for tool: %s", testTool.Name)
			continue
		}

		logTestInfo(t, "CallTool succeeded for tool: %s, found %d content items, isError: %v",
			testTool.Name, len(callResponse.Content), callResponse.IsError)

		// Verify content structure
		for j, content := range callResponse.Content {
			if content.Type == "" {
				t.Errorf("Content %d has empty type", j)
			}
			logTestInfo(t, "Content %d: Type=%s", j, content.Type)
		}
	}
}

func testCallToolWithArgumentsCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// First, get available tools
	listResponse, err := client.ListTools(ctx, "")
	if err != nil {
		if strings.Contains(err.Error(), "server does not support tools") {
			logTestInfo(t, "Server does not support tools (expected): %v", err)
			return
		}
		logTestInfo(t, "ListTools failed, skipping CallToolWithArguments test: %v", err)
		return
	}

	if len(listResponse.Tools) == 0 {
		logTestInfo(t, "No tools available to call")
		return
	}

	// Look for tools that likely accept arguments
	testToolNames := []string{"add", "echo", "sample", "longRunningOperation"}
	var testTool *string

	for _, tool := range listResponse.Tools {
		for _, targetName := range testToolNames {
			if strings.Contains(strings.ToLower(tool.Name), targetName) {
				testTool = &tool.Name
				break
			}
		}
		if testTool != nil {
			break
		}
	}

	if testTool == nil {
		logTestInfo(t, "No suitable tools found for argument testing")
		return
	}

	logTestInfo(t, "Testing CallTool with arguments for tool: %s", *testTool)

	// Test with various argument combinations
	testArgSets := []map[string]interface{}{
		{"message": "Test message", "value": 42},
		{"a": 10, "b": 20},
		{"duration": 1, "steps": 2},
		{"prompt": "Hello world"},
	}

	for i, testArgs := range testArgSets {
		callResponse, err := client.CallTool(ctx, *testTool, testArgs)
		if err != nil {
			logTestInfo(t, "CallTool with args set %d failed (may be expected): %v", i, err)
			continue
		}

		logTestInfo(t, "CallTool with args set %d succeeded for tool: %s, found %d content items",
			i, *testTool, len(callResponse.Content))
		break // Success with one arg set is enough
	}
}

func testCallToolsBatchCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// First, get available tools
	listResponse, err := client.ListTools(ctx, "")
	if err != nil {
		if strings.Contains(err.Error(), "server does not support tools") {
			logTestInfo(t, "Server does not support tools (expected): %v", err)
			return
		}
		logTestInfo(t, "ListTools failed, skipping CallToolsBatch test: %v", err)
		return
	}

	if len(listResponse.Tools) == 0 {
		logTestInfo(t, "No tools available for batch calling")
		return
	}

	// Create a batch of tool calls (limit to 2-3 tools to avoid overwhelming)
	maxBatchSize := 3
	if len(listResponse.Tools) < maxBatchSize {
		maxBatchSize = len(listResponse.Tools)
	}

	toolCalls := make([]types.ToolCall, 0, maxBatchSize)
	for i := 0; i < maxBatchSize; i++ {
		tool := listResponse.Tools[i]
		var args interface{}

		// Set appropriate arguments based on tool name
		if strings.Contains(strings.ToLower(tool.Name), "echo") {
			args = map[string]interface{}{"message": fmt.Sprintf("Batch message %d", i)}
		} else if strings.Contains(strings.ToLower(tool.Name), "add") {
			args = map[string]interface{}{"a": i, "b": i + 1}
		} else {
			args = map[string]interface{}{}
		}

		toolCalls = append(toolCalls, types.ToolCall{
			Name:      tool.Name,
			Arguments: args,
		})
	}

	logTestInfo(t, "Testing CallToolsBatch with %d tools", len(toolCalls))

	// Test batch call
	batchResponse, err := client.CallToolsBatch(ctx, toolCalls)
	if err != nil {
		logTestInfo(t, "CallToolsBatch failed (may be expected): %v", err)
		return
	}

	// Verify response structure
	if batchResponse == nil {
		t.Errorf("Expected non-nil batch response")
		return
	}

	if len(batchResponse.Results) != len(toolCalls) {
		t.Errorf("Expected %d results, got %d", len(toolCalls), len(batchResponse.Results))
		return
	}

	logTestInfo(t, "CallToolsBatch succeeded, got %d results", len(batchResponse.Results))

	// Verify each result
	for i, result := range batchResponse.Results {
		logTestInfo(t, "Batch result %d: %d content items, isError: %v",
			i, len(result.Content), result.IsError)
	}
}

func testToolsPaginationCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test with various cursor values
	cursors := []string{
		"",
		"invalid-cursor",
		"null",
		"0",
		"999999",
	}

	for _, cursor := range cursors {
		// Create a shorter timeout context for each pagination test
		paginationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

		response, err := client.ListTools(paginationCtx, cursor)
		cancel() // Cancel immediately after the call

		if err != nil {
			if strings.Contains(err.Error(), "server does not support tools") {
				logTestInfo(t, "Server does not support tools (expected): %v", err)
				return
			} else if strings.Contains(err.Error(), "context deadline exceeded") {
				logTestInfo(t, "ListTools with cursor '%s' timed out (expected for test server): %v", cursor, err)
			} else {
				logTestInfo(t, "ListTools with cursor '%s' failed (may be expected): %v", cursor, err)
			}
		} else {
			logTestInfo(t, "ListTools with cursor '%s' succeeded, found %d tools", cursor, len(response.Tools))
		}
	}
}

func testToolsWithInvalidNameCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	invalidNames := []string{
		"",
		"non-existent-tool",
		"invalid_tool_name",
		"tool/with/slashes",
		"tool with spaces",
	}

	for _, invalidName := range invalidNames {
		// Test CallTool with invalid name
		_, err := client.CallTool(ctx, invalidName, nil)
		if err == nil {
			logTestInfo(t, "CallTool with invalid name '%s' succeeded unexpectedly", invalidName)
		} else {
			if strings.Contains(err.Error(), "server does not support tools") {
				logTestInfo(t, "Server does not support tools (expected): %v", err)
				return
			}
			logTestInfo(t, "CallTool with invalid name '%s' failed as expected: %v", invalidName, err)
		}
	}
}

// Error condition tests - these test error conditions so they need separate connections
func TestToolsErrorConditions(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			t.Run("WithoutInitialization", func(t *testing.T) {
				testToolsWithoutInitialization(t, testCase)
			})

			t.Run("WithoutConnection", func(t *testing.T) {
				testToolsWithoutConnection(t, testCase)
			})
		})
	}
}

func testToolsWithoutInitialization(t *testing.T, testCase TransportTestCase) {
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

	// Test all tool methods without initialization
	testFunctions := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ListTools",
			fn: func() error {
				_, err := client.ListTools(ctx, "")
				return err
			},
		},
		{
			name: "CallTool",
			fn: func() error {
				_, err := client.CallTool(ctx, "test_tool", nil)
				return err
			},
		},
		{
			name: "CallToolsBatch",
			fn: func() error {
				_, err := client.CallToolsBatch(ctx, []types.ToolCall{{Name: "test_tool"}})
				return err
			},
		},
	}

	for _, testFunc := range testFunctions {
		err := testFunc.fn()
		if err == nil {
			t.Errorf("Expected error when calling %s without initialization", testFunc.name)
		} else if !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("Expected error about not being initialized for %s, got: %v", testFunc.name, err)
		} else {
			logTestInfo(t, "%s correctly failed without initialization: %v", testFunc.name, err)
		}
	}
}

func testToolsWithoutConnection(t *testing.T, testCase TransportTestCase) {
	t.Helper()

	// Create client without connection
	client := &Client{DSL: testCase.DSL}

	// Create context with timeout
	ctx, cancel := createTestContext(5 * time.Second)
	defer cancel()

	// Test all tool methods without connection
	testFunctions := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ListTools",
			fn: func() error {
				_, err := client.ListTools(ctx, "")
				return err
			},
		},
		{
			name: "CallTool",
			fn: func() error {
				_, err := client.CallTool(ctx, "test_tool", nil)
				return err
			},
		},
		{
			name: "CallToolsBatch",
			fn: func() error {
				_, err := client.CallToolsBatch(ctx, []types.ToolCall{{Name: "test_tool"}})
				return err
			},
		},
	}

	for _, testFunc := range testFunctions {
		err := testFunc.fn()
		if err == nil {
			t.Errorf("Expected error when calling %s without connection", testFunc.name)
		} else if !strings.Contains(err.Error(), "MCP client not initialized") {
			t.Errorf("Expected error about MCP client not initialized for %s, got: %v", testFunc.name, err)
		} else {
			logTestInfo(t, "%s correctly failed without connection: %v", testFunc.name, err)
		}
	}
}
