package openai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// Options defines the options for OpenAI extraction
type Options struct {
	ConnectorName string                   // Connector name
	Concurrent    int                      // Maximum concurrent requests
	Model         string                   // Model name (optional, can be overridden by connector)
	Temperature   float64                  // Temperature for generation (0.0-2.0)
	MaxTokens     int                      // Maximum tokens for generation
	Prompt        string                   // Custom extraction prompt (optional)
	Toolcall      *bool                    // Whether to use toolcall for extraction (nil = default true)
	Tools         []map[string]interface{} // Custom tools (optional, defaults to extraction tools)
	RetryAttempts int                      // Number of retry attempts for failed requests
	RetryDelay    time.Duration            // Delay between retry attempts
}

// Openai extraction function
type Openai struct {
	Connector     connector.Connector
	Concurrent    int
	Model         string
	Temperature   float64
	MaxTokens     int
	Prompt        string
	Toolcall      bool
	Tools         []map[string]interface{}
	RetryAttempts int
	RetryDelay    time.Duration
}

// NewOpenai create a new Openai extraction function with options
func NewOpenai(options Options) (*Openai, error) {
	c, err := connector.Select(options.ConnectorName)
	if err != nil {
		return nil, err
	}

	if !c.Is(connector.OPENAI) {
		return nil, fmt.Errorf("the connector %s is not a OpenAI connector", options.ConnectorName)
	}

	if options.Concurrent <= 0 {
		options.Concurrent = 5 // Default value for extraction (lower than embedding)
	}

	if options.Temperature < 0.0 || options.Temperature > 2.0 {
		options.Temperature = 0.1 // Low temperature for consistent extraction
	}

	if options.MaxTokens <= 0 {
		options.MaxTokens = 4000 // Default max tokens
	}

	if options.RetryAttempts <= 0 {
		options.RetryAttempts = 3 // Default retry attempts
	}

	if options.RetryDelay <= 0 {
		options.RetryDelay = time.Second // Default retry delay
	}

	// Get model from connector settings if not specified in options
	model := options.Model
	if model == "" {
		setting := c.Setting()
		if connectorModel, ok := setting["model"].(string); ok && connectorModel != "" {
			model = connectorModel
		} else {
			model = "gpt-4o-mini" // Default model for extraction
		}
	}

	// Determine if toolcall should be used
	var toolcall bool
	if options.Toolcall != nil {
		// User explicitly set toolcall value
		toolcall = *options.Toolcall
	} else {
		// Default to true when not specified
		toolcall = true
	}

	// Use custom tools if provided, otherwise use default extraction tools when toolcall is enabled
	tools := options.Tools
	if len(tools) == 0 && toolcall {
		tools = utils.ExtractionToolcall
	}

	return &Openai{
		Connector:     c,
		Concurrent:    options.Concurrent,
		Model:         model,
		Temperature:   options.Temperature,
		MaxTokens:     options.MaxTokens,
		Prompt:        options.Prompt,
		Toolcall:      toolcall,
		Tools:         tools,
		RetryAttempts: options.RetryAttempts,
		RetryDelay:    options.RetryDelay,
	}, nil
}

// NewOpenaiWithDefaults create a new Openai extraction function with default settings
func NewOpenaiWithDefaults(connectorName string) (*Openai, error) {
	toolcall := true
	return NewOpenai(Options{
		ConnectorName: connectorName,
		Concurrent:    5,
		Temperature:   0.1,
		MaxTokens:     4000,
		Toolcall:      &toolcall, // Default to toolcall mode
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	})
}

// ExtractDocuments extract entities and relationships from documents with optional progress callback
func (e *Openai) ExtractDocuments(ctx context.Context, texts []string, callback ...types.ExtractionProgress) ([]*types.ExtractionResult, error) {
	if len(texts) == 0 {
		return []*types.ExtractionResult{}, nil
	}

	var cb types.ExtractionProgress
	if len(callback) > 0 && callback[0] != nil {
		cb = callback[0]
	}

	// Report initial progress
	if cb != nil {
		cb(types.ExtractionStatusStarting, types.ExtractionPayload{
			Current: 0,
			Total:   len(texts),
			Message: "Starting document extraction...",
		})
	}

	// Use concurrent requests for better performance
	results := make([]*types.ExtractionResult, len(texts))
	errors := make([]error, len(texts))
	var wg sync.WaitGroup
	var mu sync.Mutex
	completedCount := 0
	totalTokens := 0

	// Use configured concurrent limit without artificial restriction
	maxConcurrent := e.Concurrent
	if len(texts) < maxConcurrent {
		maxConcurrent = len(texts)
	}
	// Remove the artificial limit of 10 - let users configure what they need

	// Create buffered semaphore channel to prevent blocking
	semaphore := make(chan struct{}, maxConcurrent)

	// Create a done channel to handle context cancellation
	done := make(chan struct{})

	// Start a goroutine to handle context cancellation
	go func() {
		<-ctx.Done()
		close(done)
	}()

	for i, text := range texts {
		// Check if context is cancelled before starting new goroutine
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		go func(index int, inputText string) {
			defer wg.Done()

			// Try to acquire semaphore with context cancellation support
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }() // Release semaphore
			case <-done:
				return // Context cancelled, exit early
			}

			// Check context again before processing
			select {
			case <-done:
				return // Context cancelled, exit early
			default:
			}

			// Create a callback for individual document processing
			var docCallback types.ExtractionProgress
			if cb != nil {
				docCallback = func(status types.ExtractionStatus, payload types.ExtractionPayload) {
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

			extractionResult, err := e.ExtractQuery(ctx, inputText, docCallback)

			// Store results regardless of success/failure
			mu.Lock()
			if err != nil {
				errors[index] = err
				// Report error for this item but don't fail entire batch
				if cb != nil {
					cb(types.ExtractionStatusError, types.ExtractionPayload{
						Current:       completedCount + 1,
						Total:         len(texts),
						Message:       fmt.Sprintf("Error extracting document %d: %v", index+1, err),
						DocumentIndex: &index,
						Error:         err,
					})
				}
			} else {
				results[index] = extractionResult
				// Add to total tokens count
				totalTokens += extractionResult.Usage.TotalTokens
			}

			// Update progress
			completedCount++
			if cb != nil {
				cb(types.ExtractionStatusProcessing, types.ExtractionPayload{
					Current:       completedCount,
					Total:         len(texts),
					Message:       fmt.Sprintf("Completed %d/%d documents", completedCount, len(texts)),
					DocumentIndex: &index,
				})
			}
			mu.Unlock()
		}(i, text)
	}

	// Wait for all goroutines to complete or context to be cancelled
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		// All goroutines completed normally
	case <-ctx.Done():
		// Context cancelled, return error
		return nil, ctx.Err()
	}

	// Count successful extractions vs errors
	successCount := 0
	errorCount := 0
	for i := range texts {
		if errors[i] != nil {
			errorCount++
		} else if results[i] != nil {
			successCount++
		}
	}

	// Only fail if ALL extractions failed
	if successCount == 0 && errorCount > 0 {
		if cb != nil {
			cb(types.ExtractionStatusError, types.ExtractionPayload{
				Current: len(texts),
				Total:   len(texts),
				Message: fmt.Sprintf("All %d document extractions failed", len(texts)),
				Error:   errors[0], // Return first error as example
			})
		}
		return nil, fmt.Errorf("all %d document extractions failed, first error: %w", len(texts), errors[0])
	}

	// Merge successful results
	allNodes := []types.Node{}
	allRelationships := []types.Relationship{}
	promptTokens := 0

	for _, result := range results {
		if result != nil {
			allNodes = append(allNodes, result.Nodes...)
			allRelationships = append(allRelationships, result.Relationships...)
			promptTokens += result.Usage.PromptTokens
		}
	}

	// Report completion with success/error statistics
	if cb != nil {
		message := fmt.Sprintf("Document extraction completed: %d successful, %d failed", successCount, errorCount)
		cb(types.ExtractionStatusCompleted, types.ExtractionPayload{
			Current: len(texts),
			Total:   len(texts),
			Message: message,
		})
	}

	// Return results array with same length as input texts
	// Failed extractions will have nil at corresponding index
	return results, nil
}

// ExtractQuery extract entities and relationships from a single text with optional progress callback
func (e *Openai) ExtractQuery(ctx context.Context, text string, callback ...types.ExtractionProgress) (*types.ExtractionResult, error) {
	if text == "" {
		return &types.ExtractionResult{
			Usage:         types.ExtractionUsage{},
			Model:         e.Model,
			Nodes:         []types.Node{},
			Relationships: []types.Relationship{},
		}, nil
	}

	var cb types.ExtractionProgress
	if len(callback) > 0 && callback[0] != nil {
		cb = callback[0]
	}

	// Report starting
	if cb != nil {
		cb(types.ExtractionStatusStarting, types.ExtractionPayload{
			Current: 0,
			Total:   1,
			Message: "Starting text extraction...",
		})
	}

	// Prepare extraction prompt - use JSON format instructions for non-toolcall mode
	var systemPrompt string
	if e.Toolcall {
		systemPrompt = utils.ExtractionPrompt(e.Prompt)
	} else {
		systemPrompt = utils.ExtractionPromptWithJSONFormat(e.Prompt)
	}

	// Prepare messages
	messages := []map[string]interface{}{
		{
			"role":    "system",
			"content": systemPrompt,
		},
		{
			"role":    "user",
			"content": fmt.Sprintf("Please extract entities and relationships from the following text:\n\n%s", text),
		},
	}

	payload := map[string]interface{}{
		"model":       e.Model,
		"messages":    messages,
		"temperature": e.Temperature,
		"max_tokens":  e.MaxTokens,
	}

	// Add toolcall parameters if toolcall is enabled
	if e.Toolcall {
		payload["tools"] = e.Tools
		payload["tool_choice"] = "required"
	}

	// Report processing
	if cb != nil {
		cb(types.ExtractionStatusProcessing, types.ExtractionPayload{
			Current: 0,
			Total:   1,
			Message: "Sending request to OpenAI...",
		})
	}

	// Execute with retry logic using streaming
	var extractionResult *types.ExtractionResult
	var err error
	var lastEntityCount, lastRelationshipCount int // Track previous counts

	for attempt := 0; attempt <= e.RetryAttempts; attempt++ {
		// Create stream parser for extraction tool calls
		parser := utils.NewExtractionParser()
		parser.SetToolcall(e.Toolcall) // Set toolcall mode in parser
		lastEntityCount = 0            // Reset counters for each attempt
		lastRelationshipCount = 0

		// Stream callback function with progress reporting
		streamCallback := func(data []byte) error {
			// Skip empty data chunks
			if len(data) == 0 {
				return nil
			}

			// Parse streaming chunk to get real-time extraction progress
			nodes, relationships, parseErr := parser.ParseExtractionEntities(data)
			if parseErr != nil {
				// Don't fail the entire stream for parsing errors, just log them
				return nil
			}

			// Report streaming progress only when counts change
			if cb != nil && (len(nodes) != lastEntityCount || len(relationships) != lastRelationshipCount) {
				lastEntityCount = len(nodes)
				lastRelationshipCount = len(relationships)
				message := fmt.Sprintf("Extracted %d entities, %d relationships so far...", len(nodes), len(relationships))
				cb(types.ExtractionStatusProcessing, types.ExtractionPayload{
					Current: 0,
					Total:   1,
					Message: message,
				})
			}

			return nil
		}

		// Make streaming request using utils.StreamLLM
		err = utils.StreamLLM(ctx, e.Connector, "chat/completions", payload, streamCallback)
		if err == nil {
			// Parse the accumulated data based on toolcall mode
			if e.Toolcall {
				if parser.Arguments != "" {
					nodes, relationships, parseErr := parser.ParseExtractionToolcall(parser.Arguments)
					if parseErr == nil {
						// Create extraction result from parsed data
						usage := types.ExtractionUsage{
							PromptTokens: len(strings.Fields(text)),
							TotalTokens:  len(strings.Fields(text)), // Approximate, will be updated if available
							TotalTexts:   1,
						}

						extractionResult = &types.ExtractionResult{
							Usage:         usage,
							Model:         e.Model,
							Nodes:         nodes,
							Relationships: relationships,
						}
						break // Success, exit retry loop
					} else {
						err = parseErr
						// Log the parse error for debugging
						if cb != nil {
							cb(types.ExtractionStatusProcessing, types.ExtractionPayload{
								Current: 0,
								Total:   1,
								Message: fmt.Sprintf("Parse error (attempt %d): %v", attempt+1, parseErr),
							})
						}
					}
				} else {
					err = fmt.Errorf("no tool call arguments received")
				}
			} else {
				// Non-toolcall mode: parse regular content
				if parser.Content != "" {
					nodes, relationships, parseErr := parser.ParseExtractionRegular(parser.Content)
					if parseErr == nil {
						// Create extraction result from parsed data
						usage := types.ExtractionUsage{
							PromptTokens: len(strings.Fields(text)),
							TotalTokens:  len(strings.Fields(text)), // Approximate, will be updated if available
							TotalTexts:   1,
						}

						extractionResult = &types.ExtractionResult{
							Usage:         usage,
							Model:         e.Model,
							Nodes:         nodes,
							Relationships: relationships,
						}
						break // Success, exit retry loop
					} else {
						err = parseErr
						// Log the parse error for debugging
						if cb != nil {
							cb(types.ExtractionStatusProcessing, types.ExtractionPayload{
								Current: 0,
								Total:   1,
								Message: fmt.Sprintf("Parse error (attempt %d): %v", attempt+1, parseErr),
							})
						}
					}
				} else {
					err = fmt.Errorf("no content received from LLM")
				}
			}
		}

		if attempt < e.RetryAttempts {
			if cb != nil {
				cb(types.ExtractionStatusProcessing, types.ExtractionPayload{
					Current: 0,
					Total:   1,
					Message: fmt.Sprintf("Request failed, retrying... (attempt %d/%d)", attempt+1, e.RetryAttempts),
				})
			}
			time.Sleep(e.RetryDelay)
		}
	}

	if err != nil {
		if cb != nil {
			cb(types.ExtractionStatusError, types.ExtractionPayload{
				Current: 1,
				Total:   1,
				Message: "Request failed after all retries",
				Error:   err,
			})
		}
		return nil, fmt.Errorf("request failed after %d attempts: %w", e.RetryAttempts+1, err)
	}

	// Report completion
	if cb != nil {
		cb(types.ExtractionStatusCompleted, types.ExtractionPayload{
			Current: 1,
			Total:   1,
			Message: "Text extraction completed successfully",
		})
	}

	return extractionResult, nil
}

// GetModel returns the current model being used
func (e *Openai) GetModel() string {
	return e.Model
}

// GetConcurrent returns the concurrent limit
func (e *Openai) GetConcurrent() int {
	return e.Concurrent
}

// GetTemperature returns the temperature setting
func (e *Openai) GetTemperature() float64 {
	return e.Temperature
}

// GetMaxTokens returns the max tokens setting
func (e *Openai) GetMaxTokens() int {
	return e.MaxTokens
}

// GetToolcall returns the toolcall setting
func (e *Openai) GetToolcall() bool {
	return e.Toolcall
}
