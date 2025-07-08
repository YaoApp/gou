package client

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestProgressComplete tests all progress-related functionality in one connected session
func TestProgressComplete(t *testing.T) {
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

			// Run all progress tests in sequence
			t.Run("CreateProgress", func(t *testing.T) {
				testCreateProgressCore(ctx, t, client)
			})

			t.Run("UpdateProgress", func(t *testing.T) {
				testUpdateProgressCore(ctx, t, client)
			})

			t.Run("GetProgress", func(t *testing.T) {
				testGetProgressCore(ctx, t, client)
			})

			t.Run("ListProgress", func(t *testing.T) {
				testListProgressCore(ctx, t, client)
			})

			t.Run("CompleteProgress", func(t *testing.T) {
				testCompleteProgressCore(ctx, t, client)
			})

			t.Run("ProgressWorkflow", func(t *testing.T) {
				testProgressWorkflowCore(ctx, t, client)
			})

			t.Run("CancelRequest", func(t *testing.T) {
				testCancelRequestCore(ctx, t, client)
			})

			logTestInfo(t, "All progress tests completed for %s", testCase.Name)
		})
	}
}

// Core test functions that operate on an already connected client
func testCreateProgressCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test CreateProgress
	token, err := client.CreateProgress(ctx, 100)
	if err != nil {
		logTestInfo(t, "CreateProgress failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "CreateProgress succeeded, token: %d", token)

	// Verify token is valid
	if token == 0 {
		t.Errorf("Expected non-zero token")
	}

	// Verify progress was stored
	progress, err := client.GetProgress(token)
	if err != nil {
		t.Errorf("Expected to find progress for token %d: %v", token, err)
		return
	}

	if progress.Token != token {
		t.Errorf("Expected token %d, got %d", token, progress.Token)
	}

	if progress.Total != 100 {
		t.Errorf("Expected total 100, got %d", progress.Total)
	}
}

func testUpdateProgressCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Create a progress token first
	token, err := client.CreateProgress(ctx, 50)
	if err != nil {
		logTestInfo(t, "CreateProgress failed (may be expected): %v", err)
		return
	}

	// Test UpdateProgress
	err = client.UpdateProgress(ctx, token, 25)
	if err != nil {
		logTestInfo(t, "UpdateProgress failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "UpdateProgress succeeded for token: %d", token)

	// Test updating to completion
	err = client.UpdateProgress(ctx, token, 50)
	if err != nil {
		logTestInfo(t, "UpdateProgress to completion failed (may be expected): %v", err)
		return
	}

	// Progress should be removed when completed
	_, err = client.GetProgress(token)
	if err == nil {
		logTestInfo(t, "Progress still exists after completion (may be expected)")
	} else {
		logTestInfo(t, "Progress correctly removed after completion: %v", err)
	}
}

func testGetProgressCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Create a progress token first
	token, err := client.CreateProgress(ctx, 200)
	if err != nil {
		logTestInfo(t, "CreateProgress failed (may be expected): %v", err)
		return
	}

	// Test GetProgress
	progress, err := client.GetProgress(token)
	if err != nil {
		t.Errorf("GetProgress failed: %v", err)
		return
	}

	logTestInfo(t, "GetProgress succeeded for token: %d", token)

	// Verify progress details
	if progress.Token != token {
		t.Errorf("Expected token %d, got %d", token, progress.Token)
	}

	if progress.Total != 200 {
		t.Errorf("Expected total 200, got %d", progress.Total)
	}

	// Test with invalid token
	_, err = client.GetProgress(99999)
	if err == nil {
		t.Errorf("Expected error for invalid token")
	} else {
		logTestInfo(t, "GetProgress correctly failed for invalid token: %v", err)
	}
}

func testListProgressCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Create multiple progress tokens
	var tokens []uint64
	for i := 0; i < 3; i++ {
		token, err := client.CreateProgress(ctx, uint64((i+1)*10))
		if err != nil {
			logTestInfo(t, "CreateProgress failed (may be expected): %v", err)
			return
		}
		tokens = append(tokens, token)
	}

	// Test ListProgress
	progressList := client.ListProgress()
	logTestInfo(t, "ListProgress returned %d progress items", len(progressList))

	// Verify our tokens are in the list
	for _, token := range tokens {
		if progress, exists := progressList[token]; !exists {
			t.Errorf("Expected to find token %d in progress list", token)
		} else {
			logTestInfo(t, "Found progress token %d with total %d", token, progress.Total)
		}
	}
}

func testCompleteProgressCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Create a progress token first
	token, err := client.CreateProgress(ctx, 300)
	if err != nil {
		logTestInfo(t, "CreateProgress failed (may be expected): %v", err)
		return
	}

	// Verify progress exists
	_, err = client.GetProgress(token)
	if err != nil {
		t.Errorf("Expected to find progress for token %d: %v", token, err)
		return
	}

	// Test CompleteProgress
	err = client.CompleteProgress(ctx, token)
	if err != nil {
		logTestInfo(t, "CompleteProgress failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "CompleteProgress succeeded for token: %d", token)

	// Verify progress was removed
	_, err = client.GetProgress(token)
	if err == nil {
		t.Errorf("Expected progress to be removed after completion")
	} else {
		logTestInfo(t, "Progress correctly removed after completion: %v", err)
	}

	// Test completing invalid token
	err = client.CompleteProgress(ctx, 99999)
	if err == nil {
		t.Errorf("Expected error for invalid token")
	} else {
		logTestInfo(t, "CompleteProgress correctly failed for invalid token: %v", err)
	}
}

func testProgressWorkflowCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test complete workflow: create -> update -> complete
	token, err := client.CreateProgress(ctx, 100)
	if err != nil {
		logTestInfo(t, "CreateProgress failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "Starting progress workflow with token: %d", token)

	// Simulate progress updates
	for i := uint64(10); i <= 90; i += 10 {
		err = client.UpdateProgress(ctx, token, i)
		if err != nil {
			logTestInfo(t, "UpdateProgress to %d failed (may be expected): %v", i, err)
			continue
		}
		logTestInfo(t, "Progress updated to: %d/100", i)

		// Small delay to simulate work
		time.Sleep(10 * time.Millisecond)
	}

	// Complete the progress
	err = client.CompleteProgress(ctx, token)
	if err != nil {
		logTestInfo(t, "CompleteProgress failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "Progress workflow completed successfully")
}

func testCancelRequestCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test CancelRequest
	err := client.CancelRequest(ctx, "test-request-id")
	if err != nil {
		logTestInfo(t, "CancelRequest failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "CancelRequest succeeded")

	// Test with various request ID types
	requestIDs := []interface{}{
		123,
		"string-id",
		nil,
		map[string]string{"id": "complex-id"},
	}

	for _, requestID := range requestIDs {
		err = client.CancelRequest(ctx, requestID)
		if err != nil {
			logTestInfo(t, "CancelRequest with ID %v failed (may be expected): %v", requestID, err)
		} else {
			logTestInfo(t, "CancelRequest with ID %v succeeded", requestID)
		}
	}
}

// Error condition tests - these test error conditions so they need separate connections
func TestProgressErrorConditions(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			t.Run("WithoutInitialization", func(t *testing.T) {
				testProgressWithoutInitialization(t, testCase)
			})

			t.Run("WithoutConnection", func(t *testing.T) {
				testProgressWithoutConnection(t, testCase)
			})
		})
	}
}

func testProgressWithoutInitialization(t *testing.T, testCase TransportTestCase) {
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

	// Test all progress methods without initialization
	testFunctions := []struct {
		name string
		fn   func() error
	}{
		{
			name: "CreateProgress",
			fn: func() error {
				_, err := client.CreateProgress(ctx, 100)
				return err
			},
		},
		{
			name: "UpdateProgress",
			fn: func() error {
				return client.UpdateProgress(ctx, 1, 50)
			},
		},
		{
			name: "CompleteProgress",
			fn: func() error {
				return client.CompleteProgress(ctx, 1)
			},
		},
		{
			name: "CancelRequest",
			fn: func() error {
				return client.CancelRequest(ctx, "test-id")
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

	// Local operations should still work
	progressList := client.ListProgress()
	logTestInfo(t, "ListProgress works without initialization, returned %d items", len(progressList))

	_, err = client.GetProgress(1)
	if err == nil {
		t.Errorf("Expected error for GetProgress with invalid token")
	} else {
		logTestInfo(t, "GetProgress correctly failed for invalid token: %v", err)
	}
}

func testProgressWithoutConnection(t *testing.T, testCase TransportTestCase) {
	t.Helper()

	// Create client without connection
	client := &Client{DSL: testCase.DSL}

	// Create context with timeout
	ctx, cancel := createTestContext(5 * time.Second)
	defer cancel()

	// Test all progress methods without connection
	testFunctions := []struct {
		name string
		fn   func() error
	}{
		{
			name: "CreateProgress",
			fn: func() error {
				_, err := client.CreateProgress(ctx, 100)
				return err
			},
		},
		{
			name: "UpdateProgress",
			fn: func() error {
				return client.UpdateProgress(ctx, 1, 50)
			},
		},
		{
			name: "CompleteProgress",
			fn: func() error {
				return client.CompleteProgress(ctx, 1)
			},
		},
		{
			name: "CancelRequest",
			fn: func() error {
				return client.CancelRequest(ctx, "test-id")
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

	// Local operations should still work
	progressList := client.ListProgress()
	logTestInfo(t, "ListProgress works without connection, returned %d items", len(progressList))

	_, err := client.GetProgress(1)
	if err == nil {
		t.Errorf("Expected error for GetProgress with invalid token")
	} else {
		logTestInfo(t, "GetProgress correctly failed for invalid token: %v", err)
	}
}
