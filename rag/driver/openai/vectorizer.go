package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Vectorizer implements driver.Vectorizer using OpenAI's embeddings API
type Vectorizer struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// Config contains configuration for OpenAI vectorizer
type Config struct {
	APIKey string // OpenAI API key
	Model  string // Model to use for embeddings, e.g., "text-embedding-ada-002"
}

// EmbeddingResponse represents the response from OpenAI embeddings API
type EmbeddingResponse struct {
	Data  []EmbeddingData `json:"data"`
	Model string          `json:"model"`
}

// EmbeddingData represents a single embedding in the response
type EmbeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// New creates a new OpenAI vectorizer
func New(config Config) (*Vectorizer, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_TEST_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key not provided")
		}
	}

	model := config.Model
	if model == "" {
		model = "text-embedding-ada-002" // Default model
	}

	return &Vectorizer{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Vectorize implements driver.Vectorizer
func (v *Vectorizer) Vectorize(ctx context.Context, text string) ([]float32, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"model": v.model,
		"input": text,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.apiKey)

	// Send request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	// Parse response
	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embedResp.Data[0].Embedding, nil
}

// VectorizeBatch implements driver.Vectorizer
func (v *Vectorizer) VectorizeBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"model": v.model,
		"input": texts,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.apiKey)

	// Send request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	// Parse response
	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedResp.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(embedResp.Data))
	}

	// Sort embeddings by index
	embeddings := make([][]float32, len(texts))
	for _, data := range embedResp.Data {
		embeddings[data.Index] = data.Embedding
	}

	return embeddings, nil
}

// Close implements driver.Vectorizer
func (v *Vectorizer) Close() error {
	return nil
}
