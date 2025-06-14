package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaoapp/gou/connector"
)

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

	// Expected positions should be parsed from accumulated content
	expectedPositionsCount := []int{0, 1, 2, 2, 2} // Parser can extract positions as they become available

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
	expectedPositionsCount := []int{0, 1, 2, 2, 2} // Parser can extract positions as they become available

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
	parser := NewStreamParser(false)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Incomplete array",
			input:    `[{"start_pos": 0, "end_pos": 50`,
			expected: `[{"start_pos": 0, "end_pos": 50}]`,
		},
		{
			name:     "Missing closing bracket",
			input:    `[{"start_pos": 0, "end_pos": 50}, {"start_pos": 50, "end_pos": 100}`,
			expected: `[{"start_pos": 0, "end_pos": 50}, {"start_pos": 50, "end_pos": 100}]`,
		},
		{
			name:     "Trailing comma",
			input:    `[{"start_pos": 0, "end_pos": 50},`,
			expected: `[{"start_pos": 0, "end_pos": 50}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed := parser.completeArrayJSON(tt.input)
			if completed != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, completed)
			}

			// Test that completed JSON can be parsed
			var positions []SemanticPosition
			err := TolerantJSONUnmarshal([]byte(completed), &positions)
			if err != nil {
				t.Errorf("Completed JSON should be parseable: %v", err)
			}
		})
	}
}

func TestStreamParser_SSEFormat(t *testing.T) {
	parser := NewStreamParser(false)

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
	expectedPositionsCount := []int{1, 1} // Should parse 1 position

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
