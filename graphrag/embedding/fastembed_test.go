package embedding

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/yaoapp/gou/connector"
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
	callback := func(status Status, payload Payload) {
		message := fmt.Sprintf("Status: %s, Progress: %d/%d, Message: %s",
			status, payload.Current, payload.Total, payload.Message)
		progressMessages = append(progressMessages, message)
		t.Logf("Progress: %s", message)
	}

	ctx := context.Background()
	embedding, err := fastembed.EmbedQuery(ctx, testText, callback)
	if err != nil {
		t.Fatalf("EmbedQuery failed: %v", err)
	}

	// Verify embedding
	if len(embedding) != 384 {
		t.Errorf("Expected embedding dimension 384, got %d", len(embedding))
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
		t.Error("Embedding contains all zero values")
	}

	// Verify progress callbacks were called
	if len(progressMessages) == 0 {
		t.Error("Expected progress callbacks to be called")
	}

	t.Logf("Successfully embedded text with %d dimensions", len(embedding))
	t.Logf("First 10 embedding values: %v", embedding[:10])
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
	callback := func(status Status, payload Payload) {
		message := fmt.Sprintf("Status: %s, Progress: %d/%d, Message: %s",
			status, payload.Current, payload.Total, payload.Message)
		progressMessages = append(progressMessages, message)
		t.Logf("Progress: %s", message)
	}

	ctx := context.Background()
	embeddings, err := fastembed.EmbedDocuments(ctx, testDocuments, callback)
	if err != nil {
		t.Fatalf("EmbedDocuments failed: %v", err)
	}

	// Verify embeddings
	if len(embeddings) != len(testDocuments) {
		t.Errorf("Expected %d embeddings, got %d", len(testDocuments), len(embeddings))
	}

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

	t.Logf("Successfully embedded %d documents", len(embeddings))
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
	embedding, err := fastembed.EmbedQuery(ctx, "")
	if err != nil {
		t.Fatalf("EmbedQuery with empty text failed: %v", err)
	}
	if len(embedding) != 0 {
		t.Errorf("Expected empty embedding for empty text, got %d dimensions", len(embedding))
	}

	// Test empty documents array
	embeddings, err := fastembed.EmbedDocuments(ctx, []string{})
	if err != nil {
		t.Fatalf("EmbedDocuments with empty array failed: %v", err)
	}
	if len(embeddings) != 0 {
		t.Errorf("Expected empty embeddings for empty array, got %d embeddings", len(embeddings))
	}

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

func TestFastEmbed_MissingHost(t *testing.T) {
	// Create connector without host
	fastembedDSL := `{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "FastEmbed No Host Test",
		"type": "fastembed",
		"options": {
			"model": "BAAI/bge-small-en-v1.5"
		}
	}`

	_, err := connector.New("fastembed", "test-fastembed-nohost", []byte(fastembedDSL))
	if err != nil {
		t.Fatalf("Failed to create FastEmbed connector: %v", err)
	}

	// Test with no host in options - should use connector's default host (127.0.0.1:8000)
	fastembed, err := NewFastEmbed(FastEmbedOptions{
		ConnectorName: "test-fastembed-nohost",
		Concurrent:    5,
		Dimension:     384,
		// No Host specified
	})
	if err != nil {
		t.Fatalf("Failed to create FastEmbed: %v", err)
	}

	// Should use default host
	if fastembed.GetHost() != "http://127.0.0.1:8000" {
		t.Errorf("Expected default host http://127.0.0.1:8000, got %s", fastembed.GetHost())
	}

	t.Logf("FastEmbed correctly used default host: %s", fastembed.GetHost())
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

	// Test empty text - should return empty slice without making HTTP request
	embedding, err := fastembed.EmbedQuery(ctx, "")
	if err != nil {
		t.Fatalf("EmbedQuery with empty text failed: %v", err)
	}
	if len(embedding) != 0 {
		t.Errorf("Expected empty embedding for empty text, got %d dimensions", len(embedding))
	}

	// Test empty texts array - should return empty slice without making HTTP request
	embeddings, err := fastembed.EmbedDocuments(ctx, []string{})
	if err != nil {
		t.Fatalf("EmbedDocuments with empty array failed: %v", err)
	}
	if len(embeddings) != 0 {
		t.Errorf("Expected empty embeddings for empty array, got %d embeddings", len(embeddings))
	}

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
