package embedding

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/http"
)

// FastEmbedOptions defines the options for FastEmbed embedding
type FastEmbedOptions struct {
	ConnectorName string // Connector name
	Concurrent    int    // Maximum concurrent requests
	Dimension     int    // Embedding dimension
	Model         string // Model name (optional, can be overridden by connector)
	Host          string // FastEmbed service host (optional, can be overridden by connector)
	Key           string // FastEmbed service key (optional, can be overridden by connector)
}

// FastEmbed embedding function
type FastEmbed struct {
	Connector  connector.Connector
	Concurrent int
	Dimension  int
	Model      string
	Host       string
	Key        string
}

// NewFastEmbed create a new FastEmbed embedding function with options
func NewFastEmbed(options FastEmbedOptions) (*FastEmbed, error) {
	c, err := connector.Select(options.ConnectorName)
	if err != nil {
		return nil, err
	}

	if options.Concurrent <= 0 {
		options.Concurrent = 10 // Default value
	}

	if options.Dimension <= 0 {
		options.Dimension = 384 // Default dimension for BAAI/bge-small-en-v1.5
	}

	// Get settings from connector
	setting := c.Setting()

	// Get model from connector settings if not specified in options
	model := options.Model
	if model == "" {
		if connectorModel, ok := setting["model"].(string); ok && connectorModel != "" {
			model = connectorModel
		} else {
			model = "BAAI/bge-small-en-v1.5" // Default model
		}
	}

	// Get host from connector settings if not specified in options
	host := options.Host
	if host == "" {
		if connectorHost, ok := setting["host"].(string); ok && connectorHost != "" {
			host = connectorHost
		} else {
			return nil, fmt.Errorf("FastEmbed host is required. Please set 'host' in connector options or 'Host' in FastEmbedOptions")
		}
	}

	// Add http:// prefix if not present
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	// Get key from connector settings if not specified in options
	key := options.Key
	if key == "" {
		if connectorKey, ok := setting["key"].(string); ok {
			key = connectorKey
		}
		// Note: key can be empty for no-auth cases
	}

	return &FastEmbed{
		Connector:  c,
		Concurrent: options.Concurrent,
		Dimension:  options.Dimension,
		Model:      model,
		Host:       host,
		Key:        key,
	}, nil
}

// NewFastEmbedWithDefaults create a new FastEmbed embedding function with default settings
func NewFastEmbedWithDefaults(connectorName string) (*FastEmbed, error) {
	return NewFastEmbed(FastEmbedOptions{
		ConnectorName: connectorName,
		Concurrent:    10,
		Dimension:     384,
		// Don't set Host here, let it use the connector's host setting
	})
}

// postFastEmbed sends a POST request to FastEmbed API with optional password support
func (e *FastEmbed) postFastEmbed(ctx context.Context, endpoint string, payload map[string]interface{}) (interface{}, error) {
	// Clean and build URL
	host := strings.TrimSuffix(e.Host, "/")
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	url := host + endpoint

	// Create HTTP request
	r := http.New(url)
	r.SetHeader("Content-Type", "application/json")
	r.WithContext(ctx)

	// Add authorization header if key is provided
	if e.Key != "" {
		r.SetHeader("Authorization", fmt.Sprintf("Bearer %s", e.Key))
	}

	// Make request
	resp := r.Post(payload)
	if resp.Status != 200 {
		return nil, fmt.Errorf("FastEmbed request failed with status: %d, data: %v", resp.Status, resp.Data)
	}

	return resp.Data, nil
}

// EmbedDocuments embed documents with optional progress callback
func (e *FastEmbed) EmbedDocuments(ctx context.Context, texts []string, callback ...types.EmbeddingProgress) (*types.EmbeddingResults, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	var cb types.EmbeddingProgress
	if len(callback) > 0 && callback[0] != nil {
		cb = callback[0]
	}

	// Report initial progress
	if cb != nil {
		cb(types.EmbeddingStatusStarting, types.EmbeddingPayload{
			Current: 0,
			Total:   len(texts),
			Message: "Starting document embedding with FastEmbed...",
		})
	}

	// Use concurrent requests for better performance
	embeddings := make([][]float64, len(texts))
	sparseEmbeddings := make([]types.SparseEmbedding, len(texts))
	errors := make([]error, len(texts))
	completed := make([]bool, len(texts))
	var wg sync.WaitGroup
	var mu sync.Mutex
	completedCount := 0
	totalTokens := 0
	isDense := false // Will be set based on first result

	// Limit concurrent requests
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
			var docCallback types.EmbeddingProgress
			if cb != nil {
				docCallback = func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
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

			embeddingResult, err := e.EmbedQuery(ctx, inputText, docCallback)
			if err != nil {
				errors[index] = err
				// Report error for this item
				if cb != nil {
					cb(types.EmbeddingStatusError, types.EmbeddingPayload{
						Current:       completedCount + 1,
						Total:         len(texts),
						Message:       fmt.Sprintf("Error embedding document %d", index+1),
						DocumentIndex: &index,
						Error:         err,
					})
				}
			} else {
				// Set embedding type based on first result
				mu.Lock()
				if completedCount == 0 {
					isDense = embeddingResult.IsDense()
				}

				if embeddingResult.IsDense() {
					embeddings[index] = embeddingResult.Embedding
				} else {
					indices, values := embeddingResult.GetSparseEmbedding()
					sparseEmbeddings[index] = types.SparseEmbedding{
						Indices: indices,
						Values:  values,
					}
				}
				totalTokens += embeddingResult.Usage.TotalTokens
				mu.Unlock()
			}

			// Update progress
			mu.Lock()
			completed[index] = true
			completedCount++
			if cb != nil {
				cb(types.EmbeddingStatusProcessing, types.EmbeddingPayload{
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
				cb(types.EmbeddingStatusError, types.EmbeddingPayload{
					Current: len(texts),
					Total:   len(texts),
					Message: fmt.Sprintf("Failed to embed all documents, error at index %d", i),
					Error:   err,
				})
			}
			return nil, fmt.Errorf("error embedding text at index %d: %w", i, err)
		}
	}

	// Calculate total prompt tokens
	promptTokens := 0
	for _, text := range texts {
		promptTokens += len(strings.Fields(text)) // Simple word count approximation
	}

	// Report completion
	if cb != nil {
		cb(types.EmbeddingStatusCompleted, types.EmbeddingPayload{
			Current: len(texts),
			Total:   len(texts),
			Message: "Document embedding completed successfully with FastEmbed",
		})
	}

	// Create result based on embedding type
	result := &types.EmbeddingResults{
		Usage: types.EmbeddingUsage{
			TotalTokens:  totalTokens,
			PromptTokens: promptTokens,
			TotalTexts:   len(texts),
		},
		Model: e.Model,
	}

	if isDense {
		result.Type = types.EmbeddingTypeDense
		result.Embeddings = embeddings
	} else {
		result.Type = types.EmbeddingTypeSparse
		result.SparseEmbeddings = sparseEmbeddings
	}

	return result, nil
}

// EmbedQuery embed query using FastEmbed API with optional progress callback
func (e *FastEmbed) EmbedQuery(ctx context.Context, text string, callback ...types.EmbeddingProgress) (*types.EmbeddingResult, error) {
	if text == "" {
		return nil, nil
	}

	var cb types.EmbeddingProgress
	if len(callback) > 0 && callback[0] != nil {
		cb = callback[0]
	}

	// Report starting
	if cb != nil {
		cb(types.EmbeddingStatusStarting, types.EmbeddingPayload{
			Current: 0,
			Total:   1,
			Message: "Starting text embedding with FastEmbed...",
		})
	}

	// Prepare payload for FastEmbed API
	payload := map[string]interface{}{
		"texts": []string{text},
		"model": e.Model,
	}

	// Report processing
	if cb != nil {
		cb(types.EmbeddingStatusProcessing, types.EmbeddingPayload{
			Current: 0,
			Total:   1,
			Message: "Sending request to FastEmbed...",
		})
	}

	// Use custom postFastEmbed method
	result, err := e.postFastEmbed(ctx, "embed", payload)
	if err != nil {
		if cb != nil {
			cb(types.EmbeddingStatusError, types.EmbeddingPayload{
				Current: 1,
				Total:   1,
				Message: "FastEmbed request failed",
				Error:   err,
			})
		}
		return nil, fmt.Errorf("FastEmbed request failed: %w", err)
	}

	// Parse response according to FastEmbed API format
	respMap, ok := result.(map[string]interface{})
	if !ok {
		if cb != nil {
			cb(types.EmbeddingStatusError, types.EmbeddingPayload{
				Current: 1,
				Total:   1,
				Message: "Unexpected response format",
			})
		}
		return nil, fmt.Errorf("unexpected response format")
	}

	// Parse usage information
	var usage types.EmbeddingUsage
	if usageData, ok := respMap["usage"].(map[string]interface{}); ok {
		if totalTokens, ok := usageData["total_tokens"].(float64); ok {
			usage.TotalTokens = int(totalTokens)
		}
		if totalTexts, ok := usageData["total_texts"].(float64); ok {
			usage.TotalTexts = int(totalTexts)
		}
	}

	// Fallback to simple word count if usage not provided
	if usage.TotalTokens == 0 {
		usage.PromptTokens = len(strings.Fields(text))
		usage.TotalTokens = usage.PromptTokens
	}
	if usage.TotalTexts == 0 {
		// For non-empty text, set to 1; for empty text, this should remain 0
		if text != "" {
			usage.TotalTexts = 1
		}
	}

	// Check if this is a sparse embedding response
	embeddingType := types.EmbeddingTypeDense
	if usageData, ok := respMap["usage"].(map[string]interface{}); ok {
		if embTypeStr, ok := usageData["embedding_type"].(string); ok && embTypeStr == "sparse" {
			embeddingType = types.EmbeddingTypeSparse
		}
	}

	embeddingResult := &types.EmbeddingResult{
		Usage: usage,
		Model: e.Model,
		Type:  embeddingType,
	}

	if embeddingType == types.EmbeddingTypeSparse {
		// Handle sparse embeddings
		if sparseEmbeddings, ok := respMap["embeddings"].([]interface{}); ok && len(sparseEmbeddings) > 0 {
			if sparseEmb, ok := sparseEmbeddings[0].(map[string]interface{}); ok {
				if indices, ok := sparseEmb["indices"].([]interface{}); ok {
					indicesUint32 := make([]uint32, len(indices))
					for i, idx := range indices {
						if idxFloat, ok := idx.(float64); ok {
							indicesUint32[i] = uint32(idxFloat)
						}
					}
					embeddingResult.Indices = indicesUint32
				}

				if values, ok := sparseEmb["values"].([]interface{}); ok {
					valuesFloat32 := make([]float32, len(values))
					for i, val := range values {
						if valFloat, ok := val.(float64); ok {
							valuesFloat32[i] = float32(valFloat)
						}
					}
					embeddingResult.Values = valuesFloat32
				}
			}
		}
	} else {
		// Handle dense embeddings
		embeddings, ok := respMap["embeddings"].([]interface{})
		if !ok {
			if cb != nil {
				cb(types.EmbeddingStatusError, types.EmbeddingPayload{
					Current: 1,
					Total:   1,
					Message: "No embeddings field in response",
				})
			}
			return nil, fmt.Errorf("no embeddings field in response")
		}

		if len(embeddings) == 0 {
			if cb != nil {
				cb(types.EmbeddingStatusError, types.EmbeddingPayload{
					Current: 1,
					Total:   1,
					Message: "No embedding data returned",
				})
			}
			return nil, fmt.Errorf("no embedding data returned")
		}

		// Get first embedding (since we only sent one text)
		embedding, ok := embeddings[0].([]interface{})
		if !ok {
			if cb != nil {
				cb(types.EmbeddingStatusError, types.EmbeddingPayload{
					Current: 1,
					Total:   1,
					Message: "Unexpected embedding format",
				})
			}
			return nil, fmt.Errorf("unexpected embedding format")
		}

		// Convert to []float64
		embeddingFloat := make([]float64, len(embedding))
		for i, val := range embedding {
			if floatVal, ok := val.(float64); ok {
				embeddingFloat[i] = floatVal
			} else {
				if cb != nil {
					cb(types.EmbeddingStatusError, types.EmbeddingPayload{
						Current: 1,
						Total:   1,
						Message: fmt.Sprintf("Invalid embedding value at position %d", i),
					})
				}
				return nil, fmt.Errorf("invalid embedding value at position %d", i)
			}
		}

		// Validate dimension matches expected for dense embeddings
		if len(embeddingFloat) != e.Dimension {
			if cb != nil {
				cb(types.EmbeddingStatusError, types.EmbeddingPayload{
					Current: 1,
					Total:   1,
					Message: fmt.Sprintf("Dimension mismatch: got %d, expected %d", len(embeddingFloat), e.Dimension),
				})
			}
			return nil, fmt.Errorf("received embedding dimension %d does not match expected dimension %d", len(embeddingFloat), e.Dimension)
		}

		embeddingResult.Embedding = embeddingFloat
	}

	// Report completion
	if cb != nil {
		cb(types.EmbeddingStatusCompleted, types.EmbeddingPayload{
			Current: 1,
			Total:   1,
			Message: "Text embedding completed successfully with FastEmbed",
		})
	}

	return embeddingResult, nil
}

// GetModel returns the current model being used
func (e *FastEmbed) GetModel() string {
	return e.Model
}

// GetDimension returns the embedding dimension
func (e *FastEmbed) GetDimension() int {
	return e.Dimension
}

// GetHost returns the FastEmbed service host
func (e *FastEmbed) GetHost() string {
	return e.Host
}

// HasKey returns whether key authentication is configured
func (e *FastEmbed) HasKey() bool {
	return e.Key != ""
}
