package embedding

import (
	"context"
	"fmt"
	"sync"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/utils"
)

// Status defines the status of embedding process
type Status string

// Status constants for embedding process
const (
	StatusStarting   Status = "starting"   // Starting the embedding process
	StatusProcessing Status = "processing" // Processing embeddings
	StatusCompleted  Status = "completed"  // Successfully completed
	StatusError      Status = "error"      // Error occurred
)

// Payload contains context-specific data for different embedding scenarios
type Payload struct {
	// Common fields
	Current int    `json:"current"` // Current progress count
	Total   int    `json:"total"`   // Total items to process
	Message string `json:"message"` // Status message

	// Document embedding specific
	DocumentIndex *int    `json:"document_index,omitempty"` // Index of current document being processed
	DocumentText  *string `json:"document_text,omitempty"`  // Text being processed (truncated if too long)

	// Error specific
	Error error `json:"error,omitempty"` // Error details when Status is StatusError
}

// ProgressCallback defines the callback function for progress reporting with flexible payload
type ProgressCallback func(status Status, payload Payload)

// OpenaiOptions defines the options for OpenAI embedding
type OpenaiOptions struct {
	ConnectorName string // Connector name
	Concurrent    int    // Maximum concurrent requests
	Dimension     int    // Embedding dimension
	Model         string // Model name (optional, can be overridden by connector)
}

// Openai embedding function
type Openai struct {
	Connector  connector.Connector
	Concurrent int
	Dimension  int
	Model      string
}

// NewOpenai create a new Openai embedding function with options
func NewOpenai(options OpenaiOptions) (*Openai, error) {
	c, err := connector.Select(options.ConnectorName)
	if err != nil {
		return nil, err
	}

	if !c.Is(connector.OPENAI) {
		return nil, fmt.Errorf("The connector %s is not a OpenAI connector", options.ConnectorName)
	}

	if options.Concurrent <= 0 {
		options.Concurrent = 10 // Default value
	}

	if options.Dimension <= 0 {
		options.Dimension = 1536 // Default dimension for text-embedding-3-small
	}

	// Get model from connector settings if not specified in options
	model := options.Model
	if model == "" {
		setting := c.Setting()
		if connectorModel, ok := setting["model"].(string); ok && connectorModel != "" {
			model = connectorModel
		} else {
			model = "text-embedding-3-small" // Default model
		}
	}

	return &Openai{
		Connector:  c,
		Concurrent: options.Concurrent,
		Dimension:  options.Dimension,
		Model:      model,
	}, nil
}

// NewOpenaiWithDefaults create a new Openai embedding function with default settings
func NewOpenaiWithDefaults(connectorName string) (*Openai, error) {
	return NewOpenai(OpenaiOptions{
		ConnectorName: connectorName,
		Concurrent:    10,
		Dimension:     1536,
	})
}

// EmbedDocuments embed documents with optional progress callback
func (e *Openai) EmbedDocuments(ctx context.Context, texts []string, callback ...ProgressCallback) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	var cb ProgressCallback
	if len(callback) > 0 && callback[0] != nil {
		cb = callback[0]
	}

	// Report initial progress
	if cb != nil {
		cb(StatusStarting, Payload{
			Current: 0,
			Total:   len(texts),
			Message: "Starting document embedding...",
		})
	}

	// Use concurrent requests for better performance
	embeddings := make([][]float64, len(texts))
	errors := make([]error, len(texts))
	completed := make([]bool, len(texts))
	var wg sync.WaitGroup
	var mu sync.Mutex
	completedCount := 0

	// Limit concurrent requests to avoid rate limiting
	maxConcurrent := e.Concurrent
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

			// Create a callback for individual document processing
			var docCallback ProgressCallback
			if cb != nil {
				docCallback = func(status Status, payload Payload) {
					// Enhance payload with document-specific info
					payload.DocumentIndex = &index
					truncatedText := inputText
					if len(truncatedText) > 100 {
						truncatedText = truncatedText[:100] + "..."
					}
					payload.DocumentText = &truncatedText
					cb(status, payload)
				}
			}

			embedding, err := e.EmbedQuery(ctx, inputText, docCallback)
			if err != nil {
				errors[index] = err
				// Report error for this item
				if cb != nil {
					cb(StatusError, Payload{
						Current:       completedCount + 1,
						Total:         len(texts),
						Message:       fmt.Sprintf("Error embedding document %d", index+1),
						DocumentIndex: &index,
						Error:         err,
					})
				}
			} else {
				embeddings[index] = embedding
			}

			// Update progress
			mu.Lock()
			completed[index] = true
			completedCount++
			if cb != nil {
				cb(StatusProcessing, Payload{
					Current:       completedCount,
					Total:         len(texts),
					Message:       fmt.Sprintf("Completed %d/%d documents", completedCount, len(texts)),
					DocumentIndex: &index,
				})
			}
			mu.Unlock()
		}(i, text)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			if cb != nil {
				cb(StatusError, Payload{
					Current: len(texts),
					Total:   len(texts),
					Message: fmt.Sprintf("Failed to embed all documents, error at index %d", i),
					Error:   err,
				})
			}
			return nil, fmt.Errorf("error embedding text at index %d: %w", i, err)
		}
	}

	// Report completion
	if cb != nil {
		cb(StatusCompleted, Payload{
			Current: len(texts),
			Total:   len(texts),
			Message: "Document embedding completed successfully",
		})
	}

	return embeddings, nil
}

// EmbedQuery embed query using direct POST request with optional progress callback
func (e *Openai) EmbedQuery(ctx context.Context, text string, callback ...ProgressCallback) ([]float64, error) {
	if text == "" {
		return []float64{}, nil
	}

	var cb ProgressCallback
	if len(callback) > 0 && callback[0] != nil {
		cb = callback[0]
	}

	// Report starting
	if cb != nil {
		cb(StatusStarting, Payload{
			Current: 0,
			Total:   1,
			Message: "Starting text embedding...",
		})
	}

	payload := map[string]interface{}{
		"input": text,
		"model": e.Model,
	}

	// Report processing
	if cb != nil {
		cb(StatusProcessing, Payload{
			Current: 0,
			Total:   1,
			Message: "Sending request to OpenAI...",
		})
	}

	// Use direct POST request
	result, err := utils.PostLLM(ctx, e.Connector, "embeddings", payload)
	if err != nil {
		if cb != nil {
			cb(StatusError, Payload{
				Current: 1,
				Total:   1,
				Message: "Request failed",
				Error:   err,
			})
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Parse response
	respMap, ok := result.(map[string]interface{})
	if !ok {
		if cb != nil {
			cb(StatusError, Payload{
				Current: 1,
				Total:   1,
				Message: "Unexpected response format",
			})
		}
		return nil, fmt.Errorf("unexpected response format")
	}

	data, ok := respMap["data"].([]interface{})
	if !ok {
		if cb != nil {
			cb(StatusError, Payload{
				Current: 1,
				Total:   1,
				Message: "No data field in response",
			})
		}
		return nil, fmt.Errorf("no data field in response")
	}

	if len(data) == 0 {
		if cb != nil {
			cb(StatusError, Payload{
				Current: 1,
				Total:   1,
				Message: "No embedding data returned",
			})
		}
		return nil, fmt.Errorf("no embedding data returned")
	}

	firstItem, ok := data[0].(map[string]interface{})
	if !ok {
		if cb != nil {
			cb(StatusError, Payload{
				Current: 1,
				Total:   1,
				Message: "Unexpected first item format",
			})
		}
		return nil, fmt.Errorf("unexpected first item format")
	}

	embedding, ok := firstItem["embedding"].([]interface{})
	if !ok {
		if cb != nil {
			cb(StatusError, Payload{
				Current: 1,
				Total:   1,
				Message: "No embedding field in response",
			})
		}
		return nil, fmt.Errorf("no embedding field in response")
	}

	embeddingFloat := make([]float64, len(embedding))
	for i, val := range embedding {
		if floatVal, ok := val.(float64); ok {
			embeddingFloat[i] = floatVal
		} else {
			if cb != nil {
				cb(StatusError, Payload{
					Current: 1,
					Total:   1,
					Message: fmt.Sprintf("Invalid embedding value at position %d", i),
				})
			}
			return nil, fmt.Errorf("invalid embedding value at position %d", i)
		}
	}

	// Validate dimension matches expected
	if len(embeddingFloat) != e.Dimension {
		if cb != nil {
			cb(StatusError, Payload{
				Current: 1,
				Total:   1,
				Message: fmt.Sprintf("Dimension mismatch: got %d, expected %d", len(embeddingFloat), e.Dimension),
			})
		}
		return nil, fmt.Errorf("received embedding dimension %d does not match expected dimension %d", len(embeddingFloat), e.Dimension)
	}

	// Report completion
	if cb != nil {
		cb(StatusCompleted, Payload{
			Current: 1,
			Total:   1,
			Message: "Text embedding completed successfully",
		})
	}

	return embeddingFloat, nil
}

// GetModel returns the current model being used
func (e *Openai) GetModel() string {
	return e.Model
}

// GetDimension returns the embedding dimension
func (e *Openai) GetDimension() int {
	return e.Dimension
}
