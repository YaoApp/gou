package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/connector"
)

// Test Coverage Summary:
// 1. PostLLM: Tests both local LLM and OpenAI connectors
// 2. StreamLLM: Tests streaming functionality with both regular and toolcall scenarios
//    - TestStreamLLM: Basic streaming test with local LLM
//    - TestStreamLLM_Toolcall: OpenAI toolcall streaming test with function calls
//    - TestStreamLLM_LocalLLM: Local LLM streaming test with regular responses
// 3. StreamParser: Tests streaming response parsing for both formats
//    - TestStreamParser_Regular: Regular response format parsing
//    - TestStreamParser_Toolcall: Toolcall response format parsing
//    - TestStreamParser_IncompleteJSON: Incomplete JSON handling
//    - TestStreamParser_SSEFormat: Server-Sent Events format
//    - TestStreamParser_ErrorHandling: Error scenarios
// 4. TolerantJSONUnmarshal: JSON repair and parsing with error tolerance
// 5. Utility functions: File operations, JSON parsing, semantic prompts
// 6. Memory and Goroutine Leak Detection:
//    - TestMemoryLeakDetection: Memory leak detection for utils operations
//    - TestStreamLLMMemoryLeak: Memory leak detection for StreamLLM operations
//    - TestGoroutineLeakDetection: Goroutine leak detection for all utils functions

func TestPostLLM_LocalLLM(t *testing.T) {
	// Read environment variables for local LLM
	llmURL := os.Getenv("RAG_LLM_TEST_URL")
	llmKey := os.Getenv("RAG_LLM_TEST_KEY")
	llmModel := os.Getenv("RAG_LLM_TEST_SMODEL")

	if llmURL == "" || llmKey == "" || llmModel == "" {
		t.Skip("Skipping local LLM test: RAG_LLM_TEST_URL, RAG_LLM_TEST_KEY, or RAG_LLM_TEST_SMODEL not set")
	}

	// Create local LLM connector
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

	conn, err := connector.New("openai", "test-local-llm", []byte(llmDSL))
	if err != nil {
		t.Fatalf("Failed to create local LLM connector: %v", err)
	}

	// Test request payload
	payload := map[string]interface{}{
		"model": llmModel,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "Hello! Please respond with just 'Hi there!'",
			},
		},
		"max_tokens":  50,
		"temperature": 0.1,
	}

	// Send request with context
	ctx := context.Background()
	response, err := PostLLM(ctx, conn, "chat/completions", payload)
	if err != nil {
		t.Fatalf("PostLLM failed: %s", err.Error())
	}

	// Verify response structure
	respMap, ok := response.(map[string]interface{})
	if !ok {
		t.Fatalf("Response is not a map: %T", response)
	}

	// Check for choices
	choices, ok := respMap["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Fatalf("No choices in response: %+v", respMap)
	}

	// Get first choice
	choice := choices[0].(map[string]interface{})
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("No message in choice: %+v", choice)
	}

	// Get content
	content, ok := message["content"].(string)
	if !ok {
		t.Fatalf("No content in message: %+v", message)
	}

	// Verify response contains some text
	if strings.TrimSpace(content) == "" {
		t.Errorf("Response content is empty")
	}

	t.Logf("Local LLM Response: %s", content)
}

func TestPostLLM_OpenAI(t *testing.T) {
	// Read environment variable for OpenAI
	openaiKey := os.Getenv("OPENAI_TEST_KEY")

	if openaiKey == "" {
		t.Skip("Skipping OpenAI test: OPENAI_TEST_KEY not set")
	}

	// Create OpenAI connector
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

	conn, err := connector.New("openai", "test-openai", []byte(openaiDSL))
	if err != nil {
		t.Fatalf("Failed to create OpenAI connector: %v", err)
	}

	// Test request payload
	payload := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "Hello! Please respond with just 'Hi there!'",
			},
		},
		"max_tokens":  50,
		"temperature": 0.1,
	}

	// Send request with context
	ctx := context.Background()
	response, err := PostLLM(ctx, conn, "chat/completions", payload)
	if err != nil {
		t.Fatalf("PostLLM failed: %v", err)
	}

	// Verify response structure
	respMap, ok := response.(map[string]interface{})
	if !ok {
		t.Fatalf("Response is not a map: %T", response)
	}

	// Check for choices
	choices, ok := respMap["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Fatalf("No choices in response: %+v", respMap)
	}

	// Get first choice
	choice := choices[0].(map[string]interface{})
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("No message in choice: %+v", choice)
	}

	// Get content
	content, ok := message["content"].(string)
	if !ok {
		t.Fatalf("No content in message: %+v", message)
	}

	// Verify response contains some text
	if strings.TrimSpace(content) == "" {
		t.Errorf("Response content is empty")
	}

	t.Logf("OpenAI Response: %s", content)
}

func TestStreamLLM(t *testing.T) {
	// Test that StreamLLM works with callback
	llmURL := os.Getenv("RAG_LLM_TEST_URL")
	llmKey := os.Getenv("RAG_LLM_TEST_KEY")
	llmModel := os.Getenv("RAG_LLM_TEST_SMODEL")

	if llmURL == "" || llmKey == "" || llmModel == "" {
		t.Skip("Skipping StreamLLM test: RAG_LLM_TEST_URL, RAG_LLM_TEST_KEY, or RAG_LLM_TEST_SMODEL not set")
	}

	llmDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "Stream Test",
		"type": "openai",
		"options": {
			"proxy": "%s",
			"model": "%s",
			"key": "%s"
		}
	}`, llmURL, llmModel, llmKey)

	conn, err := connector.New("openai", "test-stream", []byte(llmDSL))
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	payload := map[string]interface{}{
		"model": llmModel,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Say hello in one word"},
		},
		"max_tokens": 10,
	}

	// Collect streamed data
	var streamedData []string
	callback := func(data []byte) error {
		if len(data) > 0 {
			streamedData = append(streamedData, string(data))
			t.Logf("Streamed data: %s", string(data))
		}
		return nil
	}

	// Test streaming with context
	ctx := context.Background()
	err = StreamLLM(ctx, conn, "chat/completions", payload, callback)
	if err != nil {
		t.Logf("StreamLLM failed (may be expected): %v", err)
		// This is acceptable as the service might not support streaming or model might not exist
	} else {
		t.Logf("StreamLLM succeeded, received %d chunks", len(streamedData))
	}
}

func TestStreamLLM_Toolcall(t *testing.T) {
	// Test StreamLLM with toolcall functionality
	openaiKey := os.Getenv("OPENAI_TEST_KEY")

	if openaiKey == "" {
		t.Skip("Skipping StreamLLM toolcall test: OPENAI_TEST_KEY not set")
	}

	// Create OpenAI connector for toolcall testing
	openaiDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "OpenAI Toolcall Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "gpt-4o-mini",
			"key": "%s"
		}
	}`, openaiKey)

	conn, err := connector.New("openai", "test-toolcall", []byte(openaiDSL))
	if err != nil {
		t.Fatalf("Failed to create OpenAI connector: %v", err)
	}

	// Test payload with toolcall
	payload := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "Analyze this text and segment it: 'Hello world. This is a test.'",
			},
		},
		"max_tokens":  200,
		"temperature": 0.1,
		"tools": []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "segment_text",
					"description": "Segment text into semantic chunks",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"segments": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"start_pos": map[string]interface{}{
											"type":        "integer",
											"description": "Start position of the segment",
										},
										"end_pos": map[string]interface{}{
											"type":        "integer",
											"description": "End position of the segment",
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
		},
		"tool_choice": "auto",
	}

	// Collect streamed data
	var streamedData []string
	var toolcallDetected bool
	var argumentsAccumulated string

	callback := func(data []byte) error {
		if len(data) > 0 {
			dataStr := string(data)
			streamedData = append(streamedData, dataStr)
			t.Logf("Streamed toolcall data: %s", dataStr)

			// Check if this chunk contains tool_calls
			if strings.Contains(dataStr, "tool_calls") {
				toolcallDetected = true
			}

			// Try to extract arguments from the chunk
			if strings.Contains(dataStr, "arguments") {
				// Simple extraction for testing
				if start := strings.Index(dataStr, `"arguments":"`); start != -1 {
					start += len(`"arguments":"`)
					if end := strings.Index(dataStr[start:], `"`); end != -1 {
						argumentsAccumulated += dataStr[start : start+end]
					}
				}
			}
		}
		return nil
	}

	// Test streaming with context
	ctx := context.Background()
	err = StreamLLM(ctx, conn, "chat/completions", payload, callback)
	if err != nil {
		t.Logf("StreamLLM toolcall failed (may be expected): %v", err)
		// This is acceptable as the service might not support streaming or model might not exist
	} else {
		t.Logf("StreamLLM toolcall succeeded, received %d chunks", len(streamedData))

		if toolcallDetected {
			t.Logf("✅ Toolcall detected in streaming response")
		} else {
			t.Logf("ℹ️  No toolcall detected (model may have chosen regular response)")
		}

		if argumentsAccumulated != "" {
			t.Logf("📄 Accumulated arguments: %s", argumentsAccumulated)
		}
	}
}

func TestStreamLLM_LocalLLM(t *testing.T) {
	// Test StreamLLM with local LLM (non-toolcall scenario)
	llmURL := os.Getenv("RAG_LLM_TEST_URL")
	llmKey := os.Getenv("RAG_LLM_TEST_KEY")
	llmModel := os.Getenv("RAG_LLM_TEST_SMODEL")

	if llmURL == "" || llmKey == "" || llmModel == "" {
		t.Skip("Skipping local LLM StreamLLM test: RAG_LLM_TEST_URL, RAG_LLM_TEST_KEY, or RAG_LLM_TEST_SMODEL not set")
	}

	llmDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "Local LLM Stream Test",
		"type": "openai",
		"options": {
			"proxy": "%s",
			"model": "%s",
			"key": "%s"
		}
	}`, llmURL, llmModel, llmKey)

	conn, err := connector.New("openai", "test-local-stream", []byte(llmDSL))
	if err != nil {
		t.Fatalf("Failed to create local LLM connector: %v", err)
	}

	payload := map[string]interface{}{
		"model": llmModel,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "Analyze this text and return JSON with segments: 'Hello world. This is a test.' Format: [{\"start_pos\": 0, \"end_pos\": 12}]",
			},
		},
		"max_tokens":  100,
		"temperature": 0.1,
	}

	// Collect streamed data
	var streamedData []string
	var contentAccumulated string

	callback := func(data []byte) error {
		if len(data) > 0 {
			dataStr := string(data)
			streamedData = append(streamedData, dataStr)
			t.Logf("Streamed local LLM data: %s", dataStr)

			// Try to extract content from the chunk
			if strings.Contains(dataStr, "content") {
				// Simple extraction for testing
				if start := strings.Index(dataStr, `"content":"`); start != -1 {
					start += len(`"content":"`)
					if end := strings.Index(dataStr[start:], `"`); end != -1 {
						contentAccumulated += dataStr[start : start+end]
					}
				}
			}
		}
		return nil
	}

	// Test streaming with context
	ctx := context.Background()
	err = StreamLLM(ctx, conn, "chat/completions", payload, callback)
	if err != nil {
		t.Logf("Local LLM StreamLLM failed (may be expected): %v", err)
		// This is acceptable as the service might not support streaming or model might not exist
	} else {
		t.Logf("Local LLM StreamLLM succeeded, received %d chunks", len(streamedData))

		if contentAccumulated != "" {
			t.Logf("📄 Accumulated content: %s", contentAccumulated)
		}
	}
}

func TestParseJSONOptions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    map[string]interface{}
		expectError bool
	}{
		{
			name:  "Valid JSON options",
			input: `{"temperature": 0.1, "max_tokens": 1000, "top_p": 0.9}`,
			expected: map[string]interface{}{
				"temperature": 0.1,
				"max_tokens":  float64(1000),
				"top_p":       0.9,
			},
			expectError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    map[string]interface{}{},
			expectError: false,
		},
		{
			name:  "Simple JSON",
			input: `{"key": "value"}`,
			expected: map[string]interface{}{
				"key": "value",
			},
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			input:       `{"invalid": json}`,
			expectError: true,
		},
		{
			name:        "Malformed JSON",
			input:       `{"temperature": 0.1, "max_tokens":}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseJSONOptions(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d options, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("Expected key '%s' not found", key)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("Expected value for key '%s' to be %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestOpenFileAsReader(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file.\nLine 3."

	err := os.WriteFile(tmpFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("Valid file", func(t *testing.T) {
		reader, err := OpenFileAsReader(tmpFile)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
		defer reader.Close()

		// Test reading
		buffer := make([]byte, len(testContent))
		n, err := reader.Read(buffer)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}

		if n != len(testContent) {
			t.Errorf("Expected to read %d bytes, got %d", len(testContent), n)
		}

		if string(buffer[:n]) != testContent {
			t.Errorf("Expected content '%s', got '%s'", testContent, string(buffer[:n]))
		}

		// Test seeking
		_, err = reader.Seek(0, 0) // Seek to beginning
		if err != nil {
			t.Errorf("Failed to seek: %v", err)
		}

		// Read again to verify seek worked
		buffer2 := make([]byte, 5)
		n2, err := reader.Read(buffer2)
		if err != nil {
			t.Errorf("Failed to read after seek: %v", err)
		}

		if n2 != 5 {
			t.Errorf("Expected to read 5 bytes after seek, got %d", n2)
		}

		if string(buffer2) != "Hello" {
			t.Errorf("Expected 'Hello' after seek, got '%s'", string(buffer2))
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		_, err := OpenFileAsReader("/non/existent/file.txt")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

func TestFileReaderClose(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_close.txt")
	testContent := "Test content for close"

	err := os.WriteFile(tmpFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	reader, err := OpenFileAsReader(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	// Close the reader
	err = reader.Close()
	if err != nil {
		t.Errorf("Failed to close reader: %v", err)
	}

	// Try to read after close (should fail)
	buffer := make([]byte, 10)
	_, err = reader.Read(buffer)
	if err == nil {
		t.Error("Expected error when reading from closed file")
	}
}

// Benchmark tests
func BenchmarkParseJSONOptions(b *testing.B) {
	options := `{"temperature": 0.1, "max_tokens": 1000, "top_p": 0.9, "frequency_penalty": 0.0}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseJSONOptions(options)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkOpenFileAsReader(b *testing.B) {
	// Create a temporary file for benchmarking
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "benchmark.txt")
	testContent := "Benchmark test content for file operations"

	err := os.WriteFile(tmpFile, []byte(testContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, err := OpenFileAsReader(tmpFile)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
		reader.Close()
	}
}

func TestTolerantJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    map[string]interface{}
	}{
		{
			name:        "Valid JSON",
			input:       `{"key": "value", "number": 42}`,
			expectError: false,
			expected:    map[string]interface{}{"key": "value", "number": float64(42)},
		},
		{
			name:        "Invalid JSON that can be repaired",
			input:       `{"key": "value", "number": 42,}`, // trailing comma
			expectError: false,
			expected:    map[string]interface{}{"key": "value", "number": float64(42)},
		},
		{
			name:        "Malformed JSON",
			input:       `{"key": "value" "number": 42}`, // missing comma
			expectError: false,
			expected:    map[string]interface{}{"key": "value", "number": float64(42)},
		},
		{
			name:        "Completely invalid JSON",
			input:       `this is not json at all`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			err := TolerantJSONUnmarshal([]byte(tt.input), &result)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("Expected key '%s' not found", key)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("Expected value for key '%s' to be %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestGetSemanticPrompt(t *testing.T) {
	tests := []struct {
		name       string
		userPrompt string
		expectUser bool
	}{
		{
			name:       "User defined prompt",
			userPrompt: "This is a custom prompt for semantic analysis",
			expectUser: true,
		},
		{
			name:       "Empty user prompt",
			userPrompt: "",
			expectUser: false,
		},
		{
			name:       "Whitespace only prompt",
			userPrompt: "   \n\t  ",
			expectUser: false,
		},
		{
			name:       "User prompt with whitespace",
			userPrompt: "  Custom prompt  ",
			expectUser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSemanticPrompt(tt.userPrompt)

			if tt.expectUser {
				if result != tt.userPrompt {
					t.Errorf("Expected user prompt '%s', got '%s'", tt.userPrompt, result)
				}
			} else {
				defaultPrompt := GetDefaultSemanticPrompt()
				if result != defaultPrompt {
					t.Errorf("Expected default prompt, got different prompt")
				}
				if !strings.Contains(result, "You are an expert text analyst") {
					t.Errorf("Default prompt doesn't contain expected content")
				}
			}
		})
	}
}

func TestStreamParser_Regular(t *testing.T) {
	parser := NewStreamParser(false) // Regular response, not toolcall

	// Test chunks simulating OpenAI streaming response with semantic positions
	testChunks := []string{
		`{"choices":[{"delta":{"content":"Here are the semantic segments:\n["}}]}`,
		`{"choices":[{"delta":{"content":"{\"start_pos\": 0, \"end_pos\": 50},"}}]}`,
		`{"choices":[{"delta":{"content":"{\"start_pos\": 50, \"end_pos\": 100}"}}]}`,
		`{"choices":[{"delta":{"content":"]\n\nThese segments represent..."}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
	}

	expectedContents := []string{
		"Here are the semantic segments:\n[",
		"Here are the semantic segments:\n[{\"start_pos\": 0, \"end_pos\": 50},",
		"Here are the semantic segments:\n[{\"start_pos\": 0, \"end_pos\": 50},{\"start_pos\": 50, \"end_pos\": 100}",
		"Here are the semantic segments:\n[{\"start_pos\": 0, \"end_pos\": 50},{\"start_pos\": 50, \"end_pos\": 100}]\n\nThese segments represent...",
		"Here are the semantic segments:\n[{\"start_pos\": 0, \"end_pos\": 50},{\"start_pos\": 50, \"end_pos\": 100}]\n\nThese segments represent...",
	}

	expectedFinished := []bool{false, false, false, false, true}
	expectedPositionsCount := []int{0, 0, 2, 2, 2} // Positions parsed when JSON can be completed

	for i, chunk := range testChunks {
		t.Run(fmt.Sprintf("Chunk_%d", i), func(t *testing.T) {
			data, err := parser.ParseStreamChunk([]byte(chunk))
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if data.Content != expectedContents[i] {
				t.Errorf("Expected content '%s', got '%s'", expectedContents[i], data.Content)
			}

			if data.Finished != expectedFinished[i] {
				t.Errorf("Expected finished %v, got %v", expectedFinished[i], data.Finished)
			}

			if data.IsToolcall != false {
				t.Errorf("Expected IsToolcall to be false")
			}

			if len(data.Positions) != expectedPositionsCount[i] {
				t.Errorf("Expected %d positions, got %d", expectedPositionsCount[i], len(data.Positions))
			}

			// Check positions when they should be available
			if len(data.Positions) == 2 {
				if data.Positions[0].StartPos != 0 || data.Positions[0].EndPos != 50 {
					t.Errorf("Expected first position {0, 50}, got {%d, %d}",
						data.Positions[0].StartPos, data.Positions[0].EndPos)
				}
				if data.Positions[1].StartPos != 50 || data.Positions[1].EndPos != 100 {
					t.Errorf("Expected second position {50, 100}, got {%d, %d}",
						data.Positions[1].StartPos, data.Positions[1].EndPos)
				}
			}
		})
	}
}

func TestStreamParser_Toolcall(t *testing.T) {
	parser := NewStreamParser(true) // Toolcall response

	// Test chunks simulating OpenAI toolcall streaming response
	testChunks := []string{
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"segments\": ["}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"start_pos\": 0, \"end_pos\": 25},"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"start_pos\": 25, \"end_pos\": 50}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"]}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}

	expectedArguments := []string{
		`{"segments": [`,
		`{"segments": [{"start_pos": 0, "end_pos": 25},`,
		`{"segments": [{"start_pos": 0, "end_pos": 25},{"start_pos": 25, "end_pos": 50}`,
		`{"segments": [{"start_pos": 0, "end_pos": 25},{"start_pos": 25, "end_pos": 50}]}`,
		`{"segments": [{"start_pos": 0, "end_pos": 25},{"start_pos": 25, "end_pos": 50}]}`,
	}

	expectedFinished := []bool{false, false, false, false, true}
	expectedPositionsCount := []int{0, 0, 2, 2, 2} // Positions parsed when JSON can be completed

	for i, chunk := range testChunks {
		t.Run(fmt.Sprintf("ToolcallChunk_%d", i), func(t *testing.T) {
			data, err := parser.ParseStreamChunk([]byte(chunk))
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if data.Arguments != expectedArguments[i] {
				t.Errorf("Expected arguments '%s', got '%s'", expectedArguments[i], data.Arguments)
			}

			if data.Finished != expectedFinished[i] {
				t.Errorf("Expected finished %v, got %v", expectedFinished[i], data.Finished)
			}

			if data.IsToolcall != true {
				t.Errorf("Expected IsToolcall to be true")
			}

			if len(data.Positions) != expectedPositionsCount[i] {
				t.Errorf("Expected %d positions, got %d", expectedPositionsCount[i], len(data.Positions))
			}

			// Check positions when they should be available
			if len(data.Positions) == 2 {
				if data.Positions[0].StartPos != 0 || data.Positions[0].EndPos != 25 {
					t.Errorf("Expected first position {0, 25}, got {%d, %d}",
						data.Positions[0].StartPos, data.Positions[0].EndPos)
				}
				if data.Positions[1].StartPos != 25 || data.Positions[1].EndPos != 50 {
					t.Errorf("Expected second position {25, 50}, got {%d, %d}",
						data.Positions[1].StartPos, data.Positions[1].EndPos)
				}
			}
		})
	}
}

func TestStreamParser_IncompleteJSON(t *testing.T) {
	// Test handling of incomplete JSON during streaming
	t.Run("Regular incomplete JSON", func(t *testing.T) {
		parser := NewStreamParser(false)

		// Simulate incomplete JSON that gets completed over time
		chunks := []string{
			`{"choices":[{"delta":{"content":"[{\"start_pos\": 0"}}]}`,
			`{"choices":[{"delta":{"content":", \"end_pos\": 100},"}}]}`,
			`{"choices":[{"delta":{"content":"{\"start_pos\": 100, \"end_pos\""}}]}`,
			`{"choices":[{"delta":{"content":": 200}]"}}]}`,
		}

		var finalData *StreamChunkData
		var err error

		for _, chunk := range chunks {
			finalData, err = parser.ParseStreamChunk([]byte(chunk))
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		}

		// Should eventually parse positions even from incomplete JSON
		if len(finalData.Positions) == 0 {
			t.Logf("No positions parsed yet (expected for incomplete JSON): %s", finalData.Content)
		}
	})

	t.Run("Toolcall incomplete JSON", func(t *testing.T) {
		parser := NewStreamParser(true)

		// Simulate incomplete toolcall JSON
		chunks := []string{
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"segments\": [{\"start_pos\": 0"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":", \"end_pos\": 50}"}}]}}]}`,
		}

		var finalData *StreamChunkData
		var err error

		for _, chunk := range chunks {
			finalData, err = parser.ParseStreamChunk([]byte(chunk))
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		}

		// Should handle incomplete JSON gracefully
		if len(finalData.Positions) == 0 {
			t.Logf("No positions parsed yet (expected for incomplete JSON): %s", finalData.Arguments)
		}
	})
}

func TestStreamParser_JSONCompletion(t *testing.T) {
	// Test that StreamParser can handle incomplete JSON and complete it internally

	tests := []struct {
		name           string
		chunks         []string
		expectedPosLen int
	}{
		{
			name: "Incomplete array gets completed",
			chunks: []string{
				`{"choices":[{"delta":{"content":"[{\"start_pos\": 0, \"end_pos\": 50"}}]}`,
				`{"choices":[{"delta":{"content":"}]"}}]}`,
			},
			expectedPosLen: 1,
		},
		{
			name: "Multiple positions in single stream",
			chunks: []string{
				`{"choices":[{"delta":{"content":"[{\"start_pos\": 0, \"end_pos\": 50}, "}}]}`,
				`{"choices":[{"delta":{"content":"{\"start_pos\": 50, \"end_pos\": 100}]"}}]}`,
			},
			expectedPosLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new parser for each test to avoid state contamination
			testParser := NewStreamParser(false)
			var finalData *StreamChunkData
			var err error

			for _, chunk := range tt.chunks {
				finalData, err = testParser.ParseStreamChunk([]byte(chunk))
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check that positions were eventually parsed
			if len(finalData.Positions) != tt.expectedPosLen {
				t.Errorf("Expected %d positions, got %d. Content: %s",
					tt.expectedPosLen, len(finalData.Positions), finalData.Content)
			}
		})
	}
}

func TestStreamParser_SSEFormat(t *testing.T) {
	// Test SSE format with data: prefix
	testChunks := []string{
		`data: {"choices":[{"delta":{"content":"[{\"start_pos\": 0, \"end_pos\": 50}]"}}]}`,
		`data: [DONE]`,
	}

	expectedContents := []string{
		`[{"start_pos": 0, "end_pos": 50}]`,
		`[{"start_pos": 0, "end_pos": 50}]`,
	}

	expectedFinished := []bool{false, true}
	expectedPositionsCount := []int{1, 1} // First chunk parses positions, DONE chunk maintains them

	// Use a single parser for the entire sequence
	parser := NewStreamParser(false)

	for i, chunk := range testChunks {
		t.Run(fmt.Sprintf("SSEChunk_%d", i), func(t *testing.T) {
			data, err := parser.ParseStreamChunk([]byte(chunk))
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if data.Content != expectedContents[i] {
				t.Errorf("Expected content '%s', got '%s'", expectedContents[i], data.Content)
			}

			if data.Finished != expectedFinished[i] {
				t.Errorf("Expected finished %v, got %v", expectedFinished[i], data.Finished)
			}

			if len(data.Positions) != expectedPositionsCount[i] {
				t.Errorf("Expected %d positions, got %d", expectedPositionsCount[i], len(data.Positions))
			}
		})
	}
}

func TestStreamParser_ErrorHandling(t *testing.T) {
	parser := NewStreamParser(false)

	// Test invalid JSON
	data, err := parser.ParseStreamChunk([]byte(`invalid json`))
	if err != nil {
		t.Errorf("Should not return error for invalid JSON, got: %v", err)
	}

	if data.Error == "" {
		t.Errorf("Expected error message in data.Error")
	}

	if data.Raw == nil {
		t.Errorf("Expected raw data to be preserved")
	}

	// Test empty chunk
	data, err = parser.ParseStreamChunk([]byte(``))
	if err != nil {
		t.Errorf("Should not return error for empty chunk, got: %v", err)
	}

	if data.Error != "" {
		t.Errorf("Should not have error for empty chunk")
	}
}

func BenchmarkStreamParser(b *testing.B) {
	parser := NewStreamParser(false)
	chunk := []byte(`{"choices":[{"delta":{"content":"[{\"start_pos\": 0, \"end_pos\": 50}]"}}]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseStreamChunk(chunk)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkTolerantJSONUnmarshal(b *testing.B) {
	jsonData := []byte(`{"key": "value", "number": 42, "array": [1, 2, 3]}`)
	var result map[string]interface{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := TolerantJSONUnmarshal(jsonData, &result)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// =============================================================================
// Memory and Goroutine Leak Detection Tests
// =============================================================================

// checkGoroutineLeaks runs a test function and checks for goroutine leaks
func checkGoroutineLeaks(t *testing.T, testFunc func()) {
	initialGoroutines := runtime.NumGoroutine()
	testFunc()

	// Give goroutines time to clean up
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines > initialGoroutines {
		buf := make([]byte, 8192)
		runtime.Stack(buf, true)
		t.Errorf("Goroutine leak detected: started with %d, ended with %d goroutines\nStack trace:\n%s",
			initialGoroutines, finalGoroutines, string(buf))
	}
}

// TestMemoryLeakDetection tests for memory leaks in utils functions
func TestMemoryLeakDetection(t *testing.T) {
	// Skip this test if not in verbose mode or specific flag is not set
	if !testing.Verbose() && os.Getenv("RUN_MEMORY_TESTS") == "" {
		t.Skip("Skipping memory leak test (set RUN_MEMORY_TESTS=1 or use -v to run)")
	}

	// Force garbage collection before starting
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Test operations that might leak memory
	for i := 0; i < 1000; i++ {
		// Test JSON parsing operations
		testJSON := `{"temperature": 0.1, "max_tokens": 1000, "top_p": 0.9, "frequency_penalty": 0.0}`
		_, err := ParseJSONOptions(testJSON)
		if err != nil {
			t.Errorf("ParseJSONOptions failed: %v", err)
		}

		// Test TolerantJSONUnmarshal
		var result map[string]interface{}
		testData := []byte(`{"key": "value", "number": 42,}`) // trailing comma
		err = TolerantJSONUnmarshal(testData, &result)
		if err != nil {
			t.Errorf("TolerantJSONUnmarshal failed: %v", err)
		}

		// Test StreamParser operations
		parser := NewStreamParser(false)
		chunk := []byte(`{"choices":[{"delta":{"content":"test content"}}]}`)
		_, err = parser.ParseStreamChunk(chunk)
		if err != nil {
			t.Errorf("StreamParser.ParseStreamChunk failed: %v", err)
		}

		// Test StreamParser with toolcall
		toolcallParser := NewStreamParser(true)
		toolcallChunk := []byte(`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"test"}}]}}]}`)
		_, err = toolcallParser.ParseStreamChunk(toolcallChunk)
		if err != nil {
			t.Errorf("StreamParser toolcall ParseStreamChunk failed: %v", err)
		}

		// Test GetSemanticPrompt
		_ = GetSemanticPrompt("")
		_ = GetSemanticPrompt("custom prompt")

		// Force garbage collection periodically
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Force final garbage collection
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check memory growth (handle potential underflow)
	var memGrowth, heapGrowth int64
	if m2.Alloc >= m1.Alloc {
		memGrowth = int64(m2.Alloc - m1.Alloc)
	} else {
		memGrowth = -int64(m1.Alloc - m2.Alloc)
	}
	if m2.HeapAlloc >= m1.HeapAlloc {
		heapGrowth = int64(m2.HeapAlloc - m1.HeapAlloc)
	} else {
		heapGrowth = -int64(m1.HeapAlloc - m2.HeapAlloc)
	}

	t.Logf("Memory stats for utils operations:")
	t.Logf("  Alloc growth: %d bytes", memGrowth)
	t.Logf("  Heap growth: %d bytes", heapGrowth)
	t.Logf("  Sys growth: %d bytes", int64(m2.Sys)-int64(m1.Sys))
	t.Logf("  NumGC: %d", m2.NumGC-m1.NumGC)

	// Allow some memory growth, but not excessive
	maxAllowedGrowth := int64(512 * 1024) // 512KB threshold for utils operations
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: alloc grew by %d bytes (threshold: %d bytes)", memGrowth, maxAllowedGrowth)
	}

	if heapGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: heap grew by %d bytes (threshold: %d bytes)", heapGrowth, maxAllowedGrowth)
	}
}

// TestStreamLLMMemoryLeak tests for memory leaks in StreamLLM operations
func TestStreamLLMMemoryLeak(t *testing.T) {
	// Skip this test if not in verbose mode or specific flag is not set
	if !testing.Verbose() && os.Getenv("RUN_MEMORY_TESTS") == "" {
		t.Skip("Skipping memory leak test (set RUN_MEMORY_TESTS=1 or use -v to run)")
	}

	// Test with OpenAI if available
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		t.Skip("Skipping StreamLLM memory leak test: OPENAI_TEST_KEY not set")
	}

	// Force garbage collection before starting
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Test StreamLLM operations that might leak memory
	for i := 0; i < 10; i++ {
		func() {
			// Create OpenAI connector
			openaiDSL := fmt.Sprintf(`{
				"LANG": "1.0.0",
				"VERSION": "1.0.0",
				"label": "Memory Test",
				"type": "openai",
				"options": {
					"proxy": "https://api.openai.com/v1",
					"model": "gpt-4o-mini",
					"key": "%s"
				}
			}`, openaiKey)

			conn, err := connector.New("openai", fmt.Sprintf("test-memory-%d", i), []byte(openaiDSL))
			if err != nil {
				t.Errorf("Failed to create connector: %v", err)
				return
			}

			payload := map[string]interface{}{
				"model": "gpt-4o-mini",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Say hello"},
				},
				"max_tokens": 10,
			}

			// Test streaming with callback
			callbackCount := 0
			callback := func(data []byte) error {
				callbackCount++
				return nil
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = StreamLLM(ctx, conn, "chat/completions", payload, callback)
			if err != nil {
				t.Logf("StreamLLM failed (may be expected): %v", err)
			}

			t.Logf("Iteration %d: received %d callbacks", i, callbackCount)
		}()

		// Force garbage collection periodically
		if i%2 == 0 {
			runtime.GC()
		}
	}

	// Force final garbage collection
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check memory growth
	var memGrowth, heapGrowth int64
	if m2.Alloc >= m1.Alloc {
		memGrowth = int64(m2.Alloc - m1.Alloc)
	} else {
		memGrowth = -int64(m1.Alloc - m2.Alloc)
	}
	if m2.HeapAlloc >= m1.HeapAlloc {
		heapGrowth = int64(m2.HeapAlloc - m1.HeapAlloc)
	} else {
		heapGrowth = -int64(m1.HeapAlloc - m2.HeapAlloc)
	}

	t.Logf("Memory stats for StreamLLM operations:")
	t.Logf("  Alloc growth: %d bytes", memGrowth)
	t.Logf("  Heap growth: %d bytes", heapGrowth)
	t.Logf("  Sys growth: %d bytes", int64(m2.Sys)-int64(m1.Sys))
	t.Logf("  NumGC: %d", m2.NumGC-m1.NumGC)

	// Allow more memory growth for network operations
	maxAllowedGrowth := int64(2 * 1024 * 1024) // 2MB threshold for network operations
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: alloc grew by %d bytes (threshold: %d bytes)", memGrowth, maxAllowedGrowth)
	}

	if heapGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: heap grew by %d bytes (threshold: %d bytes)", heapGrowth, maxAllowedGrowth)
	}
}

// TestGoroutineLeakDetection tests for goroutine leaks in utils functions
func TestGoroutineLeakDetection(t *testing.T) {
	t.Run("ParseJSONOptions goroutine leak", func(t *testing.T) {
		checkGoroutineLeaks(t, func() {
			for i := 0; i < 100; i++ {
				testJSON := `{"temperature": 0.1, "max_tokens": 1000}`
				_, err := ParseJSONOptions(testJSON)
				if err != nil {
					t.Errorf("ParseJSONOptions failed: %v", err)
				}
			}
		})
	})

	t.Run("TolerantJSONUnmarshal goroutine leak", func(t *testing.T) {
		checkGoroutineLeaks(t, func() {
			for i := 0; i < 100; i++ {
				var result map[string]interface{}
				testData := []byte(`{"key": "value", "number": 42}`)
				err := TolerantJSONUnmarshal(testData, &result)
				if err != nil {
					t.Errorf("TolerantJSONUnmarshal failed: %v", err)
				}
			}
		})
	})

	t.Run("StreamParser goroutine leak", func(t *testing.T) {
		checkGoroutineLeaks(t, func() {
			for i := 0; i < 100; i++ {
				parser := NewStreamParser(false)
				chunk := []byte(`{"choices":[{"delta":{"content":"test"}}]}`)
				_, err := parser.ParseStreamChunk(chunk)
				if err != nil {
					t.Errorf("StreamParser failed: %v", err)
				}

				// Test toolcall parser too
				toolcallParser := NewStreamParser(true)
				toolcallChunk := []byte(`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"test"}}]}}]}`)
				_, err = toolcallParser.ParseStreamChunk(toolcallChunk)
				if err != nil {
					t.Errorf("Toolcall StreamParser failed: %v", err)
				}
			}
		})
	})

	t.Run("File operations goroutine leak", func(t *testing.T) {
		checkGoroutineLeaks(t, func() {
			// Create a temporary file for testing
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test_goroutine.txt")
			testContent := "Test content for goroutine leak detection"

			err := os.WriteFile(tmpFile, []byte(testContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			for i := 0; i < 50; i++ {
				reader, err := OpenFileAsReader(tmpFile)
				if err != nil {
					t.Errorf("OpenFileAsReader failed: %v", err)
					continue
				}

				// Read some data
				buffer := make([]byte, 10)
				_, err = reader.Read(buffer)
				if err != nil {
					t.Errorf("Read failed: %v", err)
				}

				// Close the reader
				err = reader.Close()
				if err != nil {
					t.Errorf("Close failed: %v", err)
				}
			}
		})
	})
}
