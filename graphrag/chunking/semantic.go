package chunking

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kaptinlin/jsonrepair"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/kun/log"
)

// SemanticPosition represents the semantic segment position returned by LLM
type SemanticPosition struct {
	StartPos int `json:"start_pos"`
	EndPos   int `json:"end_pos"`
}

// ProgressCallback represents the progress callback function
type ProgressCallback func(chunkID string, progress string, step string, data interface{}) error

// SemanticChunker implements semantic-based chunking using LLM
type SemanticChunker struct {
	structuredChunker *StructuredChunker
	progressCallback  ProgressCallback
	mutex             sync.RWMutex
}

// NewSemanticChunker creates a new semantic chunker
func NewSemanticChunker(progressCallback ProgressCallback) *SemanticChunker {
	return &SemanticChunker{
		structuredChunker: NewStructuredChunker(),
		progressCallback:  progressCallback,
	}
}

// Chunk implements semantic chunking on text
func (sc *SemanticChunker) Chunk(ctx context.Context, text string, options *types.ChunkingOptions, callback func(chunk *types.Chunk) error) error {
	// Convert text to ReadSeeker
	reader := strings.NewReader(text)
	return sc.ChunkStream(ctx, reader, options, callback)
}

// ChunkFile implements semantic chunking on file
func (sc *SemanticChunker) ChunkFile(ctx context.Context, file string, options *types.ChunkingOptions, callback func(chunk *types.Chunk) error) error {
	// Validate semantic options
	if err := sc.validateSemanticOptions(options); err != nil {
		return fmt.Errorf("invalid semantic options: %w", err)
	}

	// Use structured chunker to read file and then process semantically
	reader, err := utils.OpenFileAsReader(file)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer reader.Close()

	return sc.ChunkStream(ctx, reader, options, callback)
}

// ChunkStream implements semantic chunking on stream
func (sc *SemanticChunker) ChunkStream(ctx context.Context, stream io.ReadSeeker, options *types.ChunkingOptions, callback func(chunk *types.Chunk) error) error {
	// Step 1: Validate and prepare options
	if err := sc.validateAndPrepareOptions(options); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Step 2: Get structured chunks as context for semantic analysis
	structuredChunks, err := sc.getStructuredChunks(ctx, stream, options)
	if err != nil {
		return fmt.Errorf("failed to get structured chunks: %w", err)
	}

	if len(structuredChunks) == 0 {
		return fmt.Errorf("no structured chunks generated")
	}

	// Step 3: Process semantic chunking with concurrency
	semanticChunks, err := sc.processSemanticChunking(ctx, structuredChunks, options)
	if err != nil {
		return fmt.Errorf("failed to process semantic chunking: %w", err)
	}

	// Step 4: Build hierarchy and output chunks
	return sc.buildHierarchyAndOutput(ctx, semanticChunks, options, callback)
}

// validateSemanticOptions validates semantic options
func (sc *SemanticChunker) validateSemanticOptions(options *types.ChunkingOptions) error {
	if options.SemanticOptions == nil {
		return fmt.Errorf("semantic options cannot be nil")
	}

	semanticOpts := options.SemanticOptions
	if semanticOpts.Connector == "" {
		return fmt.Errorf("semantic connector cannot be empty")
	}

	// Validate that the connector exists
	_, err := connector.Select(semanticOpts.Connector)
	if err != nil {
		return fmt.Errorf("invalid connector '%s': %w", semanticOpts.Connector, err)
	}

	// Set defaults
	if semanticOpts.MaxRetry <= 0 {
		semanticOpts.MaxRetry = 9
	}
	if semanticOpts.MaxConcurrent <= 0 {
		semanticOpts.MaxConcurrent = 4
	}

	return nil
}

// validateAndPrepareOptions validates and prepares all options
func (sc *SemanticChunker) validateAndPrepareOptions(options *types.ChunkingOptions) error {
	// Validate semantic options
	if err := sc.validateSemanticOptions(options); err != nil {
		return err
	}

	// Set default overlap if invalid
	if options.Overlap <= 0 || options.Overlap > options.Size {
		options.Overlap = 50
	}

	// Calculate context size for structured chunking
	semanticOpts := options.SemanticOptions
	if semanticOpts.ContextSize <= 0 {
		semanticOpts.ContextSize = options.Size * options.MaxDepth * 3
	}

	// Set default prompt if empty (will be handled by utils.GetSemanticPrompt)
	// No need to set here as utils.GetSemanticPrompt handles empty prompts

	return nil
}

// getStructuredChunks gets structured chunks as context for semantic analysis
func (sc *SemanticChunker) getStructuredChunks(ctx context.Context, stream io.ReadSeeker, options *types.ChunkingOptions) ([]*types.Chunk, error) {
	// Create structured options with large chunk size and depth 1
	structuredOpts := &types.ChunkingOptions{
		Type:          options.Type,
		Size:          options.SemanticOptions.ContextSize,
		Overlap:       options.Overlap,
		MaxDepth:      1, // Only level 1 for semantic analysis
		MaxConcurrent: options.MaxConcurrent,
	}

	var chunks []*types.Chunk
	var mu sync.Mutex

	err := sc.structuredChunker.ChunkStream(ctx, stream, structuredOpts, func(chunk *types.Chunk) error {
		mu.Lock()
		defer mu.Unlock()
		chunks = append(chunks, chunk)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return chunks, nil
}

// processSemanticChunking processes semantic chunking with LLM concurrency
func (sc *SemanticChunker) processSemanticChunking(ctx context.Context, structuredChunks []*types.Chunk, options *types.ChunkingOptions) ([]*types.Chunk, error) {
	semanticOpts := options.SemanticOptions

	// Channel for controlling concurrency
	semaphore := make(chan struct{}, semanticOpts.MaxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allSemanticChunks []*types.Chunk
	var firstError error

	for i, structuredChunk := range structuredChunks {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(chunk *types.Chunk, index int) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()

			// Report progress
			sc.reportProgress(chunk.ID, "processing", "semantic_analysis", map[string]interface{}{
				"chunk_index":  index,
				"total_chunks": len(structuredChunks),
			})

			// Process semantic segmentation for this chunk
			semanticChunks, err := sc.processChunkSemanticSegmentation(ctx, chunk, options)
			if err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("failed to process chunk %s: %w", chunk.ID, err)
				}
				mu.Unlock()

				// Report error progress
				sc.reportProgress(chunk.ID, "failed", "semantic_analysis", map[string]interface{}{
					"error": err.Error(),
				})
				return
			}

			// Add to results
			mu.Lock()
			allSemanticChunks = append(allSemanticChunks, semanticChunks...)
			mu.Unlock()

			// Report completion progress
			sc.reportProgress(chunk.ID, "completed", "semantic_analysis", map[string]interface{}{
				"chunks_generated": len(semanticChunks),
			})
		}(structuredChunk, i)
	}

	wg.Wait()

	if firstError != nil {
		return nil, firstError
	}

	return allSemanticChunks, nil
}

// processChunkSemanticSegmentation processes semantic segmentation for a single chunk
func (sc *SemanticChunker) processChunkSemanticSegmentation(ctx context.Context, chunk *types.Chunk, options *types.ChunkingOptions) ([]*types.Chunk, error) {
	semanticOpts := options.SemanticOptions

	// Try with retries
	var positions []SemanticPosition
	var err error

	for retry := 0; retry <= semanticOpts.MaxRetry; retry++ {
		// Use options.Size as the target size for semantic segmentation
		positions, err = sc.callLLMForSegmentation(ctx, chunk.Text, semanticOpts, options.Size)
		if err == nil && len(positions) > 0 {
			break
		}

		if retry < semanticOpts.MaxRetry {
			log.Warn("LLM segmentation failed (retry %d/%d): %v", retry+1, semanticOpts.MaxRetry, err)
			time.Sleep(time.Duration(retry+1) * time.Second) // Exponential backoff
		}
	}

	if err != nil {
		return nil, fmt.Errorf("LLM segmentation failed after %d retries: %w", semanticOpts.MaxRetry, err)
	}

	if len(positions) == 0 {
		// If no positions returned, treat entire chunk as single semantic unit
		positions = []SemanticPosition{{StartPos: 0, EndPos: len(chunk.Text)}}
	}

	// Convert positions to chunks
	semanticChunks := sc.createSemanticChunks(chunk, positions, options)
	return semanticChunks, nil
}

// callLLMForSegmentation calls LLM to get semantic segmentation positions using streaming
func (sc *SemanticChunker) callLLMForSegmentation(ctx context.Context, text string, semanticOpts *types.SemanticOptions, maxSize int) ([]SemanticPosition, error) {
	// Get the connector
	conn, err := connector.Select(semanticOpts.Connector)
	if err != nil {
		return nil, fmt.Errorf("failed to select connector '%s': %w", semanticOpts.Connector, err)
	}

	// Build prompt using utils function with size information
	basePrompt := utils.GetSemanticPrompt(semanticOpts.Prompt)
	prompt := fmt.Sprintf("%s\n\nIMPORTANT: Do NOT create segments with regular intervals or fixed character counts. The suggested size of %d characters is only a rough guideline - ALWAYS prioritize natural semantic boundaries over size uniformity. Segments should vary significantly in size based on content structure.\n\nText to segment:\n%s", basePrompt, maxSize, text)

	// Prepare request payload
	requestData := map[string]interface{}{
		"max_tokens":  4000,
		"temperature": 0.1,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	// Set model from connector settings
	setting := conn.Setting()
	if model, ok := setting["model"].(string); ok && model != "" {
		requestData["model"] = model
	} else {
		requestData["model"] = "gpt-4o-mini" // Default model
	}

	// Add custom options if provided
	if semanticOpts.Options != "" {
		extraOptions, err := utils.ParseJSONOptions(semanticOpts.Options)
		if err != nil {
			log.Warn("Failed to parse semantic options: %v", err)
		} else {
			for k, v := range extraOptions {
				requestData[k] = v
			}
		}
	}

	// Use toolcall if enabled
	if semanticOpts.Toolcall {
		requestData["tools"] = []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "segment_text",
					"description": "Segment text into semantic chunks based on natural boundaries, NOT fixed character intervals. Prioritize topic changes, paragraph breaks, and concept shifts over uniform sizing.",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"segments": map[string]interface{}{
								"type":        "array",
								"description": "Array of semantic segments with VARIED sizes based on natural content boundaries",
								"items": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"start_pos": map[string]interface{}{
											"type":        "integer",
											"description": "Start character position of the semantic segment",
										},
										"end_pos": map[string]interface{}{
											"type":        "integer",
											"description": "End character position of the semantic segment",
										},
									},
									"required": []string{"start_pos", "end_pos"},
								},
							},
						},
						"required": []string{"segments"},
					},
				},
			},
		}
		requestData["tool_choice"] = "auto"
	}

	// Create stream parser
	parser := utils.NewStreamParser(semanticOpts.Toolcall)
	var finalContent string
	var finalArguments string

	// Stream callback function with progress reporting
	streamCallback := func(data []byte) error {
		// Skip empty data chunks
		if len(data) == 0 {
			return nil
		}

		// Parse streaming chunk
		chunkData, err := parser.ParseStreamChunk(data)
		if err != nil {
			log.Warn("Failed to parse stream chunk: %v", err)
			return nil // Don't fail the entire stream for parsing errors
		}

		fmt.Println("--------------------------------")
		fmt.Println("Streaming data", string(data))
		fmt.Println("Streaming chunk arguments", chunkData.Arguments)
		fmt.Println("Streaming chunk content", chunkData.Content)
		fmt.Println("Streaming chunk finished", chunkData.Finished)
		fmt.Println("Streaming chunk error", chunkData.Error)
		fmt.Println("Streaming chunk positions count", len(chunkData.Positions))
		if len(chunkData.Positions) > 0 {
			maxShow := 3
			if len(chunkData.Positions) < maxShow {
				maxShow = len(chunkData.Positions)
			}
			fmt.Printf("First few positions: %+v\n", chunkData.Positions[:maxShow])
		}
		fmt.Println("--------------------------------")

		// Update final content/arguments
		if semanticOpts.Toolcall {
			finalArguments = chunkData.Arguments
		} else {
			finalContent = chunkData.Content
		}

		// Report progress with streaming data including positions
		sc.reportProgress("", "streaming", "llm_response", map[string]interface{}{
			"is_toolcall":      chunkData.IsToolcall,
			"content_length":   len(chunkData.Content),
			"arguments_length": len(chunkData.Arguments),
			"positions_count":  len(chunkData.Positions),
			"finished":         chunkData.Finished,
			"has_error":        chunkData.Error != "",
		})

		// If we have positions and the stream is finished, we can potentially return early
		if chunkData.Finished && len(chunkData.Positions) > 0 {
			log.Debug("Stream finished with %d positions parsed", len(chunkData.Positions))
		}

		return nil
	}

	// Make streaming request using utils.StreamLLM
	err = utils.StreamLLM(ctx, conn, "chat/completions", requestData, streamCallback)
	if err != nil {
		return nil, fmt.Errorf("LLM streaming request failed: %w", err)
	}

	// Build response data for parsing
	var responseData map[string]interface{}
	if semanticOpts.Toolcall {
		// For toolcall, check if finalArguments is complete and valid
		if strings.TrimSpace(finalArguments) == "" {
			log.Warn("Empty finalArguments received from streaming, using fallback")
			// Use fallback: create a single segment for the entire text
			return []SemanticPosition{{StartPos: 0, EndPos: len(text)}}, nil
		}

		// Try to repair incomplete JSON arguments
		repairedArgs, err := jsonrepair.JSONRepair(finalArguments)
		if err != nil {
			log.Warn("Failed to repair finalArguments JSON: %v, using fallback", err)
			// Use fallback: create a single segment for the entire text, but apply size constraints
			fallbackPos := []SemanticPosition{{StartPos: 0, EndPos: len(text)}}
			return sc.validateAndFixPositions(fallbackPos, len(text), maxSize), nil
		}

		// Validate that repaired JSON can be parsed
		var testArgs map[string]interface{}
		if err := json.Unmarshal([]byte(repairedArgs), &testArgs); err != nil {
			log.Warn("Repaired finalArguments is still invalid: %v, using fallback", err)
			// Use fallback: create a single segment for the entire text
			return []SemanticPosition{{StartPos: 0, EndPos: len(text)}}, nil
		}

		// For toolcall, create a mock response structure with the repaired arguments
		responseData = map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"tool_calls": []interface{}{
							map[string]interface{}{
								"function": map[string]interface{}{
									"arguments": repairedArgs,
								},
							},
						},
					},
				},
			},
		}
	} else {
		// For regular response, check if finalContent is available
		if strings.TrimSpace(finalContent) == "" {
			log.Warn("Empty finalContent received from streaming, using fallback")
			// Use fallback: create a single segment for the entire text
			return []SemanticPosition{{StartPos: 0, EndPos: len(text)}}, nil
		}

		// For regular response, create a mock response structure with the accumulated content
		responseData = map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"content": finalContent,
					},
				},
			},
		}
	}

	// Convert response data to JSON bytes for existing parsing logic
	responseBytes, err := json.Marshal(responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	// Parse response using existing logic
	return sc.parseLLMResponse(responseBytes, semanticOpts.Toolcall, len(text), maxSize)
}

// parseLLMResponse parses LLM response to extract semantic positions
func (sc *SemanticChunker) parseLLMResponse(responseBody []byte, isToolcall bool, textLen, maxSize int) ([]SemanticPosition, error) {
	// First try to repair JSON if needed
	repairedJSON, err := jsonrepair.JSONRepair(string(responseBody))
	if err != nil {
		return nil, fmt.Errorf("failed to repair JSON: %w", err)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(repairedJSON), &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	var positions []SemanticPosition

	if isToolcall {
		// Parse toolcall response
		positions, err = sc.parseToolcallResponse(responseData)
	} else {
		// Parse regular response
		positions, err = sc.parseRegularResponse(responseData)
	}

	if err != nil {
		return nil, err
	}

	// Only do basic boundary checks to prevent crashes, no semantic logic
	var safePositions []SemanticPosition
	for _, pos := range positions {
		// Basic boundary safety checks only
		if pos.StartPos < 0 {
			pos.StartPos = 0
		}
		if pos.EndPos > textLen {
			pos.EndPos = textLen
		}
		if pos.StartPos >= pos.EndPos {
			continue // Skip invalid positions
		}
		safePositions = append(safePositions, pos)
	}

	return safePositions, nil
}

// parseToolcallResponse parses toolcall response format
func (sc *SemanticChunker) parseToolcallResponse(responseData map[string]interface{}) ([]SemanticPosition, error) {
	choices, ok := responseData["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in toolcall response")
	}

	choice := choices[0].(map[string]interface{})
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no message in toolcall choice")
	}

	toolCalls, ok := message["tool_calls"].([]interface{})
	if !ok || len(toolCalls) == 0 {
		return nil, fmt.Errorf("no tool_calls in message")
	}

	toolCall := toolCalls[0].(map[string]interface{})
	function, ok := toolCall["function"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no function in tool_call")
	}

	argumentsStr, ok := function["arguments"].(string)
	if !ok {
		return nil, fmt.Errorf("no arguments in function")
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsStr), &args); err != nil {
		return nil, fmt.Errorf("failed to parse function arguments: %w", err)
	}

	segments, ok := args["segments"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no segments in function arguments")
	}

	var positions []SemanticPosition
	for _, seg := range segments {
		segMap := seg.(map[string]interface{})

		// Safe conversion for start_pos
		startPos, err := sc.convertToInt(segMap["start_pos"])
		if err != nil {
			return nil, fmt.Errorf("invalid start_pos: %w", err)
		}

		// Safe conversion for end_pos
		endPos, err := sc.convertToInt(segMap["end_pos"])
		if err != nil {
			return nil, fmt.Errorf("invalid end_pos: %w", err)
		}

		positions = append(positions, SemanticPosition{
			StartPos: startPos,
			EndPos:   endPos,
		})
	}

	return positions, nil
}

// convertToInt safely converts interface{} to int, handling different number types
func (sc *SemanticChunker) convertToInt(value interface{}) (int, error) {
	if value == nil {
		return 0, fmt.Errorf("value is nil")
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		// Try to parse string as number
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed, nil
		}
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return int(parsed), nil
		}
		return 0, fmt.Errorf("cannot convert string '%s' to int", v)
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// parseRegularResponse parses regular JSON response format
func (sc *SemanticChunker) parseRegularResponse(responseData map[string]interface{}) ([]SemanticPosition, error) {
	choices, ok := responseData["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := choices[0].(map[string]interface{})
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no message in choice")
	}

	content, ok := message["content"].(string)
	if !ok {
		return nil, fmt.Errorf("no content in message")
	}

	// Check if content is empty or whitespace only
	if strings.TrimSpace(content) == "" {
		return []SemanticPosition{}, nil // Return empty positions for empty content
	}

	// Try to extract JSON from content (might be wrapped in markdown or have other text)
	jsonStr := sc.extractJSONFromText(content)

	// Check if extracted JSON is empty
	if strings.TrimSpace(jsonStr) == "" {
		return []SemanticPosition{}, nil // Return empty positions for empty JSON
	}

	// Repair and parse JSON
	repairedJSON, err := jsonrepair.JSONRepair(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to repair extracted JSON: %w", err)
	}

	var positions []SemanticPosition
	if err := json.Unmarshal([]byte(repairedJSON), &positions); err != nil {
		return nil, fmt.Errorf("failed to parse positions JSON: %w", err)
	}

	return positions, nil
}

// extractJSONFromText extracts JSON array from text content
func (sc *SemanticChunker) extractJSONFromText(text string) string {
	// Remove markdown code blocks
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")

	// Find JSON array boundaries
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")

	if start == -1 || end == -1 || start >= end {
		return text // Return as-is if no clear JSON boundaries
	}

	return text[start : end+1]
}

// validateAndFixPositions validates and fixes semantic positions
func (sc *SemanticChunker) validateAndFixPositions(positions []SemanticPosition, textLen, maxSize int) []SemanticPosition {
	if len(positions) == 0 {
		// If no positions provided, create a single segment for the entire text
		// This preserves the original content as one semantic unit
		log.Warn("No semantic positions provided, creating single segment for entire text (length: %d)", textLen)
		return []SemanticPosition{{StartPos: 0, EndPos: textLen}}
	}

	var validPositions []SemanticPosition
	lastEnd := 0

	for _, pos := range positions {
		// Only fix obvious boundary errors, don't change semantic decisions
		if pos.StartPos < 0 {
			pos.StartPos = 0
		}
		if pos.EndPos > textLen {
			pos.EndPos = textLen
		}
		if pos.StartPos >= pos.EndPos {
			continue // Skip invalid positions
		}

		// Fill gaps between segments (preserve LLM's semantic boundaries)
		if pos.StartPos > lastEnd {
			validPositions = append(validPositions, SemanticPosition{
				StartPos: lastEnd,
				EndPos:   pos.StartPos,
			})
		}

		// TRUST LLM COMPLETELY - use the segment as-is regardless of size
		// The LLM has made semantic decisions that we should respect
		validPositions = append(validPositions, pos)
		lastEnd = pos.EndPos
	}

	// Fill remaining gap if any
	if lastEnd < textLen {
		validPositions = append(validPositions, SemanticPosition{
			StartPos: lastEnd,
			EndPos:   textLen,
		})
	}

	return validPositions
}

// splitLargePosition splits a large position into smaller ones (legacy method)
func (sc *SemanticChunker) splitLargePosition(pos SemanticPosition, maxSize int) []SemanticPosition {
	return sc.splitLargePositionSemantically(pos, maxSize)
}

// splitLargePositionSemantically splits a large position while trying to preserve semantic boundaries
func (sc *SemanticChunker) splitLargePositionSemantically(pos SemanticPosition, maxSize int) []SemanticPosition {
	var positions []SemanticPosition
	currentStart := pos.StartPos

	for currentStart < pos.EndPos {
		currentEnd := currentStart + maxSize
		if currentEnd > pos.EndPos {
			currentEnd = pos.EndPos
		}

		positions = append(positions, SemanticPosition{
			StartPos: currentStart,
			EndPos:   currentEnd,
		})

		currentStart = currentEnd
	}

	return positions
}

// createSemanticChunks creates semantic chunks from positions
func (sc *SemanticChunker) createSemanticChunks(originalChunk *types.Chunk, positions []SemanticPosition, options *types.ChunkingOptions) []*types.Chunk {
	var semanticChunks []*types.Chunk

	for i, pos := range positions {
		// Extract text for this semantic segment
		chunkText := originalChunk.Text[pos.StartPos:pos.EndPos]
		if strings.TrimSpace(chunkText) == "" {
			continue // Skip empty chunks
		}

		// Calculate text position
		var textPos *types.TextPosition
		if originalChunk.TextPos != nil {
			textPos = &types.TextPosition{
				StartIndex: originalChunk.TextPos.StartIndex + pos.StartPos,
				EndIndex:   originalChunk.TextPos.StartIndex + pos.EndPos,
				StartLine:  originalChunk.TextPos.StartLine + strings.Count(originalChunk.Text[:pos.StartPos], "\n"),
				EndLine:    originalChunk.TextPos.StartLine + strings.Count(originalChunk.Text[:pos.EndPos], "\n"),
			}
		}

		// Create semantic chunk
		chunk := &types.Chunk{
			ID:       uuid.NewString(),
			Text:     chunkText,
			Type:     originalChunk.Type,
			ParentID: "",    // Will be set during hierarchy building
			Depth:    1,     // Semantic chunks start at depth 1
			Leaf:     false, // Will be determined during hierarchy building
			Root:     true,  // Semantic chunks are initially root
			Index:    i,
			Status:   types.ChunkingStatusCompleted,
			TextPos:  textPos,
			Parents:  []types.Chunk{}, // Will be populated during hierarchy building
		}

		semanticChunks = append(semanticChunks, chunk)
	}

	return semanticChunks
}

// buildHierarchyAndOutput builds hierarchy and outputs chunks
func (sc *SemanticChunker) buildHierarchyAndOutput(ctx context.Context, semanticChunks []*types.Chunk, options *types.ChunkingOptions, callback func(chunk *types.Chunk) error) error {
	// Step 1: Output semantic chunks (level 1)
	for _, chunk := range semanticChunks {
		if err := callback(chunk); err != nil {
			return fmt.Errorf("callback failed for semantic chunk %s: %w", chunk.ID, err)
		}

		sc.reportProgress(chunk.ID, "output", "semantic_chunk", nil)
	}

	// Step 2: Build hierarchy if MaxDepth > 1
	if options.MaxDepth > 1 {
		return sc.buildHierarchy(ctx, semanticChunks, options, callback)
	}

	return nil
}

// buildHierarchy builds hierarchical chunks
func (sc *SemanticChunker) buildHierarchy(ctx context.Context, baseChunks []*types.Chunk, options *types.ChunkingOptions, callback func(chunk *types.Chunk) error) error {
	currentLevelChunks := baseChunks
	currentDepth := 2

	for currentDepth <= options.MaxDepth {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Group chunks for next level
		nextLevelChunks, err := sc.createNextLevelChunks(currentLevelChunks, currentDepth, options)
		if err != nil {
			return fmt.Errorf("failed to create level %d chunks: %w", currentDepth, err)
		}

		if len(nextLevelChunks) == 0 {
			break // No more chunks to create
		}

		// Output next level chunks
		for _, chunk := range nextLevelChunks {
			if err := callback(chunk); err != nil {
				return fmt.Errorf("callback failed for level %d chunk %s: %w", currentDepth, chunk.ID, err)
			}

			sc.reportProgress(chunk.ID, "output", fmt.Sprintf("level_%d_chunk", currentDepth), nil)
		}

		// Update children's parent information
		sc.updateChildrenParents(currentLevelChunks, nextLevelChunks)

		currentLevelChunks = nextLevelChunks
		currentDepth++
	}

	return nil
}

// createNextLevelChunks creates chunks for the next hierarchy level
func (sc *SemanticChunker) createNextLevelChunks(childrenChunks []*types.Chunk, depth int, options *types.ChunkingOptions) ([]*types.Chunk, error) {
	if len(childrenChunks) <= 1 {
		return nil, nil // No need to create parent if only one child
	}

	// Calculate target size for this level
	targetSize := sc.calculateLevelSize(options.Size, depth, options.MaxDepth)

	var parentChunks []*types.Chunk
	var currentGroup []*types.Chunk
	var currentSize int

	for _, child := range childrenChunks {
		childSize := len(child.Text)

		// Check if adding this child would exceed target size
		if len(currentGroup) > 0 && currentSize+childSize > targetSize {
			// Create parent chunk for current group
			parentChunk := sc.createParentChunk(currentGroup, depth, options)
			parentChunks = append(parentChunks, parentChunk)

			// Start new group
			currentGroup = []*types.Chunk{child}
			currentSize = childSize
		} else {
			// Add to current group
			currentGroup = append(currentGroup, child)
			currentSize += childSize
		}
	}

	// Create parent for remaining group
	if len(currentGroup) > 0 {
		parentChunk := sc.createParentChunk(currentGroup, depth, options)
		parentChunks = append(parentChunks, parentChunk)
	}

	return parentChunks, nil
}

// createParentChunk creates a parent chunk from children
func (sc *SemanticChunker) createParentChunk(children []*types.Chunk, depth int, options *types.ChunkingOptions) *types.Chunk {
	if len(children) == 0 {
		return nil
	}

	// Combine text from children (without overlap since semantic chunks don't need overlap)
	var textBuilder strings.Builder
	var startIndex, endIndex int
	var startLine, endLine int

	for i, child := range children {
		if i > 0 {
			textBuilder.WriteString("\n") // Add separator between semantic segments
		}
		textBuilder.WriteString(child.Text)

		// Update position information
		if child.TextPos != nil {
			if i == 0 {
				startIndex = child.TextPos.StartIndex
				startLine = child.TextPos.StartLine
			}
			endIndex = child.TextPos.EndIndex
			endLine = child.TextPos.EndLine
		}
	}

	// Create text position
	var textPos *types.TextPosition
	if children[0].TextPos != nil {
		textPos = &types.TextPosition{
			StartIndex: startIndex,
			EndIndex:   endIndex,
			StartLine:  startLine,
			EndLine:    endLine,
		}
	}

	// Determine if this is a leaf node
	isLeaf := depth >= options.MaxDepth

	return &types.Chunk{
		ID:       uuid.NewString(),
		Text:     textBuilder.String(),
		Type:     children[0].Type,
		ParentID: "", // Will be set if there are higher levels
		Depth:    depth,
		Leaf:     isLeaf,
		Root:     false, // Parent chunks are not root
		Index:    0,     // Will be set when adding to parent
		Status:   types.ChunkingStatusCompleted,
		TextPos:  textPos,
		Parents:  []types.Chunk{}, // Will be populated later
	}
}

// updateChildrenParents updates children's parent information
func (sc *SemanticChunker) updateChildrenParents(children, parents []*types.Chunk) {
	if len(parents) == 0 {
		return
	}

	// Create mapping from child to parent
	childToParent := make(map[string]*types.Chunk)
	childIndex := 0

	for _, parent := range parents {
		// Calculate how many children belong to this parent based on text content
		parentText := parent.Text
		childrenText := ""
		startChildIndex := childIndex

		for childIndex < len(children) {
			if childIndex > startChildIndex {
				childrenText += "\n"
			}
			childrenText += children[childIndex].Text

			// Check if we've matched the parent's text content
			if strings.Contains(parentText, children[childIndex].Text) {
				childToParent[children[childIndex].ID] = parent
				children[childIndex].ParentID = parent.ID
				children[childIndex].Root = false
				children[childIndex].Index = childIndex - startChildIndex
				childIndex++

				// If parent text is fully covered, move to next parent
				if len(childrenText) >= len(parentText)-10 { // Allow some tolerance
					break
				}
			} else {
				break
			}
		}
	}
}

// calculateLevelSize calculates target size for hierarchy level
func (sc *SemanticChunker) calculateLevelSize(baseSize, depth, maxDepth int) int {
	// For semantic chunking, we use a different scaling than structured chunking
	// Higher levels should accommodate more content
	multiplier := maxDepth - depth + 2
	return baseSize * multiplier
}

// reportProgress reports progress if callback is set
func (sc *SemanticChunker) reportProgress(chunkID, progress, step string, data interface{}) {
	if sc.progressCallback != nil {
		if err := sc.progressCallback(chunkID, progress, step, data); err != nil {
			log.Warn("Progress callback error: %v", err)
		}
	}
}
