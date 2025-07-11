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
func (sc *SemanticChunker) Chunk(ctx context.Context, text string, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
	// Convert text to ReadSeeker
	reader := strings.NewReader(text)
	return sc.ChunkStream(ctx, reader, options, callback)
}

// ChunkFile implements semantic chunking on file
func (sc *SemanticChunker) ChunkFile(ctx context.Context, file string, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
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
func (sc *SemanticChunker) ChunkStream(ctx context.Context, stream io.ReadSeeker, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
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

	// Use a map to preserve order based on chunk index
	chunksMap := make(map[int]*types.Chunk)
	var mu sync.Mutex
	var maxIndex int

	err := sc.structuredChunker.ChunkStream(ctx, stream, structuredOpts, func(chunk *types.Chunk) error {
		mu.Lock()
		defer mu.Unlock()
		chunksMap[chunk.Index] = chunk
		if chunk.Index > maxIndex {
			maxIndex = chunk.Index
		}

		// Report progress for each structured chunk
		sc.reportProgress(chunk.ID, "completed", "structured_chunk", map[string]interface{}{
			"chunk_index": chunk.Index,
			"chunk_size":  len(chunk.Text),
			"chunk_text":  chunk.Text,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Rebuild ordered slice from map
	chunks := make([]*types.Chunk, 0, len(chunksMap))
	for i := 0; i <= maxIndex; i++ {
		if chunk, exists := chunksMap[i]; exists {
			chunks = append(chunks, chunk)
		}
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
			var processedChunks []*types.Chunk

			if err != nil {
				// Log the error but don't fail the entire process
				log.Warn("Failed to process semantic segmentation for chunk %s after retries: %v", chunk.ID, err)

				// Create a fallback semantic chunk from the original structured chunk
				// This ensures we don't lose any content
				fallbackChunk := &types.Chunk{
					ID:       chunk.ID + "_fallback", // Unique ID for fallback
					Text:     chunk.Text,
					Type:     chunk.Type,
					ParentID: chunk.ParentID,
					Depth:    options.MaxDepth, // Set to MaxDepth like other semantic chunks
					Leaf:     true,
					Root:     options.MaxDepth == 1,
					Index:    0, // Will be updated later
					Status:   types.ChunkingStatusCompleted,
					TextPos:  chunk.TextPos,
					Parents:  chunk.Parents,
				}
				processedChunks = []*types.Chunk{fallbackChunk}

				// Report warning progress instead of failure
				sc.reportProgress(chunk.ID, "warning", "semantic_analysis", map[string]interface{}{
					"error":  err.Error(),
					"action": "using_fallback_chunk",
				})
			} else {
				// Update indices and store results in order
				processedChunks = sc.updateSemanticChunkIndices(semanticChunks, originalIndex)
			}

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

	// We no longer fail the entire process if some chunks failed
	// Instead, we use fallback chunks and continue processing
	// Only fail if no chunks were processed at all
	if len(semanticChunksByIndex) == 0 {
		return nil, fmt.Errorf("no chunks were processed")
	}

	// Merge and finalize results
	return sc.mergeSemanticChunksInOrder(semanticChunksByIndex, len(structuredChunks)), nil
}

// updateSemanticChunkIndices updates the indices of semantic chunks based on original structured chunk order
func (sc *SemanticChunker) updateSemanticChunkIndices(semanticChunks []*types.Chunk, originalIndex int) []*types.Chunk {
	// Set temporary indices within this chunk group, will be overridden in mergeSemanticChunksInOrder
	for j, semanticChunk := range semanticChunks {
		semanticChunk.Index = j // Temporary index within this group: 0, 1, 2, ...
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

	// Update indices within MaxDepth level to be sequential 0-N
	// The hierarchy building will set appropriate indices for each level independently
	for i, chunk := range allSemanticChunks {
		chunk.Index = i // Sequential index within MaxDepth level: 0, 1, 2, 3, ...
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

	// Convert positions to chunks using generic Split method
	semanticChunks := chunk.Split(chars, positions)

	// Override depth and other properties for semantic chunks (LLM produces MaxDepth level chunks)
	for i, semanticChunk := range semanticChunks {
		semanticChunk.Depth = options.MaxDepth // LLM segmentation produces minimum granularity (MaxDepth)
		semanticChunk.Index = i                // Index within this semantic split: 0, 1, 2, ...
		semanticChunk.Root = false             // Semantic chunks are not root
		semanticChunk.Leaf = true              // At MaxDepth, these are leaf nodes
	}

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
		"temperature": 0,             // Slightly higher temperature for more semantic awareness
		"model":       "gpt-4o-mini", // Use more capable model for better semantic understanding
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
func (sc *SemanticChunker) buildHierarchyAndOutput(ctx context.Context, semanticChunks []*types.Chunk, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
	// Step 1: Set correct Root status for semantic chunks
	// If MaxDepth == 1, semantic chunks are root chunks
	// If MaxDepth > 1, semantic chunks are leaf chunks
	for _, chunk := range semanticChunks {
		chunk.Root = (options.MaxDepth == 1) // If MaxDepth==1, semantic chunks are root nodes
		chunk.Leaf = true                    // Semantic chunks are always leaf nodes
	}

	// Step 2: Output semantic chunks (MaxDepth level)
	for _, chunk := range semanticChunks {
		if err := callback(chunk); err != nil {
			return fmt.Errorf("callback failed for semantic chunk %s: %w", chunk.ID, err)
		}

		sc.reportProgress(chunk.ID, "output", "semantic_chunk", nil)
	}

	// Step 3: Build hierarchy if MaxDepth > 1
	if options.MaxDepth > 1 {
		return sc.buildHierarchy(ctx, semanticChunks, options, callback)
	}

	return nil
}

// buildHierarchy builds hierarchical chunks
func (sc *SemanticChunker) buildHierarchy(ctx context.Context, baseChunks []*types.Chunk, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
	currentLevelChunks := baseChunks
	currentDepth := options.MaxDepth - 1 // Start merging from MaxDepth-1 upwards

	for currentDepth >= 1 { // Merge up to depth=1
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

		// Set correct indices for this level (each level uses 0-N indexing)
		for i, chunk := range nextLevelChunks {
			chunk.Index = i // Each level starts from 0: 0, 1, 2, ...
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
		currentDepth-- // Move up the hierarchy, depth decreases
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

	// Calculate how many children should be grouped together
	// Higher levels should group more children together
	remainingLevels := options.MaxDepth - depth + 1
	groupSize := max(2, remainingLevels) // At least 2 children per parent, more for higher levels

	var parentChunks []*types.Chunk

	// Create groups of children, ensuring we actually combine content
	for i := 0; i < len(childrenChunks); i += groupSize {
		endIdx := min(i+groupSize, len(childrenChunks))
		currentGroup := childrenChunks[i:endIdx]

		// Only create parent if we're actually combining multiple children
		if len(currentGroup) >= 2 {
			parentChunk := sc.createParentChunk(currentGroup, depth, options)
			if parentChunk != nil {
				parentChunks = append(parentChunks, parentChunk)
			}
		} else if len(currentGroup) == 1 {
			// If only one child left, try to merge it with the last parent
			if len(parentChunks) > 0 {
				lastParent := parentChunks[len(parentChunks)-1]
				// Check if adding this child would not exceed target size too much
				if len(lastParent.Text)+len(currentGroup[0].Text) <= targetSize*2 {
					// Merge with last parent
					lastParent.Text += "\n" + currentGroup[0].Text
					// Update text position
					if lastParent.TextPos != nil && currentGroup[0].TextPos != nil {
						lastParent.TextPos.EndIndex = currentGroup[0].TextPos.EndIndex
						lastParent.TextPos.EndLine = currentGroup[0].TextPos.EndLine
					}
				} else {
					// Create separate parent for this single child (unusual case)
					parentChunk := sc.createParentChunk(currentGroup, depth, options)
					if parentChunk != nil {
						parentChunks = append(parentChunks, parentChunk)
					}
				}
			} else {
				// Create parent for single child (unusual case)
				parentChunk := sc.createParentChunk(currentGroup, depth, options)
				if parentChunk != nil {
					parentChunks = append(parentChunks, parentChunk)
				}
			}
		}
	}

	return parentChunks, nil
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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

	// Determine if this is a leaf node (depth == 1 is top level, not leaf)
	isLeaf := false      // Parent chunks are never leaf nodes
	isRoot := depth == 1 // depth == 1 is root node

	return &types.Chunk{
		ID:       uuid.NewString(),
		Text:     textBuilder.String(),
		Type:     children[0].Type,
		ParentID: "", // Will be set if there are higher levels
		Depth:    depth,
		Leaf:     isLeaf,
		Root:     isRoot, // depth == 1 is root node
		Index:    0,      // Will be set when adding to parent
		Status:   types.ChunkingStatusCompleted,
		TextPos:  textPos,
		Parents:  []types.Chunk{}, // Will be populated later
	}
}

// updateChildrenParents updates children's parent information
func (sc *SemanticChunker) updateChildrenParents(children, parents []*types.Chunk) {
	if len(parents) == 0 || len(children) == 0 {
		return
	}

	// Build parent-child mapping based on text containment
	childIndex := 0
	for _, parent := range parents {
		var parentChildren []*types.Chunk

		// Find children that belong to this parent
		for childIndex < len(children) {
			child := children[childIndex]

			// Check if child's text is contained in parent's text
			if strings.Contains(parent.Text, child.Text) {
				parentChildren = append(parentChildren, child)
				childIndex++
			} else {
				// This child doesn't belong to current parent
				break
			}
		}

		// Update parent information for all children of this parent
		for _, child := range parentChildren {
			// Set basic parent info
			child.ParentID = parent.ID
			child.Root = false
			// Keep the child's existing Index (it's already set correctly within its depth level)

			// Update Parents chain
			if len(child.Parents) == 0 {
				// Initialize Parents array with current parent
				child.Parents = []types.Chunk{*parent}
			} else {
				// Append current parent to existing Parents chain
				// Check if parent already exists to avoid duplicates
				found := false
				for j, existingParent := range child.Parents {
					if existingParent.ID == parent.ID {
						// Update existing parent entry
						child.Parents[j] = *parent
						found = true
						break
					}
				}
				if !found {
					child.Parents = append(child.Parents, *parent)
				}
			}
		}

		// If no children found by text containment, fall back to sequential assignment
		if len(parentChildren) == 0 && childIndex < len(children) {
			// Assign remaining children proportionally
			remainingChildren := len(children) - childIndex
			remainingParents := 0
			for j := range parents {
				if j >= childIndex/len(children)*len(parents) {
					remainingParents++
				}
			}
			if remainingParents == 0 {
				remainingParents = 1
			}

			childrenPerParent := max(1, remainingChildren/remainingParents)
			endIdx := min(childIndex+childrenPerParent, len(children))

			for i := childIndex; i < endIdx; i++ {
				child := children[i]
				child.ParentID = parent.ID
				child.Root = false
				// Don't modify child.Index - keep global indexing 0-N

				// Update Parents chain
				if len(child.Parents) == 0 {
					child.Parents = []types.Chunk{*parent}
				} else {
					child.Parents = append(child.Parents, *parent)
				}
			}
			childIndex = endIdx
		}
	}
}

// calculateLevelSize calculates target size for hierarchy level
func (sc *SemanticChunker) calculateLevelSize(baseSize, depth, maxDepth int) int {
	// Use original logic to maintain compatibility with existing tests
	// Higher levels (lower depth numbers) should accommodate more content
	// Formula: (maxDepth - depth + 2) * baseSize
	// This ensures L1 gets most space, L2 less, L3 (MaxDepth) least
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
