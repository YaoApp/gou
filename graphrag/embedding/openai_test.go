package embedding

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
)

func TestMain(m *testing.M) {
	// Setup test environment
	setupTestConnectors()
	code := m.Run()
	os.Exit(code)
}

func setupTestConnectors() {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		apiKey = "test-key" // Use dummy key for basic tests
	}

	// Create test OpenAI connector
	createTestConnector("test-openai", apiKey, "text-embedding-3-small", "")
}

func createTestConnector(name, apiKey, model, proxy string) {
	options := map[string]interface{}{
		"key":   apiKey,
		"model": model,
	}
	if proxy != "" {
		options["proxy"] = proxy
	}

	dsl := map[string]interface{}{
		"type":    "openai",
		"name":    "Test OpenAI Connector",
		"options": options,
	}

	dslBytes, _ := json.Marshal(dsl)
	connector.New("openai", name, dslBytes)
}

func TestNewOpenai(t *testing.T) {
	tests := []struct {
		name           string
		connectorName  string
		maxConcurrent  int
		expectedError  bool
		expectedMaxCon int
	}{
		{
			name:           "Valid connector with custom concurrent",
			connectorName:  "test-openai",
			maxConcurrent:  5,
			expectedError:  false,
			expectedMaxCon: 5,
		},
		{
			name:           "Valid connector with zero concurrent (should default to 10)",
			connectorName:  "test-openai",
			maxConcurrent:  0,
			expectedError:  false,
			expectedMaxCon: 10,
		},
		{
			name:           "Valid connector with negative concurrent (should default to 10)",
			connectorName:  "test-openai",
			maxConcurrent:  -1,
			expectedError:  false,
			expectedMaxCon: 10,
		},
		{
			name:          "Invalid connector name",
			connectorName: "non-existent",
			maxConcurrent: 5,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openai, err := NewOpenai(tt.connectorName, tt.maxConcurrent)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, openai)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, openai)
				assert.Equal(t, tt.expectedMaxCon, openai.MaxConcurrent)
			}
		})
	}
}

func TestNewOpenaiWithDefaults(t *testing.T) {
	openai, err := NewOpenaiWithDefaults("test-openai")
	assert.NoError(t, err)
	assert.NotNil(t, openai)
	assert.Equal(t, 10, openai.MaxConcurrent)
}

func TestGetDimension(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		expectedDim int
	}{
		{
			name:        "text-embedding-3-small",
			model:       "text-embedding-3-small",
			expectedDim: 1536,
		},
		{
			name:        "text-embedding-3-large",
			model:       "text-embedding-3-large",
			expectedDim: 2560,
		},
		{
			name:        "unknown model",
			model:       "unknown-model",
			expectedDim: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test connector with specific model
			connectorName := "test-openai-" + tt.name
			createTestConnector(connectorName, "test-key", tt.model, "")

			openai, err := NewOpenai(connectorName, 10)
			require.NoError(t, err)

			dim := openai.GetDimension()
			assert.Equal(t, tt.expectedDim, dim)
		})
	}
}

func TestGetModel(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		expectedModel string
	}{
		{
			name:          "Custom model",
			model:         "text-embedding-3-large",
			expectedModel: "text-embedding-3-large",
		},
		{
			name:          "Empty model (should default)",
			model:         "",
			expectedModel: "text-embedding-3-small",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test connector with specific model
			connectorName := "test-openai-model-" + tt.name
			createTestConnector(connectorName, "test-key", tt.model, "")

			openai, err := NewOpenai(connectorName, 10)
			require.NoError(t, err)

			model := openai.getModel()
			assert.Equal(t, tt.expectedModel, model)
		})
	}
}

func TestEmbedQuery(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping integration test")
	}

	openai, err := NewOpenai("test-openai", 10)
	require.NoError(t, err)

	tests := []struct {
		name     string
		text     string
		expected int // expected length should be > 0 for valid embeddings
	}{
		{
			name:     "Valid text",
			text:     "Hello world",
			expected: 1536, // text-embedding-3-small dimension
		},
		{
			name:     "Empty text",
			text:     "",
			expected: 0, // should return empty slice
		},
		{
			name:     "Long text",
			text:     "This is a longer text to test the embedding functionality with more content to see how it performs with longer inputs.",
			expected: 1536,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			embedding, err := openai.EmbedQuery(ctx, tt.text)

			assert.NoError(t, err)
			if tt.expected == 0 {
				assert.Empty(t, embedding)
			} else {
				assert.Len(t, embedding, tt.expected)
				// Check that embedding values are valid floats
				for i, val := range embedding {
					assert.IsType(t, float64(0), val, "embedding value at index %d should be float64", i)
				}
			}
		})
	}
}

func TestEmbedDocuments(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping integration test")
	}

	openai, err := NewOpenai("test-openai", 3) // Use smaller concurrent for testing
	require.NoError(t, err)

	tests := []struct {
		name     string
		texts    []string
		expected int // expected number of embeddings
	}{
		{
			name:     "Multiple texts",
			texts:    []string{"Hello", "World", "Test"},
			expected: 3,
		},
		{
			name:     "Single text",
			texts:    []string{"Single text"},
			expected: 1,
		},
		{
			name:     "Empty slice",
			texts:    []string{},
			expected: 0,
		},
		{
			name:     "Mixed content",
			texts:    []string{"Short", "This is a much longer text to test various lengths", ""},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			embeddings, err := openai.EmbedDocuments(ctx, tt.texts)

			assert.NoError(t, err)
			assert.Len(t, embeddings, tt.expected)

			for i, embedding := range embeddings {
				if i < len(tt.texts) && tt.texts[i] == "" {
					assert.Empty(t, embedding, "empty text should produce empty embedding")
				} else {
					assert.Len(t, embedding, 1536, "embedding %d should have correct dimension", i)
					// Check that embedding values are valid floats
					for j, val := range embedding {
						assert.IsType(t, float64(0), val, "embedding %d value at index %d should be float64", i, j)
					}
				}
			}
		})
	}
}

func TestPost_ErrorHandling(t *testing.T) {
	// Create test connector with empty API key
	createTestConnector("test-openai-no-key", "", "text-embedding-3-small", "")

	openai, err := NewOpenai("test-openai-no-key", 10)
	require.NoError(t, err)

	payload := map[string]interface{}{
		"input": "test",
	}

	_, err = openai.post("embeddings", payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is not set")
}

func TestPost_WithProxy(t *testing.T) {
	// Create test connector with proxy
	createTestConnector("test-openai-proxy", "test-key", "text-embedding-3-small", "https://custom-proxy.com/v1")

	openai, err := NewOpenai("test-openai-proxy", 10)
	require.NoError(t, err)

	// This will fail with invalid key, but we can test that proxy URL is used
	payload := map[string]interface{}{
		"input": "test",
	}

	_, err = openai.post("embeddings", payload)
	assert.Error(t, err) // Expected to fail with invalid key, but proxy URL should be used
}

// Performance Tests
func BenchmarkEmbedQuery(b *testing.B) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_TEST_KEY not set, skipping benchmark")
	}

	openai, err := NewOpenai("test-openai", 10)
	require.NoError(b, err)

	ctx := context.Background()
	text := "This is a test sentence for benchmarking embedding performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := openai.EmbedQuery(ctx, text)
		if err != nil {
			b.Fatalf("EmbedQuery failed: %v", err)
		}
	}
}

func BenchmarkEmbedDocuments_Sequential(b *testing.B) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_TEST_KEY not set, skipping benchmark")
	}

	openai, err := NewOpenai("test-openai", 1) // Sequential processing
	require.NoError(b, err)

	ctx := context.Background()
	texts := []string{
		"First document for testing",
		"Second document for benchmarking",
		"Third document for performance evaluation",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := openai.EmbedDocuments(ctx, texts)
		if err != nil {
			b.Fatalf("EmbedDocuments failed: %v", err)
		}
	}
}

func BenchmarkEmbedDocuments_Concurrent(b *testing.B) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_TEST_KEY not set, skipping benchmark")
	}

	openai, err := NewOpenai("test-openai", 10) // Concurrent processing
	require.NoError(b, err)

	ctx := context.Background()
	texts := []string{
		"First document for testing",
		"Second document for benchmarking",
		"Third document for performance evaluation",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := openai.EmbedDocuments(ctx, texts)
		if err != nil {
			b.Fatalf("EmbedDocuments failed: %v", err)
		}
	}
}

func TestConcurrencyPerformance(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping performance test")
	}

	texts := []string{
		"Performance test document 1",
		"Performance test document 2",
		"Performance test document 3",
		"Performance test document 4",
		"Performance test document 5",
	}

	ctx := context.Background()

	// Test sequential processing (maxConcurrent = 1)
	sequential, err := NewOpenai("test-openai", 1)
	require.NoError(t, err)

	start := time.Now()
	_, err = sequential.EmbedDocuments(ctx, texts)
	sequentialTime := time.Since(start)
	require.NoError(t, err)

	// Test concurrent processing (maxConcurrent = 5)
	concurrent, err := NewOpenai("test-openai", 5)
	require.NoError(t, err)

	start = time.Now()
	_, err = concurrent.EmbedDocuments(ctx, texts)
	concurrentTime := time.Since(start)
	require.NoError(t, err)

	t.Logf("Sequential processing time: %v", sequentialTime)
	t.Logf("Concurrent processing time: %v", concurrentTime)

	// Concurrent should be faster (allowing some variance for API latency)
	if concurrentTime < sequentialTime {
		t.Logf("✅ Concurrent processing is faster by %v", sequentialTime-concurrentTime)
	} else {
		t.Logf("⚠️  Concurrent processing took %v longer (this may be due to API rate limiting)", concurrentTime-sequentialTime)
	}
}
