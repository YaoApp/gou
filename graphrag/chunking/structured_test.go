package chunking

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// TestData holds test constants
const (
	SmallText = "Hello, World!\nThis is a test.\nLine 3.\nLine 4."
)

// getTestDataPath returns the absolute path to test data files based on current file location
func getTestDataPath(filename string) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot get current file path")
	}

	// Get the directory of the current test file (chunking directory)
	currentDir := filepath.Dir(currentFile)

	// Navigate to the tests directory: chunking -> graphrag, then add tests
	graphragDir := filepath.Dir(currentDir)
	testsDir := filepath.Join(graphragDir, "tests")

	return filepath.Join(testsDir, filename)
}

// Test data file paths - dynamically resolved
var (
	TestDataFile     = getTestDataPath("threekingdoms.txt")
	CodeTestDataFile = getTestDataPath("code.ts")
	CSVTestDataFile  = getTestDataPath("qa.csv")
)

// Test utilities
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}

	return tmpFile.Name()
}

func TestNewStructuredChunker(t *testing.T) {
	chunker := NewStructuredChunker()
	if chunker == nil {
		t.Error("NewStructuredChunker() returned nil")
	}
}

func TestNewStructuredOptions(t *testing.T) {
	tests := []struct {
		name                  string
		chunkingType          types.ChunkingType
		expectedSize          int
		expectedOverlap       int
		expectedMaxDepth      int
		expectedMaxConcurrent int
	}{
		{
			name:                  "Code type",
			chunkingType:          types.ChunkingTypeCode,
			expectedSize:          800,
			expectedOverlap:       100,
			expectedMaxDepth:      3,
			expectedMaxConcurrent: 10,
		},
		{
			name:                  "JSON type",
			chunkingType:          types.ChunkingTypeJSON,
			expectedSize:          800,
			expectedOverlap:       100,
			expectedMaxDepth:      3,
			expectedMaxConcurrent: 10,
		},
		{
			name:                  "Video type",
			chunkingType:          types.ChunkingTypeVideo,
			expectedSize:          300,
			expectedOverlap:       20,
			expectedMaxDepth:      1,
			expectedMaxConcurrent: 10,
		},
		{
			name:                  "Audio type",
			chunkingType:          types.ChunkingTypeAudio,
			expectedSize:          300,
			expectedOverlap:       20,
			expectedMaxDepth:      1,
			expectedMaxConcurrent: 10,
		},
		{
			name:                  "Image type",
			chunkingType:          types.ChunkingTypeImage,
			expectedSize:          300,
			expectedOverlap:       20,
			expectedMaxDepth:      1,
			expectedMaxConcurrent: 10,
		},
		{
			name:                  "Default type",
			chunkingType:          types.ChunkingTypeText,
			expectedSize:          300,
			expectedOverlap:       20,
			expectedMaxDepth:      1,
			expectedMaxConcurrent: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := NewStructuredOptions(tt.chunkingType)
			if options.Size != tt.expectedSize {
				t.Errorf("Size = %d, want %d", options.Size, tt.expectedSize)
			}
			if options.Overlap != tt.expectedOverlap {
				t.Errorf("Overlap = %d, want %d", options.Overlap, tt.expectedOverlap)
			}
			if options.MaxDepth != tt.expectedMaxDepth {
				t.Errorf("MaxDepth = %d, want %d", options.MaxDepth, tt.expectedMaxDepth)
			}
			if options.MaxConcurrent != tt.expectedMaxConcurrent {
				t.Errorf("MaxConcurrent = %d, want %d", options.MaxConcurrent, tt.expectedMaxConcurrent)
			}
		})
	}
}

func TestChunk(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	tests := []struct {
		name         string
		text         string
		options      *types.ChunkingOptions
		expectChunks int
	}{
		{
			name: "Small text",
			text: SmallText,
			options: &types.ChunkingOptions{
				Size:          20,
				Overlap:       5,
				MaxDepth:      2,
				MaxConcurrent: 2,
			},
			expectChunks: 4, // Will be split due to size limit
		},
		{
			name: "Large text",
			text: SmallText,
			options: &types.ChunkingOptions{
				Size:          100,
				Overlap:       10,
				MaxDepth:      1,
				MaxConcurrent: 1,
			},
			expectChunks: 1, // Single chunk
		},
		{
			name: "Empty type auto-detection",
			text: "Some text",
			options: &types.ChunkingOptions{
				Size:          50,
				Overlap:       5,
				MaxDepth:      1,
				MaxConcurrent: 1,
			},
			expectChunks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunks []*types.Chunk
			var mu sync.Mutex
			err := chunker.Chunk(ctx, tt.text, tt.options, func(chunk *types.Chunk) error {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
				return nil
			})

			if err != nil {
				t.Errorf("Chunk() error = %v", err)
				return
			}

			if len(chunks) == 0 {
				t.Error("No chunks returned")
				return
			}

			// Verify chunk structure
			for i, chunk := range chunks {
				if chunk.ID == "" {
					t.Errorf("Chunk %d has empty ID", i)
				}
				if chunk.Text == "" {
					t.Errorf("Chunk %d has empty text", i)
				}
				if chunk.Type == "" {
					t.Errorf("Chunk %d has empty type", i)
				}
				if chunk.TextPos == nil {
					t.Errorf("Chunk %d has nil TextPos", i)
				}
			}
		})
	}
}

func TestChunkFile(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	// Test with temporary files
	tests := []struct {
		name     string
		content  string
		filename string
		options  *types.ChunkingOptions
	}{
		{
			name:     "Text file",
			content:  SmallText,
			filename: "test.txt",
			options: &types.ChunkingOptions{
				Size:          30,
				Overlap:       5,
				MaxDepth:      2,
				MaxConcurrent: 2,
			},
		},
		{
			name:     "TypeScript code file",
			content:  "import { Process } from \"@yao/runtime\";\n\nexport class Excel {\n  private handle: string | null = null;\n  constructor(private file: string) {}\n}",
			filename: "test.ts",
			options: &types.ChunkingOptions{
				Size:          50,
				Overlap:       10,
				MaxDepth:      2,
				MaxConcurrent: 2,
			},
		},
		{
			name:     "JSON file",
			content:  `{"name": "test", "value": 123}`,
			filename: "test.json",
			options: &types.ChunkingOptions{
				Size:          15,
				Overlap:       3,
				MaxDepth:      1,
				MaxConcurrent: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := createTempFile(t, tt.content)
			defer os.Remove(tmpFile)

			// Rename to test extension
			newName := filepath.Join(filepath.Dir(tmpFile), tt.filename)
			err := os.Rename(tmpFile, newName)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(newName)

			var chunks []*types.Chunk
			var mu sync.Mutex
			err = chunker.ChunkFile(ctx, newName, tt.options, func(chunk *types.Chunk) error {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
				return nil
			})

			if err != nil {
				t.Errorf("ChunkFile() error = %v", err)
				return
			}

			if len(chunks) == 0 {
				t.Error("No chunks returned")
				return
			}

			// Verify auto-detection worked
			expectedType := types.GetChunkingTypeFromFilename(tt.filename)
			for _, chunk := range chunks {
				if chunk.Type != expectedType {
					t.Errorf("Expected type %s, got %s", expectedType, chunk.Type)
				}
			}
		})
	}

	// Test with non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		options := &types.ChunkingOptions{Size: 100, Overlap: 10, MaxDepth: 1, MaxConcurrent: 1}
		err := chunker.ChunkFile(ctx, "/non/existent/file.txt", options, func(chunk *types.Chunk) error {
			return nil
		})
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

func TestChunkCodeFile(t *testing.T) {
	if _, err := os.Stat(CodeTestDataFile); os.IsNotExist(err) {
		t.Skip("Code test data file not found")
	}

	chunker := NewStructuredChunker()
	ctx := context.Background()
	options := &types.ChunkingOptions{
		Type:          types.ChunkingTypeCode,
		Size:          800,
		Overlap:       100,
		MaxDepth:      3,
		MaxConcurrent: 4,
	}

	var chunks []*types.Chunk
	var totalSize int
	var mu sync.Mutex

	err := chunker.ChunkFile(ctx, CodeTestDataFile, options, func(chunk *types.Chunk) error {
		mu.Lock()
		chunks = append(chunks, chunk)
		totalSize += len(chunk.Text)
		mu.Unlock()
		return nil
	})

	if err != nil {
		t.Fatalf("ChunkFile() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("No chunks returned from code test data file")
	}

	t.Logf("Processed %d code chunks with total size %d bytes", len(chunks), totalSize)

	// Verify chunk integrity for code
	for i, chunk := range chunks {
		if chunk == nil {
			t.Errorf("Chunk %d is nil", i)
			continue
		}
		if chunk.Type != types.ChunkingTypeCode {
			t.Errorf("Chunk %d has wrong type: %s, expected %s", i, chunk.Type, types.ChunkingTypeCode)
		}
		if chunk.TextPos == nil {
			t.Errorf("Chunk %d has nil TextPos", i)
		} else {
			if chunk.TextPos.StartLine <= 0 {
				t.Errorf("Chunk %d has invalid StartLine: %d", i, chunk.TextPos.StartLine)
			}
			if chunk.TextPos.EndLine < chunk.TextPos.StartLine {
				t.Errorf("Chunk %d has EndLine < StartLine", i)
			}
		}
		// Verify code chunks contain typical code patterns
		if strings.Contains(chunk.Text, "import") || strings.Contains(chunk.Text, "export") ||
			strings.Contains(chunk.Text, "class") || strings.Contains(chunk.Text, "function") {
			// This is good - contains code-like content
		} else if len(chunk.Text) > 50 {
			t.Logf("Warning: Chunk %d may not contain typical code patterns (len: %d)", i, len(chunk.Text))
		}
	}
}

func TestChunkCSVFile(t *testing.T) {
	if _, err := os.Stat(CSVTestDataFile); os.IsNotExist(err) {
		t.Skip("CSV test data file not found")
	}

	chunker := NewStructuredChunker()
	ctx := context.Background()
	options := &types.ChunkingOptions{
		Size:          500,
		Overlap:       50,
		MaxDepth:      2,
		MaxConcurrent: 3,
	}

	var chunks []*types.Chunk
	var totalSize int
	var mu sync.Mutex

	err := chunker.ChunkFile(ctx, CSVTestDataFile, options, func(chunk *types.Chunk) error {
		mu.Lock()
		chunks = append(chunks, chunk)
		totalSize += len(chunk.Text)
		mu.Unlock()
		return nil
	})

	if err != nil {
		t.Fatalf("ChunkFile() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("No chunks returned from CSV test data file")
	}

	t.Logf("Processed %d CSV chunks with total size %d bytes", len(chunks), totalSize)

	// Verify chunk integrity for CSV
	csvChunkCount := 0
	for i, chunk := range chunks {
		if chunk == nil {
			t.Errorf("Chunk %d is nil", i)
			continue
		}
		if chunk.TextPos == nil {
			t.Errorf("Chunk %d has nil TextPos", i)
		}
		// Check if chunk contains CSV-like content (commas, quotes)
		if strings.Contains(chunk.Text, ",") {
			csvChunkCount++
		}
	}

	if csvChunkCount == 0 {
		t.Error("No chunks contain CSV-like content")
	}
}

func TestChunkStream(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	tests := []struct {
		name    string
		content string
		options *types.ChunkingOptions
	}{
		{
			name:    "Basic stream",
			content: SmallText,
			options: &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          25,
				Overlap:       5,
				MaxDepth:      2,
				MaxConcurrent: 2,
			},
		},
		{
			name:    "Large stream",
			content: strings.Repeat("Line of text\n", 100),
			options: &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          50,
				Overlap:       10,
				MaxDepth:      3,
				MaxConcurrent: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			var chunks []*types.Chunk
			var mu sync.Mutex

			err := chunker.ChunkStream(ctx, reader, tt.options, func(chunk *types.Chunk) error {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
				return nil
			})

			if err != nil {
				t.Errorf("ChunkStream() error = %v", err)
				return
			}

			if len(chunks) == 0 {
				t.Error("No chunks returned")
			}

			// Verify line numbers are correctly calculated
			for i, chunk := range chunks {
				if chunk.TextPos == nil {
					t.Errorf("Chunk %d has nil TextPos", i)
					continue
				}
				if chunk.TextPos.StartLine <= 0 {
					t.Errorf("Chunk %d has invalid StartLine: %d", i, chunk.TextPos.StartLine)
				}
				if chunk.TextPos.EndLine < chunk.TextPos.StartLine {
					t.Errorf("Chunk %d has EndLine < StartLine: %d < %d", i, chunk.TextPos.EndLine, chunk.TextPos.StartLine)
				}
			}
		})
	}
}

func TestCalculateSubSize(t *testing.T) {
	chunker := NewStructuredChunker()

	tests := []struct {
		baseSize int
		depth    int
		expected int
	}{
		{100, 1, 600}, // L1: Size * 6
		{100, 2, 300}, // L2: Size * 3
		{100, 3, 100}, // L3: Size * 1
		{100, 4, 100}, // Default
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("depth_%d", tt.depth), func(t *testing.T) {
			result := chunker.calculateSubSize(tt.baseSize, tt.depth)
			if result != tt.expected {
				t.Errorf("calculateSubSize(%d, %d) = %d, want %d", tt.baseSize, tt.depth, result, tt.expected)
			}
		})
	}
}

func TestCalculateSubOverlap(t *testing.T) {
	chunker := NewStructuredChunker()

	tests := []struct {
		baseOverlap int
		depth       int
		expected    int
	}{
		{10, 1, 60}, // L1: Overlap * 6
		{10, 2, 30}, // L2: Overlap * 3
		{10, 3, 10}, // L3: Overlap * 1
		{10, 4, 10}, // Default
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("depth_%d", tt.depth), func(t *testing.T) {
			result := chunker.calculateSubOverlap(tt.baseOverlap, tt.depth)
			if result != tt.expected {
				t.Errorf("calculateSubOverlap(%d, %d) = %d, want %d", tt.baseOverlap, tt.depth, result, tt.expected)
			}
		})
	}
}

func TestGetStreamSize(t *testing.T) {
	chunker := NewStructuredChunker()

	tests := []struct {
		name     string
		content  string
		expected int64
	}{
		{"Empty", "", 0},
		{"Small", "hello", 5},
		{"With newlines", "line1\nline2\n", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			size, err := chunker.getStreamSize(reader)
			if err != nil {
				t.Errorf("getStreamSize() error = %v", err)
				return
			}
			if size != tt.expected {
				t.Errorf("getStreamSize() = %d, want %d", size, tt.expected)
			}

			// Verify stream position is reset
			buf := make([]byte, 1)
			n, _ := reader.Read(buf)
			if n > 0 && tt.expected > 0 {
				// Should read from beginning
				if string(buf[:n]) != string(tt.content[0]) {
					t.Error("Stream position was not reset")
				}
			}
		})
	}
}

func TestProcessCurrentLevel(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	t.Run("Empty chunks", func(t *testing.T) {
		var chunks []*types.Chunk
		err := chunker.processCurrentLevel(ctx, chunks, 1, func(chunk *types.Chunk) error {
			return nil
		})
		if err != nil {
			t.Errorf("processCurrentLevel() with empty chunks error = %v", err)
		}
	})

	t.Run("Normal processing", func(t *testing.T) {
		chunks := []*types.Chunk{
			{ID: "1", Text: "test1", Type: types.ChunkingTypeText, Leaf: false, Root: false, Index: 0, Status: types.ChunkingStatusPending},
			{ID: "2", Text: "test2", Type: types.ChunkingTypeText, Leaf: false, Root: false, Index: 1, Status: types.ChunkingStatusPending},
		}

		var processed []string
		var mu sync.Mutex

		err := chunker.processCurrentLevel(ctx, chunks, 2, func(chunk *types.Chunk) error {
			mu.Lock()
			processed = append(processed, chunk.ID)
			mu.Unlock()
			return nil
		})

		if err != nil {
			t.Errorf("processCurrentLevel() error = %v", err)
		}

		if len(processed) != 2 {
			t.Errorf("Expected 2 processed chunks, got %d", len(processed))
		}
	})

	t.Run("Callback error", func(t *testing.T) {
		chunks := []*types.Chunk{
			{ID: "1", Text: "test1", Type: types.ChunkingTypeText, Leaf: false, Root: false, Index: 0, Status: types.ChunkingStatusPending},
		}

		err := chunker.processCurrentLevel(ctx, chunks, 1, func(chunk *types.Chunk) error {
			return fmt.Errorf("callback error")
		})

		if err == nil {
			t.Error("Expected error from callback")
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		chunks := []*types.Chunk{
			{ID: "1", Text: "test1", Type: types.ChunkingTypeText, Leaf: false, Root: false, Index: 0, Status: types.ChunkingStatusPending},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := chunker.processCurrentLevel(ctx, chunks, 1, func(chunk *types.Chunk) error {
			return nil
		})

		if err == nil {
			t.Error("Expected context cancellation error")
		}
	})

	t.Run("Zero max concurrent", func(t *testing.T) {
		chunks := []*types.Chunk{
			{ID: "1", Text: "test1", Type: types.ChunkingTypeText, Leaf: false, Root: false, Index: 0, Status: types.ChunkingStatusPending},
		}

		err := chunker.processCurrentLevel(ctx, chunks, 0, func(chunk *types.Chunk) error {
			return nil
		})

		if err != nil {
			t.Errorf("processCurrentLevel() with zero maxConcurrent error = %v", err)
		}
	})
}

func TestCalculateLinesFromOffset(t *testing.T) {
	chunker := NewStructuredChunker()
	content := "line1\nline2\nline3\nline4\n"
	reader := strings.NewReader(content)

	tests := []struct {
		name          string
		offset        int64
		length        int64
		expectedStart int
		expectedEnd   int
	}{
		{"Beginning", 0, 6, 1, 2},   // "line1\n"
		{"Middle", 6, 6, 2, 3},      // "line2\n"
		{"No newlines", 0, 5, 1, 1}, // "line1"
		{"End", 18, 6, 4, 5},        // "line4\n"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := chunker.calculateLinesFromOffset(reader, tt.offset, tt.length)
			if start != tt.expectedStart {
				t.Errorf("StartLine = %d, want %d", start, tt.expectedStart)
			}
			if end != tt.expectedEnd {
				t.Errorf("EndLine = %d, want %d", end, tt.expectedEnd)
			}
		})
	}
}

func TestCreateChunksWithLines(t *testing.T) {
	chunker := NewStructuredChunker()

	tests := []struct {
		name           string
		text           string
		size           int
		overlap        int
		baseStartLine  int
		expectedChunks int
	}{
		{
			name:           "Small text single chunk",
			text:           "Hello World",
			size:           20,
			overlap:        0,
			baseStartLine:  1,
			expectedChunks: 1,
		},
		{
			name:           "Multi-line with overlap",
			text:           "line1\nline2\nline3\nline4\n",
			size:           10,
			overlap:        3,
			baseStartLine:  5,
			expectedChunks: 4, // Corrected expected count
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ChunkingOptions{
				Size:          tt.size,
				Overlap:       tt.overlap,
				MaxDepth:      3,
				MaxConcurrent: 1,
			}
			chunks := chunker.createChunksWithLines(tt.text, tt.size, tt.overlap, tt.baseStartLine, "", 1, types.ChunkingTypeText, options)

			if len(chunks) != tt.expectedChunks {
				t.Errorf("Expected %d chunks, got %d", tt.expectedChunks, len(chunks))
			}

			for i, chunk := range chunks {
				if chunk.TextPos == nil {
					t.Errorf("Chunk %d has nil TextPos", i)
					continue
				}
				if chunk.TextPos.StartLine < tt.baseStartLine {
					t.Errorf("Chunk %d StartLine %d < baseStartLine %d", i, chunk.TextPos.StartLine, tt.baseStartLine)
				}
				// Test Leaf field
				expectedLeaf := 1 >= options.MaxDepth || len(chunk.Text) <= chunker.calculateSubSize(options.Size, 1)
				if chunk.Leaf != expectedLeaf {
					t.Errorf("Chunk %d Leaf = %t, expected %t", i, chunk.Leaf, expectedLeaf)
				}
			}
		})
	}
}

func TestLeafNodeDetection(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	tests := []struct {
		name             string
		text             string
		maxDepth         int
		size             int
		expectedLeafRate float64 // Expected percentage of leaf nodes
	}{
		{
			name:             "Max depth 1 - all leaves",
			text:             strings.Repeat("This is a test line.\n", 20),
			maxDepth:         1,
			size:             50,
			expectedLeafRate: 1.0, // All chunks should be leaves at max depth
		},
		{
			name:             "Max depth 3 - mixed leaves",
			text:             strings.Repeat("This is a test line with more content.\n", 50),
			maxDepth:         3,
			size:             100,
			expectedLeafRate: 0.5, // Some should be leaves, some not
		},
		{
			name:             "Small chunks - mostly leaves",
			text:             "Small text content",
			maxDepth:         3,
			size:             20,
			expectedLeafRate: 1.0, // Should all be leaves due to size
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          tt.size,
				Overlap:       10,
				MaxDepth:      tt.maxDepth,
				MaxConcurrent: 2,
			}

			var chunks []*types.Chunk
			var mu sync.Mutex
			err := chunker.Chunk(ctx, tt.text, options, func(chunk *types.Chunk) error {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
				return nil
			})

			if err != nil {
				t.Errorf("Chunk() error = %v", err)
				return
			}

			if len(chunks) == 0 {
				t.Error("No chunks returned")
				return
			}

			// Count leaf nodes
			leafCount := 0
			for _, chunk := range chunks {
				if chunk.Leaf {
					leafCount++
				}
			}

			leafRate := float64(leafCount) / float64(len(chunks))
			t.Logf("Leaf rate: %.2f (%d/%d)", leafRate, leafCount, len(chunks))

			// Verify that leaves are properly marked
			for i, chunk := range chunks {
				// Check if leaf marking is correct
				nextLevelSize := chunker.calculateSubSize(options.Size, chunk.Depth)
				shouldBeLeaf := chunk.Depth >= options.MaxDepth || len(chunk.Text) <= nextLevelSize

				if chunk.Leaf != shouldBeLeaf {
					t.Errorf("Chunk %d (depth %d, len %d) Leaf = %t, expected %t",
						i, chunk.Depth, len(chunk.Text), chunk.Leaf, shouldBeLeaf)
				}
			}

			// Verify depth constraints
			for i, chunk := range chunks {
				if chunk.Depth > options.MaxDepth {
					t.Errorf("Chunk %d has depth %d > maxDepth %d", i, chunk.Depth, options.MaxDepth)
				}
				if chunk.Depth >= options.MaxDepth && !chunk.Leaf {
					t.Errorf("Chunk %d at max depth %d should be leaf", i, chunk.Depth)
				}
			}
		})
	}
}

func TestNewFieldsHierarchy(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	tests := []struct {
		name    string
		text    string
		options *types.ChunkingOptions
	}{
		{
			name: "Root node detection",
			text: "This is a test text that should create root nodes",
			options: &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          20,
				Overlap:       5,
				MaxDepth:      2,
				MaxConcurrent: 2,
			},
		},
		{
			name: "Multi-level hierarchy",
			text: strings.Repeat("This is a test line with content that will create multiple levels.\n", 10),
			options: &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          50,
				Overlap:       10,
				MaxDepth:      3,
				MaxConcurrent: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunks []*types.Chunk
			var mu sync.Mutex
			err := chunker.Chunk(ctx, tt.text, tt.options, func(chunk *types.Chunk) error {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
				return nil
			})

			if err != nil {
				t.Errorf("Chunk() error = %v", err)
				return
			}

			if len(chunks) == 0 {
				t.Error("No chunks returned")
				return
			}

			// Test Root field
			rootCount := 0
			for _, chunk := range chunks {
				if chunk.Root {
					rootCount++
					// Root nodes should have depth 1 and no parent
					if chunk.Depth != 1 {
						t.Errorf("Root chunk has depth %d, expected 1", chunk.Depth)
					}
					if chunk.ParentID != "" {
						t.Errorf("Root chunk has ParentID %s, expected empty", chunk.ParentID)
					}
					if len(chunk.Parents) != 0 {
						t.Errorf("Root chunk has %d parents, expected 0", len(chunk.Parents))
					}
				}
			}

			if rootCount == 0 {
				t.Error("No root nodes found")
			}

			// Test Index field
			parentGroups := make(map[string][]*types.Chunk)
			for _, chunk := range chunks {
				if chunk.ParentID == "" {
					parentGroups[""] = append(parentGroups[""], chunk)
				} else {
					parentGroups[chunk.ParentID] = append(parentGroups[chunk.ParentID], chunk)
				}
			}

			for parentID, siblings := range parentGroups {
				// Sort siblings by Index to ensure proper order
				sort.Slice(siblings, func(i, j int) bool {
					return siblings[i].Index < siblings[j].Index
				})

				// Verify indexes are sequential starting from 0
				for i, chunk := range siblings {
					expectedIndex := i
					if chunk.Index != expectedIndex {
						t.Errorf("Chunk with parent %s has index %d, expected %d", parentID, chunk.Index, expectedIndex)
					}
				}
			}

			// Test Status field
			for _, chunk := range chunks {
				if chunk.Status == "" {
					t.Errorf("Chunk %s has empty status", chunk.ID)
				}

				// Leaf nodes should be completed (unless there was an error)
				if chunk.Leaf && chunk.Status != types.ChunkingStatusCompleted {
					t.Errorf("Leaf chunk %s has status %s, expected %s", chunk.ID, chunk.Status, types.ChunkingStatusCompleted)
				}
			}

			// Test Parents field for non-root nodes
			for _, chunk := range chunks {
				if !chunk.Root && chunk.ParentID != "" {
					if len(chunk.Parents) == 0 {
						t.Errorf("Non-root chunk %s has no parents", chunk.ID)
					}

					// Verify parent chain consistency
					for i, parent := range chunk.Parents {
						if i == 0 && parent.Root != true {
							t.Errorf("First parent of chunk %s is not root", chunk.ID)
						}
						if i > 0 {
							prevParent := chunk.Parents[i-1]
							if parent.ParentID != prevParent.ID {
								t.Errorf("Parent chain broken for chunk %s", chunk.ID)
							}
						}
					}
				}
			}
		})
	}
}

func TestStatusUpdate(t *testing.T) {
	chunker := NewStructuredChunker()

	// Test chunk manager status updates
	chunk1 := &types.Chunk{
		ID:     "test1",
		Text:   "test",
		Depth:  1,
		Leaf:   true,
		Status: types.ChunkingStatusPending,
	}

	chunk2 := &types.Chunk{
		ID:       "test2",
		Text:     "test",
		ParentID: "test1",
		Depth:    2,
		Leaf:     true,
		Status:   types.ChunkingStatusPending,
	}

	chunker.chunkManager.AddChunk(chunk1)
	chunker.chunkManager.AddChunk(chunk2)

	// Update child status
	chunker.chunkManager.UpdateChunkStatus("test2", types.ChunkingStatusCompleted)

	if chunk2.Status != types.ChunkingStatusCompleted {
		t.Errorf("Chunk2 status = %s, expected %s", chunk2.Status, types.ChunkingStatusCompleted)
	}
}

// Benchmark tests
func BenchmarkChunkSmallText(b *testing.B) {
	chunker := NewStructuredChunker()
	ctx := context.Background()
	options := &types.ChunkingOptions{
		Type:          types.ChunkingTypeText,
		Size:          100,
		Overlap:       20,
		MaxDepth:      2,
		MaxConcurrent: 4,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.Chunk(ctx, SmallText, options, func(chunk *types.Chunk) error {
			return nil
		})
	}
}

func BenchmarkChunkLargeText(b *testing.B) {
	if _, err := os.Stat(TestDataFile); os.IsNotExist(err) {
		b.Skip("Test data file not found")
	}

	content, err := os.ReadFile(TestDataFile)
	if err != nil {
		b.Fatal(err)
	}

	chunker := NewStructuredChunker()
	ctx := context.Background()
	options := &types.ChunkingOptions{
		Type:          types.ChunkingTypeText,
		Size:          1000,
		Overlap:       100,
		MaxDepth:      3,
		MaxConcurrent: 8,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.Chunk(ctx, string(content), options, func(chunk *types.Chunk) error {
			return nil
		})
	}
}

func BenchmarkChunkCodeFile(b *testing.B) {
	if _, err := os.Stat(CodeTestDataFile); os.IsNotExist(err) {
		b.Skip("Code test data file not found")
	}

	chunker := NewStructuredChunker()
	ctx := context.Background()
	options := &types.ChunkingOptions{
		Type:          types.ChunkingTypeCode,
		Size:          800,
		Overlap:       100,
		MaxDepth:      3,
		MaxConcurrent: 4,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.ChunkFile(ctx, CodeTestDataFile, options, func(chunk *types.Chunk) error {
			return nil
		})
	}
}

func BenchmarkChunkFileStream(b *testing.B) {
	if _, err := os.Stat(TestDataFile); os.IsNotExist(err) {
		b.Skip("Test data file not found")
	}

	chunker := NewStructuredChunker()
	ctx := context.Background()
	options := &types.ChunkingOptions{
		Type:          types.ChunkingTypeText,
		Size:          1000,
		Overlap:       100,
		MaxDepth:      2,
		MaxConcurrent: 4,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.ChunkFile(ctx, TestDataFile, options, func(chunk *types.Chunk) error {
			return nil
		})
	}
}

// Memory leak tests
func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	chunker := NewStructuredChunker()
	ctx := context.Background()

	// Run multiple iterations
	for i := 0; i < 100; i++ {
		options := &types.ChunkingOptions{
			Type:          types.ChunkingTypeText,
			Size:          100,
			Overlap:       20,
			MaxDepth:      3,
			MaxConcurrent: 4,
		}

		text := strings.Repeat("This is a test line with some content.\n", 50)

		err := chunker.Chunk(ctx, text, options, func(chunk *types.Chunk) error {
			// Process chunk
			_ = chunk.Text
			_ = chunk.ID
			return nil
		})

		if err != nil {
			t.Fatalf("Chunk failed: %v", err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check for significant memory increase
	allocDiff := m2.Alloc - m1.Alloc
	if allocDiff > 1024*1024 { // 1MB threshold
		t.Errorf("Potential memory leak detected: %d bytes increase", allocDiff)
	}
}

func TestConcurrentAccess(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Run multiple goroutines concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			options := &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          50,
				Overlap:       10,
				MaxDepth:      2,
				MaxConcurrent: 2,
			}

			text := fmt.Sprintf("Goroutine %d: %s", id, SmallText)

			err := chunker.Chunk(ctx, text, options, func(chunk *types.Chunk) error {
				// Simulate some processing time
				time.Sleep(time.Millisecond)
				return nil
			})

			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestErrorHandling(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	t.Run("Callback error propagation", func(t *testing.T) {
		options := &types.ChunkingOptions{
			Type:          types.ChunkingTypeText,
			Size:          10,
			Overlap:       2,
			MaxDepth:      1,
			MaxConcurrent: 1,
		}

		expectedErr := fmt.Errorf("callback error")
		err := chunker.Chunk(ctx, SmallText, options, func(chunk *types.Chunk) error {
			return expectedErr
		})

		if err == nil {
			t.Error("Expected error from callback")
		}
	})

	t.Run("Invalid stream operations", func(t *testing.T) {
		// Test with a reader that fails
		reader := &failingReader{}
		options := &types.ChunkingOptions{
			Type:          types.ChunkingTypeText,
			Size:          10,
			Overlap:       2,
			MaxDepth:      1,
			MaxConcurrent: 1,
		}

		err := chunker.ChunkStream(ctx, reader, options, func(chunk *types.Chunk) error {
			return nil
		})

		if err == nil {
			t.Error("Expected error from failing reader")
		}
	})
}

// Helper for testing error conditions
type failingReader struct{}

func (fr *failingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func (fr *failingReader) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("seek error")
}

// Test private methods to improve coverage
func TestProcessStreamLevels(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	tests := []struct {
		name    string
		content string
		options *types.ChunkingOptions
	}{
		{
			name:    "Max depth reached",
			content: "test content",
			options: &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          5,
				Overlap:       1,
				MaxDepth:      1, // Will not recurse
				MaxConcurrent: 1,
			},
		},
		{
			name:    "Recursive chunking with code",
			content: "import { Process } from \"@yao/runtime\";\n\nexport class Excel {\n  private handle: string | null = null;\n  constructor(private file: string) {\n    this.file = file;\n  }\n  Open(writable: boolean = false) {\n    this.handle = Process('excel.Open', this.file, writable);\n    return this.handle;\n  }\n  Close() {\n    if (this.handle) {\n      Process('excel.Close', this.handle);\n      this.handle = null;\n    }\n  }\n}",
			options: &types.ChunkingOptions{
				Type:          types.ChunkingTypeCode,
				Size:          100,
				Overlap:       20,
				MaxDepth:      3, // Will recurse multiple levels
				MaxConcurrent: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			var chunks []*types.Chunk
			var mu sync.Mutex

			err := chunker.processStreamLevels(ctx, reader, 0, int64(len(tt.content)), "", 1, tt.options, func(chunk *types.Chunk) error {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
				return nil
			})

			if err != nil {
				t.Errorf("processStreamLevels() error = %v", err)
			}

			if len(chunks) == 0 {
				t.Error("No chunks returned")
			}
		})
	}
}

func TestProcessTextLevelsWithLines(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	tests := []struct {
		name          string
		text          string
		baseStartLine int
		currentDepth  int
		options       *types.ChunkingOptions
	}{
		{
			name:          "Max depth reached",
			text:          "test content",
			baseStartLine: 1,
			currentDepth:  3,
			options: &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          5,
				Overlap:       1,
				MaxDepth:      2, // Already at max depth
				MaxConcurrent: 1,
			},
		},
		{
			name:          "Recursive processing",
			text:          strings.Repeat("This is a test line with content that will be chunked recursively.\n", 20),
			baseStartLine: 10,
			currentDepth:  1,
			options: &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          50,
				Overlap:       10,
				MaxDepth:      3,
				MaxConcurrent: 2,
			},
		},
		{
			name:          "Small chunks no recursion",
			text:          "small",
			baseStartLine: 1,
			currentDepth:  1,
			options: &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          20,
				Overlap:       5,
				MaxDepth:      3,
				MaxConcurrent: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunks []*types.Chunk
			var mu sync.Mutex

			err := chunker.processTextLevelsWithLines(ctx, tt.text, tt.baseStartLine, "parent-id", tt.currentDepth, tt.options, func(chunk *types.Chunk) error {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
				return nil
			})

			if tt.currentDepth > tt.options.MaxDepth {
				// Should return nil without processing
				if err != nil {
					t.Errorf("processTextLevelsWithLines() should not error when max depth reached, got: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("processTextLevelsWithLines() error = %v", err)
			}

			// For non-max depth cases, we should get chunks
			if tt.currentDepth <= tt.options.MaxDepth && len(chunks) == 0 {
				t.Error("No chunks returned")
			}
		})
	}
}

func TestGenerateStreamChunksWithLinesEdgeCases(t *testing.T) {
	chunker := NewStructuredChunker()

	tests := []struct {
		name      string
		content   string
		chunkSize int
		overlap   int
	}{
		{
			name:      "Exact chunk size",
			content:   "12345",
			chunkSize: 5,
			overlap:   0,
		},
		{
			name:      "Content smaller than chunk",
			content:   "abc",
			chunkSize: 10,
			overlap:   2,
		},
		{
			name:      "Large overlap",
			content:   "abcdefghijklmnop",
			chunkSize: 5,
			overlap:   4,
		},
		{
			name:      "Zero overlap",
			content:   "abcdefghijklmnop",
			chunkSize: 5,
			overlap:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			options := &types.ChunkingOptions{Size: tt.chunkSize, Overlap: tt.overlap, MaxDepth: 3, MaxConcurrent: 1}
			chunks, err := chunker.generateStreamChunksWithLines(reader, 0, int64(len(tt.content)), tt.chunkSize, tt.overlap, "parent", 1, types.ChunkingTypeText, options)

			if err != nil {
				t.Errorf("generateStreamChunksWithLines() error = %v", err)
			}

			if len(chunks) == 0 {
				t.Error("No chunks returned")
			}

			// Verify chunks
			for i, chunk := range chunks {
				if chunk.TextPos == nil {
					t.Errorf("Chunk %d has nil TextPos", i)
				}
				if chunk.ParentID != "parent" {
					t.Errorf("Chunk %d has wrong ParentID: %s", i, chunk.ParentID)
				}
				if chunk.Depth != 1 {
					t.Errorf("Chunk %d has wrong Depth: %d", i, chunk.Depth)
				}
				// Test Leaf field
				expectedLeaf := 1 >= options.MaxDepth || len(chunk.Text) <= chunker.calculateSubSize(options.Size, 1)
				if chunk.Leaf != expectedLeaf {
					t.Errorf("Chunk %d Leaf = %t, expected %t", i, chunk.Leaf, expectedLeaf)
				}
			}
		})
	}
}

func TestCalculateLinesFromOffsetEdgeCases(t *testing.T) {
	chunker := NewStructuredChunker()

	tests := []struct {
		name        string
		content     string
		offset      int64
		length      int64
		expectStart int
		expectEnd   int
	}{
		{
			name:        "Zero offset and length",
			content:     "line1\nline2\n",
			offset:      0,
			length:      0,
			expectStart: 1,
			expectEnd:   1,
		},
		{
			name:        "Offset at file end",
			content:     "line1\nline2\n",
			offset:      12,
			length:      0,
			expectStart: 3,
			expectEnd:   3,
		},
		{
			name:        "Single character",
			content:     "a",
			offset:      0,
			length:      1,
			expectStart: 1,
			expectEnd:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			start, end := chunker.calculateLinesFromOffset(reader, tt.offset, tt.length)

			if start != tt.expectStart {
				t.Errorf("Start line = %d, want %d", start, tt.expectStart)
			}
			if end != tt.expectEnd {
				t.Errorf("End line = %d, want %d", end, tt.expectEnd)
			}
		})
	}
}

func TestCreateChunksWithLinesEdgeCases(t *testing.T) {
	chunker := NewStructuredChunker()

	tests := []struct {
		name     string
		text     string
		size     int
		overlap  int
		expected int
	}{
		{
			name:     "Zero overlap",
			text:     "abcdefghij",
			size:     3,
			overlap:  0,
			expected: 4, // abc, def, ghi, j
		},
		{
			name:     "Large overlap",
			text:     "abcdef",
			size:     4,
			overlap:  2,
			expected: 3, // abcd, cdef, ef (corrected)
		},
		{
			name:     "Overlap larger than text",
			text:     "abc",
			size:     5,
			overlap:  2,
			expected: 1, // Single chunk since text is smaller than size
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ChunkingOptions{Size: tt.size, Overlap: tt.overlap, MaxDepth: 3, MaxConcurrent: 1}
			chunks := chunker.createChunksWithLines(tt.text, tt.size, tt.overlap, 1, "", 1, types.ChunkingTypeText, options)

			if len(chunks) != tt.expected {
				t.Errorf("Expected %d chunks, got %d", tt.expected, len(chunks))
			}

			// Test Leaf field
			for i, chunk := range chunks {
				expectedLeaf := 1 >= options.MaxDepth || len(chunk.Text) <= chunker.calculateSubSize(options.Size, 1)
				if chunk.Leaf != expectedLeaf {
					t.Errorf("Chunk %d Leaf = %t, expected %t", i, chunk.Leaf, expectedLeaf)
				}
			}
		})
	}
}

func TestStreamErrors(t *testing.T) {
	chunker := NewStructuredChunker()

	t.Run("Stream seek error", func(t *testing.T) {
		reader := &failingSeekReader{}
		_, err := chunker.getStreamSize(reader)
		if err == nil {
			t.Error("Expected error from failing seek")
		}
	})

	t.Run("Generate chunks with failing stream", func(t *testing.T) {
		reader := &partialFailingReader{data: "test data", failAfter: 2}
		options := &types.ChunkingOptions{Size: 100, Overlap: 10, MaxDepth: 3, MaxConcurrent: 1}
		_, err := chunker.generateStreamChunksWithLines(reader, 0, 9, 5, 1, "", 1, types.ChunkingTypeText, options)
		if err == nil {
			t.Error("Expected error from failing reader")
		}
	})
}

// Additional helper structs for error testing
type failingSeekReader struct{}

func (fsr *failingSeekReader) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (fsr *failingSeekReader) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("seek error")
}

type partialFailingReader struct {
	data      string
	pos       int
	failAfter int
}

func (pfr *partialFailingReader) Read(p []byte) (n int, err error) {
	if pfr.pos >= pfr.failAfter {
		return 0, fmt.Errorf("read error after %d bytes", pfr.failAfter)
	}
	if pfr.pos >= len(pfr.data) {
		return 0, io.EOF
	}
	n = copy(p, pfr.data[pfr.pos:])
	pfr.pos += n
	return n, nil
}

func (pfr *partialFailingReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		pfr.pos = int(offset)
	case io.SeekCurrent:
		pfr.pos += int(offset)
	case io.SeekEnd:
		pfr.pos = len(pfr.data) + int(offset)
	}
	return int64(pfr.pos), nil
}

// Integration test with real file
func TestIntegrationWithTestData(t *testing.T) {
	if _, err := os.Stat(TestDataFile); os.IsNotExist(err) {
		t.Skip("Test data file not found")
	}

	chunker := NewStructuredChunker()
	ctx := context.Background()
	options := &types.ChunkingOptions{
		Size:          500,
		Overlap:       50,
		MaxDepth:      2,
		MaxConcurrent: 4,
	}

	var chunks []*types.Chunk
	var totalSize int
	var mu sync.Mutex

	err := chunker.ChunkFile(ctx, TestDataFile, options, func(chunk *types.Chunk) error {
		mu.Lock()
		chunks = append(chunks, chunk)
		totalSize += len(chunk.Text)
		mu.Unlock()
		return nil
	})

	if err != nil {
		t.Fatalf("ChunkFile() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("No chunks returned from test data file")
	}

	t.Logf("Processed %d chunks with total size %d bytes", len(chunks), totalSize)

	// Verify chunk integrity
	for i, chunk := range chunks {
		if chunk == nil {
			t.Errorf("Chunk %d is nil", i)
			continue
		}
		if chunk.ID == "" {
			t.Errorf("Chunk %d has empty ID", i)
		}
		if chunk.TextPos == nil {
			t.Errorf("Chunk %d has nil TextPos", i)
		} else {
			if chunk.TextPos.StartLine <= 0 {
				t.Errorf("Chunk %d has invalid StartLine: %d", i, chunk.TextPos.StartLine)
			}
			if chunk.TextPos.EndLine < chunk.TextPos.StartLine {
				t.Errorf("Chunk %d has EndLine < StartLine", i)
			}
		}
	}
}

func TestMaxDepthValidation(t *testing.T) {
	chunker := NewStructuredChunker()
	ctx := context.Background()

	tests := []struct {
		name          string
		originalDepth int
		expectedDepth int
		shouldWarn    bool
	}{
		{
			name:          "Valid MaxDepth",
			originalDepth: 3,
			expectedDepth: 3,
			shouldWarn:    false,
		},
		{
			name:          "MaxDepth exceeds maximum",
			originalDepth: 5,
			expectedDepth: 3,
			shouldWarn:    true,
		},
		{
			name:          "MaxDepth below minimum",
			originalDepth: 0,
			expectedDepth: 1,
			shouldWarn:    true,
		},
		{
			name:          "Negative MaxDepth",
			originalDepth: -1,
			expectedDepth: 1,
			shouldWarn:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          100,
				Overlap:       10,
				MaxDepth:      tt.originalDepth,
				MaxConcurrent: 2,
			}

			text := "This is a test text for MaxDepth validation"

			err := chunker.Chunk(ctx, text, options, func(chunk *types.Chunk) error {
				return nil
			})

			if err != nil {
				t.Errorf("Chunk() error = %v", err)
				return
			}

			if options.MaxDepth != tt.expectedDepth {
				t.Errorf("Expected MaxDepth to be corrected to %d, got %d", tt.expectedDepth, options.MaxDepth)
			}
		})
	}
}

func TestValidateAndFixOptionsDirectly(t *testing.T) {
	chunker := NewStructuredChunker()

	tests := []struct {
		name          string
		originalDepth int
		expectedDepth int
	}{
		{"Valid depth", 2, 2},
		{"Max valid depth", 3, 3},
		{"Exceeds maximum", 4, 3},
		{"Far exceeds maximum", 10, 3},
		{"Zero depth", 0, 1},
		{"Negative depth", -5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ChunkingOptions{
				MaxDepth: tt.originalDepth,
			}

			chunker.validateAndFixOptions(options)

			if options.MaxDepth != tt.expectedDepth {
				t.Errorf("Expected MaxDepth %d, got %d", tt.expectedDepth, options.MaxDepth)
			}
		})
	}
}
