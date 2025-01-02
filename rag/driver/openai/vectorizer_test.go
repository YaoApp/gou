package openai_test

import (
	"context"
	"os"
	"testing"

	"github.com/yaoapp/gou/rag/driver/openai"
)

func TestOpenAIVectorizer(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set")
	}

	// Create vectorizer
	vectorizer, err := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "text-embedding-ada-002",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer vectorizer.Close()

	ctx := context.Background()

	// Test single text vectorization
	t.Run("single text", func(t *testing.T) {
		text := "This is a test document for OpenAI embeddings."
		embedding, err := vectorizer.Vectorize(ctx, text)
		if err != nil {
			t.Fatal(err)
		}

		// text-embedding-ada-002 produces 1536-dimensional vectors
		if len(embedding) != 1536 {
			t.Errorf("expected 1536 dimensions, got %d", len(embedding))
		}

		// Check if vector is normalized
		var sum float32
		for _, v := range embedding {
			sum += v * v
		}
		if sum < 0.99 || sum > 1.01 { // Allow for small floating point errors
			t.Errorf("vector not normalized, magnitude squared = %f", sum)
		}
	})

	// Test batch vectorization
	t.Run("batch texts", func(t *testing.T) {
		texts := []string{
			"First test document",
			"Second test document",
			"Third test document with more content",
		}

		embeddings, err := vectorizer.VectorizeBatch(ctx, texts)
		if err != nil {
			t.Fatal(err)
		}

		if len(embeddings) != len(texts) {
			t.Errorf("expected %d embeddings, got %d", len(texts), len(embeddings))
		}

		for i, embedding := range embeddings {
			if len(embedding) != 1536 {
				t.Errorf("embedding %d: expected 1536 dimensions, got %d", i, len(embedding))
			}

			// Check if vector is normalized
			var sum float32
			for _, v := range embedding {
				sum += v * v
			}
			if sum < 0.99 || sum > 1.01 {
				t.Errorf("embedding %d: vector not normalized, magnitude squared = %f", i, sum)
			}
		}
	})

	// Test semantic similarity
	t.Run("semantic similarity", func(t *testing.T) {
		pairs := []struct {
			text1, text2 string
			similar      bool
		}{
			{
				text1:   "The quick brown fox jumps over the lazy dog",
				text2:   "A fast auburn canine leaps above a sleepy hound",
				similar: true,
			},
			{
				text1:   "I love programming in Go",
				text2:   "Python is my favorite programming language",
				similar: true,
			},
			{
				text1:   "The weather is nice today",
				text2:   "Quantum mechanics is a fascinating subject",
				similar: false,
			},
		}

		for _, pair := range pairs {
			vec1, err := vectorizer.Vectorize(ctx, pair.text1)
			if err != nil {
				t.Fatal(err)
			}

			vec2, err := vectorizer.Vectorize(ctx, pair.text2)
			if err != nil {
				t.Fatal(err)
			}

			// Calculate cosine similarity
			var dotProduct float32
			for i := 0; i < len(vec1); i++ {
				dotProduct += vec1[i] * vec2[i]
			}

			t.Logf("Similarity between '%s' and '%s': %f", pair.text1, pair.text2, dotProduct)

			if pair.similar && dotProduct < 0.8 {
				t.Errorf("expected similar texts to have similarity > 0.8, got %f", dotProduct)
			}
			if !pair.similar && dotProduct > 0.8 {
				t.Errorf("expected dissimilar texts to have similarity < 0.8, got %f", dotProduct)
			}
		}
	})
}
