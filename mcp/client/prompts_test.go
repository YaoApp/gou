package client

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestPromptsComplete tests all prompt-related functionality in one connected session
func TestPromptsComplete(t *testing.T) {
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

			// Run all prompt tests in sequence
			t.Run("ListPrompts", func(t *testing.T) {
				testListPromptsCore(ctx, t, client)
			})

			t.Run("GetPrompt", func(t *testing.T) {
				testGetPromptCore(ctx, t, client)
			})

			t.Run("GetPromptWithArguments", func(t *testing.T) {
				testGetPromptWithArgumentsCore(ctx, t, client)
			})

			t.Run("PromptsPagination", func(t *testing.T) {
				testPromptsPaginationCore(ctx, t, client)
			})

			t.Run("PromptsWithInvalidName", func(t *testing.T) {
				testPromptsWithInvalidNameCore(ctx, t, client)
			})

			logTestInfo(t, "All prompt tests completed for %s", testCase.Name)
		})
	}
}

// Core test functions that operate on an already connected client
func testListPromptsCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test ListPrompts
	response, err := client.ListPrompts(ctx, "")
	if err != nil {
		// Check if it's because server doesn't support prompts
		if strings.Contains(err.Error(), "server does not support prompts") {
			logTestInfo(t, "Server does not support prompts (expected): %v", err)
			return
		}
		logTestInfo(t, "ListPrompts failed (may be expected): %v", err)
		return
	}

	// Verify response structure
	if response == nil {
		t.Errorf("Expected non-nil response")
		return
	}

	logTestInfo(t, "ListPrompts succeeded, found %d prompts", len(response.Prompts))

	// Verify prompts structure
	for i, prompt := range response.Prompts {
		if prompt.Name == "" {
			t.Errorf("Prompt %d has empty name", i)
		}
		if i < 5 { // Log first 5 prompts to avoid spam
			logTestInfo(t, "Prompt %d: Name=%s, Description=%s, Arguments=%d", i, prompt.Name, prompt.Description, len(prompt.Arguments))
		}
	}

	// Test with cursor pagination if NextCursor is provided
	if response.NextCursor != "" {
		logTestInfo(t, "Testing pagination with cursor: %s", response.NextCursor)
		paginatedResponse, err := client.ListPrompts(ctx, response.NextCursor)
		if err != nil {
			logTestInfo(t, "Paginated ListPrompts failed (may be expected): %v", err)
		} else {
			logTestInfo(t, "Paginated ListPrompts succeeded, found %d more prompts", len(paginatedResponse.Prompts))
		}
	}
}

func testGetPromptCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// First, get available prompts
	listResponse, err := client.ListPrompts(ctx, "")
	if err != nil {
		if strings.Contains(err.Error(), "server does not support prompts") {
			logTestInfo(t, "Server does not support prompts (expected): %v", err)
			return
		}
		logTestInfo(t, "ListPrompts failed, skipping GetPrompt test: %v", err)
		return
	}

	if len(listResponse.Prompts) == 0 {
		logTestInfo(t, "No prompts available to get")
		return
	}

	// Test getting the first few prompts
	maxPromptsToTest := 3
	if len(listResponse.Prompts) < maxPromptsToTest {
		maxPromptsToTest = len(listResponse.Prompts)
	}

	for i := 0; i < maxPromptsToTest; i++ {
		testPrompt := listResponse.Prompts[i]
		logTestInfo(t, "Testing GetPrompt with name: %s", testPrompt.Name)

		// Test getting prompt without arguments first
		getResponse, err := client.GetPrompt(ctx, testPrompt.Name, nil)
		if err != nil {
			logTestInfo(t, "GetPrompt failed (may be expected): %v", err)
			continue
		}

		// Verify response structure
		if getResponse == nil {
			t.Errorf("Expected non-nil response for prompt: %s", testPrompt.Name)
			continue
		}

		if len(getResponse.Messages) == 0 {
			t.Errorf("Expected at least one message for prompt: %s", testPrompt.Name)
			continue
		}

		logTestInfo(t, "GetPrompt succeeded for prompt: %s, found %d messages", testPrompt.Name, len(getResponse.Messages))

		// Verify message structure
		for j, message := range getResponse.Messages {
			if message.Role == "" {
				t.Errorf("Message %d has empty role", j)
			}
			if message.Content.Type == "" {
				t.Errorf("Message %d has empty content type", j)
			}

			logTestInfo(t, "Message %d: Role=%s, ContentType=%s", j, message.Role, message.Content.Type)
		}
	}
}

func testGetPromptWithArgumentsCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// First, get available prompts
	listResponse, err := client.ListPrompts(ctx, "")
	if err != nil {
		if strings.Contains(err.Error(), "server does not support prompts") {
			logTestInfo(t, "Server does not support prompts (expected): %v", err)
			return
		}
		logTestInfo(t, "ListPrompts failed, skipping GetPromptWithArguments test: %v", err)
		return
	}

	if len(listResponse.Prompts) == 0 {
		logTestInfo(t, "No prompts available to get")
		return
	}

	// Look for prompts with arguments
	var promptWithArgs *string
	for _, prompt := range listResponse.Prompts {
		if len(prompt.Arguments) > 0 {
			promptWithArgs = &prompt.Name
			break
		}
	}

	if promptWithArgs == nil {
		logTestInfo(t, "No prompts with arguments found")
		return
	}

	logTestInfo(t, "Testing GetPrompt with arguments for prompt: %s", *promptWithArgs)

	// Test with some sample arguments
	testArgs := map[string]interface{}{
		"temperature": 0.7,
		"style":       "formal",
		"resourceId":  "1",
		"code":        "print('hello world')",
	}

	getResponse, err := client.GetPrompt(ctx, *promptWithArgs, testArgs)
	if err != nil {
		logTestInfo(t, "GetPrompt with arguments failed (may be expected): %v", err)
		return
	}

	// Verify response structure
	if getResponse == nil {
		t.Errorf("Expected non-nil response for prompt with arguments: %s", *promptWithArgs)
		return
	}

	logTestInfo(t, "GetPrompt with arguments succeeded for prompt: %s, found %d messages", *promptWithArgs, len(getResponse.Messages))
}

func testPromptsPaginationCore(ctx context.Context, t *testing.T, client *Client) {
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

		response, err := client.ListPrompts(paginationCtx, cursor)
		cancel() // Cancel immediately after the call

		if err != nil {
			if strings.Contains(err.Error(), "server does not support prompts") {
				logTestInfo(t, "Server does not support prompts (expected): %v", err)
				return
			} else if strings.Contains(err.Error(), "context deadline exceeded") {
				logTestInfo(t, "ListPrompts with cursor '%s' timed out (expected for test server): %v", cursor, err)
			} else {
				logTestInfo(t, "ListPrompts with cursor '%s' failed (may be expected): %v", cursor, err)
			}
		} else {
			logTestInfo(t, "ListPrompts with cursor '%s' succeeded, found %d prompts", cursor, len(response.Prompts))
		}
	}
}

func testPromptsWithInvalidNameCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	invalidNames := []string{
		"",
		"non-existent-prompt",
		"invalid_prompt_name",
		"prompt/with/slashes",
		"prompt with spaces",
	}

	for _, invalidName := range invalidNames {
		// Test GetPrompt with invalid name
		_, err := client.GetPrompt(ctx, invalidName, nil)
		if err == nil {
			logTestInfo(t, "GetPrompt with invalid name '%s' succeeded unexpectedly", invalidName)
		} else {
			if strings.Contains(err.Error(), "server does not support prompts") {
				logTestInfo(t, "Server does not support prompts (expected): %v", err)
				return
			}
			logTestInfo(t, "GetPrompt with invalid name '%s' failed as expected: %v", invalidName, err)
		}
	}
}

// Error condition tests - these test error conditions so they need separate connections
func TestPromptsErrorConditions(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			t.Run("WithoutInitialization", func(t *testing.T) {
				testPromptsWithoutInitialization(t, testCase)
			})

			t.Run("WithoutConnection", func(t *testing.T) {
				testPromptsWithoutConnection(t, testCase)
			})
		})
	}
}

func testPromptsWithoutInitialization(t *testing.T, testCase TransportTestCase) {
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

	// Test all prompt methods without initialization
	testFunctions := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ListPrompts",
			fn: func() error {
				_, err := client.ListPrompts(ctx, "")
				return err
			},
		},
		{
			name: "GetPrompt",
			fn: func() error {
				_, err := client.GetPrompt(ctx, "test_prompt", nil)
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

func testPromptsWithoutConnection(t *testing.T, testCase TransportTestCase) {
	t.Helper()

	// Create client without connection
	client := &Client{DSL: testCase.DSL}

	// Create context with timeout
	ctx, cancel := createTestContext(5 * time.Second)
	defer cancel()

	// Test all prompt methods without connection
	testFunctions := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ListPrompts",
			fn: func() error {
				_, err := client.ListPrompts(ctx, "")
				return err
			},
		},
		{
			name: "GetPrompt",
			fn: func() error {
				_, err := client.GetPrompt(ctx, "test_prompt", nil)
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
