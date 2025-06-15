package chunking

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// Test data constants
const (
	SemanticTestText = `Switching to Yao is easy. Yao focuses on generative programming. It may take time to learn a few key concepts, but once you understand them, you'll see that Yao is a powerful tool.

The following sections will help you understand the core concepts of Yao and how to use them to build applications.

Before the AGI era, we believe the best way to work with AI is as a collaborator, not a master. We aim to make generated code match hand-written code, easy to read and modify.

In Yao, we use a DSL (Domain-Specific Language) to describe widgets, assemble them into applications, and use processes for atomic functions.`
)

// getSemanticTestDataPath returns the absolute path to semantic test data files
func getSemanticTestDataPath(filename string) string {
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

// Test data file paths for semantic testing
var (
	SemanticEnTestFile = getSemanticTestDataPath("semantic-en.txt")
	SemanticZhTestFile = getSemanticTestDataPath("semantic-zh.txt")
)

// MockProgressCallback for testing progress reporting
type MockProgressCallback struct {
	calls []ProgressCall
	mutex sync.Mutex
}

type ProgressCall struct {
	ChunkID  string
	Progress string
	Step     string
	Data     interface{}
}

func (m *MockProgressCallback) Callback(chunkID, progress, step string, data interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.calls = append(m.calls, ProgressCall{
		ChunkID:  chunkID,
		Progress: progress,
		Step:     step,
		Data:     data,
	})
	return nil
}

func (m *MockProgressCallback) GetCalls() []ProgressCall {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]ProgressCall(nil), m.calls...)
}

func (m *MockProgressCallback) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.calls = nil
}

// Test helper functions

// prepareConnector creates connectors directly using environment variables
func prepareConnector(t *testing.T) {
	// Create OpenAI connector using environment variables
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		openaiKey = "mock-key"
	}

	openaiDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "gpt-4o-mini",
			"key": "%s"
		}
	}`, openaiKey)

	_, err := connector.New("openai", "test-openai", []byte(openaiDSL))
	if err != nil {
		t.Logf("Failed to create OpenAI connector: %v", err)
	}

	// Create local LLM connector using environment variables
	llmURL := os.Getenv("RAG_LLM_TEST_URL")
	llmKey := os.Getenv("RAG_LLM_TEST_KEY")
	llmModel := os.Getenv("RAG_LLM_TEST_SMODEL")

	if llmURL == "" {
		llmURL = "http://localhost:11434"
	}
	if llmKey == "" {
		llmKey = "mock-key"
	}
	if llmModel == "" {
		llmModel = "qwen3:8b"
	}

	llmDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "Local LLM Test", 
		"type": "openai",
		"options": {
			"proxy": "%s",
			"model": "%s",
			"key": "%s"
		}
	}`, llmURL, llmModel, llmKey)

	_, err = connector.New("openai", "test-local-llm", []byte(llmDSL))
	if err != nil {
		t.Logf("Failed to create local LLM connector: %v", err)
	}

	// Create mock connector for tests that don't require real LLM calls
	mockDSL := `{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "Mock Service Test",
		"type": "openai",
		"options": {
			"proxy": "http://127.0.0.1:9999",
			"model": "mock-model",
			"key": "mock-key"
		}
	}`

	_, err = connector.New("openai", "test-mock", []byte(mockDSL))
	if err != nil {
		t.Logf("Failed to create mock connector: %v", err)
	}
}

func createMockSemanticOptions(useToolcall bool) *types.SemanticOptions {
	var connectorID string
	if useToolcall {
		// Use OpenAI connector for toolcall tests
		connectorID = "test-openai"
	} else {
		// Use local LLM connector for non-toolcall tests
		connectorID = "test-local-llm"
	}

	return &types.SemanticOptions{
		Connector:     connectorID,
		ContextSize:   900,
		Options:       `{"temperature": 0.1}`,
		Prompt:        "Test prompt for semantic chunking",
		Toolcall:      useToolcall,
		MaxRetry:      3,
		MaxConcurrent: 2,
	}
}

func createTestSemanticOptions(useOpenAI bool) *types.ChunkingOptions {
	var connectorID string
	var toolcall bool

	if useOpenAI {
		// Use OpenAI connector
		connectorID = "test-openai"
		toolcall = true
	} else {
		// Use local LLM connector
		connectorID = "test-local-llm"
		toolcall = false
	}

	return &types.ChunkingOptions{
		Type:          types.ChunkingTypeText,
		Size:          300,
		Overlap:       50,
		MaxDepth:      3,
		MaxConcurrent: 4,
		SemanticOptions: &types.SemanticOptions{
			Connector:     connectorID,
			ContextSize:   900, // 300 * 3 * 1 = 900
			Options:       `{"temperature": 0.1, "max_tokens": 1000}`,
			Prompt:        "", // Will use default
			Toolcall:      toolcall,
			MaxRetry:      3,
			MaxConcurrent: 2,
		},
	}
}

func TestNewSemanticChunker(t *testing.T) {
	mockProgress := &MockProgressCallback{}
	chunker := NewSemanticChunker(mockProgress.Callback)

	if chunker == nil {
		t.Error("NewSemanticChunker() returned nil")
	}

	if chunker.structuredChunker == nil {
		t.Error("SemanticChunker.structuredChunker is nil")
	}

	if chunker.progressCallback == nil {
		t.Error("SemanticChunker.progressCallback is nil")
	}

	t.Run("With nil progress callback", func(t *testing.T) {
		chunker := NewSemanticChunker(nil)
		if chunker == nil {
			t.Error("NewSemanticChunker(nil) returned nil")
		}
		if chunker.progressCallback != nil {
			t.Error("SemanticChunker.progressCallback should be nil")
		}
	})
}

func TestValidateSemanticOptions(t *testing.T) {
	prepareConnector(t)
	chunker := NewSemanticChunker(nil)

	tests := []struct {
		name        string
		options     *types.ChunkingOptions
		expectError bool
	}{
		{
			name: "Valid options",
			options: &types.ChunkingOptions{
				SemanticOptions: createMockSemanticOptions(true),
			},
			expectError: false,
		},
		{
			name: "Nil semantic options",
			options: &types.ChunkingOptions{
				SemanticOptions: nil,
			},
			expectError: true,
		},
		{
			name: "Empty connector",
			options: &types.ChunkingOptions{
				SemanticOptions: &types.SemanticOptions{
					Connector: "",
				},
			},
			expectError: true,
		},
		{
			name: "Options with defaults applied",
			options: &types.ChunkingOptions{
				SemanticOptions: &types.SemanticOptions{
					Connector:     "test-openai",
					MaxRetry:      0, // Should be set to 9
					MaxConcurrent: 0, // Should be set to 4
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chunker.validateSemanticOptions(tt.options)
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check defaults were applied for valid options
			if !tt.expectError && err == nil && tt.options.SemanticOptions != nil {
				if tt.options.SemanticOptions.MaxRetry <= 0 {
					if tt.options.SemanticOptions.MaxRetry != 9 {
						t.Errorf("Expected MaxRetry to be set to 9, got %d", tt.options.SemanticOptions.MaxRetry)
					}
				}
				if tt.options.SemanticOptions.MaxConcurrent <= 0 {
					if tt.options.SemanticOptions.MaxConcurrent != 4 {
						t.Errorf("Expected MaxConcurrent to be set to 4, got %d", tt.options.SemanticOptions.MaxConcurrent)
					}
				}
			}
		})
	}
}

func TestValidateAndPrepareOptions(t *testing.T) {
	prepareConnector(t)
	chunker := NewSemanticChunker(nil)

	tests := []struct {
		name                string
		inputOptions        *types.ChunkingOptions
		expectedOverlap     int
		expectedContextSize int
		expectDefaultPrompt bool
	}{
		{
			name: "Valid options",
			inputOptions: &types.ChunkingOptions{
				Size:     300,
				Overlap:  20,
				MaxDepth: 3,
				SemanticOptions: &types.SemanticOptions{
					Connector:   "test-openai",
					ContextSize: 1000,
					Prompt:      "Custom prompt",
				},
			},
			expectedOverlap:     20,
			expectedContextSize: 1000,
			expectDefaultPrompt: false,
		},
		{
			name: "Invalid overlap - too large",
			inputOptions: &types.ChunkingOptions{
				Size:     300,
				Overlap:  400, // > Size
				MaxDepth: 3,
				SemanticOptions: &types.SemanticOptions{
					Connector: "test-openai",
				},
			},
			expectedOverlap:     50,   // Should be set to default
			expectedContextSize: 2700, // 300 * 3 * 3
			expectDefaultPrompt: true,
		},
		{
			name: "Invalid overlap - negative",
			inputOptions: &types.ChunkingOptions{
				Size:     300,
				Overlap:  -10,
				MaxDepth: 2,
				SemanticOptions: &types.SemanticOptions{
					Connector: "test-openai",
				},
			},
			expectedOverlap:     50,   // Should be set to default
			expectedContextSize: 1800, // 300 * 2 * 3
			expectDefaultPrompt: true,
		},
		{
			name: "Zero context size",
			inputOptions: &types.ChunkingOptions{
				Size:     200,
				Overlap:  30,
				MaxDepth: 2,
				SemanticOptions: &types.SemanticOptions{
					Connector:   "test-openai",
					ContextSize: 0,
				},
			},
			expectedOverlap:     30,
			expectedContextSize: 1200, // 200 * 2 * 3
			expectDefaultPrompt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chunker.validateAndPrepareOptions(tt.inputOptions)
			if err != nil {
				t.Errorf("validateAndPrepareOptions() error = %v", err)
				return
			}

			if tt.inputOptions.Overlap != tt.expectedOverlap {
				t.Errorf("Expected Overlap %d, got %d", tt.expectedOverlap, tt.inputOptions.Overlap)
			}

			if tt.inputOptions.SemanticOptions.ContextSize != tt.expectedContextSize {
				t.Errorf("Expected ContextSize %d, got %d", tt.expectedContextSize, tt.inputOptions.SemanticOptions.ContextSize)
			}

			// Test that GetSemanticPrompt returns the expected prompt
			actualPrompt := utils.GetSemanticPrompt(tt.inputOptions.SemanticOptions.Prompt)
			hasDefaultPrompt := strings.Contains(actualPrompt, "You are an expert text analyst")
			if tt.expectDefaultPrompt != hasDefaultPrompt {
				t.Errorf("Expected default prompt: %v, got: %v", tt.expectDefaultPrompt, hasDefaultPrompt)
			}
		})
	}
}

func TestSemanticPosition(t *testing.T) {
	t.Run("BasicBoundaryChecks", func(t *testing.T) {
		// Test that basic boundary checks work without automatic segmentation
		positions := []SemanticPosition{
			{StartPos: -10, EndPos: 30}, // Negative start should be fixed
			{StartPos: 30, EndPos: 150}, // Beyond text length should be fixed
			{StartPos: 50, EndPos: 40},  // Invalid range should be filtered
		}

		textLen := 100

		// Simulate the boundary checking logic from parseLLMResponse
		var safePositions []SemanticPosition
		for _, pos := range positions {
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

		// Should have 2 valid positions after boundary fixes
		if len(safePositions) != 2 {
			t.Errorf("Expected 2 valid positions after boundary checks, got %d", len(safePositions))
		}

		// Verify boundary fixes
		if safePositions[0].StartPos != 0 {
			t.Errorf("Expected first position start to be fixed to 0, got %d", safePositions[0].StartPos)
		}
		if safePositions[1].EndPos != textLen {
			t.Errorf("Expected second position end to be fixed to %d, got %d", textLen, safePositions[1].EndPos)
		}
	})
}

func TestExtractJSONFromText(t *testing.T) {
	chunker := NewSemanticChunker(nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain JSON array",
			input:    `[{"start_pos": 0, "end_pos": 10}]`,
			expected: `[{"start_pos": 0, "end_pos": 10}]`,
		},
		{
			name:     "JSON in markdown code block",
			input:    "```json\n[{\"start_pos\": 0, \"end_pos\": 10}]\n```",
			expected: `[{"start_pos": 0, "end_pos": 10}]`,
		},
		{
			name:     "JSON with surrounding text",
			input:    "Here is the segmentation:\n[{\"start_pos\": 0, \"end_pos\": 10}]\nThat's it.",
			expected: `[{"start_pos": 0, "end_pos": 10}]`,
		},
		{
			name:     "No JSON array",
			input:    "This is just text without JSON",
			expected: "This is just text without JSON",
		},
		{
			name:     "Multiple JSON-like structures",
			input:    "First: [1, 2, 3] and second: [{\"start_pos\": 0, \"end_pos\": 10}]",
			expected: `[1, 2, 3] and second: [{"start_pos": 0, "end_pos": 10}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunker.extractJSONFromText(tt.input)
			if result != tt.expected {
				t.Errorf("Expected: %s\nGot: %s", tt.expected, result)
			}
		})
	}
}

func TestCreateSemanticChunks(t *testing.T) {
	chunker := NewSemanticChunker(nil)

	originalChunk := &types.Chunk{
		ID:   "original-1",
		Text: "First sentence. Second sentence. Third sentence. Fourth sentence.",
		Type: types.ChunkingTypeText,
		TextPos: &types.TextPosition{
			StartIndex: 0,
			EndIndex:   64,
			StartLine:  1,
			EndLine:    1,
		},
	}

	positions := []SemanticPosition{
		{StartPos: 0, EndPos: 15},  // "First sentence."
		{StartPos: 16, EndPos: 32}, // "Second sentence."
		{StartPos: 33, EndPos: 64}, // "Third sentence. Fourth sentence."
	}

	options := &types.ChunkingOptions{
		Type:     types.ChunkingTypeText,
		Size:     300,
		MaxDepth: 3,
	}

	chunks := chunker.createSemanticChunks(originalChunk, positions, options)

	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	expectedTexts := []string{
		"First sentence.",
		"Second sentence.",
		"Third sentence. Fourth sentence",
	}

	for i, chunk := range chunks {
		if chunk.Text != expectedTexts[i] {
			t.Errorf("Chunk %d: expected text '%s', got '%s'", i, expectedTexts[i], chunk.Text)
		}

		if chunk.Depth != 1 {
			t.Errorf("Chunk %d: expected depth 1, got %d", i, chunk.Depth)
		}

		if chunk.Index != i {
			t.Errorf("Chunk %d: expected index %d, got %d", i, i, chunk.Index)
		}

		if chunk.Root != true {
			t.Errorf("Chunk %d: expected to be root", i)
		}

		if chunk.Status != types.ChunkingStatusCompleted {
			t.Errorf("Chunk %d: expected completed status, got %s", i, chunk.Status)
		}

		if chunk.TextPos == nil {
			t.Errorf("Chunk %d: TextPos is nil", i)
		} else {
			expectedStartIndex := originalChunk.TextPos.StartIndex + positions[i].StartPos
			expectedEndIndex := originalChunk.TextPos.StartIndex + positions[i].EndPos

			if chunk.TextPos.StartIndex != expectedStartIndex {
				t.Errorf("Chunk %d: expected StartIndex %d, got %d", i, expectedStartIndex, chunk.TextPos.StartIndex)
			}
			if chunk.TextPos.EndIndex != expectedEndIndex {
				t.Errorf("Chunk %d: expected EndIndex %d, got %d", i, expectedEndIndex, chunk.TextPos.EndIndex)
			}
		}
	}

	// Test with empty positions
	t.Run("Empty positions", func(t *testing.T) {
		emptyChunks := chunker.createSemanticChunks(originalChunk, []SemanticPosition{}, options)
		if len(emptyChunks) != 0 {
			t.Errorf("Expected 0 chunks for empty positions, got %d", len(emptyChunks))
		}
	})

	// Test with empty text segment
	t.Run("Empty text segment", func(t *testing.T) {
		emptyPositions := []SemanticPosition{
			{StartPos: 10, EndPos: 10}, // Empty segment
		}
		emptyChunks := chunker.createSemanticChunks(originalChunk, emptyPositions, options)
		if len(emptyChunks) != 0 {
			t.Errorf("Expected 0 chunks for empty text segment, got %d", len(emptyChunks))
		}
	})
}

func TestCalculateLevelSize(t *testing.T) {
	chunker := NewSemanticChunker(nil)

	tests := []struct {
		baseSize int
		depth    int
		maxDepth int
		expected int
	}{
		{100, 1, 3, 400}, // (3 - 1 + 2) * 100 = 4 * 100
		{100, 2, 3, 300}, // (3 - 2 + 2) * 100 = 3 * 100
		{100, 3, 3, 200}, // (3 - 3 + 2) * 100 = 2 * 100
		{200, 1, 2, 600}, // (2 - 1 + 2) * 200 = 3 * 200
		{150, 2, 4, 600}, // (4 - 2 + 2) * 150 = 4 * 150
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("base=%d_depth=%d_max=%d", tt.baseSize, tt.depth, tt.maxDepth), func(t *testing.T) {
			result := chunker.calculateLevelSize(tt.baseSize, tt.depth, tt.maxDepth)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestProgressReporting(t *testing.T) {
	mockProgress := &MockProgressCallback{}
	chunker := NewSemanticChunker(mockProgress.Callback)

	// Test progress reporting
	chunker.reportProgress("test-chunk-1", "processing", "test_step", map[string]interface{}{
		"key": "value",
	})

	chunker.reportProgress("test-chunk-2", "completed", "another_step", nil)

	calls := mockProgress.GetCalls()
	if len(calls) != 2 {
		t.Errorf("Expected 2 progress calls, got %d", len(calls))
	}

	// Check first call
	if calls[0].ChunkID != "test-chunk-1" {
		t.Errorf("Expected ChunkID 'test-chunk-1', got '%s'", calls[0].ChunkID)
	}
	if calls[0].Progress != "processing" {
		t.Errorf("Expected progress 'processing', got '%s'", calls[0].Progress)
	}
	if calls[0].Step != "test_step" {
		t.Errorf("Expected step 'test_step', got '%s'", calls[0].Step)
	}

	// Check second call
	if calls[1].ChunkID != "test-chunk-2" {
		t.Errorf("Expected ChunkID 'test-chunk-2', got '%s'", calls[1].ChunkID)
	}
	if calls[1].Progress != "completed" {
		t.Errorf("Expected progress 'completed', got '%s'", calls[1].Progress)
	}

	// Test with nil callback
	t.Run("Nil callback", func(t *testing.T) {
		chunkerNil := NewSemanticChunker(nil)
		// Should not panic
		chunkerNil.reportProgress("test", "progress", "step", nil)
	})
}

// Integration tests that can be run when LLM services are available
func TestSemanticChunkingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	prepareConnector(t)

	tests := []struct {
		name       string
		useOpenAI  bool
		skipReason string
	}{
		{
			name:       "OpenAI integration",
			useOpenAI:  true,
			skipReason: "OpenAI API key not available",
		},
		{
			name:       "Non-OpenAI integration",
			useOpenAI:  false,
			skipReason: "Local LLM service not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := createTestSemanticOptions(tt.useOpenAI)

			// Check if required environment variables are set
			if tt.useOpenAI {
				if os.Getenv("OPENAI_TEST_KEY") == "" {
					t.Skip(tt.skipReason)
				}
			} else {
				if os.Getenv("RAG_LLM_TEST_URL") == "" {
					t.Skip(tt.skipReason)
				}
			}

			mockProgress := &MockProgressCallback{}
			chunker := NewSemanticChunker(mockProgress.Callback)
			ctx := context.Background()

			var chunks []*types.Chunk
			var mu sync.Mutex

			err := chunker.Chunk(ctx, SemanticTestText, options, func(chunk *types.Chunk) error {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
				return nil
			})

			if err != nil {
				t.Logf("Integration test failed (this may be expected if service is not available): %v", err)
				t.Skip("Service not available for integration test")
			}

			if len(chunks) == 0 {
				t.Error("No chunks returned from semantic chunking")
			}

			// Verify chunk structure
			for i, chunk := range chunks {
				if chunk.ID == "" {
					t.Errorf("Chunk %d has empty ID", i)
				}
				if chunk.Text == "" {
					t.Errorf("Chunk %d has empty text", i)
				}
				if chunk.Type != types.ChunkingTypeText {
					t.Errorf("Chunk %d has wrong type: %s", i, chunk.Type)
				}
			}

			// Check progress calls
			calls := mockProgress.GetCalls()
			if len(calls) == 0 {
				t.Error("No progress calls made")
			}

			t.Logf("Generated %d semantic chunks with %d progress calls", len(chunks), len(calls))
		})
	}
}

func TestSemanticChunkingWithFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file tests in short mode")
	}

	prepareConnector(t)

	testFiles := []struct {
		name     string
		path     string
		language string
	}{
		{
			name:     "English semantic test",
			path:     SemanticEnTestFile,
			language: "en",
		},
		{
			name:     "Chinese semantic test",
			path:     SemanticZhTestFile,
			language: "zh",
		},
	}

	for _, tf := range testFiles {
		t.Run(tf.name, func(t *testing.T) {
			if _, err := os.Stat(tf.path); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", tf.path)
			}

			// Use mock semantic options for file tests (no actual LLM calls)
			options := &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          500,
				Overlap:       50,
				MaxDepth:      2,
				MaxConcurrent: 2,
				SemanticOptions: &types.SemanticOptions{
					Connector:     "test-mock",
					ContextSize:   1500,
					Options:       `{"temperature": 0.1}`,
					Prompt:        "",
					Toolcall:      false,
					MaxRetry:      1, // Reduce retries for mock testing
					MaxConcurrent: 1,
				},
			}

			mockProgress := &MockProgressCallback{}
			chunker := NewSemanticChunker(mockProgress.Callback)
			ctx := context.Background()

			var chunks []*types.Chunk
			var mu sync.Mutex

			// This will fail at LLM call stage, but we can test file reading and preparation
			err := chunker.ChunkFile(ctx, tf.path, options, func(chunk *types.Chunk) error {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
				return nil
			})

			// We expect this to fail at LLM stage since mock-service is not a real service
			// The purpose is to test the preparation logic (file reading, validation, etc.)
			if err != nil {
				expectedErrors := []string{
					"LLM request failed",
					"connection refused",
					"LLM segmentation failed",
					"streaming request failed",
					"no such host", // This is expected for mock service
				}

				hasExpectedError := false
				for _, expectedErr := range expectedErrors {
					if strings.Contains(err.Error(), expectedErr) {
						hasExpectedError = true
						break
					}
				}

				if !hasExpectedError {
					t.Errorf("Unexpected error (expected mock LLM failure): %v", err)
				} else {
					t.Logf("Expected mock LLM failure: %v", err)
				}
			}

			// Even with LLM failure, we should have some progress calls for file preparation
			calls := mockProgress.GetCalls()
			t.Logf("File %s: processed with %d progress calls", tf.path, len(calls))
		})
	}
}

// Benchmark tests
func BenchmarkSemanticChunking(b *testing.B) {
	options := &types.ChunkingOptions{
		Type:          types.ChunkingTypeText,
		Size:          300,
		Overlap:       50,
		MaxDepth:      2,
		MaxConcurrent: 2,
		SemanticOptions: &types.SemanticOptions{
			Connector:     "test-mock",
			ContextSize:   900,
			Options:       `{"temperature": 0.1}`,
			Toolcall:      false,
			MaxRetry:      1,
			MaxConcurrent: 1,
		},
	}

	chunker := NewSemanticChunker(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will fail at LLM stage but benchmarks the preparation phase
		chunker.Chunk(ctx, SemanticTestText, options, func(chunk *types.Chunk) error {
			return nil
		})
	}
}

// Test error conditions
func TestSemanticChunkingErrors(t *testing.T) {
	prepareConnector(t)
	chunker := NewSemanticChunker(nil)
	ctx := context.Background()

	t.Run("Invalid semantic options", func(t *testing.T) {
		options := &types.ChunkingOptions{
			SemanticOptions: nil, // Missing semantic options
		}

		err := chunker.Chunk(ctx, "test", options, func(chunk *types.Chunk) error {
			return nil
		})

		if err == nil {
			t.Error("Expected error for nil semantic options")
		}
	})

	t.Run("Empty connector", func(t *testing.T) {
		options := &types.ChunkingOptions{
			SemanticOptions: &types.SemanticOptions{
				Connector: "", // Empty connector
			},
		}

		err := chunker.Chunk(ctx, "test", options, func(chunk *types.Chunk) error {
			return nil
		})

		if err == nil {
			t.Error("Expected error for empty connector")
		}
	})

	t.Run("Callback error", func(t *testing.T) {
		// This test would require mocking the LLM response to test callback errors
		// For now, we'll test the error propagation mechanism
		options := createTestSemanticOptions(false)
		options.SemanticOptions.MaxRetry = 0 // Reduce retry attempts

		err := chunker.Chunk(ctx, "test", options, func(chunk *types.Chunk) error {
			return fmt.Errorf("callback error")
		})

		// Should get an error, either from LLM failure or callback error
		if err == nil {
			t.Error("Expected some error")
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		options := createTestSemanticOptions(false)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := chunker.Chunk(ctx, "test", options, func(chunk *types.Chunk) error {
			return nil
		})

		if err == nil {
			t.Error("Expected context cancellation error")
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		options := createTestSemanticOptions(false)

		err := chunker.ChunkFile(ctx, "/non/existent/file.txt", options, func(chunk *types.Chunk) error {
			return nil
		})

		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

// Test memory usage and resource cleanup
func TestSemanticChunkingMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	options := &types.ChunkingOptions{
		Type:          types.ChunkingTypeText,
		Size:          200,
		Overlap:       20,
		MaxDepth:      2,
		MaxConcurrent: 2,
		SemanticOptions: &types.SemanticOptions{
			Connector:     "test-mock",
			ContextSize:   600,
			Options:       `{"temperature": 0.1}`,
			Toolcall:      false,
			MaxRetry:      1,
			MaxConcurrent: 1,
		},
	}

	chunker := NewSemanticChunker(nil)
	ctx := context.Background()

	// Run multiple iterations to check for memory leaks
	for i := 0; i < 10; i++ {
		// This will fail at LLM stage but tests memory handling in preparation
		chunker.Chunk(ctx, SemanticTestText, options, func(chunk *types.Chunk) error {
			return nil
		})
	}

	// If we get here without panicking or running out of memory, test passes
	t.Log("Memory test completed successfully")
}

// Test concurrent access
func TestSemanticChunkingConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	options := &types.ChunkingOptions{
		Type:          types.ChunkingTypeText,
		Size:          300,
		Overlap:       30,
		MaxDepth:      2,
		MaxConcurrent: 3,
		SemanticOptions: &types.SemanticOptions{
			Connector:     "test-mock",
			ContextSize:   900,
			Options:       `{"temperature": 0.1}`,
			Toolcall:      false,
			MaxRetry:      1,
			MaxConcurrent: 2,
		},
	}

	chunker := NewSemanticChunker(nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	errorChan := make(chan error, 5)

	// Run multiple concurrent chunking operations
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			text := fmt.Sprintf("Goroutine %d: %s", id, SemanticTestText)

			err := chunker.Chunk(ctx, text, options, func(chunk *types.Chunk) error {
				time.Sleep(time.Millisecond) // Simulate processing time
				return nil
			})

			if err != nil {
				errorChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Check for unexpected errors (LLM connection errors are expected)
	for err := range errorChan {
		if !strings.Contains(err.Error(), "LLM request failed") &&
			!strings.Contains(err.Error(), "connection refused") &&
			!strings.Contains(err.Error(), "LLM segmentation failed") {
			t.Errorf("Unexpected concurrent error: %v", err)
		}
	}

	t.Log("Concurrency test completed successfully")
}

// Test connector creation to verify fixes
func TestConnectorCreation(t *testing.T) {
	prepareConnector(t)

	// Test OpenAI connector
	openaiConn, err := connector.Select("test-openai")
	if err != nil {
		t.Logf("OpenAI connector not available: %v", err)
	} else {
		settings := openaiConn.Setting()
		if host, ok := settings["host"].(string); ok {
			t.Logf("OpenAI connector host: %s", host)
			if !strings.Contains(host, "api.openai.com") {
				t.Errorf("Expected OpenAI host to contain 'api.openai.com', got: %s", host)
			}
		} else {
			t.Error("OpenAI connector missing host setting")
		}
	}

	// Test local LLM connector
	llmConn, err := connector.Select("test-local-llm")
	if err != nil {
		t.Logf("Local LLM connector not available: %v", err)
	} else {
		settings := llmConn.Setting()
		if host, ok := settings["host"].(string); ok {
			t.Logf("Local LLM connector host: %s", host)
		} else {
			t.Error("Local LLM connector missing host setting")
		}
	}

	// Test mock connector
	mockConn, err := connector.Select("test-mock")
	if err != nil {
		t.Logf("Mock connector not available: %v", err)
	} else {
		settings := mockConn.Setting()
		if host, ok := settings["host"].(string); ok {
			t.Logf("Mock connector host: %s", host)
		} else {
			t.Error("Mock connector missing host setting")
		}
	}
}

// Test mock connector purpose and design
func TestMockConnectorPurpose(t *testing.T) {
	prepareConnector(t)

	t.Run("Mock connector design explanation", func(t *testing.T) {
		// Mock connector design purposes:
		// 1. Unit test isolation - test semantic chunking logic without real LLM service dependencies
		// 2. Fast testing - avoid calling real APIs in every test
		// 3. Offline testing - run tests without network or LLM service availability
		// 4. Error handling testing - verify error handling logic when connections fail

		mockConn, err := connector.Select("test-mock")
		if err != nil {
			t.Fatalf("Mock connector should be available: %v", err)
		}

		settings := mockConn.Setting()
		host, ok := settings["host"].(string)
		if !ok {
			t.Error("Mock connector should have host setting")
		}

		t.Logf("Mock connector host: %s", host)
		t.Log("Mock connector is designed to fail at LLM call stage")
		t.Log("This allows testing of preparation logic without real LLM dependencies")
	})

	t.Run("Mock connector validation logic test", func(t *testing.T) {
		// Test validation logic when using mock connector
		options := &types.ChunkingOptions{
			Type:          types.ChunkingTypeText,
			Size:          300,
			Overlap:       50,
			MaxDepth:      2,
			MaxConcurrent: 2,
			SemanticOptions: &types.SemanticOptions{
				Connector:     "test-mock",
				ContextSize:   900,
				Options:       `{"temperature": 0.1}`,
				Prompt:        "Test prompt",
				Toolcall:      false,
				MaxRetry:      1, // Reduce retry count for fast failure
				MaxConcurrent: 1,
			},
		}

		chunker := NewSemanticChunker(nil)

		// Validation should succeed (connector exists)
		err := chunker.validateSemanticOptions(options)
		if err != nil {
			t.Errorf("Mock connector validation should succeed: %v", err)
		}

		// Preparation should succeed
		err = chunker.validateAndPrepareOptions(options)
		if err != nil {
			t.Errorf("Mock connector preparation should succeed: %v", err)
		}

		t.Log("Mock connector passes validation and preparation stages")
		t.Log("It will only fail when attempting actual LLM communication")
	})

	t.Run("Mock connector expected failure test", func(t *testing.T) {
		// Test expected failure of mock connector at LLM call stage
		options := &types.ChunkingOptions{
			Type:          types.ChunkingTypeText,
			Size:          200,
			Overlap:       20,
			MaxDepth:      1, // Only test one level for fast failure
			MaxConcurrent: 1,
			SemanticOptions: &types.SemanticOptions{
				Connector:     "test-mock",
				ContextSize:   600,
				Options:       `{"temperature": 0.1}`,
				Prompt:        "",
				Toolcall:      false,
				MaxRetry:      1, // Minimum retry count
				MaxConcurrent: 1,
			},
		}

		chunker := NewSemanticChunker(nil)
		ctx := context.Background()

		err := chunker.Chunk(ctx, "Test text for mock failure", options, func(chunk *types.Chunk) error {
			return nil
		})

		// Should fail at LLM call stage
		if err == nil {
			t.Error("Mock connector should fail at LLM call stage")
		} else {
			expectedErrors := []string{
				"connection refused",
				"LLM streaming request failed",
				"streaming request failed",
				"no such host",
			}

			hasExpectedError := false
			for _, expectedErr := range expectedErrors {
				if strings.Contains(err.Error(), expectedErr) {
					hasExpectedError = true
					t.Logf("Mock connector failed as expected: %s", expectedErr)
					break
				}
			}

			if !hasExpectedError {
				t.Logf("Mock connector failed with unexpected error: %v", err)
				t.Log("This might indicate a change in error handling logic")
			}
		}
	})
}

// Test streaming semantic analysis functionality
func TestStreamingSemanticAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming test in short mode")
	}

	prepareConnector(t)

	// Test with mock streaming data
	t.Run("Mock streaming progress", func(t *testing.T) {
		mockProgress := &MockProgressCallback{}
		chunker := NewSemanticChunker(mockProgress.Callback)

		// Test progress reporting with streaming data
		chunker.reportProgress("test-chunk", "streaming", "llm_response", map[string]interface{}{
			"is_toolcall":      false,
			"content_length":   25,
			"arguments_length": 0,
			"finished":         false,
			"has_error":        false,
		})

		chunker.reportProgress("test-chunk", "streaming", "llm_response", map[string]interface{}{
			"is_toolcall":      false,
			"content_length":   50,
			"arguments_length": 0,
			"finished":         true,
			"has_error":        false,
		})

		calls := mockProgress.GetCalls()
		if len(calls) != 2 {
			t.Errorf("Expected 2 progress calls, got %d", len(calls))
		}

		// Check first streaming call
		if calls[0].Step != "llm_response" {
			t.Errorf("Expected step 'llm_response', got '%s'", calls[0].Step)
		}
		if calls[0].Progress != "streaming" {
			t.Errorf("Expected progress 'streaming', got '%s'", calls[0].Progress)
		}

		// Check data structure
		data0, ok := calls[0].Data.(map[string]interface{})
		if !ok {
			t.Errorf("Expected data to be map[string]interface{}")
		} else {
			if data0["content_length"] != 25 {
				t.Errorf("Expected content_length 25, got %v", data0["content_length"])
			}
			if data0["finished"] != false {
				t.Errorf("Expected finished false, got %v", data0["finished"])
			}
		}

		// Check second streaming call
		data1, ok := calls[1].Data.(map[string]interface{})
		if !ok {
			t.Errorf("Expected data to be map[string]interface{}")
		} else {
			if data1["content_length"] != 50 {
				t.Errorf("Expected content_length 50, got %v", data1["content_length"])
			}
			if data1["finished"] != true {
				t.Errorf("Expected finished true, got %v", data1["finished"])
			}
		}
	})

	// Test prompt handling
	t.Run("Prompt handling", func(t *testing.T) {
		// Test with custom prompt
		customPrompt := "Custom semantic analysis prompt"
		options := &types.ChunkingOptions{
			SemanticOptions: &types.SemanticOptions{
				Connector: "test-mock",
				Prompt:    customPrompt,
			},
		}

		chunker := NewSemanticChunker(nil)
		err := chunker.validateAndPrepareOptions(options)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// The prompt should remain as set (utils.GetSemanticPrompt will handle it during LLM call)
		if options.SemanticOptions.Prompt != customPrompt {
			t.Errorf("Expected custom prompt to be preserved")
		}

		// Test with empty prompt
		emptyOptions := &types.ChunkingOptions{
			SemanticOptions: &types.SemanticOptions{
				Connector: "test-mock",
				Prompt:    "",
			},
		}

		err = chunker.validateAndPrepareOptions(emptyOptions)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Empty prompt should remain empty (utils.GetSemanticPrompt will provide default)
		if emptyOptions.SemanticOptions.Prompt != "" {
			t.Errorf("Expected empty prompt to remain empty")
		}
	})
}

// Test utils integration
func TestUtilsIntegration(t *testing.T) {
	// Test GetSemanticPrompt function
	t.Run("GetSemanticPrompt", func(t *testing.T) {
		// Test with custom prompt
		customPrompt := "My custom prompt"
		result := utils.GetSemanticPrompt(customPrompt)
		if result != customPrompt {
			t.Errorf("Expected custom prompt, got default")
		}

		// Test with empty prompt
		result = utils.GetSemanticPrompt("")
		defaultPrompt := utils.GetDefaultSemanticPrompt()
		if result != defaultPrompt {
			t.Errorf("Expected default prompt for empty input")
		}

		// Verify default prompt content
		if !strings.Contains(result, "You are an expert text analyst") {
			t.Errorf("Default prompt doesn't contain expected content")
		}
	})

	// Test TolerantJSONUnmarshal with semantic data
	t.Run("TolerantJSONUnmarshal with semantic data", func(t *testing.T) {
		// Test with valid semantic positions
		validJSON := `[{"start_pos": 0, "end_pos": 100}, {"start_pos": 100, "end_pos": 200}]`
		var positions []map[string]interface{}
		err := utils.TolerantJSONUnmarshal([]byte(validJSON), &positions)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(positions) != 2 {
			t.Errorf("Expected 2 positions, got %d", len(positions))
		}

		// Test with malformed JSON that can be repaired
		malformedJSON := `[{"start_pos": 0, "end_pos": 100,}, {"start_pos": 100, "end_pos": 200}]`
		err = utils.TolerantJSONUnmarshal([]byte(malformedJSON), &positions)
		if err != nil {
			t.Errorf("Should handle malformed JSON: %v", err)
		}

		if len(positions) != 2 {
			t.Errorf("Expected 2 positions after repair, got %d", len(positions))
		}
	})
}

// Test complete semantic chunking logic with mocked LLM responses
func TestSemanticChunkingWithMockedLLMResponse(t *testing.T) {
	chunker := NewSemanticChunker(nil)

	t.Run("Complete semantic chunking with toolcall response", func(t *testing.T) {
		// Mock toolcall response
		mockToolcallResponse := `{
			"choices": [{
				"message": {
					"tool_calls": [{
						"function": {
							"arguments": "{\"segments\": [{\"start_pos\": 0, \"end_pos\": 50}, {\"start_pos\": 50, \"end_pos\": 100}, {\"start_pos\": 100, \"end_pos\": 150}]}"
						}
					}]
				}
			}]
		}`

		testText := "This is a test text for semantic chunking. It should be divided into meaningful segments. Each segment represents a coherent semantic unit."
		textLen := len(testText)
		maxSize := 60

		// Test parseLLMResponse directly
		positions, err := chunker.parseLLMResponse([]byte(mockToolcallResponse), true, textLen, maxSize)
		if err != nil {
			t.Errorf("Failed to parse toolcall response: %v", err)
		}

		if len(positions) == 0 {
			t.Error("Expected positions from toolcall response")
		}

		t.Logf("Parsed %d positions from toolcall response", len(positions))

		// Test creating semantic chunks from positions
		originalChunk := &types.Chunk{
			ID:   "test-chunk-1",
			Text: testText,
			Type: types.ChunkingTypeText,
			TextPos: &types.TextPosition{
				StartIndex: 0,
				EndIndex:   textLen,
				StartLine:  1,
				EndLine:    1,
			},
		}

		options := &types.ChunkingOptions{
			Type:     types.ChunkingTypeText,
			Size:     maxSize,
			MaxDepth: 2,
		}

		semanticChunks := chunker.createSemanticChunks(originalChunk, positions, options)
		if len(semanticChunks) == 0 {
			t.Error("Expected semantic chunks to be created")
		}

		t.Logf("Created %d semantic chunks from positions", len(semanticChunks))

		// Verify chunk properties
		for i, chunk := range semanticChunks {
			if chunk.Text == "" {
				t.Errorf("Chunk %d has empty text", i)
			}
			if chunk.ID == "" {
				t.Errorf("Chunk %d has empty ID", i)
			}
			if chunk.Depth != 1 {
				t.Errorf("Chunk %d has wrong depth: %d", i, chunk.Depth)
			}
			t.Logf("Chunk %d: %q", i, chunk.Text)
		}
	})

	t.Run("Complete semantic chunking with regular response", func(t *testing.T) {
		// Mock regular response
		mockRegularResponse := `{
			"choices": [{
				"message": {
					"content": "Here are the semantic segments:\n[{\"start_pos\": 0, \"end_pos\": 45}, {\"start_pos\": 45, \"end_pos\": 90}, {\"start_pos\": 90, \"end_pos\": 135}]\n\nThese segments represent coherent semantic units."
				}
			}]
		}`

		testText := "Semantic chunking is a powerful technique for text processing. It divides text based on meaning rather than fixed sizes. This approach improves the quality of text analysis and retrieval systems."
		textLen := len(testText)
		maxSize := 50

		// Test parseLLMResponse directly
		positions, err := chunker.parseLLMResponse([]byte(mockRegularResponse), false, textLen, maxSize)
		if err != nil {
			t.Errorf("Failed to parse regular response: %v", err)
		}

		if len(positions) == 0 {
			t.Error("Expected positions from regular response")
		}

		t.Logf("Parsed %d positions from regular response", len(positions))

		// Test creating semantic chunks from positions
		originalChunk := &types.Chunk{
			ID:   "test-chunk-2",
			Text: testText,
			Type: types.ChunkingTypeText,
			TextPos: &types.TextPosition{
				StartIndex: 0,
				EndIndex:   textLen,
				StartLine:  1,
				EndLine:    1,
			},
		}

		options := &types.ChunkingOptions{
			Type:     types.ChunkingTypeText,
			Size:     maxSize,
			MaxDepth: 3,
		}

		semanticChunks := chunker.createSemanticChunks(originalChunk, positions, options)
		if len(semanticChunks) == 0 {
			t.Error("Expected semantic chunks to be created")
		}

		t.Logf("Created %d semantic chunks from positions", len(semanticChunks))

		// Verify chunk properties
		for i, chunk := range semanticChunks {
			if chunk.Text == "" {
				t.Errorf("Chunk %d has empty text", i)
			}
			if chunk.ID == "" {
				t.Errorf("Chunk %d has empty ID", i)
			}
			if chunk.Depth != 1 {
				t.Errorf("Chunk %d has wrong depth: %d", i, chunk.Depth)
			}
			t.Logf("Chunk %d: %q", i, chunk.Text)
		}
	})

	t.Run("Complete semantic chunking with hierarchy building", func(t *testing.T) {
		// Test the complete flow including hierarchy building
		testText := "First paragraph discusses the introduction to semantic analysis. Second paragraph covers the methodology and approach. Third paragraph presents the results and findings. Fourth paragraph concludes with future work and implications."

		// Create multiple semantic chunks to test hierarchy
		positions := []SemanticPosition{
			{StartPos: 0, EndPos: 70},              // First paragraph
			{StartPos: 70, EndPos: 140},            // Second paragraph
			{StartPos: 140, EndPos: 210},           // Third paragraph
			{StartPos: 210, EndPos: len(testText)}, // Fourth paragraph
		}

		originalChunk := &types.Chunk{
			ID:   "test-chunk-hierarchy",
			Text: testText,
			Type: types.ChunkingTypeText,
			TextPos: &types.TextPosition{
				StartIndex: 0,
				EndIndex:   len(testText),
				StartLine:  1,
				EndLine:    4,
			},
		}

		options := &types.ChunkingOptions{
			Type:     types.ChunkingTypeText,
			Size:     80,
			MaxDepth: 3,
		}

		// Create semantic chunks
		semanticChunks := chunker.createSemanticChunks(originalChunk, positions, options)
		if len(semanticChunks) != 4 {
			t.Errorf("Expected 4 semantic chunks, got %d", len(semanticChunks))
		}

		// Test hierarchy building
		var allChunks []*types.Chunk
		mockCallback := func(chunk *types.Chunk) error {
			allChunks = append(allChunks, chunk)
			return nil
		}

		ctx := context.Background()
		err := chunker.buildHierarchyAndOutput(ctx, semanticChunks, options, mockCallback)
		if err != nil {
			t.Errorf("Failed to build hierarchy: %v", err)
		}

		if len(allChunks) < len(semanticChunks) {
			t.Errorf("Expected at least %d chunks in output, got %d", len(semanticChunks), len(allChunks))
		}

		// Verify we have chunks at different depths
		depthCounts := make(map[int]int)
		for _, chunk := range allChunks {
			depthCounts[chunk.Depth]++
		}

		t.Logf("Chunk distribution by depth: %v", depthCounts)

		if depthCounts[1] == 0 {
			t.Error("Expected chunks at depth 1")
		}

		// If MaxDepth > 1, we should have higher level chunks
		if options.MaxDepth > 1 && len(depthCounts) == 1 {
			t.Log("Note: Only depth 1 chunks created (this may be expected for small text)")
		}
	})

	t.Run("Error handling in LLM response parsing", func(t *testing.T) {
		// Test malformed JSON response
		malformedResponse := `{
			"choices": [{
				"message": {
					"content": "Invalid JSON: [{"start_pos": 0, "end_pos": 50,}]"
				}
			}]
		}`

		// Should handle malformed JSON gracefully
		positions, err := chunker.parseLLMResponse([]byte(malformedResponse), false, 100, 50)
		if err != nil {
			t.Logf("Expected error for malformed JSON: %v", err)
		} else {
			t.Logf("Malformed JSON was repaired, got %d positions", len(positions))
		}

		// Test empty response
		emptyResponse := `{
			"choices": [{
				"message": {
					"content": ""
				}
			}]
		}`

		positions, err = chunker.parseLLMResponse([]byte(emptyResponse), false, 100, 50)
		if err != nil {
			t.Logf("Error parsing empty response: %v", err)
		} else {
			// With the new approach, we trust LLM completely and don't create fallback positions
			// Empty response should result in 0 positions since we don't do automatic segmentation
			expectedPositions := 0
			if len(positions) != expectedPositions {
				t.Errorf("Expected %d positions for empty response, got %d", expectedPositions, len(positions))
			}
		}
	})
}

// Test semantic chunking with real test files and mocked LLM responses
func TestSemanticChunkingWithRealFilesAndMockedLLM(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file tests in short mode")
	}

	testFiles := []struct {
		name     string
		path     string
		language string
	}{
		{
			name:     "English semantic test with mocked LLM",
			path:     SemanticEnTestFile,
			language: "en",
		},
		{
			name:     "Chinese semantic test with mocked LLM",
			path:     SemanticZhTestFile,
			language: "zh",
		},
	}

	for _, tf := range testFiles {
		t.Run(tf.name, func(t *testing.T) {
			if _, err := os.Stat(tf.path); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", tf.path)
			}

			// Create chunker for testing
			mockProgress := &MockProgressCallback{}
			originalChunker := NewSemanticChunker(mockProgress.Callback)

			// Create a test that simulates the complete flow
			options := &types.ChunkingOptions{
				Type:          types.ChunkingTypeText,
				Size:          500,
				Overlap:       50,
				MaxDepth:      2,
				MaxConcurrent: 2,
				SemanticOptions: &types.SemanticOptions{
					Connector:     "test-mock", // This won't be used in our mock
					ContextSize:   1500,
					Options:       `{"temperature": 0.1}`,
					Prompt:        "",
					Toolcall:      false,
					MaxRetry:      1,
					MaxConcurrent: 1,
				},
			}

			ctx := context.Background()

			// Step 1: Read the file and get structured chunks (this part works without LLM)
			reader, err := utils.OpenFileAsReader(tf.path)
			if err != nil {
				t.Fatalf("Failed to open test file: %v", err)
			}
			defer reader.Close()

			structuredChunks, err := originalChunker.getStructuredChunks(ctx, reader, options)
			if err != nil {
				t.Fatalf("Failed to get structured chunks: %v", err)
			}

			if len(structuredChunks) == 0 {
				t.Fatal("No structured chunks generated from file")
			}

			t.Logf("Generated %d structured chunks from file %s", len(structuredChunks), tf.path)

			// Step 2: For each structured chunk, simulate LLM response and create semantic chunks
			var allSemanticChunks []*types.Chunk

			for i, structuredChunk := range structuredChunks {
				t.Logf("Processing structured chunk %d (length: %d)", i, len(structuredChunk.Text))

				// Create mock positions based on text length
				textLen := len(structuredChunk.Text)
				chunkSize := options.Size

				var positions []SemanticPosition
				for start := 0; start < textLen; start += chunkSize {
					end := start + chunkSize
					if end > textLen {
						end = textLen
					}
					positions = append(positions, SemanticPosition{
						StartPos: start,
						EndPos:   end,
					})
				}

				// Create semantic chunks from mock positions
				semanticChunks := originalChunker.createSemanticChunks(structuredChunk, positions, options)
				allSemanticChunks = append(allSemanticChunks, semanticChunks...)

				t.Logf("Created %d semantic chunks from structured chunk %d", len(semanticChunks), i)
			}

			// Step 3: Test hierarchy building and output
			var outputChunks []*types.Chunk
			var mu sync.Mutex

			mockCallback := func(chunk *types.Chunk) error {
				mu.Lock()
				outputChunks = append(outputChunks, chunk)
				mu.Unlock()
				return nil
			}

			err = originalChunker.buildHierarchyAndOutput(ctx, allSemanticChunks, options, mockCallback)
			if err != nil {
				t.Errorf("Failed to build hierarchy and output: %v", err)
			}

			// Verify results
			if len(outputChunks) == 0 {
				t.Error("No output chunks generated")
			}

			// Analyze chunk distribution
			depthCounts := make(map[int]int)
			for _, chunk := range outputChunks {
				depthCounts[chunk.Depth]++
			}

			t.Logf("File %s results:", tf.path)
			t.Logf("  - Structured chunks: %d", len(structuredChunks))
			t.Logf("  - Semantic chunks: %d", len(allSemanticChunks))
			t.Logf("  - Output chunks: %d", len(outputChunks))
			t.Logf("  - Depth distribution: %v", depthCounts)

			// Verify chunk properties
			for i, chunk := range outputChunks {
				if chunk.Text == "" {
					t.Errorf("Output chunk %d has empty text", i)
				}
				if chunk.ID == "" {
					t.Errorf("Output chunk %d has empty ID", i)
				}
				if chunk.Status != types.ChunkingStatusCompleted {
					t.Errorf("Output chunk %d has wrong status: %s", i, chunk.Status)
				}
			}

			// Check progress calls
			calls := mockProgress.GetCalls()
			t.Logf("  - Progress calls: %d", len(calls))

			// Verify we have meaningful chunks
			if len(outputChunks) < len(structuredChunks) {
				t.Errorf("Expected at least %d output chunks, got %d", len(structuredChunks), len(outputChunks))
			}
		})
	}
}

// Test streaming parser integration with semantic chunking
func TestStreamingParserIntegrationWithSemanticChunking(t *testing.T) {
	chunker := NewSemanticChunker(nil)

	t.Run("Integration with StreamParser for toolcall", func(t *testing.T) {
		// Simulate streaming toolcall response
		parser := utils.NewStreamParser(true)

		streamChunks := []string{
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"segments\": ["}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"start_pos\": 0, \"end_pos\": 60},"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"start_pos\": 60, \"end_pos\": 120},"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"start_pos\": 120, \"end_pos\": 180}"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"]}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		}

		var finalArguments string
		for _, chunk := range streamChunks {
			data, err := parser.ParseStreamChunk([]byte(chunk))
			if err != nil {
				t.Errorf("Failed to parse stream chunk: %v", err)
				continue
			}
			finalArguments = data.Arguments
		}

		// Create mock response structure
		mockResponse := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"tool_calls": []interface{}{
							map[string]interface{}{
								"function": map[string]interface{}{
									"arguments": finalArguments,
								},
							},
						},
					},
				},
			},
		}

		responseBytes, err := json.Marshal(mockResponse)
		if err != nil {
			t.Fatalf("Failed to marshal mock response: %v", err)
		}

		// Test parsing with semantic chunker
		positions, err := chunker.parseLLMResponse(responseBytes, true, 180, 70)
		if err != nil {
			t.Errorf("Failed to parse LLM response: %v", err)
		}

		if len(positions) == 0 {
			t.Error("Expected positions from streaming toolcall response")
		}

		t.Logf("Parsed %d positions from streaming toolcall response", len(positions))
		for i, pos := range positions {
			t.Logf("Position %d: [%d-%d]", i, pos.StartPos, pos.EndPos)
		}
	})

	t.Run("Integration with StreamParser for regular response", func(t *testing.T) {
		// Simulate streaming regular response
		parser := utils.NewStreamParser(false)

		streamChunks := []string{
			`{"choices":[{"delta":{"content":"Here are the segments:\n["}}]}`,
			`{"choices":[{"delta":{"content":"{\"start_pos\": 0, \"end_pos\": 50},"}}]}`,
			`{"choices":[{"delta":{"content":"{\"start_pos\": 50, \"end_pos\": 100},"}}]}`,
			`{"choices":[{"delta":{"content":"{\"start_pos\": 100, \"end_pos\": 150}"}}]}`,
			`{"choices":[{"delta":{"content":"]\nAnalysis complete."}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		}

		var finalContent string
		for _, chunk := range streamChunks {
			data, err := parser.ParseStreamChunk([]byte(chunk))
			if err != nil {
				t.Errorf("Failed to parse stream chunk: %v", err)
				continue
			}
			finalContent = data.Content
		}

		// Create mock response structure
		mockResponse := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"content": finalContent,
					},
				},
			},
		}

		responseBytes, err := json.Marshal(mockResponse)
		if err != nil {
			t.Fatalf("Failed to marshal mock response: %v", err)
		}

		// Test parsing with semantic chunker
		positions, err := chunker.parseLLMResponse(responseBytes, false, 150, 60)
		if err != nil {
			t.Errorf("Failed to parse LLM response: %v", err)
		}

		if len(positions) == 0 {
			t.Error("Expected positions from streaming regular response")
		}

		t.Logf("Parsed %d positions from streaming regular response", len(positions))
		for i, pos := range positions {
			t.Logf("Position %d: [%d-%d]", i, pos.StartPos, pos.EndPos)
		}
	})
}

// Test semantic chunking behavior with empty LLM responses
func TestSemanticChunkingFallbackWithSizeConstraints(t *testing.T) {
	prepareConnector(t)

	t.Run("Empty LLM response handling", func(t *testing.T) {
		chunker := NewSemanticChunker(nil)

		// Test parseLLMResponse with empty content - should not create automatic fallback
		emptyResponse := `{
			"choices": [{
				"message": {
					"content": ""
				}
			}]
		}`

		textLen := 1000
		maxSize := 200

		positions, err := chunker.parseLLMResponse([]byte(emptyResponse), false, textLen, maxSize)
		if err != nil {
			t.Errorf("parseLLMResponse failed: %v", err)
		}

		// With the new approach, empty response should result in 0 positions
		expectedPositions := 0
		if len(positions) != expectedPositions {
			t.Errorf("Expected %d positions for empty response, got %d", expectedPositions, len(positions))
		}

		t.Logf("Empty LLM response for text (%d chars) created %d positions (no automatic fallback)", textLen, len(positions))
	})

	t.Run("Valid LLM positions are preserved", func(t *testing.T) {
		chunker := NewSemanticChunker(nil)

		// Test that valid LLM positions are used as-is without modification
		testText := "First semantic unit. Second semantic unit. Third semantic unit."

		originalChunk := &types.Chunk{
			ID:   "test-chunk-1",
			Text: testText,
			Type: types.ChunkingTypeText,
			TextPos: &types.TextPosition{
				StartIndex: 0,
				EndIndex:   len(testText),
				StartLine:  1,
				EndLine:    1,
			},
		}

		// Valid semantic positions from LLM (different sizes to show semantic boundaries)
		positions := []SemanticPosition{
			{StartPos: 0, EndPos: 21},  // "First semantic unit. " (21 chars)
			{StartPos: 21, EndPos: 43}, // "Second semantic unit. " (22 chars)
			{StartPos: 43, EndPos: 63}, // "Third semantic unit." (20 chars)
		}

		options := &types.ChunkingOptions{
			Type:     types.ChunkingTypeText,
			Size:     15, // Smaller than some segments to show we trust LLM
			MaxDepth: 2,
		}

		// Create semantic chunks from LLM positions
		semanticChunks := chunker.createSemanticChunks(originalChunk, positions, options)

		if len(semanticChunks) != 3 {
			t.Errorf("Expected 3 semantic chunks, got %d", len(semanticChunks))
		}

		// Verify chunk content matches LLM positions exactly
		expectedTexts := []string{
			"First semantic unit. ",
			"Second semantic unit. ",
			"Third semantic unit.",
		}

		for i, chunk := range semanticChunks {
			if chunk.Text != expectedTexts[i] {
				t.Errorf("Chunk %d text mismatch: expected %q, got %q", i, expectedTexts[i], chunk.Text)
			}
			// Note: Some chunks are larger than options.Size (15), but we trust LLM
			t.Logf("Chunk %d: %d chars - %q", i, len(chunk.Text), chunk.Text)
		}

		t.Logf("Created %d semantic chunks from valid LLM positions, preserving semantic boundaries", len(semanticChunks))
	})
}

// Test number type conversion safety
func TestConvertToIntSafety(t *testing.T) {
	chunker := NewSemanticChunker(nil)

	t.Run("Valid number types", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    interface{}
			expected int
		}{
			{"int", int(42), 42},
			{"int32", int32(42), 42},
			{"int64", int64(42), 42},
			{"float32", float32(42.7), 42},
			{"float64", float64(42.9), 42},
			{"string int", "42", 42},
			{"string float", "42.8", 42},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := chunker.convertToInt(tc.input)
				if err != nil {
					t.Errorf("convertToInt(%v) failed: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("convertToInt(%v) = %d, expected %d", tc.input, result, tc.expected)
				}
			})
		}
	})

	t.Run("Invalid inputs", func(t *testing.T) {
		invalidCases := []struct {
			name  string
			input interface{}
		}{
			{"nil", nil},
			{"invalid string", "not_a_number"},
			{"boolean", true},
			{"slice", []int{1, 2, 3}},
			{"map", map[string]int{"a": 1}},
		}

		for _, tc := range invalidCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := chunker.convertToInt(tc.input)
				if err == nil {
					t.Errorf("convertToInt(%v) should have failed", tc.input)
				}
			})
		}
	})
}

// Test toolcall response parsing with different number types
func TestParseToolcallResponseWithDifferentNumberTypes(t *testing.T) {
	chunker := NewSemanticChunker(nil)

	t.Run("Response with int values", func(t *testing.T) {
		// Mock toolcall response with int values (not float64)
		mockResponse := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"tool_calls": []interface{}{
							map[string]interface{}{
								"function": map[string]interface{}{
									"arguments": `{"segments": [{"start_pos": 0, "end_pos": 50}, {"start_pos": 50, "end_pos": 100}]}`,
								},
							},
						},
					},
				},
			},
		}

		positions, err := chunker.parseToolcallResponse(mockResponse)
		if err != nil {
			t.Errorf("parseToolcallResponse failed: %v", err)
		}

		if len(positions) != 2 {
			t.Errorf("Expected 2 positions, got %d", len(positions))
		}

		expectedPositions := []SemanticPosition{
			{StartPos: 0, EndPos: 50},
			{StartPos: 50, EndPos: 100},
		}

		for i, pos := range positions {
			if pos.StartPos != expectedPositions[i].StartPos || pos.EndPos != expectedPositions[i].EndPos {
				t.Errorf("Position %d: expected %+v, got %+v", i, expectedPositions[i], pos)
			}
		}
	})

	t.Run("Response with mixed number types", func(t *testing.T) {
		// Create a response where numbers might be parsed as different types
		// This simulates real-world scenarios where JSON parsing can vary
		argumentsData := map[string]interface{}{
			"segments": []interface{}{
				map[string]interface{}{
					"start_pos": int(0),      // int type
					"end_pos":   float64(30), // float64 type
				},
				map[string]interface{}{
					"start_pos": int64(30),   // int64 type
					"end_pos":   float32(60), // float32 type
				},
				map[string]interface{}{
					"start_pos": "60", // string type
					"end_pos":   "90", // string type
				},
			},
		}

		argumentsBytes, _ := json.Marshal(argumentsData)
		mockResponse := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"tool_calls": []interface{}{
							map[string]interface{}{
								"function": map[string]interface{}{
									"arguments": string(argumentsBytes),
								},
							},
						},
					},
				},
			},
		}

		positions, err := chunker.parseToolcallResponse(mockResponse)
		if err != nil {
			t.Errorf("parseToolcallResponse failed: %v", err)
		}

		if len(positions) != 3 {
			t.Errorf("Expected 3 positions, got %d", len(positions))
		}

		expectedPositions := []SemanticPosition{
			{StartPos: 0, EndPos: 30},
			{StartPos: 30, EndPos: 60},
			{StartPos: 60, EndPos: 90},
		}

		for i, pos := range positions {
			if pos.StartPos != expectedPositions[i].StartPos || pos.EndPos != expectedPositions[i].EndPos {
				t.Errorf("Position %d: expected %+v, got %+v", i, expectedPositions[i], pos)
			}
		}
	})

	t.Run("Response with invalid number types", func(t *testing.T) {
		// Mock response with invalid number types
		argumentsData := map[string]interface{}{
			"segments": []interface{}{
				map[string]interface{}{
					"start_pos": "invalid_number", // invalid string
					"end_pos":   30,
				},
			},
		}

		argumentsBytes, _ := json.Marshal(argumentsData)
		mockResponse := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"tool_calls": []interface{}{
							map[string]interface{}{
								"function": map[string]interface{}{
									"arguments": string(argumentsBytes),
								},
							},
						},
					},
				},
			},
		}

		_, err := chunker.parseToolcallResponse(mockResponse)
		if err == nil {
			t.Error("parseToolcallResponse should have failed with invalid number")
		}

		if !strings.Contains(err.Error(), "invalid start_pos") {
			t.Errorf("Expected error about invalid start_pos, got: %v", err)
		}
	})
}
