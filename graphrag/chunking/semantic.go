package chunking

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/kun/log"
)

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

	// Step 3: Process semantic chunking
	semanticChunks, err := sc.processSemanticChunks(ctx, structuredChunks, options)
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

// processSemanticChunks processes structured chunks and merges results in correct order
func (sc *SemanticChunker) processSemanticChunks(ctx context.Context, structuredChunks []*types.Chunk, options *types.ChunkingOptions) ([]*types.Chunk, error) {
	semanticOpts := options.SemanticOptions

	// Channel for controlling concurrency
	semaphore := make(chan struct{}, semanticOpts.MaxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Use a map to store results with original index to preserve order
	semanticChunksByIndex := make(map[int][]*types.Chunk)
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

		go func(chunk *types.Chunk, originalIndex int) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()

			// Report progress
			sc.reportProgress(chunk.ID, "processing", "semantic_analysis", map[string]interface{}{
				"chunk_index":  originalIndex,
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

			// Update indices and store results in order
			processedChunks := sc.updateSemanticChunkIndices(semanticChunks, originalIndex)

			// Store results with original index to preserve order
			mu.Lock()
			semanticChunksByIndex[originalIndex] = processedChunks
			mu.Unlock()

			// Report completion progress
			sc.reportProgress(chunk.ID, "completed", "semantic_analysis", map[string]interface{}{
				"chunks_generated": len(processedChunks),
			})
		}(structuredChunk, i)
	}

	wg.Wait()

	if firstError != nil {
		return nil, firstError
	}

	// Merge and finalize results
	return sc.mergeSemanticChunksInOrder(semanticChunksByIndex, len(structuredChunks)), nil
}

// updateSemanticChunkIndices updates the indices of semantic chunks based on original structured chunk order
func (sc *SemanticChunker) updateSemanticChunkIndices(semanticChunks []*types.Chunk, originalIndex int) []*types.Chunk {
	// Update indices of semantic chunks based on original structured chunk order
	// and position within the semantic split
	for j, semanticChunk := range semanticChunks {
		// Calculate global index: originalIndex * large_number + within_chunk_index
		// This ensures proper ordering across all chunks
		semanticChunk.Index = originalIndex*10000 + j
	}
	return semanticChunks
}

// mergeSemanticChunksInOrder merges semantic chunks in the correct order based on original structured chunk order
func (sc *SemanticChunker) mergeSemanticChunksInOrder(semanticChunksByIndex map[int][]*types.Chunk, totalStructuredChunks int) []*types.Chunk {
	// Merge results in the correct order based on original structured chunk order
	var allSemanticChunks []*types.Chunk
	for i := 0; i < totalStructuredChunks; i++ {
		if chunks, exists := semanticChunksByIndex[i]; exists {
			allSemanticChunks = append(allSemanticChunks, chunks...)
		}
	}

	// Final pass to update global indices sequentially
	for globalIndex, chunk := range allSemanticChunks {
		chunk.Index = globalIndex
	}

	return allSemanticChunks
}

// processChunkSemanticSegmentation processes semantic segmentation for a single chunk
func (sc *SemanticChunker) processChunkSemanticSegmentation(ctx context.Context, chunk *types.Chunk, options *types.ChunkingOptions) ([]*types.Chunk, error) {
	semanticOpts := options.SemanticOptions

	// Try with retries
	var positions []types.Position
	var chars []string
	var err error

	for retry := 0; retry <= semanticOpts.MaxRetry; retry++ {
		// Use options.Size as the target size for semantic segmentation
		chars, positions, err = sc.callLLMForSegmentation(ctx, chunk, semanticOpts, options.Size)
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
		positions = []types.Position{{StartPos: 0, EndPos: len(chunk.Text)}}
	}

	// Convert positions to chunks
	semanticChunks := chunk.Split(chars, positions)
	return semanticChunks, nil
}

// callLLMForSegmentation calls LLM to get semantic segmentation positions using streaming
func (sc *SemanticChunker) callLLMForSegmentation(ctx context.Context, chunk *types.Chunk, semanticOpts *types.SemanticOptions, maxSize int) ([]string, []types.Position, error) {
	// Get the connector
	conn, err := connector.Select(semanticOpts.Connector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to select connector '%s': %w", semanticOpts.Connector, err)
	}

	// Build prompt using utils function with size information
	prompt := utils.SemanticPrompt(semanticOpts.Prompt, maxSize)
	// prompt = prompt + "\n\n# Text to segment:\n```text\n" + chunk.Text + "\n```"

	chars := chunk.TextWChars()
	charsJSON := ""
	for idx, char := range chars {
		charsJSON += fmt.Sprintf("%d: %s\n", idx, char)
	}

	// Prepare request payload
	requestData := map[string]interface{}{
		"temperature": 0,         // Slightly higher temperature for more semantic awareness
		"model":       "gpt-4.1", // Use more capable model for better semantic understanding
		"messages": []map[string]interface{}{
			{"role": "system", "content": prompt},
			{"role": "user", "content": charsJSON},
		},
	}

	// Set model from connector settings
	setting := conn.Setting()
	if model, ok := setting["model"].(string); ok && model != "" {
		requestData["model"] = model
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
		requestData["tools"] = utils.SemanticToolcall
		requestData["tool_choice"] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name": "segment_text",
			},
		}
	}

	// Create stream parser
	parser := utils.NewSemanticParser(semanticOpts.Toolcall)
	var finalContent string
	var finalArguments string

	// Stream callback function with progress reporting
	streamCallback := func(data []byte) error {
		// Skip empty data chunks
		if len(data) == 0 {
			return nil
		}

		// Parse streaming chunk
		positions, err := parser.ParseSemanticPositions(data)
		if err != nil {
			log.Warn("Failed to parse stream chunk: %v", err)
			return nil // Don't fail the entire stream for parsing errors
		}

		// Ignore nil positions
		if positions == nil {
			return nil
		}

		// Validate positions
		err = types.ValidatePositions(chars, positions)
		if err != nil {
			log.Warn("Invalid positions: %v", err)
			return err
		}

		// Report progress with streaming data including positions
		sc.reportProgress(chunk.ID, "streaming", "llm_response", positions)
		return nil
	}

	// Make streaming request using utils.StreamLLM
	err = utils.StreamLLM(ctx, conn, "chat/completions", requestData, streamCallback)
	if err != nil {
		return nil, nil, fmt.Errorf("LLM streaming request failed: %w", err)
	}

	// Extract final results from parser
	if semanticOpts.Toolcall {
		finalArguments = parser.Arguments
		positions, err := parser.ParseSemanticToolcall(finalArguments)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse toolcall: %w", err)
		}
		return chars, positions, nil
	}

	finalContent = parser.Content
	positions, err := parser.ParseSemanticRegular(finalContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse regular: %w", err)
	}
	return chars, positions, nil
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
