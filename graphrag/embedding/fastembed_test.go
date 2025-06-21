package embedding

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
)

// Tests that require environment variables (integration tests)

func TestNewFastEmbed(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	key := os.Getenv("FASTEMBED_TEST_KEY")
	model := os.Getenv("FASTEMBED_TEST_MODEL")
	if host == "" || model == "" {
		t.Skip("Skipping FastEmbed test: FASTEMBED_TEST_HOST or FASTEMBED_TEST_MODEL not set")
	}

	// Create connector for FastEmbed (with password)
	fastembedDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"key": "%s",
			"model": "%s"
		}
	}`, host, key, model)

	_, err := connector.New("fastembed", "test-fastembed", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	// Test with options
	options := FastEmbedOptions{
		ConnectorName: "test-fastembed",
		Concurrent:    5,
		Dimension:     384,
		Model:         "BAAI/bge-small-en-v1.5",
	}

	fastembed, err := NewFastEmbed(options)
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Verify settings
	if fastembed.GetDimension() != 384 {
		t.Errorf("Expected dimension 384, got %d", fastembed.GetDimension())
	}

	if fastembed.GetModel() != "BAAI/bge-small-en-v1.5" {
		t.Errorf("Expected model BAAI/bge-small-en-v1.5, got %s", fastembed.GetModel())
	}

	expectedHost := fmt.Sprintf("http://%s", host)
	if fastembed.GetHost() != expectedHost {
		t.Errorf("Expected host %s, got %s", expectedHost, fastembed.GetHost())
	}

	if !fastembed.HasKey() {
		t.Error("Expected key to be set")
	}

	t.Logf("FastEmbed created successfully with host: %s, model: %s, dimension: %d",
		fastembed.GetHost(), fastembed.GetModel(), fastembed.GetDimension())
}

func TestNewFastEmbedWithDefaults(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	model := os.Getenv("FASTEMBED_TEST_MODEL")
	if host == "" || model == "" {
		t.Skip("Skipping FastEmbed test: FASTEMBED_TEST_HOST or FASTEMBED_TEST_MODEL not set")
	}

	// Create connector for FastEmbed (minimal)
	fastembedDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Defaults Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"model": "%s"
		}
	}`, host, model)

	_, err := connector.New("fastembed", "test-fastembed-defaults", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	fastembed, err := NewFastEmbedWithDefaults("test-fastembed-defaults")
	if err != nil {
		t.Fatalf("Failed to create FastEmbed with defaults: %v", err)
	}

	// Verify defaults
	if fastembed.GetDimension() != 384 {
		t.Errorf("Expected default dimension 384, got %d", fastembed.GetDimension())
	}

	if fastembed.GetModel() != "BAAI/bge-small-en-v1.5" {
		t.Errorf("Expected default model BAAI/bge-small-en-v1.5, got %s", fastembed.GetModel())
	}

	expectedHost := fmt.Sprintf("http://%s", host)
	if fastembed.GetHost() != expectedHost {
		t.Errorf("Expected host %s, got %s", expectedHost, fastembed.GetHost())
	}

	if fastembed.HasKey() {
		t.Error("Expected no key to be set")
	}

	t.Logf("FastEmbed with defaults created successfully")
}

func TestFastEmbed_EmbedQuery_WithPassword(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	key := os.Getenv("FASTEMBED_TEST_KEY")
	model := os.Getenv("FASTEMBED_TEST_MODEL")

	if host == "" || key == "" || model == "" {
		t.Skip("Skipping FastEmbed test: required environment variables not set")
	}

	// Create connector for FastEmbed with password
	fastembedDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed With Password Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"key": "%s",
			"model": "%s"
		}
	}`, host, key, model)

	_, err := connector.New("fastembed", "test-fastembed-password", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	options := FastEmbedOptions{
		ConnectorName: "test-fastembed-password",
		Concurrent:    5,
		Dimension:     384,
	}

	fastembed, err := NewFastEmbed(options)
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Test single text embedding with progress callback
	testText := "Hello, this is a test text for embedding."

	var progressMessages []string
	callback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		message := fmt.Sprintf("Status: %s, Progress: %d/%d, Message: %s",
			status, payload.Current, payload.Total, payload.Message)
		progressMessages = append(progressMessages, message)
		t.Logf("Progress: %s", message)
	}

	ctx := context.Background()
	embeddingResult, err := fastembed.EmbedQuery(ctx, testText, callback)
	if err != nil {
		t.Fatalf("EmbedQuery failed: %v", err)
	}

	// Verify embedding result
	assert.NotNil(t, embeddingResult)
	assert.Equal(t, 1, embeddingResult.Usage.TotalTexts)
	assert.Greater(t, embeddingResult.Usage.TotalTokens, 0)
	assert.Equal(t, types.EmbeddingTypeDense, embeddingResult.Type)

	if len(embeddingResult.Embedding) != 384 {
		t.Errorf("Expected embedding dimension 384, got %d", len(embeddingResult.Embedding))
	}

	// Check if embedding contains non-zero values
	hasNonZero := false
	for _, val := range embeddingResult.Embedding {
		if val != 0.0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("Embedding contains all zero values")
	}

	// Verify progress callbacks were called
	if len(progressMessages) == 0 {
		t.Error("Expected progress callbacks to be called")
	}

	t.Logf("Successfully embedded text with %d dimensions", len(embeddingResult.Embedding))
	t.Logf("First 10 embedding values: %v", embeddingResult.Embedding[:10])
}

func TestFastEmbed_EmbedQuery_NoPassword(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	model := os.Getenv("FASTEMBED_TEST_MODEL")

	if host == "" || model == "" {
		t.Skip("Skipping FastEmbed test: required environment variables not set")
	}

	// Create connector for FastEmbed without password
	fastembedDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed No Password Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"model": "%s"
		}
	}`, host, model)

	_, err := connector.New("fastembed", "test-fastembed-nopassword", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	options := FastEmbedOptions{
		ConnectorName: "test-fastembed-nopassword",
		Concurrent:    5,
		Dimension:     384,
	}

	fastembed, err := NewFastEmbed(options)
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Test single text embedding - this should fail if server requires auth
	testText := "Hello, this is a test text for embedding."
	ctx := context.Background()
	_, err = fastembed.EmbedQuery(ctx, testText)

	// We expect this to fail if the server requires authentication
	if err == nil {
		t.Log("EmbedQuery succeeded without password - server allows anonymous access")
	} else {
		t.Logf("EmbedQuery failed without password as expected: %v", err)
		// This is expected behavior if server requires auth
	}
}

func TestFastEmbed_EmbedDocuments(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	key := os.Getenv("FASTEMBED_TEST_KEY")
	model := os.Getenv("FASTEMBED_TEST_MODEL")

	if host == "" || key == "" || model == "" {
		t.Skip("Skipping FastEmbed test: required environment variables not set")
	}

	// Create connector for FastEmbed with password
	fastembedDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Documents Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"key": "%s",
			"model": "%s"
		}
	}`, host, key, model)

	_, err := connector.New("fastembed", "test-fastembed-documents", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	options := FastEmbedOptions{
		ConnectorName: "test-fastembed-documents",
		Concurrent:    3,
		Dimension:     384,
	}

	fastembed, err := NewFastEmbed(options)
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Test multiple documents embedding
	testDocuments := []string{
		"This is the first test document.",
		"This is the second test document.",
		"This is the third test document.",
		"This is the fourth test document.",
		"This is the fifth test document.",
	}

	var progressMessages []string
	callback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		message := fmt.Sprintf("Status: %s, Progress: %d/%d, Message: %s",
			status, payload.Current, payload.Total, payload.Message)
		progressMessages = append(progressMessages, message)
		t.Logf("Progress: %s", message)
	}

	ctx := context.Background()
	embeddingResults, err := fastembed.EmbedDocuments(ctx, testDocuments, callback)
	if err != nil {
		t.Fatalf("EmbedDocuments failed: %v", err)
	}

	// Verify embeddings
	assert.NotNil(t, embeddingResults)
	assert.Equal(t, len(testDocuments), embeddingResults.Count())
	assert.Equal(t, len(testDocuments), embeddingResults.Usage.TotalTexts)
	assert.Greater(t, embeddingResults.Usage.TotalTokens, 0)
	assert.Equal(t, types.EmbeddingTypeDense, embeddingResults.Type)

	embeddings := embeddingResults.GetDenseEmbeddings()
	assert.NotNil(t, embeddings)

	for i, embedding := range embeddings {
		if len(embedding) != 384 {
			t.Errorf("Expected embedding %d dimension 384, got %d", i, len(embedding))
		}

		// Check if embedding contains non-zero values
		hasNonZero := false
		for _, val := range embedding {
			if val != 0.0 {
				hasNonZero = true
				break
			}
		}
		if !hasNonZero {
			t.Errorf("Embedding %d contains all zero values", i)
		}
	}

	// Verify progress callbacks were called
	if len(progressMessages) == 0 {
		t.Error("Expected progress callbacks to be called")
	}

	t.Logf("Successfully embedded %d documents", embeddingResults.Count())
}

func TestFastEmbed_EmbedEmptyTexts(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	model := os.Getenv("FASTEMBED_TEST_MODEL")
	if host == "" || model == "" {
		t.Skip("Skipping FastEmbed test: FASTEMBED_TEST_HOST or FASTEMBED_TEST_MODEL not set")
	}

	// Create minimal connector
	fastembedDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Empty Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"model": "%s"
		}
	}`, host, model)

	_, err := connector.New("fastembed", "test-fastembed-empty", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	fastembed, err := NewFastEmbedWithDefaults("test-fastembed-empty")
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	ctx := context.Background()

	// Test empty text
	embeddingResult, err := fastembed.EmbedQuery(ctx, "")
	if err != nil {
		t.Fatalf("EmbedQuery with empty text failed: %v", err)
	}
	assert.Nil(t, embeddingResult)

	// Test empty documents array
	embeddingResults, err := fastembed.EmbedDocuments(ctx, []string{})
	if err != nil {
		t.Fatalf("EmbedDocuments with empty array failed: %v", err)
	}
	assert.Nil(t, embeddingResults)

	t.Log("Empty text handling works correctly")
}

func TestFastEmbed_ErrorHandling(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	model := os.Getenv("FASTEMBED_TEST_MODEL")
	if host == "" || model == "" {
		t.Skip("Skipping FastEmbed test: FASTEMBED_TEST_HOST or FASTEMBED_TEST_MODEL not set")
	}

	// Create connector with invalid port to test network errors
	fastembedDSL := `{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Error Test",
		"type": "fastembed",
		"options": {
			"host": "127.0.0.1:9999",
			"key": "test-password",
			"model": "BAAI/bge-small-en-v1.5"
		}
	}`

	_, err := connector.New("fastembed", "test-fastembed-error", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	options := FastEmbedOptions{
		ConnectorName: "test-fastembed-error",
		Concurrent:    5,
		Dimension:     384,
	}

	fastembed, err := NewFastEmbed(options)
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	ctx := context.Background()

	// Test network error
	_, err = fastembed.EmbedQuery(ctx, "test text")
	if err == nil {
		t.Error("Expected network error, but got success")
	} else {
		t.Logf("Got expected network error: %v", err)
	}
}

// Basic tests that don't require external services

func TestFastEmbed_BasicStructure(t *testing.T) {
	// Create a basic FastEmbed connector
	fastembedDSL := `{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Basic Test",
		"type": "fastembed",
		"options": {
			"host": "127.0.0.1:8000",
			"key": "test-password",
			"model": "BAAI/bge-small-en-v1.5"
		}
	}`

	_, err := connector.New("fastembed", "test-fastembed-basic", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	// Test with explicit options
	options := FastEmbedOptions{
		ConnectorName: "test-fastembed-basic",
		Concurrent:    5,
		Dimension:     384,
		Model:         "BAAI/bge-small-en-v1.5",
		Host:          "http://127.0.0.1:8000",
		Key:           "test-password",
	}

	fastembed, err := NewFastEmbed(options)
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Test getters
	if fastembed.GetDimension() != 384 {
		t.Errorf("Expected dimension 384, got %d", fastembed.GetDimension())
	}

	if fastembed.GetModel() != "BAAI/bge-small-en-v1.5" {
		t.Errorf("Expected model BAAI/bge-small-en-v1.5, got %s", fastembed.GetModel())
	}

	if fastembed.GetHost() != "http://127.0.0.1:8000" {
		t.Errorf("Expected host http://127.0.0.1:8000, got %s", fastembed.GetHost())
	}

	if !fastembed.HasKey() {
		t.Error("Expected key to be set")
	}

	t.Logf("FastEmbed created successfully with host: %s, model: %s, dimension: %d, has_key: %v",
		fastembed.GetHost(), fastembed.GetModel(), fastembed.GetDimension(), fastembed.HasKey())
}

func TestFastEmbed_WithDefaultsBasic(t *testing.T) {
	// Create a minimal FastEmbed connector (no password)
	fastembedDSL := `{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Defaults Test",
		"type": "fastembed",
		"options": {
			"host": "127.0.0.1:8000",
			"model": "BAAI/bge-small-en-v1.5"
		}
	}`

	_, err := connector.New("fastembed", "test-fastembed-defaults-basic", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	fastembed, err := NewFastEmbedWithDefaults("test-fastembed-defaults-basic")
	if err != nil {
		t.Fatalf("Failed to create FastEmbed with defaults: %v", err)
	}

	// Test defaults
	if fastembed.GetDimension() != 384 {
		t.Errorf("Expected default dimension 384, got %d", fastembed.GetDimension())
	}

	if fastembed.GetModel() != "BAAI/bge-small-en-v1.5" {
		t.Errorf("Expected default model BAAI/bge-small-en-v1.5, got %s", fastembed.GetModel())
	}

	if fastembed.GetHost() != "http://127.0.0.1:8000" {
		t.Errorf("Expected host http://127.0.0.1:8000, got %s", fastembed.GetHost())
	}

	if fastembed.HasKey() {
		t.Error("Expected no key to be set")
	}

	t.Logf("FastEmbed with defaults created successfully: host: %s, model: %s, dimension: %d, has_key: %v",
		fastembed.GetHost(), fastembed.GetModel(), fastembed.GetDimension(), fastembed.HasKey())
}

func TestFastEmbed_MissingConnector(t *testing.T) {
	// Test with non-existent connector
	_, err := NewFastEmbed(FastEmbedOptions{
		ConnectorName: "non-existent-connector",
	})
	if err == nil {
		t.Error("Expected error for non-existent connector")
	}
	t.Logf("Correctly got error for non-existent connector: %v", err)
}

func TestFastEmbed_EmptyInputs(t *testing.T) {
	// Create minimal connector for empty text tests
	fastembedDSL := `{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Empty Test",
		"type": "fastembed",
		"options": {
			"host": "127.0.0.1:8000",
			"model": "BAAI/bge-small-en-v1.5"
		}
	}`

	_, err := connector.New("fastembed", "test-fastembed-empty-basic", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	fastembed, err := NewFastEmbedWithDefaults("test-fastembed-empty-basic")
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	ctx := context.Background()

	// Test empty text - should return nil without making HTTP request
	embeddingResult, err := fastembed.EmbedQuery(ctx, "")
	if err != nil {
		t.Fatalf("EmbedQuery with empty text failed: %v", err)
	}
	assert.Nil(t, embeddingResult)

	// Test empty texts array - should return nil without making HTTP request
	embeddingResults, err := fastembed.EmbedDocuments(ctx, []string{})
	if err != nil {
		t.Fatalf("EmbedDocuments with empty array failed: %v", err)
	}
	assert.Nil(t, embeddingResults)

	t.Log("Empty text handling works correctly")
}

func TestFastEmbed_CustomOptions(t *testing.T) {
	// Create connector with custom settings
	fastembedDSL := `{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Custom Test",
		"type": "fastembed",
		"options": {
			"host": "custom-host:9000",
			"key": "custom-password",
			"model": "custom-model"
		}
	}`

	_, err := connector.New("fastembed", "test-fastembed-custom", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	// Test overriding with explicit options
	options := FastEmbedOptions{
		ConnectorName: "test-fastembed-custom",
		Concurrent:    10,
		Dimension:     512,
		Model:         "overridden-model",
		Host:          "http://override-host:8888",
		Key:           "override-password",
	}

	fastembed, err := NewFastEmbed(options)
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Test that explicit options override connector settings
	if fastembed.GetDimension() != 512 {
		t.Errorf("Expected dimension 512, got %d", fastembed.GetDimension())
	}

	if fastembed.GetModel() != "overridden-model" {
		t.Errorf("Expected model overridden-model, got %s", fastembed.GetModel())
	}

	if fastembed.GetHost() != "http://override-host:8888" {
		t.Errorf("Expected host http://override-host:8888, got %s", fastembed.GetHost())
	}

	if !fastembed.HasKey() {
		t.Error("Expected key to be set")
	}

	t.Logf("Custom options test passed: host: %s, model: %s, dimension: %d, concurrent: %d",
		fastembed.GetHost(), fastembed.GetModel(), fastembed.GetDimension(), 10)
}

func TestFastEmbed_DefaultPortHandling(t *testing.T) {
	// Test default port handling
	fastembed, err := NewFastEmbed(FastEmbedOptions{
		ConnectorName: "test-fastembed-defaults-basic", // reuse existing connector
		Dimension:     384,
	})
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Should use default port 8000
	if fastembed.GetHost() != "http://127.0.0.1:8000" {
		t.Errorf("Expected default host http://127.0.0.1:8000, got %s", fastembed.GetHost())
	}

	t.Logf("Default port handling test passed: %s", fastembed.GetHost())
}

// ===== Sparse Vector Tests =====

func TestFastEmbed_EmbedQuery_Sparse(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	key := os.Getenv("FASTEMBED_TEST_KEY")

	if host == "" {
		t.Skip("Skipping FastEmbed sparse test: FASTEMBED_TEST_HOST not set")
	}

	// Create connector for FastEmbed with BM25 sparse model
	fastembedDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Sparse Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"key": "%s",
			"model": "Qdrant/bm25"
		}
	}`, host, key)

	_, err := connector.New("fastembed", "test-fastembed-sparse", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	options := FastEmbedOptions{
		ConnectorName: "test-fastembed-sparse",
		Concurrent:    5,
		Dimension:     0, // BM25 doesn't have fixed dimension
		Model:         "Qdrant/bm25",
	}

	fastembed, err := NewFastEmbed(options)
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Test single text embedding for sparse vectors
	testText := "The quick brown fox jumps over the lazy dog"

	var progressMessages []string
	callback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		message := fmt.Sprintf("Status: %s, Progress: %d/%d, Message: %s",
			status, payload.Current, payload.Total, payload.Message)
		progressMessages = append(progressMessages, message)
		t.Logf("Progress: %s", message)
	}

	ctx := context.Background()
	embeddingResult, err := fastembed.EmbedQuery(ctx, testText, callback)
	if err != nil {
		t.Fatalf("EmbedQuery failed: %v", err)
	}

	// Verify sparse embedding result
	assert.NotNil(t, embeddingResult)
	assert.Equal(t, 1, embeddingResult.Usage.TotalTexts)
	assert.Greater(t, embeddingResult.Usage.TotalTokens, 0)
	assert.Equal(t, types.EmbeddingTypeSparse, embeddingResult.Type)
	assert.Equal(t, "Qdrant/bm25", embeddingResult.Model)

	// Check sparse embedding structure
	assert.True(t, embeddingResult.IsSparse())
	assert.False(t, embeddingResult.IsDense())
	assert.Nil(t, embeddingResult.GetDenseEmbedding())

	// Get sparse embedding data
	indices, values := embeddingResult.GetSparseEmbedding()
	assert.NotNil(t, indices)
	assert.NotNil(t, values)
	assert.Equal(t, len(indices), len(values))
	assert.Greater(t, len(indices), 0, "Sparse embedding should have non-zero elements")

	// Verify indices and values
	for i, idx := range indices {
		assert.Greater(t, idx, uint32(0), "Index should be positive")
		assert.Greater(t, values[i], float32(0), "Value should be positive")
		t.Logf("Sparse element %d: index=%d, value=%f", i, idx, values[i])
	}

	// Verify progress callbacks were called
	assert.Greater(t, len(progressMessages), 0, "Expected progress callbacks to be called")

	t.Logf("Successfully embedded text as sparse vector with %d non-zero elements", len(indices))
	t.Logf("Model: %s, Type: %s", embeddingResult.Model, embeddingResult.Type)
}

func TestFastEmbed_EmbedDocuments_Sparse(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	key := os.Getenv("FASTEMBED_TEST_KEY")

	if host == "" {
		t.Skip("Skipping FastEmbed sparse test: FASTEMBED_TEST_HOST not set")
	}

	// Create connector for FastEmbed with BM25 sparse model
	fastembedDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Sparse Documents Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"key": "%s",
			"model": "Qdrant/bm25"
		}
	}`, host, key)

	_, err := connector.New("fastembed", "test-fastembed-sparse-docs", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	options := FastEmbedOptions{
		ConnectorName: "test-fastembed-sparse-docs",
		Concurrent:    3,
		Dimension:     0, // BM25 doesn't have fixed dimension
		Model:         "Qdrant/bm25",
	}

	fastembed, err := NewFastEmbed(options)
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Test multiple documents embedding for sparse vectors
	testDocuments := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning and artificial intelligence are transforming technology",
		"Natural language processing enables computers to understand human text",
		"Vector databases store and retrieve high-dimensional data efficiently",
	}

	var progressMessages []string
	callback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		message := fmt.Sprintf("Status: %s, Progress: %d/%d, Message: %s",
			status, payload.Current, payload.Total, payload.Message)
		progressMessages = append(progressMessages, message)
		t.Logf("Progress: %s", message)
	}

	ctx := context.Background()
	embeddingResults, err := fastembed.EmbedDocuments(ctx, testDocuments, callback)
	if err != nil {
		t.Fatalf("EmbedDocuments failed: %v", err)
	}

	// Verify sparse embeddings results
	assert.NotNil(t, embeddingResults)
	assert.Equal(t, len(testDocuments), embeddingResults.Count())
	assert.Equal(t, len(testDocuments), embeddingResults.Usage.TotalTexts)
	assert.Greater(t, embeddingResults.Usage.TotalTokens, 0)
	assert.Equal(t, types.EmbeddingTypeSparse, embeddingResults.Type)
	assert.Equal(t, "Qdrant/bm25", embeddingResults.Model)

	// Check sparse embeddings structure
	assert.True(t, embeddingResults.IsSparse())
	assert.False(t, embeddingResults.IsDense())
	assert.Nil(t, embeddingResults.GetDenseEmbeddings())

	// Get sparse embeddings data
	sparseEmbeddings := embeddingResults.GetSparseEmbeddings()
	assert.NotNil(t, sparseEmbeddings)
	assert.Equal(t, len(testDocuments), len(sparseEmbeddings))

	// Verify each sparse embedding
	for i, sparseEmb := range sparseEmbeddings {
		assert.Greater(t, len(sparseEmb.Indices), 0, "Document %d should have non-zero elements", i)
		assert.Equal(t, len(sparseEmb.Indices), len(sparseEmb.Values), "Document %d indices and values length should match", i)

		// Check indices and values
		for j, idx := range sparseEmb.Indices {
			assert.Greater(t, idx, uint32(0), "Document %d index %d should be positive", i, j)
			assert.Greater(t, sparseEmb.Values[j], float32(0), "Document %d value %d should be positive", i, j)
		}

		t.Logf("Document %d: %d non-zero elements", i, len(sparseEmb.Indices))
	}

	// Verify progress callbacks were called
	assert.Greater(t, len(progressMessages), 0, "Expected progress callbacks to be called")

	t.Logf("Successfully embedded %d documents as sparse vectors", embeddingResults.Count())
	t.Logf("Model: %s, Type: %s, Total tokens: %d",
		embeddingResults.Model, embeddingResults.Type, embeddingResults.Usage.TotalTokens)
}

func TestFastEmbed_SparseDenseComparison(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	key := os.Getenv("FASTEMBED_TEST_KEY")

	if host == "" {
		t.Skip("Skipping FastEmbed comparison test: FASTEMBED_TEST_HOST not set")
	}

	testText := "The quick brown fox jumps over the lazy dog"
	ctx := context.Background()

	// Test Dense Embedding (BAAI/bge-small-en-v1.5)
	denseDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Dense Comparison Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"key": "%s",
			"model": "BAAI/bge-small-en-v1.5"
		}
	}`, host, key)

	_, err := connector.New("fastembed", "test-fastembed-dense-comp", []byte(denseDSL))
	if err != nil {
		t.Fatalf("Failed to create dense FastEmbed connector: %v", err)
	}

	denseOptions := FastEmbedOptions{
		ConnectorName: "test-fastembed-dense-comp",
		Dimension:     384,
		Model:         "BAAI/bge-small-en-v1.5",
	}

	denseFastembed, err := NewFastEmbed(denseOptions)
	if err != nil {
		t.Fatalf("Failed to create dense FastEmbed: %v", err)
	}

	denseResult, err := denseFastembed.EmbedQuery(ctx, testText)
	if err != nil {
		t.Fatalf("Dense EmbedQuery failed: %v", err)
	}

	// Test Sparse Embedding (Qdrant/bm25)
	sparseDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Sparse Comparison Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"key": "%s",
			"model": "Qdrant/bm25"
		}
	}`, host, key)

	_, err = connector.New("fastembed", "test-fastembed-sparse-comp", []byte(sparseDSL))
	if err != nil {
		t.Fatalf("Failed to create sparse FastEmbed connector: %v", err)
	}

	sparseOptions := FastEmbedOptions{
		ConnectorName: "test-fastembed-sparse-comp",
		Dimension:     0,
		Model:         "Qdrant/bm25",
	}

	sparseFastembed, err := NewFastEmbed(sparseOptions)
	if err != nil {
		t.Fatalf("Failed to create sparse FastEmbed: %v", err)
	}

	sparseResult, err := sparseFastembed.EmbedQuery(ctx, testText)
	if err != nil {
		t.Fatalf("Sparse EmbedQuery failed: %v", err)
	}

	// Compare results
	assert.NotNil(t, denseResult)
	assert.NotNil(t, sparseResult)

	// Dense embedding checks
	assert.Equal(t, types.EmbeddingTypeDense, denseResult.Type)
	assert.True(t, denseResult.IsDense())
	assert.False(t, denseResult.IsSparse())
	assert.NotNil(t, denseResult.GetDenseEmbedding())
	assert.Equal(t, 384, len(denseResult.GetDenseEmbedding()))
	denseIndices, denseValues := denseResult.GetSparseEmbedding()
	assert.Nil(t, denseIndices)
	assert.Nil(t, denseValues)

	// Sparse embedding checks
	assert.Equal(t, types.EmbeddingTypeSparse, sparseResult.Type)
	assert.False(t, sparseResult.IsDense())
	assert.True(t, sparseResult.IsSparse())
	assert.Nil(t, sparseResult.GetDenseEmbedding())
	sparseIndices, sparseValues := sparseResult.GetSparseEmbedding()
	assert.NotNil(t, sparseIndices)
	assert.NotNil(t, sparseValues)
	assert.Greater(t, len(sparseIndices), 0)
	assert.Equal(t, len(sparseIndices), len(sparseValues))

	t.Logf("Dense embedding: Model=%s, Type=%s, Dimension=%d",
		denseResult.Model, denseResult.Type, len(denseResult.GetDenseEmbedding()))
	t.Logf("Sparse embedding: Model=%s, Type=%s, Non-zero elements=%d",
		sparseResult.Model, sparseResult.Type, len(sparseIndices))

	// Log some sparse values for inspection
	if len(sparseIndices) > 0 {
		maxShow := 5
		if len(sparseIndices) < maxShow {
			maxShow = len(sparseIndices)
		}
		t.Logf("First %d sparse elements:", maxShow)
		for i := 0; i < maxShow; i++ {
			t.Logf("  [%d]: index=%d, value=%f", i, sparseIndices[i], sparseValues[i])
		}
	}
}

func TestFastEmbed_SparseEmptyText(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	key := os.Getenv("FASTEMBED_TEST_KEY")

	if host == "" {
		t.Skip("Skipping FastEmbed sparse empty test: FASTEMBED_TEST_HOST not set")
	}

	// Create connector for sparse model
	fastembedDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed Sparse Empty Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"key": "%s",
			"model": "Qdrant/bm25"
		}
	}`, host, key)

	_, err := connector.New("fastembed", "test-fastembed-sparse-empty", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	fastembed, err := NewFastEmbed(FastEmbedOptions{
		ConnectorName: "test-fastembed-sparse-empty",
		Model:         "Qdrant/bm25",
	})
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	ctx := context.Background()

	// Test empty text for sparse model
	embeddingResult, err := fastembed.EmbedQuery(ctx, "")
	if err != nil {
		t.Fatalf("EmbedQuery with empty text failed: %v", err)
	}
	assert.Nil(t, embeddingResult)

	// Test empty documents array for sparse model
	embeddingResults, err := fastembed.EmbedDocuments(ctx, []string{})
	if err != nil {
		t.Fatalf("EmbedDocuments with empty array failed: %v", err)
	}
	assert.Nil(t, embeddingResults)

	t.Log("Sparse model empty text handling works correctly")
}

func TestFastEmbed_SparseModelVariations(t *testing.T) {
	// Skip if no test environment
	host := os.Getenv("FASTEMBED_TEST_HOST")
	key := os.Getenv("FASTEMBED_TEST_KEY")

	if host == "" {
		t.Skip("Skipping FastEmbed sparse variations test: FASTEMBED_TEST_HOST not set")
	}

	testText := "artificial intelligence machine learning natural language processing"
	ctx := context.Background()

	// Test BM25 model specifically
	bm25DSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed BM25 Test",
		"type": "fastembed",
		"options": {
			"host": "%s",
			"key": "%s",
			"model": "Qdrant/bm25"
		}
	}`, host, key)

	_, err := connector.New("fastembed", "test-fastembed-bm25", []byte(bm25DSL))
	if err != nil {
		t.Fatalf("Failed to create BM25 FastEmbed connector: %v", err)
	}

	bm25Fastembed, err := NewFastEmbed(FastEmbedOptions{
		ConnectorName: "test-fastembed-bm25",
		Model:         "Qdrant/bm25",
	})
	if err != nil {
		t.Fatalf("Failed to create BM25 FastEmbed: %v", err)
	}

	bm25Result, err := bm25Fastembed.EmbedQuery(ctx, testText)
	if err != nil {
		t.Fatalf("BM25 EmbedQuery failed: %v", err)
	}

	// Verify BM25 results
	assert.NotNil(t, bm25Result)
	assert.Equal(t, types.EmbeddingTypeSparse, bm25Result.Type)
	assert.Equal(t, "Qdrant/bm25", bm25Result.Model)
	assert.True(t, bm25Result.IsSparse())

	indices, values := bm25Result.GetSparseEmbedding()
	assert.NotNil(t, indices)
	assert.NotNil(t, values)
	assert.Greater(t, len(indices), 0)

	t.Logf("BM25 model test passed: %d non-zero elements", len(indices))
	t.Logf("Usage: TotalTokens=%d, TotalTexts=%d",
		bm25Result.Usage.TotalTokens, bm25Result.Usage.TotalTexts)

	// Verify specific BM25 characteristics
	// BM25 typically assigns higher weights to less frequent terms
	for i, val := range values {
		assert.Greater(t, val, float32(0), "BM25 values should be positive")
		t.Logf("BM25 element %d: index=%d, value=%f", i, indices[i], val)
	}
}
