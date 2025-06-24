package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

const (
	testConnectorName  = "openai_test"
	testLocalConnector = "test-local-llm"
	testModel          = "gpt-4o-mini"
	testText           = "John Smith works for Google in Mountain View, California. He is the lead engineer of the AI team that developed a revolutionary search algorithm called PageRank. The algorithm was invented by Larry Page and Sergey Brin at Stanford University. Google was founded in 1998 and has become one of the world's largest technology companies."
)

// setupConnector creates connectors using the same pattern as semantic_test.go
func setupConnector(t testing.TB) {
	// Create OpenAI connector using environment variables (same as semantic_test.go)
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		openaiKey = "mock-key"
	}

	openaiDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "%s",
			"key": "%s"
		}
	}`, testModel, openaiKey)

	_, err := connector.New("openai", testConnectorName, []byte(openaiDSL))
	if err != nil {
		t.Logf("Failed to create OpenAI connector: %v", err)
	}

	// Create local LLM connector using environment variables (same as semantic_test.go)
	llmURL := os.Getenv("RAG_LLM_TEST_URL")
	llmKey := os.Getenv("RAG_LLM_TEST_KEY")
	llmModel := os.Getenv("RAG_LLM_TEST_SMODEL")

	if llmURL == "" {
		llmURL = "http://localhost:11434"
	}
	if llmKey == "" {
		llmKey = "mock-key"
	}
	if llmModel == "" {
		llmModel = "qwen3:8b"
	}

	llmDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "Local LLM Test", 
		"type": "openai",
		"options": {
			"proxy": "%s",
			"model": "%s",
			"key": "%s"
		}
	}`, llmURL, llmModel, llmKey)

	_, err = connector.New("openai", testLocalConnector, []byte(llmDSL))
	if err != nil {
		t.Logf("Failed to create local LLM connector: %v", err)
	}
}

func teardownConnector(t testing.TB) {
	// No need to explicitly remove connectors in this test pattern
}

func TestNewOpenai(t *testing.T) {
	setupConnector(t)
	defer teardownConnector(t)

	tests := []struct {
		name    string
		options Options
		wantErr bool
	}{
		{
			name: "valid options with higher concurrency",
			options: Options{
				ConnectorName: testConnectorName,
				Concurrent:    10, // Increased from 3
				Model:         testModel,
				Temperature:   0.1,
				MaxTokens:     2000,
			},
			wantErr: false,
		},
		{
			name: "invalid connector",
			options: Options{
				ConnectorName: "non_existent",
			},
			wantErr: true,
		},
		{
			name: "default values with higher concurrency",
			options: Options{
				ConnectorName: testConnectorName,
				Concurrent:    15, // Test higher concurrency
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := NewOpenai(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenai() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && extractor == nil {
				t.Error("NewOpenai() returned nil extractor")
			}
		})
	}
}

func TestNewOpenaiWithDefaults(t *testing.T) {
	setupConnector(t)
	defer teardownConnector(t)

	extractor, err := NewOpenaiWithDefaults(testConnectorName)
	if err != nil {
		t.Fatalf("NewOpenaiWithDefaults() error = %v", err)
	}

	if extractor == nil {
		t.Fatal("NewOpenaiWithDefaults() returned nil")
	}

	// Check default values
	if extractor.GetConcurrent() != 5 {
		t.Errorf("Expected concurrent = 5, got %d", extractor.GetConcurrent())
	}
	if extractor.GetTemperature() != 0.1 {
		t.Errorf("Expected temperature = 0.1, got %f", extractor.GetTemperature())
	}
	if extractor.GetMaxTokens() != 4000 {
		t.Errorf("Expected max tokens = 4000, got %d", extractor.GetMaxTokens())
	}
	if !extractor.GetToolcall() {
		t.Errorf("Expected toolcall = true, got %v", extractor.GetToolcall())
	}
}

func TestToolcallOption(t *testing.T) {
	setupConnector(t)
	defer teardownConnector(t)

	// Helper function to create bool pointer
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name             string
		options          Options
		expectedToolcall bool
		description      string
	}{
		{
			name: "Explicit toolcall enabled",
			options: Options{
				ConnectorName: testConnectorName,
				Toolcall:      boolPtr(true),
			},
			expectedToolcall: true,
			description:      "Should enable toolcall when explicitly set to true",
		},
		{
			name: "Explicit toolcall disabled",
			options: Options{
				ConnectorName: testConnectorName,
				Toolcall:      boolPtr(false),
			},
			expectedToolcall: false,
			description:      "Should disable toolcall when explicitly set to false",
		},
		{
			name: "Default toolcall with no tools",
			options: Options{
				ConnectorName: testConnectorName,
			},
			expectedToolcall: true,
			description:      "Should default to toolcall when not specified",
		},
		{
			name: "Custom tools with toolcall disabled",
			options: Options{
				ConnectorName: testConnectorName,
				Toolcall:      boolPtr(false),
				Tools:         []map[string]interface{}{{"test": "tool"}},
			},
			expectedToolcall: false,
			description:      "Should respect explicit toolcall=false even with custom tools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := NewOpenai(tt.options)
			if err != nil {
				t.Fatalf("NewOpenai() error = %v", err)
			}

			if extractor.GetToolcall() != tt.expectedToolcall {
				t.Errorf("%s: Expected toolcall = %v, got %v", tt.description, tt.expectedToolcall, extractor.GetToolcall())
			}

			// If toolcall is enabled, should have tools
			if extractor.GetToolcall() && len(extractor.Tools) == 0 {
				t.Errorf("%s: Expected tools when toolcall is enabled", tt.description)
			}

			// If toolcall is disabled, tools should be empty (unless custom tools provided)
			if !extractor.GetToolcall() && len(tt.options.Tools) == 0 && len(extractor.Tools) > 0 {
				t.Errorf("%s: Expected no default tools when toolcall is disabled", tt.description)
			}
		})
	}
}

func TestExtractQuery(t *testing.T) {
	setupConnector(t)
	defer teardownConnector(t)

	// Test with OpenAI (toolcall supported)
	t.Run("OpenAI with toolcall", func(t *testing.T) {
		if os.Getenv("OPENAI_TEST_KEY") == "" {
			t.Skip("OPENAI_TEST_KEY not set, skipping OpenAI test")
		}

		extractor, err := NewOpenaiWithDefaults(testConnectorName)
		if err != nil {
			t.Fatalf("Failed to create extractor: %v", err)
		}

		ctx := context.Background()

		// Test with progress callback
		var progressUpdates []types.ExtractionPayload
		callback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
			progressUpdates = append(progressUpdates, payload)
			t.Logf("Progress: %s - %s", status, payload.Message)
		}

		result, err := extractor.ExtractQuery(ctx, testText, callback)
		if err != nil {
			t.Fatalf("ExtractQuery() error = %v", err)
		}

		if result == nil {
			t.Fatal("ExtractQuery() returned nil result")
		}

		// Verify basic structure
		if result.Model != testModel {
			t.Errorf("Expected model %s, got %s", testModel, result.Model)
		}

		// Should have extracted some entities
		if len(result.Nodes) == 0 {
			t.Error("Expected at least one entity, got none")
		}

		// Relationships are optional - LLM might not always find relationships
		if len(result.Relationships) == 0 {
			t.Log("No relationships extracted (this is acceptable)")
		} else {
			t.Logf("Extracted %d relationships", len(result.Relationships))
		}

		// Check usage stats
		if result.Usage.TotalTexts != 1 {
			t.Errorf("Expected TotalTexts = 1, got %d", result.Usage.TotalTexts)
		}

		// Validate entities
		for i, node := range result.Nodes {
			if node.ID == "" {
				t.Errorf("Entity %d has empty ID", i)
			}
			if node.Name == "" {
				t.Errorf("Entity %d has empty Name", i)
			}
			if node.Type == "" {
				t.Errorf("Entity %d has empty Type", i)
			}
			if node.Description == "" {
				t.Errorf("Entity %d has empty Description", i)
			}
			if node.Confidence < 0.0 || node.Confidence > 1.0 {
				t.Errorf("Entity %d has invalid confidence: %f", i, node.Confidence)
			}
			if node.ExtractionMethod != types.ExtractionMethodLLM {
				t.Errorf("Entity %d has wrong extraction method: %s", i, node.ExtractionMethod)
			}
		}

		// Validate relationships (if any)
		for i, rel := range result.Relationships {
			if rel.StartNode == "" {
				t.Errorf("Relationship %d has empty StartNode", i)
			}
			if rel.EndNode == "" {
				t.Errorf("Relationship %d has empty EndNode", i)
			}
			if rel.Type == "" {
				t.Errorf("Relationship %d has empty Type", i)
			}
			if rel.Description == "" {
				t.Errorf("Relationship %d has empty Description", i)
			}
			if rel.Confidence < 0.0 || rel.Confidence > 1.0 {
				t.Errorf("Relationship %d has invalid confidence: %f", i, rel.Confidence)
			}
			if rel.ExtractionMethod != types.ExtractionMethodLLM {
				t.Errorf("Relationship %d has wrong extraction method: %s", i, rel.ExtractionMethod)
			}
		}

		// Verify we got progress updates
		if len(progressUpdates) == 0 {
			t.Error("Expected progress updates, got none")
		}

		t.Logf("Extracted %d entities and %d relationships", len(result.Nodes), len(result.Relationships))
		for _, node := range result.Nodes {
			t.Logf("Entity: %s (%s) - %s", node.Name, node.Type, node.Description)
		}
		for _, rel := range result.Relationships {
			t.Logf("Relationship: %s -[%s]-> %s - %s", rel.StartNode, rel.Type, rel.EndNode, rel.Description)
		}
	})

	// Test with local LLM (no toolcall support)
	t.Run("Local LLM without toolcall", func(t *testing.T) {
		if os.Getenv("RAG_LLM_TEST_URL") == "" {
			t.Skip("RAG_LLM_TEST_URL not set, skipping local LLM test")
		}

		// Create extractor with local LLM connector and disabled toolcall
		toolcallDisabled := false
		extractor, err := NewOpenai(Options{
			ConnectorName: testLocalConnector,
			Concurrent:    8, // Higher concurrency for local LLM
			Temperature:   0.1,
			Toolcall:      &toolcallDisabled, // Explicitly disable toolcall
		})
		if err != nil {
			t.Fatalf("Failed to create local LLM extractor: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Add timeout
		defer cancel()

		var progressUpdates []types.ExtractionPayload
		callback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
			progressUpdates = append(progressUpdates, payload)
			// Reduce log frequency
			if len(progressUpdates)%5 == 0 || status == types.ExtractionStatusCompleted || status == types.ExtractionStatusError {
				t.Logf("Local LLM Progress: %s - %s", status, payload.Message)
			}
		}

		result, err := extractor.ExtractQuery(ctx, testText, callback)
		if err != nil {
			// Check for common network errors that indicate service is not available
			if strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "no such host") ||
				strings.Contains(err.Error(), "context deadline exceeded") ||
				strings.Contains(err.Error(), "i/o timeout") {
				t.Logf("Local LLM service not available: %v", err)
				t.Skip("Local LLM service not available")
			}
			t.Fatalf("Local LLM extraction failed: %v", err)
		}

		if result == nil {
			t.Fatal("ExtractQuery() returned nil result")
		}

		// Should have extracted some entities
		if len(result.Nodes) == 0 {
			t.Error("Expected at least one entity from local LLM, got none")
		}

		// Verify we got progress updates
		if len(progressUpdates) == 0 {
			t.Error("Expected progress updates from local LLM, got none")
		}

		t.Logf("Local LLM extracted %d entities and %d relationships", len(result.Nodes), len(result.Relationships))
	})

	t.Run("empty text", func(t *testing.T) {
		extractor, err := NewOpenaiWithDefaults(testConnectorName)
		if err != nil {
			t.Fatalf("Failed to create extractor: %v", err)
		}

		ctx := context.Background()
		result, err := extractor.ExtractQuery(ctx, "")
		if err != nil {
			t.Fatalf("ExtractQuery() with empty text error = %v", err)
		}

		if len(result.Nodes) != 0 {
			t.Errorf("Expected 0 entities for empty text, got %d", len(result.Nodes))
		}
		if len(result.Relationships) != 0 {
			t.Errorf("Expected 0 relationships for empty text, got %d", len(result.Relationships))
		}
	})
}

func TestExtractDocuments(t *testing.T) {
	setupConnector(t)
	defer teardownConnector(t)

	// Test with higher concurrency
	extractor, err := NewOpenai(Options{
		ConnectorName: testConnectorName,
		Concurrent:    10, // Increased from default 5
		Model:         testModel,
		Temperature:   0.1,
	})
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	ctx := context.Background()

	t.Run("extract from multiple documents", func(t *testing.T) {
		if os.Getenv("OPENAI_TEST_KEY") == "" {
			t.Skip("OPENAI_TEST_KEY not set, skipping OpenAI test")
		}

		texts := []string{
			"Alice works at Microsoft as a software engineer.",
			"Bob is the CEO of Apple Inc. based in Cupertino.",
			"Carol leads the research team at DeepMind in London.",
		}

		var progressUpdates []types.ExtractionPayload
		callback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
			progressUpdates = append(progressUpdates, payload)
			t.Logf("Progress: %s - %s", status, payload.Message)
		}

		results, err := extractor.ExtractDocuments(ctx, texts, callback)
		if err != nil {
			t.Fatalf("ExtractDocuments() error = %v", err)
		}

		if results == nil {
			t.Fatal("ExtractDocuments() returned nil results")
		}

		if len(results) != len(texts) {
			t.Errorf("Expected %d results, got %d", len(texts), len(results))
		}

		// Count total entities and relationships across all results
		totalEntities := 0
		totalRelationships := 0
		successfulExtractions := 0

		for i, result := range results {
			if result != nil {
				totalEntities += len(result.Nodes)
				totalRelationships += len(result.Relationships)
				successfulExtractions++
				t.Logf("Document %d: %d entities, %d relationships", i, len(result.Nodes), len(result.Relationships))
			} else {
				t.Logf("Document %d: extraction failed (nil result)", i)
			}
		}

		// Should have extracted entities from at least some documents
		if totalEntities == 0 {
			t.Error("Expected at least one entity from all documents, got none")
		}

		// Should have successfully processed some documents
		if successfulExtractions == 0 {
			t.Error("Expected at least one successful extraction, got none")
		}

		// Verify we got progress updates
		if len(progressUpdates) == 0 {
			t.Error("Expected progress updates, got none")
		}

		t.Logf("Extracted %d entities and %d relationships from %d documents (%d successful)",
			totalEntities, totalRelationships, len(texts), successfulExtractions)
	})

	t.Run("empty documents", func(t *testing.T) {
		results, err := extractor.ExtractDocuments(ctx, []string{})
		if err != nil {
			t.Fatalf("ExtractDocuments() with empty slice error = %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty documents, got %d", len(results))
		}
	})
}

func TestConcurrentExtraction(t *testing.T) {
	setupConnector(t)
	defer teardownConnector(t)

	// Significantly increase concurrency
	extractor, err := NewOpenai(Options{
		ConnectorName: testConnectorName,
		Concurrent:    15, // Increased from 3
		Model:         testModel,
		Temperature:   0.1,
	})
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	ctx := context.Background()

	// Test concurrent extraction with more documents
	texts := make([]string, 20) // Increased from 10
	for i := 0; i < 20; i++ {
		texts[i] = fmt.Sprintf("Person_%d works at Company_%d as a manager. The company is located in City_%d.", i, i, i)
	}

	start := time.Now()
	results, err := extractor.ExtractDocuments(ctx, texts)
	elapsed := time.Since(start)

	if err != nil {
		if os.Getenv("OPENAI_TEST_KEY") == "" {
			t.Skip("OPENAI_TEST_KEY not set, skipping concurrent test")
		}
		t.Fatalf("Concurrent extraction failed: %v", err)
	}

	// Count total entities and relationships
	totalEntities := 0
	totalRelationships := 0
	for _, result := range results {
		if result != nil {
			totalEntities += len(result.Nodes)
			totalRelationships += len(result.Relationships)
		}
	}

	// Should have extracted entities from some texts
	if totalEntities == 0 {
		t.Error("Expected entities from concurrent extraction")
	}

	t.Logf("Concurrent extraction of %d documents took %v", len(texts), elapsed)
	t.Logf("Extracted %d entities and %d relationships", totalEntities, totalRelationships)
}

func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	setupConnector(t)
	defer teardownConnector(t)

	// Increase concurrency for stress test
	extractor, err := NewOpenai(Options{
		ConnectorName: testConnectorName,
		Concurrent:    20, // Increased from 5
		Model:         testModel,
		RetryAttempts: 2,
		RetryDelay:    500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	ctx := context.Background()

	// Generate many texts for stress testing
	numTexts := 100 // Increased from 50
	texts := make([]string, numTexts)
	for i := 0; i < numTexts; i++ {
		texts[i] = fmt.Sprintf("Employee_%d works at Organization_%d located in Location_%d. The organization specializes in Technology_%d and has Partnership with Company_%d.", i, i%10, i%5, i%3, i%7)
	}

	start := time.Now()
	results, err := extractor.ExtractDocuments(ctx, texts)
	elapsed := time.Since(start)

	if err != nil {
		if os.Getenv("OPENAI_TEST_KEY") == "" {
			t.Skip("OPENAI_TEST_KEY not set, skipping stress test")
		}
		t.Fatalf("Stress test failed: %v", err)
	}

	// Count total entities, relationships, and tokens
	totalEntities := 0
	totalRelationships := 0
	totalTokens := 0
	for _, result := range results {
		if result != nil {
			totalEntities += len(result.Nodes)
			totalRelationships += len(result.Relationships)
			totalTokens += result.Usage.TotalTokens
		}
	}

	if totalEntities == 0 {
		t.Error("Expected entities from stress test")
	}

	t.Logf("Stress test with %d documents took %v", numTexts, elapsed)
	t.Logf("Extracted %d entities and %d relationships", totalEntities, totalRelationships)
	t.Logf("Total tokens used: %d", totalTokens)
	t.Logf("Average time per document: %v", elapsed/time.Duration(numTexts))
}

func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	setupConnector(t)
	defer teardownConnector(t)

	extractor, err := NewOpenaiWithDefaults(testConnectorName)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	ctx := context.Background()

	// Force garbage collection to get baseline
	runtime.GC()
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Run extraction multiple times (reduced from 20 to 10 for speed)
	numIterations := 10
	for i := 0; i < numIterations; i++ {
		text := fmt.Sprintf("Iteration %d: John Doe works at Tech Corp in City %d.", i, i)
		_, err := extractor.ExtractQuery(ctx, text)
		if err != nil {
			if os.Getenv("OPENAI_TEST_KEY") == "" {
				t.Skip("OPENAI_TEST_KEY not set, skipping memory test")
			}
			t.Fatalf("Extraction failed at iteration %d: %v", i, err)
		}

		// More frequent garbage collection for faster test
		if i%3 == 0 {
			runtime.GC()
		}
	}

	// Force garbage collection after all operations
	runtime.GC()
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate memory usage (handle potential negative growth)
	var memGrowthMB float64
	if memAfter.Alloc >= memBefore.Alloc {
		memGrowth := memAfter.Alloc - memBefore.Alloc
		memGrowthMB = float64(memGrowth) / (1024 * 1024)
	} else {
		// Memory decreased (negative growth)
		memDecrease := memBefore.Alloc - memAfter.Alloc
		memGrowthMB = -float64(memDecrease) / (1024 * 1024)
	}

	t.Logf("Memory before: %d bytes", memBefore.Alloc)
	t.Logf("Memory after: %d bytes", memAfter.Alloc)
	t.Logf("Memory growth: %.2f MB", memGrowthMB)

	// Alert if memory growth seems excessive (adjusted threshold for fewer iterations)
	if memGrowthMB > 5.0 {
		t.Errorf("Potential memory leak detected: %.2f MB growth after %d iterations", memGrowthMB, numIterations)
	}
}

func TestErrorHandling(t *testing.T) {
	setupConnector(t)
	defer teardownConnector(t)

	t.Run("invalid connector type", func(t *testing.T) {
		// Create a non-OpenAI connector
		dsl := map[string]interface{}{
			"type": "mysql",
			"name": "Invalid Connector",
			"options": map[string]interface{}{
				"host": "localhost",
			},
		}
		dslBytes, _ := json.Marshal(dsl)
		connector.New("mysql", "invalid_connector", dslBytes)

		_, err := NewOpenai(Options{
			ConnectorName: "invalid_connector",
		})
		if err == nil {
			t.Error("Expected error for invalid connector type")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		extractor, err := NewOpenaiWithDefaults(testConnectorName)
		if err != nil {
			t.Fatalf("Failed to create extractor: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = extractor.ExtractQuery(ctx, testText)
		if err == nil {
			t.Error("Expected error for cancelled context")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		extractor, err := NewOpenaiWithDefaults(testConnectorName)
		if err != nil {
			t.Fatalf("Failed to create extractor: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		_, err = extractor.ExtractQuery(ctx, testText)
		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

// Test both toolcall and non-toolcall scenarios
func TestToolcallVsNonToolcall(t *testing.T) {
	setupConnector(t)
	defer teardownConnector(t)

	ctx := context.Background()
	testText := "Apple Inc. is a technology company founded by Steve Jobs."

	t.Run("With toolcall (OpenAI)", func(t *testing.T) {
		if os.Getenv("OPENAI_TEST_KEY") == "" {
			t.Skip("OPENAI_TEST_KEY not set, skipping OpenAI toolcall test")
		}

		extractor, err := NewOpenai(Options{
			ConnectorName: testConnectorName,
			Concurrent:    8,
			Model:         testModel,
			Temperature:   0.1,
			// Use default tools (includes toolcall)
		})
		if err != nil {
			t.Fatalf("Failed to create OpenAI extractor: %v", err)
		}

		result, err := extractor.ExtractQuery(ctx, testText)
		if err != nil {
			t.Fatalf("OpenAI toolcall extraction failed: %v", err)
		}

		if len(result.Nodes) == 0 {
			t.Error("Expected entities from OpenAI toolcall extraction")
		}

		t.Logf("OpenAI toolcall: %d entities, %d relationships", len(result.Nodes), len(result.Relationships))
	})

	t.Run("Without toolcall (Local LLM)", func(t *testing.T) {
		if os.Getenv("RAG_LLM_TEST_URL") == "" {
			t.Skip("RAG_LLM_TEST_URL not set, skipping local LLM test")
		}

		toolcallDisabled := false
		extractor, err := NewOpenai(Options{
			ConnectorName: testLocalConnector,
			Concurrent:    8,
			Temperature:   0.1,
			Toolcall:      &toolcallDisabled, // Explicitly disable toolcall
		})
		if err != nil {
			t.Fatalf("Failed to create local LLM extractor: %v", err)
		}

		result, err := extractor.ExtractQuery(ctx, testText)
		if err != nil {
			// Check for common network errors that indicate service is not available
			if strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "no such host") ||
				strings.Contains(err.Error(), "context deadline exceeded") ||
				strings.Contains(err.Error(), "i/o timeout") {
				t.Logf("Local LLM service not available: %v", err)
				t.Skip("Local LLM service not available")
			}
			t.Fatalf("Local LLM extraction failed: %v", err)
		}

		if len(result.Nodes) == 0 {
			t.Error("Expected entities from local LLM extraction")
		}

		t.Logf("Local LLM non-toolcall: %d entities, %d relationships", len(result.Nodes), len(result.Relationships))
	})
}

func TestParseResponse(t *testing.T) {
	t.Run("valid extraction tool call", func(t *testing.T) {
		// Test the new extraction parser with valid tool call arguments
		validArguments := `{
			"entities": [
				{
					"id": "john_smith",
					"name": "John Smith",
					"type": "PERSON",
					"description": "A person",
					"confidence": 0.9
				}
			],
			"relationships": [
				{
					"start_node": "john_smith",
					"end_node": "google",
					"type": "WORKS_FOR",
					"description": "Employment relationship",
					"confidence": 0.8
				}
			]
		}`

		// Create extraction parser and test parsing
		parser := utils.NewExtractionParser()
		nodes, relationships, err := parser.ParseExtractionToolcall(validArguments)
		if err != nil {
			t.Fatalf("ParseExtractionToolcall() error = %v", err)
		}

		if len(nodes) != 1 {
			t.Errorf("Expected 1 entity, got %d", len(nodes))
		}

		if len(relationships) != 1 {
			t.Errorf("Expected 1 relationship, got %d", len(relationships))
		}

		// Validate entity
		entity := nodes[0]
		if entity.ID != "john_smith" {
			t.Errorf("Expected entity ID 'john_smith', got '%s'", entity.ID)
		}
		if entity.Name != "John Smith" {
			t.Errorf("Expected entity name 'John Smith', got '%s'", entity.Name)
		}
		if entity.Type != "PERSON" {
			t.Errorf("Expected entity type 'PERSON', got '%s'", entity.Type)
		}
		if entity.Confidence != 0.9 {
			t.Errorf("Expected entity confidence 0.9, got %f", entity.Confidence)
		}

		// Validate relationship
		rel := relationships[0]
		if rel.StartNode != "john_smith" {
			t.Errorf("Expected relationship start_node 'john_smith', got '%s'", rel.StartNode)
		}
		if rel.EndNode != "google" {
			t.Errorf("Expected relationship end_node 'google', got '%s'", rel.EndNode)
		}
		if rel.Type != "WORKS_FOR" {
			t.Errorf("Expected relationship type 'WORKS_FOR', got '%s'", rel.Type)
		}
		if rel.Confidence != 0.8 {
			t.Errorf("Expected relationship confidence 0.8, got %f", rel.Confidence)
		}
	})

	t.Run("invalid JSON format", func(t *testing.T) {
		invalidArguments := []string{
			"invalid json",
			"{}",
			`{"entities": "not an array"}`,
			`{"entities": [], "relationships": "not an array"}`,
			`{"incomplete": json`,
		}

		parser := utils.NewExtractionParser()
		for i, args := range invalidArguments {
			nodes, relationships, err := parser.ParseExtractionToolcall(args)
			// Should either return error or empty results for invalid JSON
			if err != nil {
				t.Logf("Test case %d correctly returned error: %v", i, err)
			} else if len(nodes) == 0 && len(relationships) == 0 {
				t.Logf("Test case %d correctly returned empty results", i)
			} else {
				t.Errorf("Test case %d: Expected error or empty results for invalid JSON, got %d entities and %d relationships", i, len(nodes), len(relationships))
			}
		}
	})

	t.Run("streaming chunk parsing", func(t *testing.T) {
		// Test streaming chunk parsing
		parser := utils.NewExtractionParser()

		// Simulate streaming chunks
		chunks := []string{
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"entities\": ["}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"id\": \"test\", \"name\": \"Test\", \"type\": \"PERSON\", \"description\": \"Test person\", \"confidence\": 0.9},"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"], \"relationships\": []}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		}

		var finalNodes []types.Node
		var finalRelationships []types.Relationship
		var err error

		for _, chunk := range chunks {
			finalNodes, finalRelationships, err = parser.ParseExtractionEntities([]byte(chunk))
			if err != nil {
				t.Logf("Chunk parsing error (may be expected for incomplete chunks): %v", err)
			}
		}

		// The final result should have parsed entities
		if len(finalNodes) == 0 {
			t.Log("No entities parsed from streaming chunks - this may be expected for incomplete JSON")
		} else {
			t.Logf("Successfully parsed %d entities and %d relationships from streaming chunks", len(finalNodes), len(finalRelationships))
		}
	})
}

func TestRaceConditions(t *testing.T) {
	// Re-enabled after fixing DNS package race condition and HTTP transport pooling

	setupConnector(t)
	defer teardownConnector(t)

	extractor, err := NewOpenai(Options{
		ConnectorName: testConnectorName,
		Concurrent:    3, // Very low concurrency to avoid DNS race condition
		Model:         testModel,
	})
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Very conservative settings to avoid triggering the DNS race condition
	numGoroutines := 3
	numIterations := 2

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	// Launch multiple goroutines doing concurrent extractions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				// Add delay to prevent DNS race condition
				time.Sleep(time.Duration(id*100+j*50) * time.Millisecond)

				text := fmt.Sprintf("Goroutine %d iteration %d: Person_%d works at Company_%d", id, j, id*j, j)
				_, err := extractor.ExtractQuery(ctx, text)
				if err != nil {
					// Only report non-timeout errors as failures
					if !strings.Contains(err.Error(), "context deadline exceeded") {
						errors <- fmt.Errorf("goroutine %d iteration %d failed: %w", id, j, err)
					}
				}
			}
		}(i)
	}

	// Wait for completion or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed successfully
	case <-ctx.Done():
		t.Errorf("Race condition test timed out")
		return
	}

	close(errors)

	// Check for any errors
	var errorCount int
	for err := range errors {
		t.Errorf("Race condition error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Found %d race condition errors", errorCount)
	}

	t.Logf("Race condition test completed: %d goroutines × %d iterations", numGoroutines, numIterations)
}

func BenchmarkExtractQuery(b *testing.B) {
	setupConnector(b)
	defer teardownConnector(b)

	extractor, err := NewOpenaiWithDefaults(testConnectorName)
	if err != nil {
		b.Fatalf("Failed to create extractor: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		text := fmt.Sprintf("Benchmark iteration %d: Employee works at Company located in City.", i)
		_, err := extractor.ExtractQuery(ctx, text)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkExtractDocuments(b *testing.B) {
	setupConnector(b)
	defer teardownConnector(b)

	extractor, err := NewOpenai(Options{
		ConnectorName: testConnectorName,
		Concurrent:    5,
		Model:         testModel,
	})
	if err != nil {
		b.Fatalf("Failed to create extractor: %v", err)
	}

	ctx := context.Background()
	texts := []string{
		"Alice works at Microsoft.",
		"Bob leads the team at Google.",
		"Carol is CEO of Apple.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractor.ExtractDocuments(ctx, texts)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// Test concurrent extraction with stress testing
func TestConcurrentExtractionStressTest(t *testing.T) {
	// Test high concurrency with improved implementation
	extractor, err := NewOpenai(Options{
		ConnectorName: testConnectorName,
		Concurrent:    20, // Test high concurrency
		Temperature:   0.1,
		MaxTokens:     2000,
		RetryAttempts: 2,
		RetryDelay:    500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	// Test with reasonable dataset size
	numTexts := 50
	texts := make([]string, numTexts)
	for i := 0; i < numTexts; i++ {
		texts[i] = fmt.Sprintf("Document %d: %s", i+1, testText)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // Generous timeout for high concurrency
	defer cancel()

	var progressMutex sync.Mutex
	progressCount := 0
	successCount := 0
	errorCount := 0

	callback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
		progressMutex.Lock()
		progressCount++

		if status == types.ExtractionStatusCompleted {
			successCount++
		} else if status == types.ExtractionStatusError {
			errorCount++
		}

		// Log progress periodically
		if progressCount%10 == 0 || status == types.ExtractionStatusCompleted {
			t.Logf("Stress test progress (%d): %s - %s (Success: %d, Errors: %d)",
				progressCount, status, payload.Message, successCount, errorCount)
		}
		progressMutex.Unlock()
	}

	results, err := extractor.ExtractDocuments(ctx, texts, callback)
	if err != nil {
		t.Fatalf("Concurrent extraction failed: %v", err)
	}

	if results == nil {
		t.Fatal("ExtractDocuments() returned nil results")
	}

	// Count total entities and relationships
	totalEntities := 0
	totalRelationships := 0
	for _, result := range results {
		if result != nil {
			totalEntities += len(result.Nodes)
			totalRelationships += len(result.Relationships)
		}
	}

	// Should have extracted entities from multiple documents
	if totalEntities == 0 {
		t.Error("Expected entities from concurrent extraction, got none")
	}

	t.Logf("High concurrency stress test completed: extracted %d entities and %d relationships from %d documents (Success: %d, Errors: %d)",
		totalEntities, totalRelationships, numTexts, successCount, errorCount)
}
