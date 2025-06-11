package embedding

import (
	"context"
	"fmt"
	"sync"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/http"
)

// Default embedding function

// Openai embedding function
type Openai struct {
	Connector     connector.Connector
	MaxConcurrent int
}

// NewOpenai create a new Openai embedding function
func NewOpenai(connectorName string, maxConcurrent int) (*Openai, error) {
	c, err := connector.Select(connectorName)
	if err != nil {
		return nil, err
	}

	if !c.Is(connector.OPENAI) {
		return nil, fmt.Errorf("The connector %s is not a OpenAI connector", connectorName)
	}

	if maxConcurrent <= 0 {
		maxConcurrent = 10 // Default value
	}

	return &Openai{
		Connector:     c,
		MaxConcurrent: maxConcurrent,
	}, nil
}

// NewOpenaiWithDefaults create a new Openai embedding function with default settings
func NewOpenaiWithDefaults(connectorName string) (*Openai, error) {
	return NewOpenai(connectorName, 10)
}

// EmbedDocuments embed documents
func (e *Openai) EmbedDocuments(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	// Use concurrent requests for better performance
	embeddings := make([][]float64, len(texts))
	errors := make([]error, len(texts))
	var wg sync.WaitGroup

	// Limit concurrent requests to avoid rate limiting
	maxConcurrent := e.MaxConcurrent
	if len(texts) < maxConcurrent {
		maxConcurrent = len(texts)
	}

	semaphore := make(chan struct{}, maxConcurrent)

	for i, text := range texts {
		wg.Add(1)
		go func(index int, inputText string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			embedding, err := e.EmbedQuery(ctx, inputText)
			if err != nil {
				errors[index] = err
				return
			}
			embeddings[index] = embedding
		}(i, text)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("error embedding text at index %d: %w", i, err)
		}
	}

	return embeddings, nil
}

// EmbedQuery embed query
func (e *Openai) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return []float64{}, nil
	}

	payload := map[string]interface{}{
		"input": text,
	}

	response, err := e.post("embeddings", payload)
	if err != nil {
		return nil, err
	}

	// Parse response
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	data, ok := respMap["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no data field in response")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	firstItem, ok := data[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected first item format")
	}

	embedding, ok := firstItem["embedding"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no embedding field in response")
	}

	embeddingFloat := make([]float64, len(embedding))
	for i, val := range embedding {
		if floatVal, ok := val.(float64); ok {
			embeddingFloat[i] = floatVal
		} else {
			return nil, fmt.Errorf("invalid embedding value at position %d", i)
		}
	}

	return embeddingFloat, nil
}

// GetDimension get dimension
func (e *Openai) GetDimension() int {
	model := e.getModel()
	switch model {
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 2560
	}
	return 0
}

func (e *Openai) getModel() string {
	setting := e.Connector.Setting()
	model := setting["model"].(string)
	if model == "" {
		model = "text-embedding-3-small"
	}
	return model
}

func (e *Openai) post(endpoint string, payload map[string]interface{}) (interface{}, error) {
	setting := e.Connector.Setting()
	host := "https://api.openai.com/v1"

	// Proxy
	if proxy, ok := setting["proxy"].(string); ok {
		host = proxy
	}

	apiKey := setting["key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("API key is not set")
	}
	url := fmt.Sprintf("%s/%s", host, endpoint)
	payload["model"] = e.getModel()

	r := http.New(url)
	r.SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	r.SetHeader("Content-Type", "application/json")

	resp := r.Post(payload)
	if resp.Status != 200 {
		return nil, fmt.Errorf("request failed with status: %d", resp.Status)
	}

	return resp.Data, nil
}
