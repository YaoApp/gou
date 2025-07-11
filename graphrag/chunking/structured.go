package chunking

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/kun/log"
)

// ChunkManager manages chunk hierarchy and status
type ChunkManager struct {
	chunks   map[string]*types.Chunk   // ID -> Chunk mapping
	children map[string][]*types.Chunk // Parent ID -> Children mapping
	mutex    sync.RWMutex
}

// NewChunkManager creates a new chunk manager
func NewChunkManager() *ChunkManager {
	return &ChunkManager{
		chunks:   make(map[string]*types.Chunk),
		children: make(map[string][]*types.Chunk),
	}
}

// AddChunk adds a chunk to the manager
func (cm *ChunkManager) AddChunk(chunk *types.Chunk) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.chunks[chunk.ID] = chunk
	if chunk.ParentID != "" {
		cm.children[chunk.ParentID] = append(cm.children[chunk.ParentID], chunk)
	}
}

// UpdateChunkStatus updates a chunk's status and propagates to parents if needed
func (cm *ChunkManager) UpdateChunkStatus(chunkID string, status types.ChunkingStatus) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	chunk, exists := cm.chunks[chunkID]
	if !exists {
		return
	}

	chunk.Status = status

	// If this is a completed leaf node, check parent status
	if status == types.ChunkingStatusCompleted && chunk.Leaf && chunk.ParentID != "" {
		cm.checkParentCompletion(chunk.ParentID)
	}
}

// checkParentCompletion checks if all children of a parent are completed
func (cm *ChunkManager) checkParentCompletion(parentID string) {
	parent, exists := cm.chunks[parentID]
	if !exists {
		return
	}

	children := cm.children[parentID]
	if len(children) == 0 {
		return
	}

	// Check if all children are completed
	allCompleted := true
	for _, child := range children {
		if child.Status != types.ChunkingStatusCompleted {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		parent.Status = types.ChunkingStatusCompleted
		// Recursively check parent's parent
		if parent.ParentID != "" {
			cm.checkParentCompletion(parent.ParentID)
		}
	}
}

// GetParents returns the parents chain for a chunk
func (cm *ChunkManager) GetParents(chunk *types.Chunk) []types.Chunk {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var parents []types.Chunk
	currentID := chunk.ParentID

	for currentID != "" {
		if parent, exists := cm.chunks[currentID]; exists {
			parents = append([]types.Chunk{*parent}, parents...) // Prepend to maintain order
			currentID = parent.ParentID
		} else {
			break
		}
	}

	return parents
}

// StructuredChunker is the chunker for structured data
type StructuredChunker struct {
	chunkManager *ChunkManager
	// Atomic counters for index generation by depth
	indexCounters [4]int64 // Support depths 1-3, plus index 0 unused
}

// NewStructuredChunker creates a new structured chunker
func NewStructuredChunker() *StructuredChunker {
	return &StructuredChunker{
		chunkManager: NewChunkManager(),
	}
}

// validateAndFixOptions validates and fixes chunking options
func (chunker *StructuredChunker) validateAndFixOptions(options *types.ChunkingOptions) {
	if options.MaxDepth > 5 {
		log.Warn("MaxDepth is set to %d which exceeds the recommended maximum of 5. Setting MaxDepth to 5 for optimal performance.", options.MaxDepth)
		options.MaxDepth = 5
	}
	if options.MaxDepth < 1 {
		log.Warn("MaxDepth is set to %d which is below the minimum of 1. Setting MaxDepth to 1.", options.MaxDepth)
		options.MaxDepth = 1
	}
	if options.SizeMultiplier <= 0 {
		options.SizeMultiplier = 3 // Default multiplier
	}
}

// NewStructuredOptions creates a new structured chunker options by chunking type
func NewStructuredOptions(chunkingType types.ChunkingType) *types.ChunkingOptions {

	switch chunkingType {
	case types.ChunkingTypeCode, types.ChunkingTypeJSON:
		return &types.ChunkingOptions{Size: 800, Overlap: 100, MaxDepth: 3, SizeMultiplier: 3, MaxConcurrent: 10}

	case types.ChunkingTypeVideo, types.ChunkingTypeAudio, types.ChunkingTypeImage:
		return &types.ChunkingOptions{Size: 300, Overlap: 20, MaxDepth: 1, SizeMultiplier: 3, MaxConcurrent: 10}

	default:
		return &types.ChunkingOptions{Size: 300, Overlap: 20, MaxDepth: 1, SizeMultiplier: 3, MaxConcurrent: 10}
	}
}

// Chunk is the main function to chunk text
func (chunker *StructuredChunker) Chunk(ctx context.Context, text string, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
	// Validate and fix options
	chunker.validateAndFixOptions(options)

	// Auto-detect content type if not provided
	if options.Type == "" {
		options.Type = types.ChunkingTypeText // Default to text for string input
	}

	// Convert text to ReadSeeker
	reader := strings.NewReader(text)
	return chunker.ChunkStream(ctx, reader, options, callback)
}

// ChunkFile is the function to chunk file
func (chunker *StructuredChunker) ChunkFile(ctx context.Context, file string, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
	// Validate and fix options
	chunker.validateAndFixOptions(options)

	// Open file
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()

	// Auto-detect content type if not provided
	if options.Type == "" {
		// Try to detect MIME type from file content
		buffer := make([]byte, 512)
		n, _ := f.Read(buffer)
		if n > 0 {
			mimeType := http.DetectContentType(buffer[:n])
			options.Type = types.GetChunkingTypeFromMime(mimeType)
		}

		// Fallback to filename-based detection
		if options.Type == "" || options.Type == types.ChunkingTypeText {
			options.Type = types.GetChunkingTypeFromFilename(file)
		}

		// Reset file pointer to beginning
		f.Seek(0, io.SeekStart)
	}

	// Use ChunkStream to process the file
	return chunker.ChunkStream(ctx, f, options, callback)
}

// ChunkStream is the function to chunk stream
func (chunker *StructuredChunker) ChunkStream(ctx context.Context, stream io.ReadSeeker, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
	// Validate and fix options
	chunker.validateAndFixOptions(options)

	// Get stream size
	streamSize, err := chunker.getStreamSize(stream)
	if err != nil {
		return fmt.Errorf("failed to get stream size: %w", err)
	}

	// Process level 1 chunks directly from stream
	return chunker.processStreamLevels(ctx, stream, 0, streamSize, "", 1, options, callback)
}

// getStreamSize gets the total size of the stream
func (chunker *StructuredChunker) getStreamSize(stream io.ReadSeeker) (int64, error) {
	// Seek to end to get size
	size, err := stream.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	// Seek back to beginning
	_, err = stream.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}

	return size, nil
}

// processStreamLevels processes chunks at different levels directly from stream
func (chunker *StructuredChunker) processStreamLevels(ctx context.Context, stream io.ReadSeeker, offset, size int64, parentID string, currentDepth int, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
	// Validate and fix options to ensure defaults are set
	chunker.validateAndFixOptions(options)

	// Check if maximum depth is reached
	if currentDepth > options.MaxDepth {
		return nil
	}

	// Calculate chunk size for current level
	chunkSize := chunker.calculateSubSize(options.Size, currentDepth, options.MaxDepth, options.SizeMultiplier)
	overlap := chunker.calculateSubOverlap(options.Overlap, currentDepth, options.MaxDepth, options.SizeMultiplier)

	// Generate chunks for current level and process them concurrently
	chunks, err := chunker.generateStreamChunksWithLines(stream, offset, size, chunkSize, overlap, parentID, currentDepth, options.Type, options)
	if err != nil {
		return err
	}

	// Process current level chunks concurrently with status tracking
	if err := chunker.processCurrentLevel(ctx, chunks, options.MaxConcurrent, func(chunk *types.Chunk) error {
		// Call the original callback
		err := callback(chunk)
		if err != nil {
			// Update status to failed on callback error
			chunker.chunkManager.UpdateChunkStatus(chunk.ID, types.ChunkingStatusFailed)
			return err
		}

		// If this is a leaf node and callback succeeded, mark as completed
		if chunk.Leaf {
			chunker.chunkManager.UpdateChunkStatus(chunk.ID, types.ChunkingStatusCompleted)
		}

		return nil
	}); err != nil {
		return err
	}

	// If not at maximum depth, process sub-levels
	if currentDepth < options.MaxDepth {

		for _, chunk := range chunks {
			// Only create sub-chunks if current chunk is large enough
			subChunkSize := chunker.calculateSubSize(options.Size, currentDepth+1, options.MaxDepth, options.SizeMultiplier)

			if int64(len(chunk.Text)) > int64(subChunkSize) {
				// Update chunk status to processing
				chunker.chunkManager.UpdateChunkStatus(chunk.ID, types.ChunkingStatusProcessing)

				// Process sub-chunks from the text content with line tracking
				baseStartLine := 1
				if chunk.TextPos != nil {
					baseStartLine = chunk.TextPos.StartLine
				}
				err := chunker.processTextLevelsWithLines(ctx, chunk.Text, baseStartLine, chunk.ID, currentDepth+1, options, callback)
				if err != nil {
					// Update chunk status to failed on error
					chunker.chunkManager.UpdateChunkStatus(chunk.ID, types.ChunkingStatusFailed)
					return err
				}
			}
		}
	}

	return nil
}

// generateStreamChunksWithLines generates chunks by reading from stream with line number tracking
func (chunker *StructuredChunker) generateStreamChunksWithLines(stream io.ReadSeeker, offset, totalSize int64, chunkSize, overlap int, parentID string, depth int, chunkType types.ChunkingType, options *types.ChunkingOptions) ([]*types.Chunk, error) {
	var chunks []*types.Chunk

	if totalSize <= int64(chunkSize) {
		// If total size is smaller than chunk size, read all as single chunk
		_, err := stream.Seek(offset, io.SeekStart)
		if err != nil {
			return nil, err
		}

		data := make([]byte, totalSize)
		n, err := io.ReadFull(stream, data)
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, err
		}

		text := chunker.fixUTF8Chunk(string(data[:n]))
		startLine, endLine := chunker.calculateLinesFromOffset(stream, offset, int64(len(text)))

		// Determine if this is a leaf node
		isLeaf := depth >= options.MaxDepth || int64(len(text)) <= int64(chunker.calculateSubSize(options.Size, depth+1, options.MaxDepth, options.SizeMultiplier))

		// Determine if this is a root node
		isRoot := depth == 1 && parentID == ""

		// Determine initial status
		var status types.ChunkingStatus
		if isLeaf {
			status = types.ChunkingStatusCompleted
		} else {
			status = types.ChunkingStatusPending
		}

		chunk := &types.Chunk{
			ID:       uuid.NewString(),
			Text:     text,
			ParentID: parentID,
			Depth:    depth,
			Type:     chunkType,
			Leaf:     isLeaf,
			Root:     isRoot,
			Index:    chunker.getNextIndex(depth),
			Status:   status,
			TextPos: &types.TextPosition{
				StartIndex: int(offset),
				EndIndex:   int(offset) + n,
				StartLine:  startLine,
				EndLine:    endLine,
			},
		}

		// Add to chunk manager and set parents
		chunker.chunkManager.AddChunk(chunk)
		chunk.Parents = chunker.chunkManager.GetParents(chunk)

		chunks = append(chunks, chunk)
		return chunks, nil
	}

	// Process in chunks
	pos := int64(0)
	for pos < totalSize {
		// Calculate current chunk end position
		end := pos + int64(chunkSize)
		if end > totalSize {
			end = totalSize
		}

		// Seek to current position
		_, err := stream.Seek(offset+pos, io.SeekStart)
		if err != nil {
			return nil, err
		}

		// Read chunk data
		chunkLen := end - pos
		data := make([]byte, chunkLen)
		n, err := io.ReadFull(stream, data)
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, err
		}

		text := chunker.fixUTF8Chunk(string(data[:n]))
		startLine, endLine := chunker.calculateLinesFromOffset(stream, offset+pos, int64(len(text)))

		// Determine if this is a leaf node
		isLeaf := depth >= options.MaxDepth || int64(len(text)) <= int64(chunker.calculateSubSize(options.Size, depth+1, options.MaxDepth, options.SizeMultiplier))

		// Determine if this is a root node
		isRoot := depth == 1 && parentID == ""

		// Determine initial status
		var status types.ChunkingStatus
		if isLeaf {
			status = types.ChunkingStatusCompleted
		} else {
			status = types.ChunkingStatusPending
		}

		chunk := &types.Chunk{
			ID:       uuid.NewString(),
			Text:     text,
			ParentID: parentID,
			Depth:    depth,
			Type:     chunkType,
			Leaf:     isLeaf,
			Root:     isRoot,
			Index:    chunker.getNextIndex(depth),
			Status:   status,
			TextPos: &types.TextPosition{
				StartIndex: int(offset + pos),
				EndIndex:   int(offset+pos) + n,
				StartLine:  startLine,
				EndLine:    endLine,
			},
		}

		// Add to chunk manager and set parents
		chunker.chunkManager.AddChunk(chunk)
		chunk.Parents = chunker.chunkManager.GetParents(chunk)

		chunks = append(chunks, chunk)

		// Calculate next position considering overlap
		pos += int64(chunkSize) - int64(overlap)
		if pos >= totalSize {
			break
		}
	}

	return chunks, nil
}

// processTextLevelsWithLines processes chunks from text content with line tracking (for sub-levels)
func (chunker *StructuredChunker) processTextLevelsWithLines(ctx context.Context, text string, baseStartLine int, parentID string, currentDepth int, options *types.ChunkingOptions, callback types.ChunkingProgress) error {
	// Validate and fix options to ensure defaults are set
	chunker.validateAndFixOptions(options)

	// Check if maximum depth is reached
	if currentDepth > options.MaxDepth {
		return nil
	}

	// Calculate chunk size for current level
	chunkSize := chunker.calculateSubSize(options.Size, currentDepth, options.MaxDepth, options.SizeMultiplier)
	overlap := chunker.calculateSubOverlap(options.Overlap, currentDepth, options.MaxDepth, options.SizeMultiplier)

	// Create chunks from text with line tracking
	chunks := chunker.createChunksWithLines(text, chunkSize, overlap, baseStartLine, parentID, currentDepth, options.Type, options)

	// Process current level chunks concurrently with status tracking
	if err := chunker.processCurrentLevel(ctx, chunks, options.MaxConcurrent, func(chunk *types.Chunk) error {
		// Call the original callback
		err := callback(chunk)
		if err != nil {
			// Update status to failed on callback error
			chunker.chunkManager.UpdateChunkStatus(chunk.ID, types.ChunkingStatusFailed)
			return err
		}

		// If this is a leaf node and callback succeeded, mark as completed
		if chunk.Leaf {
			chunker.chunkManager.UpdateChunkStatus(chunk.ID, types.ChunkingStatusCompleted)
		}

		return nil
	}); err != nil {
		return err
	}

	// If not at maximum depth, recursively process next level
	if currentDepth < options.MaxDepth {
		for _, chunk := range chunks {
			subChunkSize := chunker.calculateSubSize(options.Size, currentDepth+1, options.MaxDepth, options.SizeMultiplier)
			if len(chunk.Text) > subChunkSize {
				// Update chunk status to processing
				chunker.chunkManager.UpdateChunkStatus(chunk.ID, types.ChunkingStatusProcessing)

				baseStartLine := 1
				if chunk.TextPos != nil {
					baseStartLine = chunk.TextPos.StartLine
				}
				err := chunker.processTextLevelsWithLines(ctx, chunk.Text, baseStartLine, chunk.ID, currentDepth+1, options, callback)
				if err != nil {
					// Update chunk status to failed on error
					chunker.chunkManager.UpdateChunkStatus(chunk.ID, types.ChunkingStatusFailed)
					return err
				}
			}
		}
	}

	return nil
}

// createChunksWithLines creates text chunks with specified size, overlap and line tracking
func (chunker *StructuredChunker) createChunksWithLines(text string, size, overlap, baseStartLine int, parentID string, depth int, chunkType types.ChunkingType, options *types.ChunkingOptions) []*types.Chunk {
	var chunks []*types.Chunk
	textBytes := []byte(text)
	totalLen := len(textBytes)

	if totalLen <= size {
		// If text length is less than or equal to specified size, return single chunk
		endLine := baseStartLine + strings.Count(text, "\n")

		// Determine if this is a leaf node
		isLeaf := depth >= options.MaxDepth || len(text) <= chunker.calculateSubSize(options.Size, depth+1, options.MaxDepth, options.SizeMultiplier)

		// Determine if this is a root node
		isRoot := depth == 1 && parentID == ""

		// Determine initial status
		var status types.ChunkingStatus
		if isLeaf {
			status = types.ChunkingStatusCompleted
		} else {
			status = types.ChunkingStatusPending
		}

		chunk := &types.Chunk{
			ID:       uuid.NewString(),
			Text:     text,
			ParentID: parentID,
			Depth:    depth,
			Type:     chunkType,
			Leaf:     isLeaf,
			Root:     isRoot,
			Index:    chunker.getNextIndex(depth),
			Status:   status,
			TextPos: &types.TextPosition{
				StartIndex: 0,
				EndIndex:   totalLen,
				StartLine:  baseStartLine,
				EndLine:    endLine,
			},
		}

		// Add to chunk manager and set parents
		chunker.chunkManager.AddChunk(chunk)
		chunk.Parents = chunker.chunkManager.GetParents(chunk)

		chunks = append(chunks, chunk)
		return chunks
	}

	pos := 0
	currentLine := baseStartLine
	for pos < totalLen {
		end := pos + size
		if end > totalLen {
			end = totalLen
		}

		chunkText := chunker.fixUTF8Chunk(string(textBytes[pos:end]))
		linesInChunk := strings.Count(chunkText, "\n")
		endLine := currentLine + linesInChunk

		// Determine if this is a leaf node
		isLeaf := depth >= options.MaxDepth || len(chunkText) <= chunker.calculateSubSize(options.Size, depth+1, options.MaxDepth, options.SizeMultiplier)

		// Determine if this is a root node
		isRoot := depth == 1 && parentID == ""

		// Determine initial status
		var status types.ChunkingStatus
		if isLeaf {
			status = types.ChunkingStatusCompleted
		} else {
			status = types.ChunkingStatusPending
		}

		chunk := &types.Chunk{
			ID:       uuid.NewString(),
			Text:     chunkText,
			ParentID: parentID,
			Depth:    depth,
			Type:     chunkType,
			Leaf:     isLeaf,
			Root:     isRoot,
			Index:    chunker.getNextIndex(depth),
			Status:   status,
			TextPos: &types.TextPosition{
				StartIndex: pos,
				EndIndex:   end,
				StartLine:  currentLine,
				EndLine:    endLine,
			},
		}

		// Add to chunk manager and set parents
		chunker.chunkManager.AddChunk(chunk)
		chunk.Parents = chunker.chunkManager.GetParents(chunk)

		chunks = append(chunks, chunk)

		// Calculate next position considering overlap
		pos += size - overlap
		if pos >= totalLen {
			break
		}

		// Update current line for next chunk (considering overlap)
		if overlap > 0 && pos < totalLen {
			// Ensure we don't go before the beginning of the text
			overlapStart := pos - overlap
			if overlapStart < 0 {
				overlapStart = 0
			}
			if overlapStart < pos {
				overlapText := string(textBytes[overlapStart:pos])
				overlapLines := strings.Count(overlapText, "\n")
				currentLine = endLine - overlapLines
			} else {
				currentLine = endLine
			}
		} else {
			currentLine = endLine
		}
	}

	return chunks
}

// calculateLinesFromOffset calculates start and end line numbers for a chunk at given offset
func (chunker *StructuredChunker) calculateLinesFromOffset(stream io.ReadSeeker, offset, length int64) (int, int) {
	// Save current position
	currentPos, err := stream.Seek(0, io.SeekCurrent)
	if err != nil {
		return 1, 1 // fallback to line 1 if error
	}
	defer stream.Seek(currentPos, io.SeekStart) // restore position

	// Seek to beginning to count lines before offset
	_, err = stream.Seek(0, io.SeekStart)
	if err != nil {
		return 1, 1 // fallback to line 1 if error
	}

	startLine := 1
	if offset > 0 {
		// Read data from start to offset to count newlines
		buffer := make([]byte, 4096) // 4KB buffer for efficient reading
		bytesRead := int64(0)

		for bytesRead < offset {
			remaining := offset - bytesRead
			readSize := int64(len(buffer))
			if remaining < readSize {
				readSize = remaining
			}

			n, err := stream.Read(buffer[:readSize])
			if err != nil && err != io.EOF {
				return 1, 1 // fallback if error
			}
			if n == 0 {
				break
			}

			startLine += strings.Count(string(buffer[:n]), "\n")
			bytesRead += int64(n)
		}
	}

	// Now count lines in the chunk itself
	_, err = stream.Seek(offset, io.SeekStart)
	if err != nil {
		return startLine, startLine
	}

	chunkData := make([]byte, length)
	n, err := io.ReadFull(stream, chunkData)
	if err != nil && err != io.ErrUnexpectedEOF {
		return startLine, startLine
	}

	linesInChunk := strings.Count(string(chunkData[:n]), "\n")
	endLine := startLine + linesInChunk

	return startLine, endLine
}

// processCurrentLevel processes all chunks at current level concurrently
func (chunker *StructuredChunker) processCurrentLevel(ctx context.Context, chunks []*types.Chunk, maxConcurrent int, callback types.ChunkingProgress) error {
	if len(chunks) == 0 {
		return nil
	}

	// Limit concurrent goroutines
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // default value
	}

	// Create semaphore to control concurrency
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstError error

	for _, chunk := range chunks {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{} // acquire semaphore

		go func(c *types.Chunk) {
			defer func() {
				<-semaphore // release semaphore
				wg.Done()
			}()

			if err := callback(c); err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = err
				}
				mu.Unlock()
			}
		}(chunk)
	}

	wg.Wait()
	return firstError
}

// calculateSubSize calculates sub-level chunk size based on current depth, max depth, and multiplier
func (chunker *StructuredChunker) calculateSubSize(baseSize, depth, maxDepth, multiplier int) int {
	// Calculate remaining levels: maxDepth - depth
	// Size grows linearly with remaining levels: baseSize * max(1, remainingLevels * multiplier)
	remainingLevels := maxDepth - depth
	if remainingLevels <= 0 {
		return baseSize
	}
	return baseSize * remainingLevels * multiplier
}

// calculateSubOverlap calculates sub-level overlap based on current depth, max depth, and multiplier
func (chunker *StructuredChunker) calculateSubOverlap(baseOverlap, depth, maxDepth, multiplier int) int {
	// Calculate remaining levels: maxDepth - depth
	// Overlap grows linearly with remaining levels: baseOverlap * max(1, remainingLevels * multiplier)
	remainingLevels := maxDepth - depth
	if remainingLevels <= 0 {
		return baseOverlap
	}
	return baseOverlap * remainingLevels * multiplier
}

// getNextIndex returns the next index for the given depth level
func (chunker *StructuredChunker) getNextIndex(depth int) int {
	if depth < 1 || depth > 3 {
		depth = 1 // fallback to depth 1
	}
	return int(atomic.AddInt64(&chunker.indexCounters[depth], 1) - 1)
}

// resetIndexCounters resets all index counters to 0 (mainly for testing)
func (chunker *StructuredChunker) resetIndexCounters() {
	for i := range chunker.indexCounters {
		atomic.StoreInt64(&chunker.indexCounters[i], 0)
	}
}

// fixUTF8Chunk removes broken UTF-8 characters from the beginning and end of a chunk
func (chunker *StructuredChunker) fixUTF8Chunk(text string) string {
	if text == "" {
		return text
	}

	data := []byte(text)
	start := 0
	end := len(data)

	// Remove broken UTF-8 characters from the beginning
	for start < len(data) {
		if (data[start] & 0x80) == 0 {
			// ASCII character, valid start
			break
		}
		if (data[start] & 0xC0) != 0x80 {
			// Valid UTF-8 character start
			break
		}
		// This is a continuation byte, skip it
		start++
	}

	// Remove broken UTF-8 characters from the end
	// We need to be more careful here - check if the string is valid
	// and if not, find the last valid UTF-8 character boundary
	for end > start {
		candidate := string(data[start:end])
		if utf8.ValidString(candidate) {
			break
		}
		// Move back one byte and try again
		end--
	}

	if start >= end {
		return ""
	}

	result := string(data[start:end])

	// Double-check that the result is valid UTF-8
	if !utf8.ValidString(result) {
		// If still invalid, return empty string to be safe
		return ""
	}

	return result
}
