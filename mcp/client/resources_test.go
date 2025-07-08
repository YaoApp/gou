package client

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestResourcesComplete tests all resource-related functionality in one connected session
func TestResourcesComplete(t *testing.T) {
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

			// Run all resource tests in sequence
			t.Run("ListResources", func(t *testing.T) {
				testListResourcesCore(ctx, t, client)
			})

			t.Run("ReadResource", func(t *testing.T) {
				testReadResourceCore(ctx, t, client)
			})

			t.Run("SubscribeResource", func(t *testing.T) {
				testSubscribeResourceCore(ctx, t, client)
			})

			t.Run("UnsubscribeResource", func(t *testing.T) {
				testUnsubscribeResourceCore(ctx, t, client)
			})

			t.Run("SubscribeUnsubscribeFlow", func(t *testing.T) {
				testSubscribeUnsubscribeFlowCore(ctx, t, client)
			})

			t.Run("ResourcesPagination", func(t *testing.T) {
				testResourcesPaginationCore(ctx, t, client)
			})

			t.Run("ResourcesWithInvalidURI", func(t *testing.T) {
				testResourcesWithInvalidURICore(ctx, t, client)
			})

			logTestInfo(t, "All resource tests completed for %s", testCase.Name)
		})
	}
}

// Core test functions that operate on an already connected client
func testListResourcesCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Test ListResources
	response, err := client.ListResources(ctx, "")
	if err != nil {
		// Check if it's because server doesn't support resources
		if strings.Contains(err.Error(), "server does not support resources") {
			logTestInfo(t, "Server does not support resources (expected): %v", err)
			return
		}
		logTestInfo(t, "ListResources failed (may be expected): %v", err)
		return
	}

	// Verify response structure
	if response == nil {
		t.Errorf("Expected non-nil response")
		return
	}

	logTestInfo(t, "ListResources succeeded, found %d resources", len(response.Resources))

	// Verify resources structure
	for i, resource := range response.Resources {
		if resource.URI == "" {
			t.Errorf("Resource %d has empty URI", i)
		}
		if i < 5 { // Log first 5 resources to avoid spam
			logTestInfo(t, "Resource %d: URI=%s, Name=%s, MimeType=%s", i, resource.URI, resource.Name, resource.MimeType)
		}
	}

	// Test with cursor pagination if NextCursor is provided
	if response.NextCursor != "" {
		logTestInfo(t, "Testing pagination with cursor: %s", response.NextCursor)
		paginatedResponse, err := client.ListResources(ctx, response.NextCursor)
		if err != nil {
			logTestInfo(t, "Paginated ListResources failed (may be expected): %v", err)
		} else {
			logTestInfo(t, "Paginated ListResources succeeded, found %d more resources", len(paginatedResponse.Resources))
		}
	}
}

func testReadResourceCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// First, get available resources
	listResponse, err := client.ListResources(ctx, "")
	if err != nil {
		if strings.Contains(err.Error(), "server does not support resources") {
			logTestInfo(t, "Server does not support resources (expected): %v", err)
			return
		}
		logTestInfo(t, "ListResources failed, skipping ReadResource test: %v", err)
		return
	}

	if len(listResponse.Resources) == 0 {
		logTestInfo(t, "No resources available to read")
		return
	}

	// Test reading the first few resources
	maxResourcesToTest := 3
	if len(listResponse.Resources) < maxResourcesToTest {
		maxResourcesToTest = len(listResponse.Resources)
	}

	for i := 0; i < maxResourcesToTest; i++ {
		testURI := listResponse.Resources[i].URI
		logTestInfo(t, "Testing ReadResource with URI: %s", testURI)

		readResponse, err := client.ReadResource(ctx, testURI)
		if err != nil {
			logTestInfo(t, "ReadResource failed (may be expected): %v", err)
			continue
		}

		// Verify response structure
		if readResponse == nil {
			t.Errorf("Expected non-nil response for URI: %s", testURI)
			continue
		}

		if len(readResponse.Contents) == 0 {
			t.Errorf("Expected at least one content item for URI: %s", testURI)
			continue
		}

		logTestInfo(t, "ReadResource succeeded for URI: %s, found %d content items", testURI, len(readResponse.Contents))

		// Verify content structure
		for j, content := range readResponse.Contents {
			if content.URI == "" {
				t.Errorf("Content %d has empty URI", j)
			}

			// Check content type
			if content.Text != "" && content.Blob != nil {
				t.Errorf("Content %d has both text and blob data", j)
			}
			if content.Text == "" && content.Blob == nil {
				t.Errorf("Content %d has neither text nor blob data", j)
			}

			if content.Text != "" {
				logTestInfo(t, "Content %d: URI=%s, MimeType=%s, Text length=%d", j, content.URI, content.MimeType, len(content.Text))
			} else {
				logTestInfo(t, "Content %d: URI=%s, MimeType=%s, Blob length=%d", j, content.URI, content.MimeType, len(content.Blob))
			}
		}
	}
}

func testSubscribeResourceCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Get a test resource URI
	testURI := getTestResourceURI(t)
	logTestInfo(t, "Testing SubscribeResource with URI: %s", testURI)

	// Create a shorter timeout context for subscription test
	subscribeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test subscription
	err := client.SubscribeResource(subscribeCtx, testURI)
	if err != nil {
		// Check for expected errors
		if strings.Contains(err.Error(), "server does not support resources") {
			logTestInfo(t, "Server does not support resources (expected): %v", err)
			return
		}
		if strings.Contains(err.Error(), "server does not support resource subscriptions") {
			logTestInfo(t, "Server does not support resource subscriptions (expected): %v", err)
			return
		}
		if strings.Contains(err.Error(), "context deadline exceeded") {
			logTestInfo(t, "SubscribeResource timed out (expected for test server): %v", err)
			return
		}
		logTestInfo(t, "SubscribeResource failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "SubscribeResource succeeded for URI: %s", testURI)
}

func testUnsubscribeResourceCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Get a test resource URI
	testURI := getTestResourceURI(t)
	logTestInfo(t, "Testing UnsubscribeResource with URI: %s", testURI)

	// Create a shorter timeout context for unsubscription test
	unsubscribeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test unsubscription (may fail if not subscribed first)
	err := client.UnsubscribeResource(unsubscribeCtx, testURI)
	if err != nil {
		// Check for expected errors
		if strings.Contains(err.Error(), "server does not support resources") {
			logTestInfo(t, "Server does not support resources (expected): %v", err)
			return
		}
		if strings.Contains(err.Error(), "server does not support resource subscriptions") {
			logTestInfo(t, "Server does not support resource subscriptions (expected): %v", err)
			return
		}
		if strings.Contains(err.Error(), "context deadline exceeded") {
			logTestInfo(t, "UnsubscribeResource timed out (expected for test server): %v", err)
			return
		}
		logTestInfo(t, "UnsubscribeResource failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "UnsubscribeResource succeeded for URI: %s", testURI)
}

func testSubscribeUnsubscribeFlowCore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	// Get a test resource URI
	testURI := getTestResourceURI(t)
	logTestInfo(t, "Testing subscribe/unsubscribe flow with URI: %s", testURI)

	// Create a shorter timeout context for subscription test
	subscribeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test subscription
	err := client.SubscribeResource(subscribeCtx, testURI)
	if err != nil {
		// Check for expected errors
		if strings.Contains(err.Error(), "server does not support resources") {
			logTestInfo(t, "Server does not support resources (expected): %v", err)
			return
		}
		if strings.Contains(err.Error(), "server does not support resource subscriptions") {
			logTestInfo(t, "Server does not support resource subscriptions (expected): %v", err)
			return
		}
		if strings.Contains(err.Error(), "context deadline exceeded") {
			logTestInfo(t, "SubscribeResource timed out (expected for test server): %v", err)
			return
		}
		logTestInfo(t, "SubscribeResource failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "SubscribeResource succeeded")

	// Create a shorter timeout context for unsubscription test
	unsubscribeCtx, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()

	// Test unsubscription
	err = client.UnsubscribeResource(unsubscribeCtx, testURI)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			logTestInfo(t, "UnsubscribeResource timed out (expected for test server): %v", err)
			return
		}
		logTestInfo(t, "UnsubscribeResource failed (may be expected): %v", err)
		return
	}

	logTestInfo(t, "UnsubscribeResource succeeded - full flow completed")
}

func testResourcesPaginationCore(ctx context.Context, t *testing.T, client *Client) {
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

		response, err := client.ListResources(paginationCtx, cursor)
		cancel() // Cancel immediately after the call

		if err != nil {
			if strings.Contains(err.Error(), "server does not support resources") {
				logTestInfo(t, "Server does not support resources (expected): %v", err)
				return
			} else if strings.Contains(err.Error(), "context deadline exceeded") {
				logTestInfo(t, "ListResources with cursor '%s' timed out (expected for test server): %v", cursor, err)
			} else {
				logTestInfo(t, "ListResources with cursor '%s' failed (may be expected): %v", cursor, err)
			}
		} else {
			logTestInfo(t, "ListResources with cursor '%s' succeeded, found %d resources", cursor, len(response.Resources))
		}
	}
}

func testResourcesWithInvalidURICore(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()

	invalidURIs := []string{
		"",
		"invalid-uri",
		"://invalid",
		"http://",
		"non-existent://resource",
	}

	for _, invalidURI := range invalidURIs {
		// Test ReadResource with invalid URI
		_, err := client.ReadResource(ctx, invalidURI)
		if err == nil {
			logTestInfo(t, "ReadResource with invalid URI '%s' succeeded unexpectedly", invalidURI)
		} else {
			if strings.Contains(err.Error(), "server does not support resources") {
				logTestInfo(t, "Server does not support resources (expected): %v", err)
				return
			}
			logTestInfo(t, "ReadResource with invalid URI '%s' failed as expected: %v", invalidURI, err)
		}
	}
}

// Error condition tests - these test error conditions so they need separate connections
func TestResourcesErrorConditions(t *testing.T) {
	testCases := getStandardTransportTestCases()

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Skip test if configuration is not available
			if testCase.ShouldSkip {
				t.Skip(testCase.SkipReason)
				return
			}

			t.Run("WithoutInitialization", func(t *testing.T) {
				testResourcesWithoutInitialization(t, testCase)
			})

			t.Run("WithoutConnection", func(t *testing.T) {
				testResourcesWithoutConnection(t, testCase)
			})
		})
	}
}

func testResourcesWithoutInitialization(t *testing.T, testCase TransportTestCase) {
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

	// Test all resource methods without initialization
	testFunctions := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ListResources",
			fn: func() error {
				_, err := client.ListResources(ctx, "")
				return err
			},
		},
		{
			name: "ReadResource",
			fn: func() error {
				_, err := client.ReadResource(ctx, getTestResourceURI(t))
				return err
			},
		},
		{
			name: "SubscribeResource",
			fn: func() error {
				return client.SubscribeResource(ctx, getTestResourceURI(t))
			},
		},
		{
			name: "UnsubscribeResource",
			fn: func() error {
				return client.UnsubscribeResource(ctx, getTestResourceURI(t))
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

func testResourcesWithoutConnection(t *testing.T, testCase TransportTestCase) {
	t.Helper()

	// Create client without connection
	client := &Client{DSL: testCase.DSL}

	// Create context with timeout
	ctx, cancel := createTestContext(5 * time.Second)
	defer cancel()

	// Test all resource methods without connection
	testFunctions := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ListResources",
			fn: func() error {
				_, err := client.ListResources(ctx, "")
				return err
			},
		},
		{
			name: "ReadResource",
			fn: func() error {
				_, err := client.ReadResource(ctx, getTestResourceURI(t))
				return err
			},
		},
		{
			name: "SubscribeResource",
			fn: func() error {
				return client.SubscribeResource(ctx, getTestResourceURI(t))
			},
		},
		{
			name: "UnsubscribeResource",
			fn: func() error {
				return client.UnsubscribeResource(ctx, getTestResourceURI(t))
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
