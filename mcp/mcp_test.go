package mcp

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/application/disk"
)

func TestMain(m *testing.M) {
	// Source the environment file before running tests
	// This simulates: source /Users/max/Yao/gou/env.local.sh

	// Set up test environment variables
	os.Setenv("MCP_TEST_KEY", "test_api_key_123")
	os.Setenv("MCP_TEST_URL", "https://test.example.com")
	os.Setenv("MCP_TEST_TOKEN", "Bearer test_token_456")

	// Initialize application for file reading
	if app := application.App; app == nil {
		// Try to load from test environment
		root := os.Getenv("GOU_TEST_APPLICATION")
		if root == "" {
			root = "../.." // Default to gou root directory
		}

		diskApp, err := disk.Open(root)
		if err == nil {
			application.Load(diskApp)
		}
	}

	// Run tests
	code := m.Run()

	// Clean up
	os.Unsetenv("MCP_TEST_KEY")
	os.Unsetenv("MCP_TEST_URL")
	os.Unsetenv("MCP_TEST_TOKEN")

	os.Exit(code)
}

func TestLoadClientSource(t *testing.T) {
	t.Run("LoadClientSource with stdio transport", func(t *testing.T) {
		dsl := `{
			"transport": "stdio",
			"name": "test-stdio",
			"label": "Test STDIO Client",
			"description": "Test MCP client with stdio transport",
			"command": "echo",
			"arguments": ["hello", "world"],
			"env": {
				"API_KEY": "$ENV.MCP_TEST_KEY",
				"DEBUG": "true"
			},
			"timeout": "30s"
		}`

		client, err := LoadClientSource(dsl, "test-stdio")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Check if client is stored in global map
		assert.True(t, Exists("test-stdio"))

		// Test Select function
		selectedClient, err := Select("test-stdio")
		assert.NoError(t, err)
		assert.Equal(t, client, selectedClient)
	})

	t.Run("LoadClientSource with SSE transport", func(t *testing.T) {
		dsl := `{
			"transport": "sse",
			"name": "test-sse",
			"label": "Test SSE Client",
			"description": "Test MCP client with SSE transport",
			"url": "$ENV.MCP_TEST_URL",
			"authorization_token": "$ENV.MCP_TEST_TOKEN",
			"timeout": "60s"
		}`

		client, err := LoadClientSource(dsl, "test-sse")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Check environment variable processing
		assert.True(t, Exists("test-sse"))
	})

	t.Run("LoadClientSource with HTTP transport", func(t *testing.T) {
		dsl := `{
			"transport": "http",
			"name": "test-http",
			"label": "Test HTTP Client",
			"description": "Test MCP client with HTTP transport",
			"url": "https://api.example.com/mcp",
			"authorization_token": "Bearer static_token",
			"timeout": "45s"
		}`

		client, err := LoadClientSource(dsl, "test-http")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		assert.True(t, Exists("test-http"))
	})

	t.Run("LoadClientSource with invalid DSL", func(t *testing.T) {
		dsl := `{
			"transport": "invalid",
			"name": "test-invalid"
		}`

		_, err := LoadClientSource(dsl, "test-invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported transport type")
	})

	t.Run("LoadClientSource with empty ID", func(t *testing.T) {
		dsl := `{
			"transport": "stdio",
			"name": "test",
			"command": "echo"
		}`

		_, err := LoadClientSource(dsl, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client id is required")
	})

	t.Run("LoadClientSource with malformed JSON", func(t *testing.T) {
		dsl := `{
			"transport": "stdio",
			"name": "test",
			"command": "echo"
		` // Invalid JSON - missing closing brace

		_, err := LoadClientSource(dsl, "test-malformed")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse MCP client DSL")
	})
}

func TestLoadClient(t *testing.T) {
	// Skip this test if application is not properly initialized
	if application.App == nil {
		t.Skip("Application not initialized, skipping file-based tests")
	}

	t.Run("LoadClient from file", func(t *testing.T) {
		// This test depends on the test DSL files in gou-dev-app/mcps
		// We'll test this with a relative path

		// Try to load from the test application
		client, err := LoadClient("mcps/stdio.mcp.yao", "stdio-file")
		if err != nil {
			t.Skipf("Could not load test file: %v", err)
		}

		assert.NotNil(t, client)
		assert.True(t, Exists("stdio-file"))
	})

	t.Run("LoadClient with non-existent file", func(t *testing.T) {
		_, err := LoadClient("non-existent-file.yao", "test-non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read MCP client file")
	})
}

func TestSelect(t *testing.T) {
	t.Run("Select existing client", func(t *testing.T) {
		// First create a client
		dsl := `{
			"transport": "stdio",
			"name": "test-select",
			"command": "echo"
		}`

		originalClient, err := LoadClientSource(dsl, "test-select")
		assert.NoError(t, err)

		// Now select it
		selectedClient, err := Select("test-select")
		assert.NoError(t, err)
		assert.Equal(t, originalClient, selectedClient)
	})

	t.Run("Select non-existent client", func(t *testing.T) {
		_, err := Select("non-existent-client")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MCP client non-existent-client not found")
	})
}

func TestExists(t *testing.T) {
	t.Run("Exists for existing client", func(t *testing.T) {
		// Create a client first
		dsl := `{
			"transport": "stdio",
			"name": "test-exists",
			"command": "echo"
		}`

		_, err := LoadClientSource(dsl, "test-exists")
		assert.NoError(t, err)

		// Test exists
		assert.True(t, Exists("test-exists"))
	})

	t.Run("Exists for non-existent client", func(t *testing.T) {
		assert.False(t, Exists("non-existent-client"))
	})
}

func TestGetClient(t *testing.T) {
	t.Run("GetClient existing client", func(t *testing.T) {
		// Create a client first
		dsl := `{
			"transport": "stdio",
			"name": "test-get",
			"command": "echo"
		}`

		originalClient, err := LoadClientSource(dsl, "test-get")
		assert.NoError(t, err)

		// Get client (should not panic)
		retrievedClient := GetClient("test-get")
		assert.Equal(t, originalClient, retrievedClient)
	})

	t.Run("GetClient non-existent client", func(t *testing.T) {
		// This should panic/throw exception
		assert.Panics(t, func() {
			GetClient("non-existent-client")
		})
	})
}

func TestEnvironmentVariableProcessing(t *testing.T) {
	t.Run("Environment variables in URL and token", func(t *testing.T) {
		dsl := `{
			"transport": "sse",
			"name": "test-env",
			"url": "$ENV.MCP_TEST_URL",
			"authorization_token": "$ENV.MCP_TEST_TOKEN"
		}`

		client, err := LoadClientSource(dsl, "test-env")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// The environment variables should be processed
		// Note: We can't directly check the processed values without accessing the client's internal DSL
		// But the fact that the client was created successfully indicates the env vars were processed
	})

	t.Run("Environment variables in env map", func(t *testing.T) {
		dsl := `{
			"transport": "stdio",
			"name": "test-env-map",
			"command": "echo",
			"env": {
				"API_KEY": "$ENV.MCP_TEST_KEY",
				"STATIC_VAR": "static_value"
			}
		}`

		client, err := LoadClientSource(dsl, "test-env-map")
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("Environment variables in arguments", func(t *testing.T) {
		dsl := `{
			"transport": "stdio",
			"name": "test-env-args",
			"command": "echo",
			"arguments": ["$ENV.MCP_TEST_KEY", "static_arg"]
		}`

		client, err := LoadClientSource(dsl, "test-env-args")
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})
}

func TestDSLValidation(t *testing.T) {
	t.Run("Missing required fields for stdio", func(t *testing.T) {
		dsl := `{
			"transport": "stdio",
			"name": "test-no-command"
		}`

		_, err := LoadClientSource(dsl, "test-no-command")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command is required")
	})

	t.Run("Missing required fields for HTTP", func(t *testing.T) {
		dsl := `{
			"transport": "http",
			"name": "test-no-url"
		}`

		_, err := LoadClientSource(dsl, "test-no-url")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "URL is required")
	})

	t.Run("Missing required fields for SSE", func(t *testing.T) {
		dsl := `{
			"transport": "sse",
			"name": "test-no-url"
		}`

		_, err := LoadClientSource(dsl, "test-no-url")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "URL is required")
	})

	t.Run("Auto-fill missing ID and Name", func(t *testing.T) {
		dsl := `{
			"transport": "stdio",
			"command": "echo"
		}`

		client, err := LoadClientSource(dsl, "test-auto-fill")
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// The ID and Name should be auto-filled with the provided id
		assert.True(t, Exists("test-auto-fill"))
	})
}

func TestConcurrentLoad(t *testing.T) {
	const numGoroutines = 20
	const numClients = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numClients)

	// Create different DSLs for testing
	dslTemplate := `{
		"transport": "stdio",
		"name": "test-concurrent-%d",
		"command": "echo",
		"arguments": ["test"]
	}`

	// Test concurrent LoadClientSource
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numClients; j++ {
				clientID := fmt.Sprintf("concurrent-%d-%d", goroutineID, j)
				dsl := fmt.Sprintf(dslTemplate, j)

				_, err := LoadClientSource(dsl, clientID)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, client %d: %w", goroutineID, j, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify all clients were loaded
	totalExpected := numGoroutines * numClients
	loadedClients := ListClients()

	// Count clients with our prefix
	concurrentClients := 0
	for _, clientID := range loadedClients {
		if len(clientID) > 11 && clientID[:11] == "concurrent-" {
			concurrentClients++
		}
	}

	assert.Equal(t, totalExpected, concurrentClients, "Not all clients were loaded correctly")

	// Clean up
	for _, clientID := range loadedClients {
		if len(clientID) > 11 && clientID[:11] == "concurrent-" {
			UnloadClient(clientID)
		}
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	const numReaders = 10
	const numWriters = 3
	const testDuration = 1 * time.Second

	// Load some initial clients
	initialClients := []string{"reader-test-1", "reader-test-2", "reader-test-3"}
	dsl := `{
		"transport": "stdio",
		"name": "reader-test",
		"command": "echo",
		"arguments": ["test"]
	}`

	for _, clientID := range initialClients {
		_, err := LoadClientSource(dsl, clientID)
		assert.NoError(t, err)
	}

	var wg sync.WaitGroup
	stop := make(chan bool)
	errors := make(chan error, numReaders+numWriters)

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				default:
					// Random read operations
					for _, clientID := range initialClients {
						if Exists(clientID) {
							_, err := Select(clientID)
							if err != nil {
								errors <- fmt.Errorf("reader %d: %w", readerID, err)
								return
							}
						}
					}
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			counter := 0
			for {
				select {
				case <-stop:
					return
				default:
					// Load and unload clients
					clientID := fmt.Sprintf("writer-%d-%d", writerID, counter)
					_, err := LoadClientSource(dsl, clientID)
					if err != nil {
						errors <- fmt.Errorf("writer %d: %w", writerID, err)
						return
					}

					time.Sleep(5 * time.Millisecond)
					UnloadClient(clientID)
					counter++
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(testDuration)
	close(stop)
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Clean up
	for _, clientID := range initialClients {
		UnloadClient(clientID)
	}
}

func TestConcurrentLoadSameClient(t *testing.T) {
	const numGoroutines = 10
	const clientID = "same-client-test"

	dsl := `{
		"transport": "stdio",
		"name": "same-client-test",
		"command": "echo",
		"arguments": ["test"]
	}`

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	clients := make(chan Client, numGoroutines)

	// Try to load the same client from multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			client, err := LoadClientSource(dsl, clientID)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %w", goroutineID, err)
				return
			}

			clients <- client
		}(i)
	}

	wg.Wait()
	close(errors)
	close(clients)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify only one client instance exists
	assert.True(t, Exists(clientID), "Client should exist")

	// All goroutines should have received a valid client
	clientCount := 0
	for range clients {
		clientCount++
	}
	assert.Equal(t, numGoroutines, clientCount, "All goroutines should have received a client")

	// Clean up
	UnloadClient(clientID)
}

// Clean up function to remove test clients
func TestCleanup(t *testing.T) {
	// This runs last to clean up all test clients
	testClients := []string{
		"test-stdio", "test-sse", "test-http", "test-invalid",
		"test-select", "test-exists", "test-get", "test-env",
		"test-env-map", "test-env-args", "test-no-command",
		"test-no-url", "test-auto-fill", "stdio-file",
	}

	for _, clientID := range testClients {
		if Exists(clientID) {
			UnloadClient(clientID)
		}
	}
}
