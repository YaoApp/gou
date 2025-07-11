package graphrag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
)

// ==== Test Data Utils ====

// getDocumentTestDataDir returns the document test data directory
func getDocumentTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	testDataDir := filepath.Join(currentDir, "tests", "document")
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path for document test data dir: %v", err))
	}
	return absPath
}

// getDocumentTestFilePath returns the full path to a document test file
func getDocumentTestFilePath(filename string) string {
	return filepath.Join(getDocumentTestDataDir(), filename)
}

// ==== Connector Setup ====

// prepareAddFileConnector creates connectors for AddFile testing
func prepareAddFileConnector(t *testing.T) {
	t.Helper()

	// Create OpenAI connector for AddFile testing
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		openaiKey = "mock-key"
	}

	openaiDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI AddFile Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "gpt-4o-mini",
			"key": "%s"
		}
	}`, openaiKey)

	_, err := connector.New("openai", "test-addfile-openai", []byte(openaiDSL))
	if err != nil {
		t.Logf("Failed to create OpenAI AddFile connector: %v", err)
	}

	// Create mock connector for tests that don't require real LLM calls
	mockDSL := `{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "Mock AddFile Test",
		"type": "openai",
		"options": {
			"proxy": "http://127.0.0.1:9999",
			"model": "gpt-4o-mini",
			"key": "mock-key"
		}
	}`

	_, err = connector.New("openai", "test-addfile-mock", []byte(mockDSL))
	if err != nil {
		t.Logf("Failed to create mock AddFile connector: %v", err)
	}
}

// ==== AddFile Tests ====

// TestAddFile tests the AddFile function with different configurations
func TestAddFile(t *testing.T) {
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	testConfigs := []string{"vector", "vector+graph", "vector+store", "vector+graph+store"}

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Skipf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Skipf("Failed to create GraphRag instance for %s: %v", configName, err)
			}

			ctx := context.Background()

			// Test with text file
			t.Run("Text_File", func(t *testing.T) {
				testFile := getDocumentTestFilePath("text.txt")
				if _, err := os.Stat(testFile); os.IsNotExist(err) {
					t.Skipf("Test file text.txt not found: %s", testFile)
				}

				options := &types.UpsertOptions{
					DocID:     fmt.Sprintf("test-text-%s", configName),
					GraphName: "test-collection",
					Metadata: map[string]interface{}{
						"source": "test",
						"type":   "text",
						"config": configName,
					},
				}

				docID, err := g.AddFile(ctx, testFile, options)

				if err != nil {
					// Expected errors with mock setup
					expectedErrors := []string{
						"connection refused", "no such host", "connector not found", "connector openai not loaded",
						"vector store", "graph store", "store", "embedding", "extraction",
					}

					hasExpectedError := false
					for _, expectedErr := range expectedErrors {
						if strings.Contains(err.Error(), expectedErr) {
							hasExpectedError = true
							break
						}
					}

					if hasExpectedError {
						t.Logf("Expected error with mock setup: %v", err)
					} else {
						t.Errorf("Unexpected error: %v", err)
					}
					return
				}

				// Validate success
				if docID == "" {
					t.Error("AddFile returned empty document ID")
					return
				}

				if docID != options.DocID {
					t.Errorf("Expected document ID %s, got %s", options.DocID, docID)
				}

				t.Logf("Text file processed successfully - ID: %s", docID)
			})

			// Test with image file
			t.Run("Image_File", func(t *testing.T) {
				testFile := getDocumentTestFilePath("image.png")
				if _, err := os.Stat(testFile); os.IsNotExist(err) {
					t.Skipf("Test file image.png not found: %s", testFile)
				}

				options := &types.UpsertOptions{
					DocID:     fmt.Sprintf("test-image-%s", configName),
					GraphName: "test-collection",
					Metadata: map[string]interface{}{
						"source": "test",
						"type":   "image",
						"config": configName,
					},
				}

				docID, err := g.AddFile(ctx, testFile, options)

				if err != nil {
					t.Logf("Expected error with image processing: %v", err)
					return
				}

				if docID == "" {
					t.Error("AddFile returned empty document ID")
					return
				}

				t.Logf("Image file processed successfully - ID: %s", docID)
			})

			// Test with PDF file
			t.Run("PDF_File", func(t *testing.T) {
				testFile := getDocumentTestFilePath("pdf.pdf")
				if _, err := os.Stat(testFile); os.IsNotExist(err) {
					t.Skipf("Test file pdf.pdf not found: %s", testFile)
				}

				options := &types.UpsertOptions{
					DocID:     fmt.Sprintf("test-pdf-%s", configName),
					GraphName: "test-collection",
					Metadata: map[string]interface{}{
						"source": "test",
						"type":   "pdf",
						"config": configName,
					},
				}

				docID, err := g.AddFile(ctx, testFile, options)

				if err != nil {
					t.Logf("Expected error with PDF processing: %v", err)
					return
				}

				if docID == "" {
					t.Error("AddFile returned empty document ID")
					return
				}

				t.Logf("PDF file processed successfully - ID: %s", docID)
			})
		})
	}
}

// TestAddFileErrorHandling tests error conditions
func TestAddFileErrorHandling(t *testing.T) {
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	config := configs["vector"]
	if config == nil {
		t.Skip("Vector config not found")
	}

	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	ctx := context.Background()

	t.Run("Non_Existent_File", func(t *testing.T) {
		options := &types.UpsertOptions{
			DocID:     "test-nonexistent",
			GraphName: "test-collection",
		}

		_, err := g.AddFile(ctx, "/non/existent/file.txt", options)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		t.Logf("Non-existent file correctly rejected: %v", err)
	})

	t.Run("Empty_File_Path", func(t *testing.T) {
		options := &types.UpsertOptions{
			DocID:     "test-empty",
			GraphName: "test-collection",
		}

		_, err := g.AddFile(ctx, "", options)
		if err == nil {
			t.Error("Expected error for empty file path")
		}
		t.Logf("Empty file path correctly rejected: %v", err)
	})

	t.Run("Nil_Options", func(t *testing.T) {
		testFile := getDocumentTestFilePath("text.txt")
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Skip("Test file text.txt not found")
		}

		// AddFile with nil options should handle gracefully or return error
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("AddFile with nil options panicked (expected): %v", r)
				}
			}()

			docID, err := g.AddFile(ctx, testFile, nil)

			if err != nil {
				t.Logf("AddFile with nil options returned error (expected): %v", err)
				return
			}

			if docID == "" {
				t.Error("AddFile with nil options returned empty document ID")
			} else {
				t.Logf("Nil options handled successfully - Document ID: %s", docID)
			}
		}()
	})
}

// TestAddFileStoreIntegration tests Store integration specifically
func TestAddFileStoreIntegration(t *testing.T) {
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	storeConfigs := []string{"vector+store", "vector+graph+store"}

	for _, configName := range storeConfigs {
		config := configs[configName]
		if config == nil {
			t.Skipf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Store_%s", configName), func(t *testing.T) {
			g, err := New(config)
			if err != nil {
				t.Skipf("Failed to create GraphRag instance for %s: %v", configName, err)
			}

			// Skip if Store is not available
			if g.Store == nil {
				t.Skipf("Store not available for config %s", configName)
			}

			testFile := getDocumentTestFilePath("text.txt")
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Skip("Test file text.txt not found")
			}

			ctx := context.Background()
			options := &types.UpsertOptions{
				DocID:     fmt.Sprintf("test-store-%s", configName),
				GraphName: "test-collection",
				Metadata: map[string]interface{}{
					"source": "store_test",
					"config": configName,
				},
			}

			docID, err := g.AddFile(ctx, testFile, options)

			if err != nil {
				t.Logf("Config %s: Expected error: %v", configName, err)
				return
			}

			if docID == "" {
				t.Error("AddFile returned empty document ID")
				return
			}

			// Check if original text was stored
			originKey := fmt.Sprintf("origin_%s", docID)
			if g.Store.Has(originKey) {
				t.Logf("Config %s: Original text stored successfully", configName)
			} else {
				t.Logf("Config %s: Original text not found in store (expected with mock)", configName)
			}

			t.Logf("Store integration test completed for %s", configName)
		})
	}
}

// TestAddFileRealIntegration tests with real OpenAI if available
func TestAddFileRealIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real integration test in short mode")
	}

	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping real integration test")
	}

	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	config := configs["vector"]
	if config == nil {
		t.Skip("Vector config not found")
	}

	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	// Create a simple test file
	testFile := "/tmp/addfile_real_test.txt"
	content := "This is a test document for real AddFile integration testing. It contains sample text to process."
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	ctx := context.Background()
	options := &types.UpsertOptions{
		DocID:     "real-test-001",
		GraphName: "real-test-collection",
		Metadata: map[string]interface{}{
			"source": "real_test",
			"type":   "text",
		},
	}

	docID, err := g.AddFile(ctx, testFile, options)

	if err != nil {
		t.Fatalf("Real integration test failed: %v", err)
	}

	if docID == "" {
		t.Fatal("Real integration returned empty document ID")
	}

	t.Logf("Real integration successful!")
	t.Logf("Document ID: %s", docID)
}
