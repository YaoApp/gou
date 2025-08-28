package converter

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/yaoapp/gou/graphrag/types"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

const (
	// BufferSize for streaming processing
	BufferSize = 2 * 1024 * 1024 // 2MB chunks
)

// UTF8 implements the Converter interface for UTF-8 conversion
type UTF8 struct{}

// NewUTF8 creates a new UTF8 instance
func NewUTF8() *UTF8 {
	return &UTF8{}
}

// Convert converts a file to UTF-8 plain text with progress callbacks
func (c *UTF8) Convert(ctx context.Context, file string, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	c.reportProgress(types.ConverterStatusPending, "Opening file", 0.0, callback...)

	f, err := os.Open(file)
	if err != nil {
		c.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to open file: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()

	result, err := c.ConvertStream(ctx, f, callback...)
	if err != nil {
		return nil, err
	}

	c.reportProgress(types.ConverterStatusSuccess, "File conversion completed", 1.0, callback...)
	return result, nil
}

// ConvertStream converts a stream to UTF-8 plain text with gzip support using streaming
func (c *UTF8) ConvertStream(ctx context.Context, stream io.ReadSeeker, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	c.reportProgress(types.ConverterStatusPending, "Starting conversion", 0.0, callback...)

	// Check if gzipped
	var reader io.Reader
	var isGzipped bool
	peekBuffer := make([]byte, 2)
	n, err := io.ReadFull(stream, peekBuffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		if err == io.EOF {
			return nil, fmt.Errorf("empty stream")
		}
		c.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to read stream: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to read stream: %w", err)
	}

	// Reset to beginning
	_, err = stream.Seek(0, io.SeekStart)
	if err != nil {
		c.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to reset stream: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to reset stream: %w", err)
	}

	// Check gzip magic number (0x1f, 0x8b)
	if n >= 2 && peekBuffer[0] == 0x1f && peekBuffer[1] == 0x8b {
		c.reportProgress(types.ConverterStatusPending, "Decompressing gzip", 0.2, callback...)
		gzipReader, err := gzip.NewReader(stream)
		if err != nil {
			c.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to create gzip reader: %v", err), 0.0, callback...)
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
		isGzipped = true
	} else {
		reader = stream
		isGzipped = false
	}

	// Fast path: check if already UTF-8 and text (for non-gzipped files)
	if !isGzipped {
		if isUTF8, isText := c.quickTextCheck(stream); isUTF8 && isText {
			c.reportProgress(types.ConverterStatusPending, "File already UTF-8, using fast path", 0.3, callback...)
			return c.fastReadUTF8(ctx, stream, callback...)
		}
		// Reset stream position after quick check
		_, err = stream.Seek(0, io.SeekStart)
		if err != nil {
			c.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to reset stream: %v", err), 0.0, callback...)
			return nil, fmt.Errorf("failed to reset stream: %w", err)
		}
	}

	c.reportProgress(types.ConverterStatusPending, "Processing stream", 0.4, callback...)

	// Stream processing to save memory
	text, err := c.streamToUTF8(ctx, reader, callback...)
	if err != nil {
		c.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to process stream: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to process stream: %w", err)
	}

	c.reportProgress(types.ConverterStatusSuccess, "Conversion completed", 1.0, callback...)

	// Create metadata with basic information
	metadata := map[string]interface{}{
		"encoding":    "utf-8",
		"gzipped":     isGzipped,
		"text_length": len(text),
	}

	return &types.ConvertResult{
		Text:     text,
		Metadata: metadata,
	}, nil
}

// streamToUTF8 processes the stream in chunks to save memory
func (c *UTF8) streamToUTF8(ctx context.Context, reader io.Reader, callback ...types.ConverterProgress) (string, error) {
	var result strings.Builder
	buf := make([]byte, BufferSize)
	var leftover []byte // Handle incomplete UTF-8 sequences at chunk boundaries
	totalProcessed := 0
	firstChunk := true
	var detectedEncoding encoding.Encoding // Store detected encoding for consistent conversion

	bufferedReader := bufio.NewReaderSize(reader, BufferSize)

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		n, err := bufferedReader.Read(buf)
		if n == 0 {
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("failed to read chunk: %w", err)
			}
			continue
		}

		totalProcessed += n

		// Combine leftover from previous chunk with current data
		var chunk []byte
		if len(leftover) > 0 {
			chunk = make([]byte, len(leftover)+n)
			copy(chunk, leftover)
			copy(chunk[len(leftover):], buf[:n])
			leftover = leftover[:0] // Clear leftover
		} else {
			chunk = buf[:n]
		}

		// Handle incomplete UTF-8 at the end of chunk
		validEnd := len(chunk)
		if err != io.EOF { // Only check for incomplete sequences if not at end of file
			validEnd = c.findLastValidUTF8Boundary(chunk)
			if validEnd < len(chunk) {
				// Save incomplete bytes for next iteration
				leftover = make([]byte, len(chunk)-validEnd)
				copy(leftover, chunk[validEnd:])
			}
		}

		// Process the valid part
		if validEnd > 0 {
			validChunk := chunk[:validEnd]

			// Check if first chunk is text content (binary detection) and detect encoding
			if firstChunk {
				if !c.isTextContent(validChunk) {
					return "", fmt.Errorf("content appears to be binary, not text")
				}

				// Detect encoding on the first chunk for consistent conversion
				if !utf8.Valid(validChunk) {
					detectedEncoding = c.detectEncoding(validChunk)
				}
				firstChunk = false
			}

			// Remove BOM if this is the first chunk
			if result.Len() == 0 && len(validChunk) >= 3 &&
				validChunk[0] == 0xEF && validChunk[1] == 0xBB && validChunk[2] == 0xBF {
				validChunk = validChunk[3:]
			}

			// Convert chunk to UTF-8
			text := c.convertChunkToUTF8(validChunk, detectedEncoding)
			result.WriteString(text)
		}

		if err == io.EOF {
			break
		}

		// Report progress occasionally
		if totalProcessed%(BufferSize*4) == 0 {
			progress := 0.4 + (0.4 * float64(totalProcessed) / float64(totalProcessed+BufferSize))
			c.reportProgress(types.ConverterStatusPending, fmt.Sprintf("Processed %d bytes", totalProcessed), progress, callback...)
		}
	}

	// Process any remaining leftover bytes
	if len(leftover) > 0 {
		text := c.convertChunkToUTF8(leftover, detectedEncoding)
		result.WriteString(text)
	}

	finalText := result.String()
	if finalText == "" {
		return "", fmt.Errorf("no valid data to convert")
	}

	// Clean boundaries of the final result
	return c.cleanUTF8Boundaries(finalText), nil
}

// findLastValidUTF8Boundary finds the last complete UTF-8 character boundary
func (c *UTF8) findLastValidUTF8Boundary(data []byte) int {
	// Start from the end and work backwards
	for i := len(data) - 1; i >= 0 && i >= len(data)-4; i-- {
		if utf8.ValidString(string(data[:i+1])) {
			return i + 1
		}
	}
	// If we can't find a valid boundary, return the original length
	return len(data)
}

// convertChunkToUTF8 converts a chunk to valid UTF-8 using detected encoding
func (c *UTF8) convertChunkToUTF8(chunk []byte, detectedEncoding encoding.Encoding) string {
	// If already valid UTF-8, return as-is
	if utf8.Valid(chunk) {
		return string(chunk)
	}

	// If we have a detected encoding, use it
	if detectedEncoding != nil {
		decoder := detectedEncoding.NewDecoder()
		result, err := decoder.Bytes(chunk)
		if err == nil {
			return string(result)
		}
	}

	// Fallback: try to detect and convert from other encodings
	if converted, ok := c.detectAndConvert(chunk); ok {
		return converted
	}

	// Last resort: Convert to string, Go will replace invalid UTF-8 with replacement character
	return string(chunk)
}

// detectEncoding detects the most likely encoding for the given data
func (c *UTF8) detectEncoding(data []byte) encoding.Encoding {
	// List of encodings to try, in order of preference
	encodings := []encoding.Encoding{
		traditionalchinese.Big5,
		simplifiedchinese.GBK,
		simplifiedchinese.GB18030,
		japanese.ShiftJIS,
		japanese.EUCJP,
		korean.EUCKR,
		charmap.ISO8859_1,
		charmap.Windows1252,
		charmap.Windows1251,
	}

	bestEncoding := encoding.Encoding(nil)
	bestScore := 0.0

	for _, enc := range encodings {
		decoder := enc.NewDecoder()
		result, err := decoder.Bytes(data)
		if err == nil {
			resultStr := string(result)
			score := c.calculateTextScore(resultStr)
			if score > bestScore {
				bestScore = score
				bestEncoding = enc
			}
		}
	}

	// Only return encoding if it has a reasonable score
	if bestScore > 0.5 {
		return bestEncoding
	}

	return nil
}

// calculateTextScore calculates how "text-like" the converted string is
func (c *UTF8) calculateTextScore(text string) float64 {
	if len(text) == 0 {
		return 0.0
	}

	totalRunes := 0
	replacementRunes := 0
	printableRunes := 0
	chineseRunes := 0

	for _, r := range text {
		totalRunes++

		if r == '\uFFFD' { // Unicode replacement character
			replacementRunes++
		} else if r >= 32 && r < 127 { // ASCII printable
			printableRunes++
		} else if r >= 0x4E00 && r <= 0x9FFF { // CJK Unified Ideographs
			chineseRunes++
			printableRunes++
		} else if r == '\t' || r == '\n' || r == '\r' { // Common whitespace
			printableRunes++
		}
	}

	if totalRunes == 0 {
		return 0.0
	}

	// Calculate score based on various factors
	replacementRatio := float64(replacementRunes) / float64(totalRunes)
	printableRatio := float64(printableRunes) / float64(totalRunes)
	chineseRatio := float64(chineseRunes) / float64(totalRunes)

	// Penalize high replacement character ratio
	score := 1.0 - replacementRatio

	// Bonus for high printable ratio
	score *= printableRatio

	// Bonus for Chinese characters (since we're dealing with Big5)
	if chineseRatio > 0.1 {
		score *= (1.0 + chineseRatio)
	}

	return score
}

// detectAndConvert tries to detect the encoding and convert to UTF-8
func (c *UTF8) detectAndConvert(data []byte) (string, bool) {
	// List of encodings to try, in order of preference
	encodings := []struct {
		name string
		enc  encoding.Encoding
	}{
		{"Big5", traditionalchinese.Big5},
		{"GBK", simplifiedchinese.GBK},
		{"GB18030", simplifiedchinese.GB18030},
		{"Shift_JIS", japanese.ShiftJIS},
		{"EUC-JP", japanese.EUCJP},
		{"EUC-KR", korean.EUCKR},
		{"ISO-8859-1", charmap.ISO8859_1},
		{"Windows-1252", charmap.Windows1252},
		{"Windows-1251", charmap.Windows1251},
	}

	for _, encoding := range encodings {
		decoder := encoding.enc.NewDecoder()
		result, err := decoder.Bytes(data)
		if err == nil {
			// Check if the result looks reasonable (contains valid characters)
			resultStr := string(result)
			if c.isReasonableText(resultStr) {
				return resultStr, true
			}
		}
	}

	return "", false
}

// isReasonableText checks if the converted text looks reasonable
func (c *UTF8) isReasonableText(text string) bool {
	if len(text) == 0 {
		return false
	}

	// Count valid characters vs replacement characters
	totalRunes := 0
	replacementRunes := 0

	for _, r := range text {
		totalRunes++
		if r == '\uFFFD' { // Unicode replacement character
			replacementRunes++
		}
	}

	// If more than 50% are replacement characters, probably not the right encoding
	if totalRunes > 0 && replacementRunes > totalRunes/2 {
		return false
	}

	return true
}

// cleanUTF8Boundaries removes broken UTF-8 characters only from start and end
func (c *UTF8) cleanUTF8Boundaries(text string) string {
	if text == "" {
		return text
	}

	data := []byte(text)
	start := 0
	end := len(data)

	// Remove broken UTF-8 from beginning (only first few bytes)
	for start < len(data) && start < 4 {
		if (data[start] & 0x80) == 0 {
			break // ASCII
		}
		if (data[start] & 0xC0) != 0x80 {
			break // Valid UTF-8 start
		}
		start++ // Skip continuation byte
	}

	// Remove broken UTF-8 from end (only last few bytes)
	for end > start && len(data)-end < 4 {
		if utf8.ValidString(string(data[start:end])) {
			break
		}
		end--
	}

	if start >= end {
		return ""
	}

	return string(data[start:end])
}

// quickTextCheck performs fast UTF-8 and text format detection
func (c *UTF8) quickTextCheck(stream io.ReadSeeker) (isUTF8 bool, isText bool) {
	// Read first 8KB for detection
	buffer := make([]byte, 8*1024)
	n, err := stream.Read(buffer)
	if err != nil && err != io.EOF {
		return false, false
	}

	if n == 0 {
		return true, true // Empty file is considered UTF-8 text
	}

	data := buffer[:n]

	// Check if valid UTF-8
	isUTF8 = utf8.Valid(data)

	// Check if it's text (not binary)
	isText = c.isTextContent(data)

	return isUTF8, isText
}

// fastReadUTF8 reads UTF-8 content directly with minimal processing
func (c *UTF8) fastReadUTF8(ctx context.Context, stream io.ReadSeeker, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	c.reportProgress(types.ConverterStatusPending, "Reading UTF-8 content", 0.5, callback...)

	// Reset to beginning
	_, err := stream.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to reset stream: %w", err)
	}

	var result strings.Builder
	buf := make([]byte, BufferSize)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		n, err := stream.Read(buf)
		if n > 0 {
			data := buf[:n]

			// Remove BOM from first chunk
			if result.Len() == 0 && len(data) >= 3 &&
				data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
				data = data[3:]
			}

			result.Write(data)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read stream: %w", err)
		}
	}

	text := result.String()
	if text == "" {
		return nil, fmt.Errorf("no content to convert")
	}

	c.reportProgress(types.ConverterStatusSuccess, "Fast UTF-8 read completed", 1.0, callback...)

	// Create metadata for fast path
	metadata := map[string]interface{}{
		"encoding":    "utf-8",
		"fast_path":   true,
		"text_length": len(text),
	}

	return &types.ConvertResult{
		Text:     text,
		Metadata: metadata,
	}, nil
}

// isTextContent checks if the data appears to be text (not binary)
func (c *UTF8) isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	// Check for common binary file signatures first
	if len(data) >= 4 {
		// Check for common binary signatures
		signatures := [][]byte{
			{0x89, 0x50, 0x4E, 0x47}, // PNG
			{0xFF, 0xD8, 0xFF},       // JPEG
			{0x47, 0x49, 0x46},       // GIF
			{0x25, 0x50, 0x44, 0x46}, // PDF
			{0x50, 0x4B, 0x03, 0x04}, // ZIP
			{0x50, 0x4B, 0x05, 0x06}, // ZIP (empty)
			{0x50, 0x4B, 0x07, 0x08}, // ZIP (spanned)
			{0x7F, 0x45, 0x4C, 0x46}, // ELF executable
			{0x4D, 0x5A},             // PE executable
		}

		for _, sig := range signatures {
			if len(data) >= len(sig) {
				match := true
				for i, b := range sig {
					if data[i] != b {
						match = false
						break
					}
				}
				if match {
					return false
				}
			}
		}
	}

	// If it's valid UTF-8, it's very likely to be text
	if utf8.Valid(data) {
		// For UTF-8 content, use a more lenient approach
		return c.isUTF8TextContent(data)
	}

	// For non-UTF-8 data, use byte-level analysis
	return c.isByteTextContent(data)
}

// isUTF8TextContent checks UTF-8 valid data for text characteristics
func (c *UTF8) isUTF8TextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	// Convert to string and analyze runes instead of bytes
	text := string(data)
	totalRunes := 0
	controlRunes := 0
	printableRunes := 0

	for _, r := range text {
		totalRunes++

		// Allow common text control characters
		if r == '\t' || r == '\n' || r == '\r' || r == '\f' || r == '\v' {
			printableRunes++
			continue
		}

		// Count ASCII control characters (0-31, 127) as control
		// But be more lenient with extended Unicode ranges
		if r < 32 || r == 127 {
			controlRunes++
		} else {
			printableRunes++
		}
	}

	// For UTF-8 text, allow up to 50% control characters
	// This is more lenient since UTF-8 can contain various Unicode characters
	if totalRunes > 10 && controlRunes > totalRunes/2 {
		return false
	}

	return true
}

// isByteTextContent checks non-UTF-8 data using byte-level analysis
func (c *UTF8) isByteTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	// Count control characters and printable characters
	controlChars := 0
	printableChars := 0

	for _, b := range data {
		// Allow common text control characters
		if b == '\t' || b == '\n' || b == '\r' {
			printableChars++
			continue
		}

		// Count control characters (0-31, 127) but be more lenient with extended range
		// Don't count 127-159 as control chars since they might be valid in other encodings
		if b < 32 || b == 127 {
			controlChars++
		} else {
			printableChars++
		}
	}

	// Use a more lenient threshold for non-UTF-8 content
	if len(data) > 10 && controlChars > len(data)/2 {
		return false
	}

	return true
}

// reportProgress reports conversion progress
func (c *UTF8) reportProgress(status types.ConverterStatus, message string, progress float64, callbacks ...types.ConverterProgress) {
	if len(callbacks) == 0 {
		return
	}

	payload := types.ConverterPayload{
		Status:   status,
		Message:  message,
		Progress: progress,
	}

	for _, callback := range callbacks {
		if callback != nil {
			callback(status, payload)
		}
	}
}
